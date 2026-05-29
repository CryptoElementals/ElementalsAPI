package redis

const (
	LPUSH_COMMAND  = "LPUSH"
	RPUSH_COMMAND  = "RPUSH"
	LPOP_COMMAND   = "LPOP"
	RPOP_COMMAND   = "RPOP"
	LLEN_COMMAND   = "LLEN"
	LRANGE_COMMAND = "LRANGE"
	LREM_COMMAND   = "LREM"
	LSET_COMMAND   = "LSET"
	LINDEX_COMMAND = "LINDEX"
	LTRIM_COMMAND  = "LTRIM"
)

func LPush(key string, values ...interface{}) (int, error) {
	return mustDefault().LPush(key, values...)
}

func RPush(key string, values ...interface{}) (int, error) {
	return mustDefault().RPush(key, values...)
}

func LPop(key string) (string, error) {
	return mustDefault().LPop(key)
}

func RPop(key string) (string, error) {
	return mustDefault().RPop(key)
}

func LLen(key string) (int, error) {
	return mustDefault().LLen(key)
}

func LRange(key string, start int, stop int) ([]string, error) {
	return mustDefault().LRange(key, start, stop)
}

func LRem(key string, count int, value interface{}) (int, error) {
	return mustDefault().LRem(key, count, value)
}

func LSet(key string, index int, value interface{}) error {
	return mustDefault().LSet(key, index, value)
}

func LIndex(key string, index int) (string, error) {
	return mustDefault().LIndex(key, index)
}

func LTrim(key string, start int, stop int) error {
	return mustDefault().LTrim(key, start, stop)
}
