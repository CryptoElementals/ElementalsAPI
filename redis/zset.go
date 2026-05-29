package redis

const (
	ZADD_COMMAND              = "ZADD"
	ZREM_COMMAND              = "ZREM"
	ZSCORE_COMMAND            = "ZSCORE"
	ZCARD_COMMAND             = "ZCARD"
	ZCOUNT_COMMAND            = "ZCOUNT"
	ZRANGE_COMMAND            = "ZRANGE"
	ZREVRANGE_COMMAND         = "ZREVRANGE"
	ZRANGEBYSCORE_COMMAND     = "ZRANGEBYSCORE"
	ZREVRANGEBYSCORE_COMMAND  = "ZREVRANGEBYSCORE"
	ZRANK_COMMAND             = "ZRANK"
	ZREVRANK_COMMAND          = "ZREVRANK"
	ZINCRBY_COMMAND           = "ZINCRBY"
)

func ZAdd(key string, score float64, member interface{}) (int, error) {
	return mustDefault().ZAdd(key, score, member)
}

func ZRem(key string, members ...interface{}) (int, error) {
	return mustDefault().ZRem(key, members...)
}

func ZScore(key string, member interface{}) (float64, error) {
	return mustDefault().ZScore(key, member)
}

// ZScoreMemberExists reports whether member is present in the sorted set (ZSCORE is not nil).
func ZScoreMemberExists(key string, member interface{}) (bool, error) {
	return mustDefault().ZScoreMemberExists(key, member)
}

// ZScoreInt64IfMember returns the score as int64 when the member exists (e.g. lobby queue ZSET scores are join time in milliseconds).
func ZScoreInt64IfMember(key string, member interface{}) (score int64, ok bool, err error) {
	return mustDefault().ZScoreInt64IfMember(key, member)
}

func ZCard(key string) (int, error) {
	return mustDefault().ZCard(key)
}

func ZCount(key string, min interface{}, max interface{}) (int, error) {
	return mustDefault().ZCount(key, min, max)
}

func ZRange(key string, start int, stop int) ([]string, error) {
	return mustDefault().ZRange(key, start, stop)
}

func ZRevRange(key string, start int, stop int) ([]string, error) {
	return mustDefault().ZRevRange(key, start, stop)
}

func ZRangeByScore(key string, min interface{}, max interface{}, offset int, count int) ([]string, error) {
	return mustDefault().ZRangeByScore(key, min, max, offset, count)
}

func ZRevRangeByScore(key string, max interface{}, min interface{}, offset int, count int) ([]string, error) {
	return mustDefault().ZRevRangeByScore(key, max, min, offset, count)
}

func ZRank(key string, member interface{}) (int, error) {
	return mustDefault().ZRank(key, member)
}

func ZRevRank(key string, member interface{}) (int, error) {
	return mustDefault().ZRevRank(key, member)
}

func ZIncrBy(key string, increment float64, member interface{}) (float64, error) {
	return mustDefault().ZIncrBy(key, increment, member)
}
