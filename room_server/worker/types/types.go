package types

import (
	"encoding/json"
	"fmt"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

type PlayerAddress struct {
	Id               int64
	TemporaryAddress string
}

func NewPlayerAddress(id int64, temporaryAddress string) *PlayerAddress {
	return &PlayerAddress{
		Id:               id,
		TemporaryAddress: strings.ToLower(temporaryAddress),
	}
}

func (a *PlayerAddress) String() string {
	return fmt.Sprintf("%d_%s", a.Id, a.TemporaryAddress)
}

func (a *PlayerAddress) Parse(str string) error {
	parts := strings.Split(str, "_")
	if len(parts) != 2 {
		return fmt.Errorf("invalid player address")
	}
	var id int64
	_, err := fmt.Sscanf(parts[0], "%d", &id)
	if err != nil {
		return fmt.Errorf("invalid player id: %w", err)
	}
	a.Id = id
	a.TemporaryAddress = strings.ToLower(parts[1])
	return nil
}

func (a *PlayerAddress) ToDao() *dao.GamePlayerInfo {
	return &dao.GamePlayerInfo{
		PlayerId:         a.Id,
		TemporaryAddress: a.TemporaryAddress,
	}
}

func (a *PlayerAddress) ToProto() *proto.PlayerAddress {
	return &proto.PlayerAddress{
		Id:               a.Id,
		TemporaryAddress: a.TemporaryAddress,
	}
}

func (a *PlayerAddress) ToProtoNoWallet() *proto.PlayerAddress {
	return &proto.PlayerAddress{
		Id:               0, // Id not included when using ToProtoNoWallet
		TemporaryAddress: strings.ToLower(a.TemporaryAddress),
	}
}

func (a *PlayerAddress) FromDao(player dao.GamePlayerInfo) {
	a.Id = player.PlayerId
	a.TemporaryAddress = strings.ToLower(player.TemporaryAddress)
}

func (a *PlayerAddress) FromProto(player *proto.PlayerAddress) {
	a.Id = player.Id
	a.TemporaryAddress = strings.ToLower(player.TemporaryAddress)
}

func ToJsonLoggable(obj any) string {
	res, _ := json.Marshal(obj)
	return string(res)
}

func ToJsonLoggableIndent(obj any) string {
	res, _ := json.MarshalIndent(obj, "", "  ")
	return string(res)
}
