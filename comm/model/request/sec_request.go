package request

import (
	"time"

	"secondkill/comm/model/response"
)

type SecRequest struct {
	// 商品Id
	GoodsId int
	// 客户端来源
	Source string
	// 授权码
	Authcode string
	// 秒杀时间
	SecTime string
	// 随机数
	Nance string
	// 用户Id
	UserId int
	// 用户cookie 用户登录商城 登录的时候下发cookie
	// cookie:{userId,token}
	// cookie:{userId:1234567890,userAuth:xhaohgsohosnxaingjfajl}
	// 访问秒杀传入 userId 和 userAuth 对用户进行校验
	// 保证所有用户必须在登录的情况下才能进行秒杀抢购
	// 用于校验用户是否有参与秒杀的权限
	UserAuthSign string

	// 用户访问秒杀 /seckill 接口时间
	AccessTime time.Time

	// 客户端IP
	ClientIP string

	// 客户端referer 进入秒杀只有几个referer是合法的
	ClientReferer string

	// 客户端参与秒杀后是否关闭通知
	ClientCloseNotify <-chan bool `json:"-"`

	// 该秒杀请求对应的处理结果
	SecReqRelatedSecRespChan chan *response.SecResponse `json:"-"`
}
