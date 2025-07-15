package chain

import (
	"context"
)

type Service struct {
	ctx   context.Context
	chain *Chain
}

func NewService(ctx context.Context) *Service {
	return &Service{ctx: ctx}
}

func (s *Service) ReceiveTransactions(tx [][]byte) error {
	return nil
}
