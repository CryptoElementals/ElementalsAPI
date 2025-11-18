package main

var (
	playerAddress string // Used for legacy token commands (deprecated, use playerId instead)
	playerId      int64  // Used for queue and game commands

	points          int64
	tokens          int64
	playerName      string
	playerAvatarUrl string
	backgroundUrl   string

	endpoint           string
	user               string
	password           string
	dbName             string
	chainRpc           string
	roomServerEndpoint string
	tempWalletPath     string
	gameID             uint
	tempAddress        string
)
