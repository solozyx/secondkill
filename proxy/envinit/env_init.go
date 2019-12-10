package envinit

import (
	"encoding/json"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/coreos/etcd/clientv3"
	"github.com/garyburd/redigo/redis"

	"secondkill/comm/utils"
	"secondkill/proxy/conf"
)

var (
	G_envInit *envInit
)

type envInit struct {
	RedisBlacklistPool    *redis.Pool
	EtcdClient            *clientv3.Client
	RedisProxyToLogicPool *redis.Pool
	RedisLogicToProxyPool *redis.Pool
}

func InitProxy() {
	G_envInit = &envInit{}
	initLogs()
	initRedis()
	initEtcd()
	logs.Info("seckill proxy env init success")
}

func initLogs() {
	logConf := make(map[string]interface{})
	logConf["filename"] = conf.G_proxyConf.BeegoLogConf.BeegoLogPath
	logConf["level"] = utils.ConvertLogLevel(conf.G_proxyConf.BeegoLogConf.BeegoLogLevel)
	confBytes, err := json.Marshal(logConf)
	if err != nil {
		logs.Error("seckill proxy json Marshal logConf err: %v", err)
		panic(err)
	}
	logs.SetLogger(logs.AdapterFile, string(confBytes))
	// 日志默认输出调用的文件名和文件行号,如果你期望不输出调用的文件名和文件行号
	// 开启传入参数 true,关闭传入参数 false,默认是关闭的
	logs.EnableFuncCallDepth(true)
}

func initRedis() {
	var (
		redisPool *redis.Pool
		conn      redis.Conn
	)
	// redis blacklist 实例
	redisPool = &redis.Pool{
		MaxIdle:   conf.G_proxyConf.RedisBlacklistConf.RedisMaxIdle,
		MaxActive: conf.G_proxyConf.RedisBlacklistConf.RedisMaxActive,
		// 超时时间 单位 秒 time.Duration 单位是纳秒
		IdleTimeout: time.Duration(conf.G_proxyConf.RedisBlacklistConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.G_proxyConf.RedisBlacklistConf.RedisAddr)
		},
	}
	conn = redisPool.Get()
	defer conn.Close()
	// 检测Redis连接是否可用
	// redis 连接失败 主动把程序挂掉
	if _, err := conn.Do("ping"); err != nil {
		logs.Error("seckill proxy ping redis Blacklist server err: ", err)
		panic(err) // return env_init.go:44:3: err is shadowed during return
	} else {
		logs.Info("seckill proxy ping redis Blacklist server success")
	}
	G_envInit.RedisBlacklistPool = redisPool

	// redis proxy->logic 接入层到逻辑层实例
	redisPool = &redis.Pool{
		MaxIdle:     conf.G_proxyConf.RedisProxyToLogicConf.RedisMaxIdle,
		MaxActive:   conf.G_proxyConf.RedisProxyToLogicConf.RedisMaxActive,
		IdleTimeout: time.Duration(conf.G_proxyConf.RedisProxyToLogicConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.G_proxyConf.RedisProxyToLogicConf.RedisAddr)
		},
	}
	conn = redisPool.Get()
	defer conn.Close()
	if _, err := conn.Do("ping"); err != nil {
		logs.Error("seckill proxy ping redis ProxyToLogic server err: ", err)
		panic(err)
	} else {
		logs.Info("seckill proxy ping redis ProxyToLogic server success")
	}
	G_envInit.RedisProxyToLogicPool = redisPool

	// redis access<-logic 逻辑层到接入层实例
	redisPool = &redis.Pool{
		MaxIdle:     conf.G_proxyConf.RedisLogicToProxyConf.RedisMaxIdle,
		MaxActive:   conf.G_proxyConf.RedisLogicToProxyConf.RedisMaxActive,
		IdleTimeout: time.Duration(conf.G_proxyConf.RedisLogicToProxyConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.G_proxyConf.RedisLogicToProxyConf.RedisAddr)
		},
	}
	conn = redisPool.Get()
	defer conn.Close()
	// redis 连接失败 主动把程序挂掉
	if _, err := conn.Do("ping"); err != nil {
		logs.Error("seckill proxy ping redis LogicToProxy server err: ", err)
		panic(err)
	} else {
		logs.Info("seckill proxy ping redis LogicToProxy server success")
	}
	G_envInit.RedisLogicToProxyPool = redisPool
}

func initEtcd() {
	config := clientv3.Config{
		Endpoints:   []string{conf.G_proxyConf.EtcdConf.EtcdAddr},
		DialTimeout: time.Duration(conf.G_proxyConf.EtcdConf.EtcdTimeout) * time.Second,
	}
	client, err := clientv3.New(config)
	if err != nil {
		// etcd 连接失败 主动把程序挂掉
		logs.Error("seckill proxy connect etcd server err: ", err)
		panic(err)
	} else {
		logs.Info("seckill proxy connect etcd server success")
	}
	// TODO defer etcdClient.Close() ?
	G_envInit.EtcdClient = client
}
