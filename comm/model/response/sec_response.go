package response

type SecResponse struct {
	UserId  int
	GoodsId int
	Code    int
	// 用于在蜜小蜂APP优惠费率借款 或商城系统加购物车
	Token string
	// 有效期
	TokenTime int64
}
