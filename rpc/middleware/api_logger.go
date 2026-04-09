package middleware

import (
	"context"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"google.golang.org/grpc"
)

func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	if err != nil {
		log.Errorw("rpc called", "method", info.FullMethod, "req", types.ToJsonLoggable(req), "err", err, "duration", duration.Seconds())
	} else {
		log.Debugw("rpc called", "method", info.FullMethod, "req", types.ToJsonLoggable(req), "resp", types.ToJsonLoggable(resp), "duration", duration.Seconds())
	}

	return resp, err
}
