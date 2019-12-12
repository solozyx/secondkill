package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"

	"secondkill/comm/config/constant"
	"secondkill/comm/model/request"
	"secondkill/comm/model/response"
	"secondkill/proxy/service"
)

// 控制器是入口 不处理具体业务逻辑
// 业务在service层处理 后续需要切换Beego框架 service层代码方便复用

type SeckillController struct {
	// 继承Beego控制器
	beego.Controller
}

func (c *SeckillController) SecKill() {
	var (
		respMap    map[string]interface{}
		goodsId    int
		source     string
		authcode   string
		secTime    string
		nance      string
		secRequest *request.SecRequest

		err        error
		statusCode int
		goodsInfo  map[string]interface{}
	)
	respMap = make(map[string]interface{})
	respMap["code"] = constant.SuccessActivityCode
	respMap["message"] = "success"

	// 向浏览器客户端返回数据
	defer func() {
		c.Data["json"] = respMap
		c.ServeJSON()
	}()

	if goodsId, err = c.GetInt("goods_id"); err != nil {
		respMap["code"] = constant.ErrInvalidRequest
		respMap["message"] = "invalid goods_id"
		return
	}

	source = c.GetString("source")
	authcode = c.GetString("authcode")
	secTime = c.GetString("time")
	nance = c.GetString("nance")

	secRequest = &request.SecRequest{
		GoodsId:  goodsId,  // 商品Id
		Source:   source,   // 客户端类型
		SecTime:  secTime,  // 秒杀请求时间
		Authcode: authcode, // 授权码
		Nance:    nance,    // 随机数
		// 1个 SecRequest 对应 1个 SecResponse
		SecReqRelatedSecRespChan: make(chan *response.SecResponse, 1),
	}

	// TODO NOTICE - 简化测试去掉用户cookie校验 生产环境请接入商城系统获取cookie
	// 简化改为从 URL 获取 userId
	if secRequest.UserId, err = c.GetInt("user_id"); err != nil {
		respMap["code"] = constant.ErrInvalidRequest
		respMap["message"] = "invalid input user_id"
		return
	}

	// 获取cookie
	secRequest.UserAuthSign = c.Ctx.GetCookie("userAuthSign")
	if secRequest.UserId, err = strconv.Atoi(c.Ctx.GetCookie("userId")); err != nil {
		// cookie 获取 userId 失败
		respMap["code"] = constant.ErrInvalidRequest
		respMap["message"] = "invalid cookie userId"
		return
	}

	// 接入层 用户访问 /seckill 接口时间
	secRequest.AccessTime = time.Now()
	// 客户端IP 去掉端口
	if len(c.Ctx.Request.RemoteAddr) > 0 {
		secRequest.ClientIP = strings.Split(c.Ctx.Request.RemoteAddr, ":")[0]
		logs.Debug("seckill proxy ClientIP = ", secRequest.ClientIP)
	}
	secRequest.ClientReferer = c.Ctx.Request.Referer()

	// 用户是否关闭浏览器客户端
	secRequest.ClientCloseNotify = c.Ctx.ResponseWriter.CloseNotify()

	if goodsInfo, statusCode, err = service.G_proxyService.SecKill(secRequest); err != nil {
		respMap["code"] = statusCode
		respMap["message"] = err.Error()
		return
	}

	respMap["data"] = goodsInfo
	respMap["code"] = statusCode
}

func (c *SeckillController) SecInfo() {
	var (
		respMap    map[string]interface{}
		goodsId    int
		err        error
		secInfo    map[string]interface{}
		statusCode int
	)
	respMap = make(map[string]interface{})
	respMap["code"] = constant.SuccessActivityCode
	respMap["message"] = "success"

	// 向浏览器客户端返回数据
	defer func() {
		c.Data["json"] = respMap
		c.ServeJSON()
	}()

	// 获取浏览器传入参数
	if goodsId, err = c.GetInt("goods_id"); err != nil {
		logs.Error("seckill proxy invalid request get goods_id err: %v", err)
		respMap["code"] = constant.ErrInvalidRequest
		respMap["message"] = "invalid goods_id"
		return
	}
	if secInfo, statusCode, err = service.G_proxyService.SecInfo(goodsId); err != nil {
		respMap["code"] = statusCode
		respMap["message"] = err.Error()
		return
	}

	respMap["data"] = secInfo
}

func (c *SeckillController) SecList() {
	var (
		err       error
		respMap   map[string]interface{}
		goodsList []map[string]interface{}
	)
	respMap = make(map[string]interface{})
	respMap["code"] = constant.SuccessActivityCode
	respMap["message"] = "success"
	defer func() {
		c.Data["json"] = respMap
		c.ServeJSON()
	}()
	if goodsList, err = service.G_proxyService.SecList(); err != nil {
		logs.Error("seckill proxy invalid request get goodsList err: %v", err)
		respMap["code"] = constant.ErrNotFoundGoodsId
		respMap["message"] = err.Error()
		return
	}
	respMap["data"] = goodsList
}
