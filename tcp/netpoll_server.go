package tcp

import (
	"context"
	"fmt"
	"go-net-demo/http"
	"runtime"
	"time"

	"github.com/cloudwego/netpoll"
	"github.com/fatih/color"
	"go-net-demo/logger"
)

type NetPoll struct {
	Port int

	eventLoop netpoll.EventLoop
}

func (np *NetPoll) Serve() {
	netpoll.SetNumLoops(runtime.GOMAXPROCS(0))
	netpoll.DisableGopool()

	listener, err := netpoll.CreateListener("tcp", fmt.Sprintf(":%d", np.Port))
	if err != nil {
		logger.Logf(logger.FATAL, "create netpoll listener failed")
	}

	if np.eventLoop, err = netpoll.NewEventLoop(
		handle,
		netpoll.WithOnPrepare(prepare),
		netpoll.WithReadTimeout(time.Second),
		netpoll.WithIdleTimeout(30*time.Minute),
	); err != nil {
		logger.Logf(logger.FATAL, "failed to new netpoll event loop: %v", err)
	}

	fmt.Printf(color.New(color.FgBlue).SprintFunc()(
		"RUNNING, powered by netpoll.\nStatus of Server:\nListening\t:%d\n"),
		np.Port)

	err = np.eventLoop.Serve(listener)
	if err != nil {
		logger.Logf(logger.FATAL, "failed to run netpoll server: %v", err)
	}
}

func (np *NetPoll) Stop() {
	if np.eventLoop == nil {
		return
	}
	err := np.eventLoop.Shutdown(context.Background())
	if err != nil {
		fmt.Printf("failed to stop netpoll: %v", err)
	}
	fmt.Println("tcp server is stopped")
	fmt.Println("Waiting for 3 seconds for the remaining request detection to be completed")
	time.Sleep(3 * time.Second)
	return
}

func prepare(conn netpoll.Connection) context.Context {
	conn.AddCloseCallback(func(connection netpoll.Connection) error {
		return nil
	})
	return context.Background()
}

func handle(ctx context.Context, conn netpoll.Connection) error {
	var err error
	var request []byte
	var req http.Request

	//now := time.Now()
	codec := Codec{}
	remote := conn.RemoteAddr().String()

	defer func() {
	}()

	for {
		request, err = codec.DecodeNetPoll(conn)
		if err == ErrEmptyPackage {
			break
		}

		if err == ErrIncompletePacket {
			logger.Logf(logger.VERBOSE, "incomplete packet from `%s`, close this connection", remote)
			if _, err = conn.Write([]byte("OK;" + err.Error() + "\n")); err != nil {
			}
			_ = conn.Close()
			return nil
		}

		if err != nil {
			logger.Logf(logger.VERBOSE, "failed to decode tcp from `%s`: %v", remote, err)
			if conn.IsActive() {
				conn.Close()
			}
			return nil
		}

		// serialize request
		//req, err = string2xttp(string(request))
		req, err = http.NewRequest(string(request))
		if err != nil {
			logger.Logf(logger.VERBOSE, "failed to decode http from `%s`: %v", remote, err)
			if _, err = conn.Write([]byte("OK;" + err.Error() + "\n")); err != nil {
			}
			continue
		}

		code, reason, _ := http.Detect(&req)

		if _, err = conn.Write([]byte(string(code) + ";" + reason + "\n")); err != nil {
			logger.Logf(logger.ERROR, "failed to write to conn `%s`: %v", conn.RemoteAddr().String(), err)
		}
	}

	return nil
}
