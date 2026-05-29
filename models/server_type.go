package dao

import "strings"

const (
	ServerTypeTrial  = "trial"
	ServerTypeNormal = "normal"
)

// DefaultServerTypeForNewUser is assigned when creating API-server user profiles.
const DefaultServerTypeForNewUser = ServerTypeTrial

// DefaultServerTypeForExistingUser is used when backfilling legacy rows.
const DefaultServerTypeForExistingUser = ServerTypeNormal

// NormalizeServerType returns a valid type, defaulting empty/unknown to trial.
func NormalizeServerType(serverType string) string {
	switch strings.ToLower(strings.TrimSpace(serverType)) {
	case ServerTypeNormal:
		return ServerTypeNormal
	case ServerTypeTrial, "":
		return ServerTypeTrial
	default:
		return ServerTypeTrial
	}
}
