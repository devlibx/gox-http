package main

import (
	"context"
	"fmt"
	"github.com/devlibx/gox-http/example/perf/helper"
	"time"
)

func main() {
	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Duration(2*time.Second))
	defer cancelFunc()

	overChannel, err := helper.StartMockServer(ctx, 4567)
	if err != nil {
		panic(err)
	}

	<-overChannel
	fmt.Println("Done...")
}
