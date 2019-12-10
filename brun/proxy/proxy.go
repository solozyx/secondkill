package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"

	"secondkill/proxy/conf"
	"secondkill/proxy/dao"
	"secondkill/proxy/envinit"
	"secondkill/proxy/limite"
	// router模块不需要使用它里面的函数方法,但需要执行router.init()初始化,下划线 _ 导入做初始化
	_ "secondkill/proxy/router"
	"secondkill/proxy/service"
)

func main(){
	// 初始化配置
	err := conf.InitConfig()
	if err != nil {
		logs.Error("seckill proxy read config err: %v",err)
		return
	}
	// 初始化 日志 Redis etcd
	envinit.InitProxy()

	// 初始化秒杀商品模块
	err = dao.InitGoodsDao()
	if err != nil {
		logs.Error("seckill proxy InitGoodsMgr err: %v",err)
		return
	}

	// 初始化秒杀接入层
	err = service.InitProxyService()
	if err != nil {
		logs.Error("seckill proxy init AccessService err: %v",err)
		return
	}

	// 初始化用户流控模块
	limite.InitSecondLimitMgr()

	beego.Run()
}