package conf

import (
	"fmt"
	"strings"
	"sync"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

var (
	G_proxyConf *proxyConfig
)

type redisConfig struct {
	RedisAddr        string
	RedisMaxIdle     int
	RedisMaxActive   int
	RedisIdleTimeout int
}

type etcdConfig struct {
	EtcdAddr    string
	EtcdTimeout int
	// 秒杀活动key前缀 秒杀在etcd配置都在该key下
	EtcdSeckillKey        string
	EtcdSeckillProductKey string
}

type beegoLogConfig struct {
	BeegoLogPath  string
	BeegoLogLevel string
}

type proxyConfig struct {
	RedisBlacklistConf    redisConfig
	RedisProxyToLogicConf redisConfig
	RedisLogicToProxyConf redisConfig

	GoroutineNumWriteProxyRequestToRedis   int
	GoroutineNumReadLogicResponseFromRedis int

	ChanSizeWriteProxyRequest int
	ChanSizeReadLogicResponse int

	EtcdConf etcdConfig

	// beego框架日志配置
	BeegoLogConf beegoLogConfig

	// cookie密钥
	CookieSecretKey string

	// 单用户每秒访问秒杀业务最大次数
	UserAccessSeckillSecondLimit int
	// 单IP每秒访问秒杀业务最大次数
	IPAccessSeckillSecondLimit int

	// 客户端referer白名单
	ClientRefererWhiteList []string
	// userid 黑名单
	UserBlackList *sync.Map
	// clientip 黑名单
	IPBlackList *sync.Map
}

func InitConfig() (err error) {
	var (
		redisAddr string
		etcdAddr  string

		redisMaxIdle     int
		redisMaxActive   int
		redisIdleTimeout int

		etcdTimeout int

		beegoLogPath  string
		beegoLogLevel string

		etcdSeckillKey        string
		etcdSeckillProductKey string

		cookieSecretKey string

		userAccessSeckillSecondLimit int
		ipAccessSeckillSecondLimit   int

		clientRefererWhiteList string

		goroutineNum int
		chanSize     int
	)

	// 读取 main包/conf/app.conf 配置文件
	redisAddr = beego.AppConfig.String("redis_blacklist_addr")
	etcdAddr = beego.AppConfig.String("etcd_addr")
	if len(redisAddr) == 0 || len(etcdAddr) == 0 {
		err = fmt.Errorf("beego read redis[%s] or etcd[%s] is null", redisAddr, etcdAddr)
		return
	}

	if redisMaxIdle, err = beego.AppConfig.Int("redis_blacklist_max_idle"); err != nil {
		err = fmt.Errorf("beego read redis_blacklist_max_idle err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_blacklist_max_idle: %v", redisMaxIdle)

	if redisMaxActive, err = beego.AppConfig.Int("redis_blacklist_max_active"); err != nil {
		err = fmt.Errorf("beego read redis_blacklist_max_active err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_blacklist_max_active: %v", redisMaxActive)

	if redisIdleTimeout, err = beego.AppConfig.Int("redis_blacklist_idle_timeout"); err != nil {
		err = fmt.Errorf("beego read redis_blacklist_idle_timeout err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_blacklist_idle_timeout: %v", redisIdleTimeout)

	if etcdTimeout, err = beego.AppConfig.Int("etcd_timeout"); err != nil {
		err = fmt.Errorf("beego read etcd_timeout err: %v", err)
		return
	}
	logs.Debug("beego read success for etcd_timeout: %v", etcdTimeout)

	beegoLogPath = beego.AppConfig.String("log_path")
	beegoLogLevel = beego.AppConfig.String("log_level")
	if len(beegoLogPath) == 0 || len(beegoLogLevel) == 0 {
		err = fmt.Errorf("beego read log_path[%s] or log_level[%s] is null", beegoLogPath, beegoLogLevel)
		return
	}

	etcdSeckillKey = beego.AppConfig.String("etcd_seckill_key")
	if len(etcdSeckillKey) == 0 {
		err = fmt.Errorf("beego read etcd_seckill_key is null")
		return
	}

	etcdSeckillProductKey = beego.AppConfig.String("etcd_seckill_product_key")
	if len(etcdSeckillProductKey) == 0 {
		err = fmt.Errorf("beego read etcd_seckill_product_key is null")
		return
	}

	cookieSecretKey = beego.AppConfig.String("cookie_secret_key")
	if len(cookieSecretKey) == 0 {
		err = fmt.Errorf("beego read cookie_secret_key is null")
		return
	}

	if userAccessSeckillSecondLimit, err = beego.AppConfig.Int("user_access_seckill_second_limit"); err != nil {
		err = fmt.Errorf("beego read user_access_seckill_second_limit err: %v", err)
		return
	}
	logs.Debug("beego read success for user_access_seckill_second_limit: %v", userAccessSeckillSecondLimit)

	if ipAccessSeckillSecondLimit, err = beego.AppConfig.Int("ip_access_seckill_second_limit"); err != nil {
		err = fmt.Errorf("beego read ip_access_seckill_second_limit err: %v", err)
		return
	}
	logs.Debug("beego read success for ip_access_seckill_second_limit: %v", ipAccessSeckillSecondLimit)

	clientRefererWhiteList = beego.AppConfig.String("client_referer_white_list")
	if len(clientRefererWhiteList) == 0 {
		err = fmt.Errorf("beego read client_referer_white_list is null")
		return
	}

	G_proxyConf = &proxyConfig{
		RedisBlacklistConf: redisConfig{
			RedisAddr:        redisAddr,
			RedisMaxIdle:     redisMaxIdle,
			RedisMaxActive:   redisMaxActive,
			RedisIdleTimeout: redisIdleTimeout,
		},
		EtcdConf: etcdConfig{
			EtcdAddr:              etcdAddr,
			EtcdTimeout:           etcdTimeout,
			EtcdSeckillKey:        etcdSeckillKey,
			EtcdSeckillProductKey: etcdSeckillProductKey,
		},
		BeegoLogConf: beegoLogConfig{
			BeegoLogPath:  beegoLogPath,
			BeegoLogLevel: beegoLogLevel,
		},
		CookieSecretKey:              cookieSecretKey,
		UserAccessSeckillSecondLimit: userAccessSeckillSecondLimit,
		IPAccessSeckillSecondLimit:   ipAccessSeckillSecondLimit,
		ClientRefererWhiteList:       strings.Split(clientRefererWhiteList, ","),
		UserBlackList:                &sync.Map{},
		IPBlackList:                  &sync.Map{},
	}

	// 读取 main包/conf/app.conf 配置文件 redis_access_to_logic_xxx
	redisAddr = beego.AppConfig.String("redis_proxy_to_logic_addr")
	if len(redisAddr) == 0 {
		err = fmt.Errorf("beego read redis_proxy_to_logic_addr err")
		return
	}

	if redisMaxIdle, err = beego.AppConfig.Int("redis_proxy_to_logic_idle"); err != nil {
		err = fmt.Errorf("beego read redis_proxy_to_logic_idle err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_proxy_to_logic_idle: %v", redisMaxIdle)

	if redisMaxActive, err = beego.AppConfig.Int("redis_proxy_to_logic_active"); err != nil {
		err = fmt.Errorf("beego read redis_proxy_to_logic_active err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_proxy_to_logic_active: %v", redisMaxActive)

	if redisIdleTimeout, err = beego.AppConfig.Int("redis_proxy_to_logic_idle_timeout"); err != nil {
		err = fmt.Errorf("beego read redis_proxy_to_logic_idle_timeout err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_proxy_to_logic_idle_timeout: %v", redisIdleTimeout)

	G_proxyConf.RedisProxyToLogicConf = redisConfig{
		RedisAddr:        redisAddr,
		RedisMaxIdle:     redisMaxIdle,
		RedisMaxActive:   redisMaxActive,
		RedisIdleTimeout: redisIdleTimeout,
	}

	// 读取 redis_logic_to_proxy_xxx
	redisAddr = beego.AppConfig.String("redis_logic_to_proxy_addr")
	if len(redisAddr) == 0 {
		err = fmt.Errorf("beego read redis_logic_to_proxy_addr err")
		return
	}

	if redisMaxIdle, err = beego.AppConfig.Int("redis_logic_to_proxy_idle"); err != nil {
		err = fmt.Errorf("beego read redis_logic_to_proxy_idle err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_logic_to_proxy_idle: %v", redisMaxIdle)

	if redisMaxActive, err = beego.AppConfig.Int("redis_logic_to_proxy_active"); err != nil {
		err = fmt.Errorf("beego read redis_logic_to_proxy_active err: %v", err)
		return
	}
	logs.Debug("beego read success for redis_logic_to_proxy_active: %v", redisMaxActive)

	if redisIdleTimeout, err = beego.AppConfig.Int("redis_logic_to_proxy_idle_timeout"); err != nil {
		err = fmt.Errorf("beego read redis_logic_to_proxy_idle_timeout err: ", err)
		return
	}
	logs.Debug("beego read success for redis_logic_to_proxy_idle_timeout: %v", redisIdleTimeout)

	G_proxyConf.RedisLogicToProxyConf = redisConfig{
		RedisAddr:        redisAddr,
		RedisMaxIdle:     redisMaxIdle,
		RedisMaxActive:   redisMaxActive,
		RedisIdleTimeout: redisIdleTimeout,
	}

	// 读写redis goroutine数量
	if goroutineNum, err = beego.AppConfig.Int("goroutine_num_write_proxy_request_to_redis"); err != nil {
		err = fmt.Errorf("beego read goroutine_num_write_proxy_request_to_redis err: %v", err)
		return
	}
	G_proxyConf.GoroutineNumWriteProxyRequestToRedis = goroutineNum

	if goroutineNum, err = beego.AppConfig.Int("goroutine_num_read_logic_response_from_redis"); err != nil {
		err = fmt.Errorf("beego read goroutine_num_read_logic_response_from_redis err: %v", err)
		return
	}
	G_proxyConf.GoroutineNumReadLogicResponseFromRedis = goroutineNum

	// channel size
	if chanSize, err = beego.AppConfig.Int("channel_size_write_access_request"); err != nil {
		err = fmt.Errorf("beego read channel_size_write_access_request err: %v", err)
		return
	}
	G_proxyConf.ChanSizeWriteProxyRequest = chanSize

	if chanSize, err = beego.AppConfig.Int("channel_size_read_logic_response"); err != nil {
		err = fmt.Errorf("beego read channel_size_read_logic_response err: %v", err)
		return
	}
	G_proxyConf.ChanSizeReadLogicResponse = chanSize

	return
}
