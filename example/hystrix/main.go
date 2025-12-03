package main

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/devlibx/gox-base/errors"
	"sync"
	"sync/atomic"
	"time"
)

func main() {

	hystrix.ConfigureCommand("test", hystrix.CommandConfig{
		Timeout:                100,
		MaxConcurrentRequests:  1000,
		RequestVolumeThreshold: 100,
		SleepWindow:            5000,
		ErrorPercentThreshold:  50,
	})

	var maxCount int32 = 10000
	var count int32
	var errorCount int32
	wg := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for {
				c := atomic.AddInt32(&count, 1)
				if c >= maxCount {
					wg.Done()
					break
				}
				err := hystrix.Do("test", func() error {
					time.Sleep(10 * time.Millisecond)
					if c%2 == 0 {
						return errors.New("error")
					}
					return nil
				}, nil)
				if err != nil {
					fmt.Println(err.Error())
					atomic.AddInt32(&errorCount, 1)
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("max count: %d, error count: %d\n", maxCount, errorCount)
}
