package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/coreos/etcd/clientv3"

	"secondkill/comm/model/goods"
)

var (
	etcdClient *clientv3.Client
	testKey    = "/solozyx/seckill/goods"
)

func etcdInit() {
	config := clientv3.Config{
		Endpoints:   []string{"192.168.174.134:2379"},
		DialTimeout: time.Duration(5) * time.Second,
	}
	client, err := clientv3.New(config)
	if err != nil {
		logs.Error("connect etcd server err: ", err)
	}
	etcdClient = client
}

func clearEtcd() {
	var (
		delResp   *clientv3.DeleteResponse
		err       error
		goodsList []*goods.Goods
	)
	delResp, err = etcdClient.KV.Delete(context.TODO(), testKey, clientv3.WithPrevKV())
	if err != nil {
		logs.Error("clear testKey from etcd err: %v", err)
	}
	// NOTICE 不可 delResp.PrevKvs[0] 取不到 0要删除的key不存在
	if len(delResp.PrevKvs) > 0 {
		fmt.Println("clear etcd success")
		err = json.Unmarshal(delResp.PrevKvs[0].Value, &goodsList)
		if err == nil {
			fmt.Println(string(delResp.PrevKvs[0].Value))
		}
	}
}

func TestEtcdSeckillGoodsWorkFlow(t *testing.T) {
	etcdInit()
	clearEtcd()
	t.Run("addGoodsToEtcd", testAddGoodsToEtcd)
	t.Run("getGoodsFromEtcd", testGetGoodsFromEtcd)
	clearEtcd()
}

func testAddGoodsToEtcd(t *testing.T) {
	var (
		goodsList  []*goods.Goods
		ctx        context.Context
		cancelFunc context.CancelFunc
	)
	goodsList = make([]*goods.Goods, 0)
	goodsList = append(goodsList, &goods.Goods{
		GoodsId:   1001,
		StartTime: time.Now().Unix() + 500,
		EndTime:   time.Now().Unix() + 1000,
		Status:    0,
		Total:     1,
		Left:      1,
	})
	goodsList = append(goodsList, &goods.Goods{
		GoodsId:   1002,
		StartTime: time.Now().Unix() - 500,
		EndTime:   time.Now().Unix() - 200,
		Status:    0,
		Total:     2,
		Left:      2,
	})
	goodsList = append(goodsList, &goods.Goods{
		GoodsId:   1003,
		StartTime: time.Now().Unix() - 500,
		EndTime:   time.Now().Unix() + 200,
		Status:    0,
		Total:     3,
		Left:      3,
	})
	fmt.Println(goodsList)

	data, err := json.Marshal(goodsList)
	if err != nil {
		t.Errorf("json marshal failed ")
	}
	ctx, cancelFunc = context.WithTimeout(context.Background(), 10*time.Second)
	_, err = etcdClient.KV.Put(ctx, testKey, string(data))
	if err != nil {
		t.Errorf("put testKey to etcd err: %v", err)
	}
	cancelFunc()
}

func testGetGoodsFromEtcd(t *testing.T) {
	var etcdGetResp *clientv3.GetResponse
	etcdGetResp, err := etcdClient.KV.Get(context.Background(), testKey)
	if err != nil {
		t.Errorf("get testKey from etcd err: %v", err)
	}
	for k, v := range etcdGetResp.Kvs {
		logs.Info("get key[%s] value[%s] \n", string(k), v)
	}
}
