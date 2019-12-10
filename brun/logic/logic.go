package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"


	"secondkill/logic/conf"
	"secondkill/logic/dao"
	"secondkill/logic/envinit"
	"secondkill/logic/service"
)

func main(){
	err := conf.InitConfig()
	if err != nil {
		logs.Error("seckill logic read config err: ",err)
		return
	}
	envinit.InitLogic()

	// 初始化秒杀商品模块
	if err = dao.InitGoodsDao(); err != nil {
		return
	}
	// 初始化 秒杀逻辑层
	if err = service.InitLogicService(); err != nil {
		logs.Error("seckill logic init LogicService err: ",err)
		return
	}

	beego.Run()
}