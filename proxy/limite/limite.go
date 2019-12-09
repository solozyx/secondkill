package limite

import (
	"fmt"
	"sync"

	"secondkill/comm/model/request"
	"secondkill/proxy/conf"
)

var (
	G_secondLimitMgr *secondLimitMgr
)

type secondLimit struct {
	// 计数器
	counter int
	// 时间戳 没必要用具体time时间类型
	currentTime int64
}

type secondLimitMgr struct {
	// UserLimitMap map[int]*SecondLimit
	// 同一时刻 大量用户秒杀请求
	// lock sync.Mutex
	userLimitMap *sync.Map
	// IPLimitMap map[string]*SecondLimit
	ipLimitMap *sync.Map
}

func InitSecondLimitMgr() {
	G_secondLimitMgr = &secondLimitMgr{}
	// G_secondLimitMgr.UserLimitMap = make(map[int]*SecondLimit,10000)
	// G_secondLimitMgr.lock = sync.Mutex{}
	G_secondLimitMgr.userLimitMap = &sync.Map{}
	G_secondLimitMgr.ipLimitMap = &sync.Map{}
}

// 用户请求对其进行计数
func (slm *secondLimitMgr) UserAndIpAccessSecondCountCheck(secRequest *request.SecRequest) (err error) {
	// slm.lock.Lock()
	// 新用户访问创建流控限制
	// sl,ok := slm.UserLimitMap[secRequest.UserId]
	var sl *secondLimit
	user, ok := slm.userLimitMap.Load(secRequest.UserId)
	if !ok {
		sl = &secondLimit{}
		// slm.UserLimitMap[secRequest.UserId] = sl
		slm.userLimitMap.Store(secRequest.UserId, sl)
	} else {
		sl = user.(*secondLimit)
	}
	// 当前时间启动计数器
	counter := sl.count(secRequest.AccessTime.Unix())
	// slm.lock.Unlock()
	// 流控限制
	if counter > conf.G_proxyConf.UserAccessSeckillSecondLimit {
		err = fmt.Errorf("seckill service user_id busy :( ")
		return
	}

	// TODO 单user 和 单ip 限流 现在是同步机制 会有性能问题 优化为异步
	// 把本次 /seckill 请求的 userid 和 clientip 放到 channel 后台协程去计数
	// 1秒钟该userid clientip 超过最大次数限制 就把userid clientip加黑名单 禁掉用户
	// 也可以把数据放到一个统一的数据中心
	// 现在考虑的只是单台机器
	// 生产环境 前面是负载均衡 后面有 20台机器 每台机器刷1次
	// 把20台机器 该用户 userid clientip 汇总到一台中心服务器做分析
	// 得出一个 userid clientip 黑名单 达到频率控制阈值
	// 把该黑名单存储到etcd 形成业务闭环
	// 前面 20次可以正常秒杀 到21次 中心服务器发现异常 设置黑名单 写入redis 程序读取到 拒绝
	// 黑名单数据比较多 存储在etcd不合适 etcd用来存储一些配置项 少量数据
	ip, ok := slm.ipLimitMap.Load(secRequest.ClientIP)
	if !ok {
		sl = &secondLimit{}
		slm.ipLimitMap.Store(secRequest.ClientIP, sl)
	} else {
		sl = ip.(*secondLimit)
	}
	ipcounter := sl.count(secRequest.AccessTime.Unix())
	if ipcounter > conf.G_proxyConf.IPAccessSeckillSecondLimit {
		err = fmt.Errorf("seckill service client_ip busy :( ")
		return
	}

	return
}

// 计算某个用户在这1秒访问次数
func (sl *secondLimit) count(nowTime int64) (currentCount int) {
	// 计数的时间 和 上次计数时间 不相等 说明不是同1秒
	if sl.currentTime != nowTime {
		sl.currentTime = nowTime
		sl.counter = 1
		currentCount = sl.counter
		return
	}
	// 是同一秒 则 计数器累加 1
	sl.counter++
	currentCount = sl.counter
	return
}

// 检查当前秒内用户访问次数
func (sl *secondLimit) check(nowTime int64) int {
	if sl.currentTime != nowTime {
		return 0
	}
	return sl.counter
}
