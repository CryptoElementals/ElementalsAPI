package conversion

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbUserTokenToProtoGetPlayerTokenResponse(userToken *dao.UserToken) *proto.GetPlayerTokenResponse {
	lockedPoints := 0
	for _, locked := range userToken.LockedTokens {
		lockedPoints += int(locked.TokenAmount)
	}

	return &proto.GetPlayerTokenResponse{
		WalletAddress: userToken.WalletAddress,
		Tokens:        uint64(userToken.TokenAmount),
		Points:        uint64(userToken.Points),
		LockedTokens:  uint64(lockedPoints),
	}
}
