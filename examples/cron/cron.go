package main

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/cron"
)

func main() {
	s := cron.NewScheduler()
	s.Register("t1", time.Second*5, func(ctx context.Context) {
		fmt.Println("fire")
	}, true)
	ctx, ccl := context.WithTimeout(context.Background(), time.Second*10)
	defer ccl()
	s.Start(ctx)
	time.Sleep(time.Second * 10)
}
