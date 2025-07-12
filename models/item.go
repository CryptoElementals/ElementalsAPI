package dao

type Item struct {
	BaseModel
	Effects []Effect
}

type Effect struct {
	BaseModel
}
