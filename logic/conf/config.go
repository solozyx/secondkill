package conf

import (
	"fmt"

	"github.com/astaxie/beego"
)

var (
	G_logicConf *logicConf
)

type redisConf struct {
	RedisAddr        string
	RedisMaxIdle     int
	RedisMaxActive   int
	RedisIdleTimeout int
}

type etcdConf struct {
	EtcdAddr    string
	EtcdTimeout int
	// 秒杀活动key前缀 秒杀在etcd配置都在该key下
	EtcdSeckillKey        string
	EtcdSeckillProductKey string
}

type beegoLogConf struct {
	BeegoLogPath  string
	BeegoLogLevel string
}

type logicConf struct {
	// beego 框架日志配置
	BeegoLogConf beegoLogConf

	EtcdConf etcdConf

	RedisLogicToProxyConf redisConf
	RedisProxyToLogicConf redisConf

	GoroutineNumReadProxyRequestFromRedis int
	GoroutineNumWriteLogicResponseToRedis int
	GoroutineNumSeckillHandle             int

	ChanSizeReadProxyRequest   int
	ChanSizeWriteLogicResponse int

	TimeoutProxyMaxAlive    int
	TimeoutLogicSecReqChan  int
	TimeoutLogicSecRespChan int

	// 秒杀逻辑层生成token返回接入层 用户到商城系统加购物车
	TokenPasswd string
}

func InitConfig() (err error) {
	var (
		beegoLogPath  string
		beegoLogLevel string

		redisAddr        string
		redisMaxIdle     int
		redisMaxActive   int
		redisIdleTimeout int

		etcdAddr              string
		etcdTimeout           int
		etcdSeckillKey        string
		etcdSeckillProductKey string

		goroutineNum int
		chanSize     int

		timeout int

		tokenPasswd string
	)

	G_logicConf = &logicConf{}
	// 读取 main包/conf/app.conf 配置文件
	// beego日志配置
	beegoLogPath = beego.AppConfig.String("logs::log_path")
	beegoLogLevel = beego.AppConfig.String("logs::log_level")
	if len(beegoLogPath) == 0 {
		err = fmt.Errorf("beego read log_path is null")
		beegoLogPath = "./logs"
	}
	if len(beegoLogLevel) == 0 {
		err = fmt.Errorf("beego read log_level is null")
		beegoLogLevel = "debug"
	}
	G_logicConf.BeegoLogConf = beegoLogConf{
		BeegoLogLevel: beegoLogLevel,
		BeegoLogPath:  beegoLogPath,
	}

	// redis
	redisAddr = beego.AppConfig.String("redis::redis_proxy_to_logic_addr")
	if len(redisAddr) == 0 {
		err = fmt.Errorf("beego read redis_proxy_to_logic_addr is null")
		return
	}
	if redisMaxIdle, err = beego.AppConfig.Int("redis::redis_proxy_to_logic_idle"); err != nil {
		err = fmt.Errorf("beego read redis_proxy_to_logic_idle err: %v", err)
		return
	}
	if redisMaxActive, err = beego.AppConfig.Int("redis::redis_proxy_to_logic_active"); err != nil {
		err = fmt.Errorf("beego read redis_proxy_to_logic_active err: %v", err)
		return
	}
	if redisIdleTimeout, err = beego.AppConfig.Int("redis::redis_proxy_to_logic_idle_timeout"); err != nil {
		err = fmt.Errorf("beego read redis_proxy_to_logic_idle_timeout err: %v", err)
		return
	}
	G_logicConf.RedisProxyToLogicConf = redisConf{
		RedisAddr:        redisAddr,
		RedisMaxIdle:     redisMaxIdle,
		RedisMaxActive:   redisMaxActive,
		RedisIdleTimeout: redisIdleTimeout,
	}

	redisAddr = beego.AppConfig.String("redis::redis_logic_to_proxy_addr")
	if len(redisAddr) == 0 {
		err = fmt.Errorf("beego read redis_logic_to_proxy_addr is null")
		return
	}
	if redisMaxIdle, err = beego.AppConfig.Int("redis::redis_logic_to_proxy_idle"); err != nil {
		err = fmt.Errorf("beego read redis_logic_to_proxy_idle err: %v", err)
		return
	}
	if redisMaxActive, err = beego.AppConfig.Int("redis::redis_logic_to_proxy_active"); err != nil {
		err = fmt.Errorf("beego read redis_logic_to_proxy_active err: %v", err)
		return
	}
	if redisIdleTimeout, err = beego.AppConfig.Int("redis::redis_logic_to_proxy_idle_timeout"); err != nil {
		err = fmt.Errorf("beego read redis_logic_to_proxy_idle_timeout err: %v", err)
		return
	}
	G_logicConf.RedisLogicToProxyConf = redisConf{
		RedisAddr:        redisAddr,
		RedisMaxIdle:     redisMaxIdle,
		RedisMaxActive:   redisMaxActive,
		RedisIdleTimeout: redisIdleTimeout,
	}

	// etcd
	etcdAddr = beego.AppConfig.String("etcd::etcd_addr")
	if len(etcdAddr) == 0 {
		err = fmt.Errorf("beego read etcd_addr is null")
		return
	}
	if etcdTimeout, err = beego.AppConfig.Int("etcd::etcd_timeout"); err != nil {
		err = fmt.Errorf("beego read etcd_timeout config err: %v", err)
		return
	}
	etcdSeckillKey = beego.AppConfig.String("etcd::etcd_seckill_key")
	if len(etcdSeckillKey) == 0 {
		err = fmt.Errorf("beego read etcd_seckill_key is null")
		return
	}
	etcdSeckillProductKey = beego.AppConfig.String("etcd::etcd_seckill_product_key")
	if len(etcdSeckillProductKey) == 0 {
		err = fmt.Errorf("beego read etcd_seckill_product_key is null")
		return
	}
	G_logicConf.EtcdConf = etcdConf{
		EtcdAddr:              etcdAddr,
		EtcdTimeout:           etcdTimeout,
		EtcdSeckillKey:        etcdSeckillKey,
		EtcdSeckillProductKey: etcdSeckillProductKey,
	}

	// 读写redis goroutine数量
	if goroutineNum, err = beego.AppConfig.Int("goroutine::goroutine_num_read_proxy_request_from_redis"); err != nil {
		err = fmt.Errorf("beego read goroutine_num_read_proxy_request_from_redis err: %v", err)
		return
	}
	G_logicConf.GoroutineNumReadProxyRequestFromRedis = goroutineNum

	if goroutineNum, err = beego.AppConfig.Int("goroutine::goroutine_num_write_logic_response_to_redis"); err != nil {
		err = fmt.Errorf("beego read goroutine_num_write_logic_response_to_redis err: %v", err)
		return
	}
	G_logicConf.GoroutineNumWriteLogicResponseToRedis = goroutineNum

	if goroutineNum, err = beego.AppConfig.Int("goroutine::goroutine_num_seckill_handle"); err != nil {
		err = fmt.Errorf("beego read goroutine_num_seckill_handle config err: %v", err)
		return
	}
	G_logicConf.GoroutineNumSeckillHandle = goroutineNum

	// channel size
	if chanSize, err = beego.AppConfig.Int("channel::channel_size_read_proxy_request"); err != nil {
		err = fmt.Errorf("beego read channel_size_read_proxy_request err: %v", err)
		return
	}
	G_logicConf.ChanSizeReadProxyRequest = chanSize

	if chanSize, err = beego.AppConfig.Int("channel::channel_size_write_logic_response"); err != nil {
		err = fmt.Errorf("beego read channel_size_write_logic_response err: %v", err)
		return
	}
	G_logicConf.ChanSizeWriteLogicResponse = chanSize

	// timeout settings
	if timeout, err = beego.AppConfig.Int("timeout::timeout_proxy_max_alive"); err != nil {
		err = fmt.Errorf("beego read timeout_proxy_max_alive err: %v", err)
		return
	}
	G_logicConf.TimeoutProxyMaxAlive = timeout

	if timeout, err = beego.AppConfig.Int("timeout::timeout_logic_secrequest_chan"); err != nil {
		err = fmt.Errorf("beego read timeout_logic_secrequest_chan err: %v", err)
		return
	}
	G_logicConf.TimeoutLogicSecReqChan = timeout

	if timeout, err = beego.AppConfig.Int("timeout::timeout_logic_secresponse_chan"); err != nil {
		err = fmt.Errorf("beego read timeout_logic_secresponse_chan err: %v", err)
		return
	}
	G_logicConf.TimeoutLogicSecRespChan = timeout

	// token生成所需密钥种子
	tokenPasswd = beego.AppConfig.String("token::seckill_token_passwd")
	if len(tokenPasswd) == 0 {
		err = fmt.Errorf("beego read seckill_token_passwd is null ")
		return
	}
	G_logicConf.TokenPasswd = tokenPasswd

	return
}
