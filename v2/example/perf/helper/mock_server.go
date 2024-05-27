package helper

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type delayFunc struct {
	delayInMs int
}

func (d *delayFunc) server() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(d.delayInMs) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}
}

func (d *delayFunc) path() string {
	return fmt.Sprintf("/delay/%d_ms", d.delayInMs)
}

func StartMockServer(ctx context.Context, port int) (chan bool, error) {
	var server *http.Server
	var err error
	completed := make(chan bool, 2)

	go func() {
		mux := http.NewServeMux()
		server = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}

		delayFuncObj10Ms := delayFunc{delayInMs: 10}
		mux.HandleFunc(delayFuncObj10Ms.path(), delayFuncObj10Ms.server())

		if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		completed <- true
		close(completed)
	}()

	go func() {
		select {
		case <-ctx.Done():
			_ = server.Shutdown(context.Background())
		}
	}()

	return completed, err
}
