package game

import (
	"github.com/CryptoElementals/common/rpc/proto"
)

// --- rpc/server.PlayerRequestHandler (game + chain only; queue / pre-game state is on LobbyService) ---

func (s *Service) ConfirmBattle(req *proto.ConfirmBattleRequest) error {
	return s.gameManager.HandleConfirmBattle(req)
}

func (s *Service) Surrender(req *proto.SurrenderRequest) error {
	return s.gameManager.HandleSurrender(req)
}

func (s *Service) SubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error {
	return s.gameManager.HandleSubmitPlayerCommitment(req)
}

func (s *Service) SubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error {
	return s.gameManager.HandleSubmitPlayerCard(req)
}
