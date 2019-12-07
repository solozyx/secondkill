package router

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"

	"secondkill/proxy/controller"
)

func init() {
	logs.Debug("seckill proxy RESTful API routers init")
	// api
	// controller
	// get post 都支持用* : 具体的处理方法 SecKill  get:SecKill post:SecKill
	// 前端 get post 请求 /seckill 调用 SkillController 的 SecKill 方法处理
	beego.Router("/seckill",&controller.SeckillController{},"*:SecKill")
	beego.Router("/secinfo",&controller.SeckillController{},"*:SecInfo")
	beego.Router("/seclist",&controller.SeckillController{},"*:SecList")
}