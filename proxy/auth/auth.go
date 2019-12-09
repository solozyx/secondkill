package auth

import (
	"crypto/md5"
	"fmt"

	"github.com/astaxie/beego/logs"

	"secondkill/comm/model/request"
	"secondkill/proxy/conf"
)

func UserCheck(secReq *request.SecRequest) (isUserValid bool) {
	var (
		authData string
		authSign string
		referer  string
		found    bool
	)
	for _, referer = range conf.G_proxyConf.ClientRefererWhiteList {
		if referer == secReq.ClientReferer {
			found = true
			break
		}
	}
	if !found {
		logs.Warn("user is reject by referer white list", secReq.UserId, secReq)
		return found
	}

	authData = fmt.Sprintf("%s:%d", conf.G_proxyConf.CookieSecretKey, secReq.UserId)
	// 16进制格式化
	authSign = fmt.Sprintf("%x", md5.Sum([]byte(authData)))
	if authSign == secReq.UserAuthSign {
		isUserValid = true
	}
	logs.Warn("UserAuthSign check fail")
	return
}
