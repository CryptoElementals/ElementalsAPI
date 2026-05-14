package redis

const (
	XADD_COMMAND                  = "XADD"
	XDEL_COMMAND                  = "XDEL"
	XLEN_COMMAND                  = "XLEN"
	XRANGE_COMMAND                = "XRANGE"
	XREVRANGE_COMMAND             = "XREVRANGE"
	XREAD_COMMAND                 = "XREAD"
	XTRIM_COMMAND                 = "XTRIM"
	XGROUP_COMMAND                = "XGROUP"
	XREADGROUP_COMMAND            = "XREADGROUP"
	XACK_COMMAND                  = "XACK"
	XPENDING_COMMAND              = "XPENDING"
	XCLAIM_COMMAND                = "XCLAIM"
	XAUTOCLAIM_COMMAND            = "XAUTOCLAIM"
	XINFO_COMMAND                 = "XINFO"
	XGROUP_CREATE_SUBCOMMAND      = "CREATE"
	XGROUP_DESTROY_SUBCOMMAND     = "DESTROY"
	XGROUP_DELCONSUMER_SUBCOMMAND = "DELCONSUMER"
	MKSTREAM_OPTION               = "MKSTREAM"
	STREAMS_OPTION                = "STREAMS"
	BLOCK_OPTION                  = "BLOCK"
	COUNT_OPTION                  = "COUNT"
	MINID_OPTION                  = "MINID"
	MAXLEN_OPTION                 = "MAXLEN"
	APPROX_OPTION                 = "~"
)

func XAdd(stream string, id string, fields map[string]interface{}) (string, error) {
	return mustDefault().XAdd(stream, id, fields)
}

func XDel(stream string, ids ...string) (int, error) {
	return mustDefault().XDel(stream, ids...)
}

func XLen(stream string) (int, error) {
	return mustDefault().XLen(stream)
}

func XRange(stream string, start string, end string, count int) ([]interface{}, error) {
	return mustDefault().XRange(stream, start, end, count)
}

func XRevRange(stream string, end string, start string, count int) ([]interface{}, error) {
	return mustDefault().XRevRange(stream, end, start, count)
}

func XRead(stream string, startID string, count int, blockMs int) ([]interface{}, error) {
	return mustDefault().XRead(stream, startID, count, blockMs)
}

func XTrimMaxLen(stream string, maxLen int, approximate bool) (int, error) {
	return mustDefault().XTrimMaxLen(stream, maxLen, approximate)
}

func XTrimMinID(stream string, minID string) (int, error) {
	return mustDefault().XTrimMinID(stream, minID)
}

func XGroupCreate(stream string, group string, startID string, mkstream bool) error {
	return mustDefault().XGroupCreate(stream, group, startID, mkstream)
}

func XGroupDestroy(stream string, group string) (int, error) {
	return mustDefault().XGroupDestroy(stream, group)
}

func XGroupDelConsumer(stream string, group string, consumer string) (int, error) {
	return mustDefault().XGroupDelConsumer(stream, group, consumer)
}

func XReadGroup(group string, consumer string, stream string, id string, count int, blockMs int) ([]interface{}, error) {
	return mustDefault().XReadGroup(group, consumer, stream, id, count, blockMs)
}

func XAck(stream string, group string, ids ...string) (int, error) {
	return mustDefault().XAck(stream, group, ids...)
}

func XPending(stream string, group string) ([]interface{}, error) {
	return mustDefault().XPending(stream, group)
}

func XClaim(stream string, group string, consumer string, minIdleTimeMs int, ids ...string) ([]interface{}, error) {
	return mustDefault().XClaim(stream, group, consumer, minIdleTimeMs, ids...)
}

// XAutoClaim runs XAUTOCLAIM key group consumer min-idle-time start [COUNT count].
// Returns the raw Redis array: [next-start, entries...] (entries in XRANGE shape).
func XAutoClaim(stream string, group string, consumer string, minIdleTimeMs int, start string, count int) ([]interface{}, error) {
	return mustDefault().XAutoClaim(stream, group, consumer, minIdleTimeMs, start, count)
}

func XInfoStream(stream string) ([]interface{}, error) {
	return mustDefault().XInfoStream(stream)
}
