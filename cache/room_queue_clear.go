package cache

// Prefixes must match room_server/worker/queue.Queue (queue_info / locked_token).
const (
	roomQueueInfoPrefix   = "queue_info"
	roomLockedTokenPrefix = "locked_token"
)

// ClearRoomServerQueueAndTokenKeys deletes all keys under the room server matchmaking cache prefixes.
func ClearRoomServerQueueAndTokenKeys(c Cache) (queueKeysRemoved, tokenKeysRemoved int, err error) {
	qc := WithPrefix(roomQueueInfoPrefix, c)
	keys, err := qc.List("")
	if err != nil {
		return 0, 0, err
	}
	for _, k := range keys {
		if err := qc.Delete(k); err != nil {
			return queueKeysRemoved, tokenKeysRemoved, err
		}
		queueKeysRemoved++
	}

	tc := WithPrefix(roomLockedTokenPrefix, c)
	tkeys, err := tc.List("")
	if err != nil {
		return queueKeysRemoved, 0, err
	}
	for _, k := range tkeys {
		if err := tc.Delete(k); err != nil {
			return queueKeysRemoved, tokenKeysRemoved, err
		}
		tokenKeysRemoved++
	}
	return queueKeysRemoved, tokenKeysRemoved, nil
}
