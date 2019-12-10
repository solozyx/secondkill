package envinit

import (
	"encoding/json"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/coreos/etcd/clientv3"
	"github.com/garyburd/redigo/redis"

	"secondkill/comm/utils"
	"secondkill/logic/conf"
)

var (
	G_envInit *envInit
)

type envInit struct {
	RedisProxyToLogicPool *redis.Pool
	RedisLogicToProxyPool *redis.Pool
	EtcdClient            *clientv3.Client
}

func InitLogic() {
	// 赋值单例
	G_envInit = &envInit{}
	// 先初始化日志 如果打印日志的时候日志没有初始化 会打到终端上
	initLogs()
	initRedis()
	initEtcd()
	logs.Info("seckill logic env init success")
}

func initLogs() {
	logConf := make(map[string]interface{})
	logConf["filename"] = conf.G_logicConf.BeegoLogConf.BeegoLogPath
	logConf["level"] = utils.ConvertLogLevel(conf.G_logicConf.BeegoLogConf.BeegoLogLevel)
	confBytes, err := json.Marshal(logConf)
	if err != nil {
		logs.Error("seckill logic json Marshal logConf err")
		panic(err)
	}
	logs.SetLogger(logs.AdapterFile, string(confBytes))
	// 日志默认输出调用的文件名和文件行号,如果你期望不输出调用的文件名和文件行号
	// 开启传入参数 true,关闭传入参数 false,默认是关闭的.
	logs.EnableFuncCallDepth(true)
}

func initRedis() {
	var (
		redisPool *redis.Pool
		conn      redis.Conn
	)
	// redis proxy --> logic
	redisPool = &redis.Pool{
		MaxIdle:   conf.G_logicConf.RedisProxyToLogicConf.RedisMaxIdle,
		MaxActive: conf.G_logicConf.RedisProxyToLogicConf.RedisMaxActive,
		// 超时时间 单位 秒 time.Duration 单位是纳秒
		IdleTimeout: time.Duration(conf.G_logicConf.RedisProxyToLogicConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.G_logicConf.RedisProxyToLogicConf.RedisAddr)
		},
	}
	conn = redisPool.Get()
	defer conn.Close()
	// 检测Redis连接是否可用 redis 连接失败 主动把程序挂掉
	if _, err := conn.Do("ping"); err != nil {
		logs.Error("seckill logic ping redis ProxyToLogic server err: ", err)
		panic(err) // return envinit.go:44:3: err is shadowed during return
	} else {
		logs.Info("seckill logic ping redis ProxyToLogic server success")
	}
	G_envInit.RedisProxyToLogicPool = redisPool

	// redis logic --> proxy
	redisPool = &redis.Pool{
		MaxIdle:     conf.G_logicConf.RedisLogicToProxyConf.RedisMaxIdle,
		MaxActive:   conf.G_logicConf.RedisLogicToProxyConf.RedisMaxActive,
		IdleTimeout: time.Duration(conf.G_logicConf.RedisLogicToProxyConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", conf.G_logicConf.RedisLogicToProxyConf.RedisAddr)
		},
	}
	conn = redisPool.Get()
	defer conn.Close()
	if _, err := conn.Do("ping"); err != nil {
		logs.Error("seckill logic ping redis LogicToProxy server err: ", err)
		panic(err)
	} else {
		logs.Info("seckill logic ping redis LogicToProxy server success")
	}
	G_envInit.RedisLogicToProxyPool = redisPool
}

func initEtcd() {
	config := clientv3.Config{
		Endpoints:   []string{conf.G_logicConf.EtcdConf.EtcdAddr},
		DialTimeout: time.Duration(conf.G_logicConf.EtcdConf.EtcdTimeout) * time.Second,
	}
	client, err := clientv3.New(config)
	if err != nil {
		logs.Error("seckill logic connect etcd server err: ", err)
		panic(err)
	} else {
		logs.Info("seckill logic connect etcd server success")
	}
	// TODO defer client.Close() ?
	G_envInit.EtcdClient = client
}
