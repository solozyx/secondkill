package limite

type GoodsSecondRateLimit struct {
	Counter     int   `json:"counter"`
	CurrentTime int64 `json:"currentTime"`
}

func (l *GoodsSecondRateLimit) Count(nowTime int64) (currentCount int) {
	// 计数的时间 和 上次计数时间 不相等 说明不是同1秒
	if l.CurrentTime != nowTime {
		l.CurrentTime = nowTime
		l.Counter = 1
		currentCount = l.Counter
		return
	}
	// 是同一秒 则 计数器累加 1
	l.Counter++
	currentCount = l.Counter
	return
}

// 检查当前秒内用户访问次数
func (l *GoodsSecondRateLimit) Check(nowTime int64) int {
	if l.CurrentTime != nowTime {
		return 0
	}
	return l.Counter
}
