package goods

import "secondkill/logic/limite"

// 秒杀商品
// json序列化 反序列化 结构体字段首字母大写
// json标签存储到etcd
// db标签存储到mysql
// orm 结构体tag和mysql数据表做映射
type Goods struct {
	// 商品Id
	GoodsId int `json:"goodsId" db:"id"`
	// 商品名称
	GoodsName string `json:"goodsName" db:"name"`
	// 开始秒杀抢购时间
	StartTime int64 `json:"startTime"`
	// 结束秒杀抢购时间
	EndTime int64 `json:"endTime"`
	// 商品状态
	// 秒杀抢购是一方面 后台也可以强制控制该商品卖光
	// 平台有1000库存 不想抢购这1000库存 平台设置商品状态
	Status int `json:"status" db:"status"`
	// 商品数量总库存量
	Total int `json:"total" db:"total"`
	// 商品当前剩余量
	Left int `json:"left"`

	// 限速控制 该商品每秒秒杀卖出数量
	SecondLimit *limite.GoodsSecondRateLimit `json:"secondLimit"`

	// 该商品秒杀每秒最大卖出数量限制
	SecondSaleMaxLimit int `json:"secondSaleMaxLimit"`
	// 单用户购买数量限制
	OneUserBuyLimit int `json:"oneUserBuyLimit"`

	// 随机抽奖 用户秒杀成功概率 [0.1 - 1.0]
	BuyProbability float64 `json:"buyProbability"`
}
