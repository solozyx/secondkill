package service

import (
	"encoding/json"
	"fmt"
	"secondkill/proxy/auth"
	"strconv"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/garyburd/redigo/redis"

	"secondkill/comm/config/constant"
	"secondkill/comm/model/goods"
	"secondkill/comm/model/request"
	"secondkill/comm/model/response"

	"secondkill/proxy/conf"
	"secondkill/proxy/dao"
	"secondkill/proxy/envinit"
	"secondkill/proxy/limite"
)

var (
	G_proxyService *proxyService
)

type proxyService struct {
	// waitGroup *sync.WaitGroup
	secReqChan chan *request.SecRequest
}

func InitProxyService() (err error) {
	G_proxyService = &proxyService{}
	// G_proxyService.waitGroup = &sync.WaitGroup{}
	G_proxyService.secReqChan = make(chan *request.SecRequest,
		conf.G_proxyConf.ChanSizeWriteProxyRequest)
	if err = G_proxyService.loadBlackList(); err != nil {
		logs.Warn("seckill proxy load userid and IP blacklist err: ", err)
		return
	}
	G_proxyService.run()
	logs.Debug("seckill proxy init ProxyService success")
	return
}

// GET / POST http://127.0.0.1:9091/secinfo?goods_id=0000
func (s *proxyService) SecInfo(goodsId int) (goodsInfo map[string]interface{}, statusCode int, err error) {
	return s.goodsInfoById(goodsId)
}

// GET / POST http://127.0.0.1:9091/seclist
func (s *proxyService) SecList() (goodsList []map[string]interface{}, err error) {
	var (
		goodsInfo map[string]interface{}
	)
	goodsList = make([]map[string]interface{}, 0)
	goodsInfo = make(map[string]interface{})
	// TODO dao包的读写锁应该公开么 RWLock ?
	dao.G_goodsDao.RWLock.RLock()
	defer dao.G_goodsDao.RWLock.RUnlock()
	for _, aGoods := range dao.G_goodsDao.GoodsMap {
		if goodsInfo, _, err = s.goodsInfoById(aGoods.GoodsId); err != nil {
			logs.Error("seckill proxy get goods[%d] from etcd err: %v", aGoods.GoodsId, err)
			continue
		}
		goodsList = append(goodsList, goodsInfo)
	}
	return
}

func (s *proxyService) goodsInfoById(goodsId int) (goodsInfo map[string]interface{}, statusCode int, err error) {
	var (
		aGoods      *goods.Goods
		ok          bool
		now         int64
		isStart     bool
		isEnd       bool
		goodsStatus string
	)
	goodsInfo = make(map[string]interface{})

	// TODO dao包的读写锁应该公开么 RWLock ?
	dao.G_goodsDao.RWLock.RLock()
	if aGoods, ok = dao.G_goodsDao.GoodsMap[goodsId]; !ok {
		statusCode = constant.ErrNotFoundGoodsId
		err = fmt.Errorf("not found goods_id")
		return
	}
	dao.G_goodsDao.RWLock.RUnlock()

	goodsInfo["goods_id"] = goodsId
	// 把1个秒杀商品的 开始 结束 时间返回给客户端 问题?
	// 客户端 和 服务端 时间不一样 时钟不同步
	// 造成有些客户端秒杀开始 有些客户端还没有开始
	// goodsInfo["start_time"] = goods.StartTime
	// goodsInfo["end_time"] = goods.EndTime
	//
	// 秒杀商品开始时间以服务端时间为准
	goodsStatus = "success"

	now = time.Now().Unix()
	if now-aGoods.StartTime < 0 {
		isStart = false
		isEnd = false
		goodsStatus = "the goods seckill in not start :( "
		statusCode = constant.ErrActivityNotStart
	}
	if now-aGoods.StartTime > 0 {
		isStart = true
		isEnd = false
	}
	if now-aGoods.EndTime > 0 {
		isStart = false
		isEnd = true
		goodsStatus = "the goods seckill is end :( "
		statusCode = constant.ErrActivityAlreadyEnd
	}
	if aGoods.Status == constant.GoodsStatusForceSaleOut || aGoods.Status == constant.GoodsStatusSaleOut {
		isStart = false
		isEnd = true
		goodsStatus = "goods is sale out"
		statusCode = constant.ErrActivitySaleOut
	}

	goodsInfo["start"] = isStart
	goodsInfo["end"] = isEnd

	// 返回客户端商品状态 不要返回商品数量 会出危机
	// 某一天搞1元秒杀 0件商品 大家抢购 一个也没抢到 耍猴
	// 服务端的数据都是保密的 策略都很容易被人猜到
	goodsInfo["status"] = goodsStatus // goods.Status
	return
}

// 加载 UserBlackList IPBlackList 黑名单
func (s *proxyService) loadBlackList() (err error) {
	var (
		conn       redis.Conn
		reply      interface{}
		userIdList []string
		userIdStr  string
		userId     int
		ipList     []string
		ipStr      string
	)
	conn = envinit.G_envInit.RedisBlacklistPool.Get()
	defer conn.Close()

	// userId blacklist
	if reply, err = conn.Do("hgetall", constant.RedisQueueUserIdBlacklist); err != nil {
		logs.Warn("redis load userIdBlackList fail err: ", err)
	}
	if userIdList, err = redis.Strings(reply, err); err != nil {
		return
	}
	for _, userIdStr = range userIdList {
		if userId, err = strconv.Atoi(userIdStr); err != nil {
			logs.Warn("redis invalid userId err: ", err)
			continue
		}
		conf.G_proxyConf.UserBlackList.Store(userId, true)
	}

	// IP blacklist
	if reply, err = conn.Do("hgetall", constant.RedisQueueIpBlacklist); err != nil {
		logs.Warn("redis load clientIpBlackList fail err: ", err)
	}
	if ipList, err = redis.Strings(reply, err); err != nil {
		return
	}
	for _, ipStr = range ipList {
		conf.G_proxyConf.IPBlackList.Store(ipStr, true)
	}

	// 子协程 同步 idblacklist ipblacklist 黑名单数据
	go s.syncIPBlackList()
	go s.syncUserIdBlackList()
	return
}

func (s *proxyService) syncIPBlackList() {
	var (
		conn  redis.Conn
		reply interface{}
		err   error
		ipStr string
	)
	conn = envinit.G_envInit.RedisBlacklistPool.Get()
	defer conn.Close()
	for {
		// 黑名单数据存储到队列 用的时候POP
		// BLPOP 操作是阻塞的 队列没有数据元素 没有新增数据元素 会阻塞在那里
		// 有超时时间，如果队列没有数据元素会阻塞 直到超时
		// 阻塞在那里比较实时，一有元素就能获取到，没有元素就阻塞等待
		// 如果每隔5秒钟取1次，新的黑名单可能就加入了，实时获取到，所以用阻塞操作
		reply, err = conn.Do("BLPOP", constant.RedisQueueIpBlacklist, time.Second)
		if ipStr, err = redis.String(reply, err); err != nil {
			continue
		}
		// TODO 优化 用普通map 读写锁 缓存ipStr 到一定数量 加一次锁写入普通map
		conf.G_proxyConf.IPBlackList.Store(ipStr, true)
	}
}

func (s *proxyService) syncUserIdBlackList() {
	var (
		conn   redis.Conn
		reply  interface{}
		err    error
		userId int
	)
	conn = envinit.G_envInit.RedisBlacklistPool.Get()
	defer conn.Close()
	for {
		reply, err = conn.Do("BLPOP", constant.RedisQueueUserIdBlacklist, time.Second)
		if userId, err = redis.Int(reply, err); err != nil {
			continue
		}
		// TODO 优化 用普通map 读写锁 缓存userId 到一定数量 加一次锁写入普通map
		conf.G_proxyConf.UserBlackList.Store(userId, true)
	}
}

// GET / POST http://127.0.0.1:9091/seckill?goods_id=0000&source=ios&authcode=xxxx&nance=xxxx
func (s *proxyService) SecKill(secRequest *request.SecRequest) (seckillInfo map[string]interface{}, statusCode int, err error) {
	var (
		isUserValid         bool
		userSeckillGoodsKey string
	)
	seckillInfo = make(map[string]interface{})
	// TODO dao包的读写锁应该公开么 RWLock ?
	dao.G_goodsDao.RWLock.RLock()
	defer dao.G_goodsDao.RWLock.RUnlock()
	//TODO NOTICE 简化测试去掉用户cookie校验 生产环境请接入商城系统获取cookie
	// 用户从商城系统 登录 进入秒杀系统 用户鉴权
	if isUserValid = auth.UserCheck(secRequest); !isUserValid {
		statusCode = constant.ErrUserAuthCheckFailed
		err = fmt.Errorf("invalid user cookie auth")
		return
	}

	if err = limite.G_secondLimitMgr.UserAndIpAccessSecondCountCheck(secRequest); err != nil {
		statusCode = constant.ErrUserServiceBusy
		return
	}

	if _, statusCode, err = s.goodsInfoById(secRequest.GoodsId); err != nil {
		return
	}
	if statusCode != constant.GoodsStatusNormal {
		logs.Warn("SeckillProxy goods_id Un Normal ")
		return
	}

	userSeckillGoodsKey = fmt.Sprintf("%d_%d", secRequest.UserId, secRequest.GoodsId)

	// 秒杀接入层 把能参与秒杀的请求 放到redis队列
	// redis做了一个黑名单实例 再做1个redis实例
	s.secReqChan <- secRequest
	//...
	return
}

func (s *proxyService) run() {
	// redis write ProxyService SecReqChan into redis
	for i := 0; i < conf.G_proxyConf.GoroutineNumWriteProxyRequestToRedis; i++ {
		// s.waitGroup.Add(1)
		go s.writeProxySecRequestToRedis(i)
	}
	logs.Debug("seckill proxy run all goroutine start ... ")
	// s.waitGroup.Wait()
	time.Sleep(10 * time.Second)
	logs.Debug("seckill proxy run all goroutine end ")
}

func (s *proxyService) writeProxySecRequestToRedis(goroutineNo int) {
	var (
		secReq *request.SecRequest
		conn   redis.Conn
		err    error
		data   []byte
	)
	logs.Debug("seckill proxy goroutine starting 1 writeProxySecRequestToRedis gNo = %d", goroutineNo)

	conn = envinit.G_envInit.RedisProxyToLogicPool.Get()
	defer conn.Close()

	for {
		if conn == nil {
			conn = envinit.G_envInit.RedisProxyToLogicPool.Get()
		}

		secReq = <-s.secReqChan
		if data, err = json.Marshal(secReq); err != nil {
			logs.Error("seckill proxy json marshal secRequest err: ", err)
			continue
		}
		if _, err = conn.Do("LPUSH", constant.RedisQueueSecReq, string(data)); err != nil {
			logs.Error("seckill proxy Redis LPUSH err: ", err)
			conn.Close()
			continue
		}
	}
}
