// Package snowflake provides a process-wide Snowflake ID generator (github.com/bwmarrin/snowflake).
// Call [Init] or [InitFromConfig] once during application bootstrap before any [GenerateID] (typical single-threaded startup satisfies this).
package snowflake

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/CryptoElementals/common/log"
	sf "github.com/bwmarrin/snowflake"
)

// maxNodeID is the inclusive upper bound for the default bwmarrin layout (10 node bits).
const maxNodeID = 1023

var node *sf.Node

// Init configures the snowflake worker id (0–1023 for the default bit layout). Safe to call more than once;
// later calls are no-ops after the first successful init.
func Init(nodeID int64) error {
	if node != nil {
		return nil
	}
	n, err := sf.NewNode(nodeID)
	if err != nil {
		return err
	}
	node = n
	return nil
}

// InitFromConfig uses nodeID when non-zero; zero picks a cryptographically random id in [0, maxNodeID].
// Returns the resolved node id (for logging) and an error if [Init] fails.
func InitFromConfig(nodeID int64) (resolved int64, err error) {
	if nodeID == 0 {
		resolved, err = randomNodeID()
		if err != nil {
			return 0, fmt.Errorf("snowflake random node id: %w", err)
		}
	} else {
		if nodeID < 0 || nodeID > maxNodeID {
			return 0, fmt.Errorf("snowflake node id out of range [1,%d]: %d", maxNodeID, nodeID)
		}
		resolved = nodeID
	}
	if err := Init(resolved); err != nil {
		return 0, err
	}
	return resolved, nil
}

func randomNodeID() (int64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	u := binary.BigEndian.Uint64(buf[:])
	return int64(u % uint64(maxNodeID+1)), nil
}

// GenerateID returns a new unique int64 snowflake id. Init must have completed successfully first.
func GenerateID() int64 {
	if node == nil {
		log.Fatal("snowflake: Init must be called before GenerateID")
	}
	return node.Generate().Int64()
}
