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
	"secondkill/proxy/conf"
	"secondkill/proxy/envinit"
)

var (
	// 公有包变量
	G_goodsDao *goodsDao
)

type goodsDao struct {
	// 私有包变量
	goodsKey    string
	etcdWatcher clientv3.Watcher
	// 商品数据
	GoodsMap map[int]*goods.Goods
	// 避免多协程竞争资源的读写锁 读多写少的情况 配置可能半天才更新1次
	// rwLock sync.RWMutex
	RWLock sync.RWMutex
}

func InitGoodsDao() (err error) {
	G_goodsDao = &goodsDao{}
	G_goodsDao.goodsKey = fmt.Sprintf("%s%s",
		conf.G_proxyConf.EtcdConf.EtcdSeckillKey, conf.G_proxyConf.EtcdConf.EtcdSeckillProductKey)
	G_goodsDao.GoodsMap = make(map[int]*goods.Goods, 1024)
	G_goodsDao.etcdWatcher = clientv3.NewWatcher(envinit.G_envInit.EtcdClient)
	// 初始化 读写锁
	// G_goodsDao.rwLock = sync.RWMutex{}
	G_goodsDao.RWLock = sync.RWMutex{}
	// 加载goodsList
	if err = G_goodsDao.listGoods(); err != nil {
		return
	}
	// 启动etcd key监听
	if err = G_goodsDao.watchGoodsKey(); err != nil {
		return
	}
	return
}

func (dao *goodsDao) listGoods() (err error) {
	var (
		etcdGetResp *clientv3.GetResponse
		kvPair      *mvccpb.KeyValue
		goodsList   []*goods.Goods
	)
	goodsList = make([]*goods.Goods, 0)
	etcdGetResp, err = envinit.G_envInit.EtcdClient.KV.Get(context.Background(), dao.goodsKey)
	if err != nil {
		logs.Error("seckill proxy get key [%s] from etcd err: %v", dao.goodsKey, err)
		return
	}
	for _, kvPair = range etcdGetResp.Kvs {
		err = json.Unmarshal(kvPair.Value, &goodsList)
		if err != nil {
			logs.Error("seckill proxy json Unmarshal seckill goods err: %v", err)
		}
	}
	dao.updateGoodsMap(goodsList)
	return
}

// TODO 优化 goodsList []goods.Goods slice 的更新不是很方便 商品单个化
func (dao *goodsDao) updateGoodsMap(goodsList []*goods.Goods) {
	// 这里不加锁也没问题 这是程序启动 main协程里跑的 其他后台goroutine还没有启动 为了保险 加读写锁

	// 这样写数据量 几十个 几百个 商品问题不大;数据量比较大 几万个 十几万个 加锁的时间会比较长
	//	dao.rwLock.Lock()
	//	for _,goods = range goodsList{
	//		dao.GoodsMap[goods.GoodsId] = &goods
	//	}
	//	dao.rwLock.Unlock()

	// 优化
	// 填充这个临时的Map不需要加锁 只有1个子协程在操作临时Map
	// 临时Map不加锁 不会影响main协程的逻辑处理
	var tmpMap = make(map[int]*goods.Goods, 1024)

	// TODO NOTICE for...range loop variable reuse err
	//	for _,goods = range goodsList{
	//		tmpMap[goods.GoodsId] = &goods
	//	}

	for i := 0; i < len(goodsList); i++ {
		tmpMap[goodsList[i].GoodsId] = goodsList[i]
	}

	// 在需要更新数据的时候加锁 程序性能会提升很多倍 锁住的仅仅是1次赋值操作
	// Map是引用类型 Map赋值只是把地址拷贝 轻量级拷贝
	// dao.rwLock.Lock()
	dao.RWLock.Lock()
	dao.GoodsMap = tmpMap
	// dao.rwLock.Unlock()
	dao.RWLock.Unlock()
}

// 观察etcd中秒杀商品配置变化 实时更新商品状态
// NOTICE goodsKey --> []*goods.Goods 非 单个Goods 待优化
func (dao *goodsDao) watchGoodsKey() (err error) {
	var (
		goodsList []*goods.Goods
		// 是否从etcd读取到秒杀商品配置
		isGetGoodsList bool
		watchChan      clientv3.WatchChan
		watchResp      clientv3.WatchResponse
		watchEvent     *clientv3.Event
	)
	goodsList = make([]*goods.Goods, 0)

	// 1.获取 /solozyx/seckill/goods 目录下所有秒杀商品
	// 获取当前etcd集群Revision
	logs.Debug("seckill proxy begin watch etcd goodsKey: %s", dao.goodsKey)

	// 2.从该Revision开始向后监听kv变化事件 启动监听协程
	go func() {
		// 监听etcd集群变化 {goodsKey} 后续变化
		// NOTICE channel在etcd设计中的表现
		watchChan = dao.etcdWatcher.Watch(context.Background(), dao.goodsKey, clientv3.WithPrefix())
		for watchResp = range watchChan {
			// 每次etcd监听返回的是多个Event事件
			// 每个Event是不同key的事件 不一定是同一个key
			// Event有类型
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.DELETE:
					// 删除了1个商品
					logs.Warn("seckill proxy etcd delete goodsKey: %s", dao.goodsKey)
					isGetGoodsList = false
					continue
				case mvccpb.PUT:
					// 新建 修改 获取 goodKey 在etcd中最新的数据
					if string(watchEvent.Kv.Key) == dao.goodsKey {
						if err = json.Unmarshal(watchEvent.Kv.Value, &goodsList); err != nil {
							// json字符串反序列化失败 非法json 一般不会出现 静默处理 继续监听后续变化
							logs.Error("seckill proxy goodsKey = []*goods.Goods json Unmarshal err: %v", err)
							isGetGoodsList = false
							continue
						}
						isGetGoodsList = true
						logs.Debug("seckill proxy etcd get goodsList are: ", goodsList)
					}
				}
			}
			if isGetGoodsList {
				// 从etcd获取到goodsKey对应的商品列表 直接赋值 也可以
				// 但是直接赋值有问题 这里 go func(){}() 后台协程再跑
				// 生产线上运行之前配置好的 秒杀商品
				// main.main()协程 ListGoods() 的列表 和 go func(){}() 的列表
				// 在不同的协程 存在多协程并发抢占1个变量的问题
				// 直接赋值 有多线程问题 在 go func(){}() 在这个后台把
				// 新的列表准备好 然后再去更新线上配置
				// 避免多协程竞争问题 加锁
				dao.updateGoodsMap(goodsList)
			}
		}
	}()
	return
}
