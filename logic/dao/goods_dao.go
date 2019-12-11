package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/astaxie/beego/logs"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"

	"secondkill/comm/model/goods"
	"secondkill/logic/conf"
	"secondkill/logic/envinit"
	"secondkill/logic/limite"
)

var (
	G_goodsDao *goodsDao
)

type goodsDao struct {
	// 秒杀商品在etcd配置key
	goodsKey    string
	etcdWatcher clientv3.Watcher
	// 所有秒杀商品列表
	GoodsMap map[int]*goods.Goods
	RWLock   sync.RWMutex
	// 已卖出秒杀商品统计列表 sync.Map[goodsId int]商品卖出数量count int
	goodsSaledMap *sync.Map
}

func InitGoodsDao() (err error) {
	G_goodsDao = &goodsDao{}
	G_goodsDao.goodsKey = fmt.Sprintf("%s%s",
		conf.G_logicConf.EtcdConf.EtcdSeckillKey, conf.G_logicConf.EtcdConf.EtcdSeckillProductKey)
	G_goodsDao.GoodsMap = make(map[int]*goods.Goods, 1024)
	G_goodsDao.etcdWatcher = clientv3.NewWatcher(envinit.G_envInit.EtcdClient)
	G_goodsDao.RWLock = sync.RWMutex{}
	G_goodsDao.goodsSaledMap = &sync.Map{}
	G_goodsDao.watchGoodsKey()
	return
}

func (dao *goodsDao) watchGoodsKey() {
	var (
		goodsList      []goods.Goods
		isGetGoodsList bool
		watchChan      clientv3.WatchChan
		watchResp      clientv3.WatchResponse
		watchEvent     *clientv3.Event
	)
	goodsList = make([]goods.Goods, 0)
	logs.Debug("seckill logic begin watch etcd goodsKey: ", dao.goodsKey)

	go func() {
		watchChan = dao.etcdWatcher.Watch(context.Background(), dao.goodsKey, clientv3.WithPrefix())
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.DELETE:
					logs.Warn("seckill logic etcd delete goodsKey: ", dao.goodsKey)
					isGetGoodsList = false
					continue
				case mvccpb.PUT:
					if string(watchEvent.Kv.Key) == dao.goodsKey {
						if err := json.Unmarshal(watchEvent.Kv.Value, &goodsList); err != nil {
							logs.Error("seckill logic []*model.Goods json Unmarshal err: ", err)
							isGetGoodsList = false
							continue
						}
						isGetGoodsList = true
						logs.Debug("seckill logic etcd get goodsList are: ", goodsList)
					}
				}
			}
			if isGetGoodsList {
				dao.updateGoodsMap(goodsList)
			}
		}
	}()
	return
}

func (dao *goodsDao) updateGoodsMap(goodsList []goods.Goods) {
	var (
		tmpMap map[int]*goods.Goods
		aGoods goods.Goods
	)
	tmpMap = make(map[int]*goods.Goods, 1024)
	for i := 0; i < len(goodsList); i++ {
		aGoods = goodsList[i]
		aGoods.SecondLimit = &limite.GoodsSecondRateLimit{}
		tmpMap[goodsList[i].GoodsId] = &aGoods
	}
	dao.RWLock.Lock()
	dao.GoodsMap = tmpMap
	dao.RWLock.Unlock()
}

func (dao *goodsDao) CountGoodsSaledById(goodsId int) (count int) {
	// 没必要判断goodsId是否存在 不存在则 count 为 0
	value, _ := dao.goodsSaledMap.Load(goodsId)
	// count,ok := value.(int)
	// if !ok {
	//	return 0
	//}
	count = value.(int)
	return
}

func (dao *goodsDao) AddGoodsSaledById(goodsId, count int) {
	var (
		currentCount int
		value        interface{}
		ok           bool
	)
	if value, ok = dao.goodsSaledMap.Load(goodsId); !ok {
		currentCount = count
	}
	if currentCount, ok = value.(int); !ok {
		currentCount = count
	} else {
		currentCount += count
	}
	dao.goodsSaledMap.Store(goodsId, currentCount)
}
