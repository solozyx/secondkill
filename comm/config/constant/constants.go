package constant

const (
	SuccessActivityCode = 0000

	ErrInvalidRequest      = 1001
	ErrNotFoundGoodsId     = 1002
	ErrUserAuthCheckFailed = 1003
	// 对于机器人 提示系统繁忙 正常用户不会收到该错误
	// 恶意刷的用户才会收到该错误
	// 提示要模糊 不能让机器人猜测到服务端策略
	ErrUserServiceBusy = 1004

	ErrActivityNotStart   = 1005
	ErrActivityAlreadyEnd = 1006
	ErrActivitySaleOut    = 1007
	ErrActivityRetry      = 1008
	ErrActivityAlreadyBuy = 1009

	// 秒杀接入层 等待 秒杀逻辑层 返回处理结果超时
	ErrProxyWaitLogicRespTimeout = 1010
	ErrProxyUserClientCloesd     = 1011

	GoodsStatusNormal       = 0 // 秒杀商品正常抢购
	GoodsStatusSaleOut      = 1 // 秒杀商品正常卖光
	GoodsStatusForceSaleOut = 2 // 秒杀商品强制卖光 结束抢购

	ActivityStatusNormal   = 0
	ActivityStatusDisabled = 1
	ActivityStatusExpired  = 2

	ActivityNormal   = "秒杀活动正常"
	ActivityDisabled = "秒杀活动禁止"
	ActivityExpired  = "秒杀活动过期"

	GolangTimeSeed = "2006-01-02 15:04:05"

	RedisQueueUserIdBlacklist = "secondkill:queue_userid_blacklist"
	RedisQueueIpBlacklist     = "secondkill:queue_ip_blacklist"
	RedisQueueSecReq          = "secondkill:queue_sec_request"
	RedisQueueSecResp         = "secondkill:queue_sec_response"
)
