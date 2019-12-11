package user

import "sync"

type UserBuyHistroy struct {
	// sync.Map[goodsId int]该用户对该商品购买count int
	BuyHistory *sync.Map
}

func (ubh *UserBuyHistroy) Count(goodsId int) int {
	var (
		value           interface{}
		alreadyBuyCount int
		ok              bool
	)
	value, _ = ubh.BuyHistory.Load(goodsId)
	if alreadyBuyCount, ok = value.(int); !ok {
		return 0
	} else {
		return alreadyBuyCount
	}
}

func (ubh *UserBuyHistroy) Add(goodsId, count int) {
	var (
		value        interface{}
		currentCount int
		ok           bool
	)
	if value, ok = ubh.BuyHistory.Load(goodsId); !ok {
		currentCount = count
	}
	if currentCount, ok = value.(int); !ok {
		currentCount = count
	} else {
		currentCount += count
	}
	ubh.BuyHistory.Store(goodsId, currentCount)
}
