package redis

const (
	SADD_COMMAND     = "SADD"
	SREM_COMMAND     = "SREM"
	SMOVE_COMMAND    = "SMOVE"
	SMEMBERS_COMMAND = "SMEMBERS"
	SISMEMBER_CMD    = "SISMEMBER"
	SCARD_COMMAND    = "SCARD"
	SPOP_COMMAND     = "SPOP"
)

func SAdd(key string, members ...interface{}) (int, error) {
	return mustDefault().SAdd(key, members...)
}

func SRem(key string, members ...interface{}) (int, error) {
	return mustDefault().SRem(key, members...)
}

func SMove(source, destination string, member interface{}) (bool, error) {
	return mustDefault().SMove(source, destination, member)
}

func SMembers(key string) ([]string, error) {
	return mustDefault().SMembers(key)
}

func SIsMember(key string, member interface{}) (bool, error) {
	return mustDefault().SIsMember(key, member)
}

func SCard(key string) (int, error) {
	return mustDefault().SCard(key)
}

func SPop(key string) (string, error) {
	return mustDefault().SPop(key)
}
