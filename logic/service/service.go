package service

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/garyburd/redigo/redis"

	"secondkill/comm/config/constant"
	"secondkill/comm/model/goods"
	"secondkill/comm/model/request"
	"secondkill/comm/model/response"
	"secondkill/logic/conf"
	"secondkill/logic/dao"
	"secondkill/logic/envinit"
	"secondkill/logic/user"
)

var (
	G_logicService *logicService
)

type logicService struct {
	// waitGroup *sync.WaitGroup
	// 秒杀接入层把SecRequest序列化写入redis队列 秒杀逻辑层读取SecRequest到chan
	secReqChan chan *request.SecRequest
	// 秒杀逻辑层处理SecRequest得到SecResponse 把SecResponse投递到chan
	secRespChan chan *response.SecResponse

	// 所有用户参与秒杀成功购买商品列表
	// 1个用户 user_id 可以购买1个商品 goods_id 若干个 count
	// userBuyGoodsHistory = map [user_id(int)] *sync.Map [goods_id(int)] 购买数量count(int)
	userBuyGoodsHistory     map[int]*user.UserBuyHistroy
	userBuyGoodsHistoryLock *sync.RWMutex
}

func InitLogicService() (err error) {
	G_logicService = &logicService{}
	// G_logicService.waitGroup = &sync.WaitGroup{}
	G_logicService.secReqChan = make(chan *request.SecRequest, conf.G_logicConf.ChanSizeReadProxyRequest)
	G_logicService.secRespChan = make(chan *response.SecResponse, conf.G_logicConf.ChanSizeWriteLogicResponse)
	G_logicService.userBuyGoodsHistory = make(map[int]*user.UserBuyHistroy, 100000)
	G_logicService.userBuyGoodsHistoryLock = &sync.RWMutex{}
	G_logicService.run()
	logs.Debug("seckill logic init LogicService success")
	return
}

func (s *logicService) run() {
	// redis read ProxyService SecReqChan put into redis
	for i := 0; i < conf.G_logicConf.GoroutineNumReadProxyRequestFromRedis; i++ {
		// logic.waitGroup.Add(1)
		go s.readProxySecRequestFromRedis(i)
	}
	// logic service handle Seckill SecRequest
	for i := 0; i < conf.G_logicConf.GoroutineNumSeckillHandle; i++ {
		// logic.waitGroup.Add(1)
		go s.handleSecRequest(i)
	}
	// redis write LogicService SecResponse
	for i := 0; i < conf.G_logicConf.GoroutineNumWriteLogicResponseToRedis; i++ {
		// logic.waitGroup.Add(1)
		go s.writeSecResponseToRedis(i)
	}
	logs.Debug("seckill logic run all goroutine start ... ")
	// logic.waitGroup.Wait()
	// 主协程等待10秒 启动48个子协程
	time.Sleep(time.Second * 10)
	logs.Debug("seckill logic run all goroutine end ")
}

// 秒杀接入层把 SecRequest 写入redis队列
// 秒杀逻辑层第1组协程 从redis队列读取 SecRequest到 secReqChan
func (s *logicService) readProxySecRequestFromRedis(goroutineNo int) {
	var (
		conn            redis.Conn
		reply           interface{} // string
		redisBlpopValue []interface{}
		ok              bool
		data            []byte
		err             error
		secReq          request.SecRequest
		ticker          *time.Ticker
	)
	logs.Debug("seckill logic goroutine starting 1 readProxySecRequestFromRedis gNo = ", goroutineNo)

	conn = envinit.G_envInit.RedisProxyToLogicPool.Get()
	defer conn.Close()

	for {
		if conn == nil {
			conn = envinit.G_envInit.RedisProxyToLogicPool.Get()
		}
		for {
			// 传入1个超时时间 阻塞10秒 如果队列没有数据 10秒就会返回
			// 阻塞秒数 timeout传0表示无限阻塞 只要队列无数据该协程就一直阻塞在这里 不占资源
			// TODO ERROR - redigo: unexpected type for String, got type []interface {}
			// 执行 BLPOP 返回数组 数组[0] 队列名称 数组[1]真实的POP数据
			if reply, err = conn.Do("BLPOP", constant.RedisQueueSecReq, 0); err != nil {
				logs.Error("seckill logic redis conn redis ProxySecRequest err: ", err)
				break
			}
			if redisBlpopValue, ok = reply.([]interface{}); !ok || len(redisBlpopValue) != 2 {
				logs.Error("seckill logic redis BLPOP redis ProxySecRequest value []interface{} err ")
				continue
			}
			if data, ok = redisBlpopValue[1].([]byte); !ok {
				logs.Error("seckill logic redis BLPOP redis ProxySecRequest value []byte err ")
				continue
			}
			logs.Debug("seckill logic redis ProxySecRequest success SecRequest = ", string(data))

			if err = json.Unmarshal(data, &secReq); err != nil {
				logs.Error("SeckillLogic json Unmarshal SecReq err: ", err)
				continue
			}
			// TODO NOTICE 过载保护
			// 避免处理过期请求 新的请求一直在队列末尾 一直处理过期请求 导致新请求又过期
			// SecRequest.AccessTime 每个秒杀请求携带时间 如果后台处理不过来了 请求会堆积
			// 设置客户端最大等待时间是 10秒-15秒 根据实际情况定
			// 请求过期的话 直接扔掉 因为接入层与客户端的连接已经关闭了
			if time.Now().Unix()-secReq.AccessTime.Unix() >= int64(conf.G_logicConf.TimeoutProxyMaxAlive) {
				logs.Warn("seckill logic SecRequest is expired ")
				continue
			}

			// logic.secReqChan <- &secReq
			ticker = time.NewTicker(time.Millisecond * time.Duration(conf.G_logicConf.TimeoutLogicSecReqChan)) // 100毫秒
			select {
			case s.secReqChan <- &secReq:
			case <-ticker.C:
				logs.Warn("seckill logic warning SecRequest chan 100000size full ")
				break
			}
		}
	}
}

// 秒杀逻辑层第2组协程 消费secReqChan中的SecRequest请求
// 把处理SecRequest得到的对应的SecResponse投递到 secRespChan
func (s *logicService) handleSecRequest(goroutineNo int) {
	var (
		secReq  *request.SecRequest
		secResp *response.SecResponse
		err     error
		ticker  *time.Ticker
	)
	logs.Debug("seckill logic goroutine starting 2 handleSeckillRequest gNo = ", goroutineNo)

	for secReq = range s.secReqChan {
		logs.Debug("seckill logic begin process SecRequest ...")
		if secResp, err = s.seckill(secReq); err != nil {
			logs.Warn("seckill logic handle SecRequest err: ", err)
			secResp = &response.SecResponse{
				Code: constant.ErrUserServiceBusy,
			}
		}
		// channel满了继续写入会阻塞
		// s.secRespChan <- secResp
		// secResp 处理不过来 又会影响到 secReqChan 的投递
		// 1个channel堆积导致 另外1个channel堆积
		// 在线业务 要考虑超时 不能让用户无限等待 量大了堆积 一直消耗不完导致性能恶化
		// channel满了有堆积迹象 就主动把一些请求丢弃
		ticker = time.NewTicker(time.Millisecond * time.Duration(conf.G_logicConf.TimeoutLogicSecRespChan)) // 100毫秒
		select {
		case s.secRespChan <- secResp:
			logs.Debug("seckill logic writing SecResponse to s.secRespChan size(s.secRespChan) = %d", len(s.secRespChan))
		case <-ticker.C:
			// 100毫秒 secRespChan 没有投递 secResp 超时 丢弃该请求对应的响应
			// channel size设定的是 10万 如果达到10万 说明很多请求堆积了
			// 要快速清空channel 即时处理后续秒杀请求
			logs.Warn("seckill logic writing SecResponse chan 100000size full ")
			break
		}
	}
}

// 秒杀逻辑层第3组协程
// 把secRespChan存放的处理SecRequest得到的对应的SecResponse 写入redis队列
func (s *logicService) writeSecResponseToRedis(goroutineNo int) {
	var (
		secResp *response.SecResponse
		err     error
		data    []byte
		conn    redis.Conn
	)
	logs.Debug("seckill logic goroutine starting 3 writeSecResponseToRedis gNo = ", goroutineNo)

	conn = envinit.G_envInit.RedisLogicToProxyPool.Get()
	defer conn.Close()

	for secResp = range s.secRespChan {
		if conn == nil {
			conn = envinit.G_envInit.RedisLogicToProxyPool.Get()
		}
		if data, err = json.Marshal(secResp); err != nil {
			logs.Error("seckill logic json Marshal SecResp err: ", err)
			continue
		}
		// if _,err = redis.String(conn.Do("RPUSH",constant.RedisQueueSecResp,string(data))); err != nil{
		if _, err = conn.Do("RPUSH", constant.RedisQueueSecResp, string(data)); err != nil {
			logs.Error("seckill logic redis RPUSH LogicService SecResponse err: ", err)
			conn.Close()
			continue
		}
		logs.Debug("seckill logic redis write SecResponse success SecResponse = ", *secResp)
	}
}

// 秒杀业务具体逻辑处理
func (s *logicService) seckill(secReq *request.SecRequest) (secResp *response.SecResponse, err error) {
	var (
		aGoods                               *goods.Goods
		ok                                   bool
		aGoodsCurrentSecondAlreadySaledCount int
		userBuyHistory                       *user.UserBuyHistroy
		aUserAlreadyBuyAgoodsCount           int
		aGoodsAlreadySaledTotalCount         int
		buyProbability                       float64
		now                                  int64
	)
	secResp = &response.SecResponse{}
	secResp.UserId = secReq.UserId
	secResp.GoodsId = secReq.GoodsId

	// 加读锁
	dao.G_goodsDao.RWLock.RLock()
	// 用户抢购商品是否存在
	if aGoods, ok = dao.G_goodsDao.GoodsMap[secReq.GoodsId]; !ok {
		logs.Error("seckill logic seckill not found goods : ", secReq.GoodsId)
		secResp.Code = constant.ErrNotFoundGoodsId
		return
	}
	dao.G_goodsDao.RWLock.RUnlock()

	// 该秒杀商品状态
	// 是否售罄
	if aGoods.Status == constant.GoodsStatusSaleOut {
		secResp.Code = constant.ErrActivitySaleOut
		return
	}

	// 是否超速
	// 当前这1秒已经卖出该商品多少个
	aGoodsCurrentSecondAlreadySaledCount = aGoods.SecondLimit.Check(time.Now().Unix())
	if aGoodsCurrentSecondAlreadySaledCount >= aGoods.SecondSaleMaxLimit {
		// 这1秒没有资格 下一秒再来
		secResp.Code = constant.ErrActivityRetry
		return
	}

	// 是否已经购买
	// 加读锁
	s.userBuyGoodsHistoryLock.Lock()
	if userBuyHistory, ok = s.userBuyGoodsHistory[secReq.UserId]; !ok {
		userBuyHistory = &user.UserBuyHistroy{
			BuyHistory: &sync.Map{},
		}
		s.userBuyGoodsHistory[secReq.UserId] = userBuyHistory
	}
	aUserAlreadyBuyAgoodsCount = userBuyHistory.Count(aGoods.GoodsId)
	s.userBuyGoodsHistoryLock.Unlock()

	if aUserAlreadyBuyAgoodsCount >= aGoods.OneUserBuyLimit {
		secResp.Code = constant.ErrActivityAlreadyBuy
		return
	}

	// 是否总数超限
	aGoodsAlreadySaledTotalCount = dao.G_goodsDao.CountGoodsSaledById(aGoods.GoodsId)
	if aGoodsAlreadySaledTotalCount >= aGoods.Total {
		secResp.Code = constant.ErrActivitySaleOut
		aGoods.Status = constant.GoodsStatusSaleOut
		return
	}

	// 用户等级 用户黑名单

	// 随机抽奖 生成1个[0-1]随机数 比如0.8的概率买到 <0.8就买到 >=0.8就买不到
	if buyProbability = rand.Float64(); buyProbability >= aGoods.BuyProbability {
		secResp.Code = constant.ErrActivityRetry
		return
	}

	// 走到这里 用户才可能秒杀成功
	// 更新总数 商品的卖出数量 + 1
	userBuyHistory.Add(aGoods.GoodsId, 1)
	dao.G_goodsDao.AddGoodsSaledById(aGoods.GoodsId, 1)

	// 用户秒杀成功 更新秒杀速度 ...
	secResp.Code = constant.SuccessActivityCode

	// 生成token
	now = time.Now().Unix()
	secResp.Token = s.tokenGenerate(secReq.UserId, secReq.GoodsId, now, conf.G_logicConf.TokenPasswd)
	secResp.TokenTime = now

	return
}

// userId&goodsId&nowTimeStamp判断是否过期&密钥种子
func (s *logicService) tokenGenerate(userId, goodsId int, timestamp int64, seed string) (token string) {
	tokenStr := fmt.Sprintf("userId=%d&goodsId=%d&timestamp=%v&security=%s",
		userId, goodsId, timestamp, seed)
	tokenBytes := md5.Sum([]byte(tokenStr))
	// md5返回byte数组 16进制格式化
	token = fmt.Sprintf("%x", tokenBytes)
	return
}
