package domain

type User struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Password   string `json:"password"`
	CreateTime int64  `json:"create_time"`
}

type CreateUserReq struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginReq struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
type LoginResp struct {
	Token string `json:"token"`
}
