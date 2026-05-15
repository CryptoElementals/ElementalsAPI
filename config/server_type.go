package config

import dao "github.com/CryptoElementals/common/models"

const (
	ServerTypeTrial  = dao.ServerTypeTrial
	ServerTypeNormal = dao.ServerTypeNormal
)

const (
	DefaultServerTypeForNewUser      = dao.DefaultServerTypeForNewUser
	DefaultServerTypeForExistingUser = dao.DefaultServerTypeForExistingUser
)

// NormalizeServerType returns a valid type, defaulting empty/unknown to trial.
func NormalizeServerType(serverType string) string {
	return dao.NormalizeServerType(serverType)
}

// RequiredEnvironmentNames are the environment entries apiserver must configure.
func RequiredEnvironmentNames() []string {
	return []string{ServerTypeTrial, ServerTypeNormal}
}
