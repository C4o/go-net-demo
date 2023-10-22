package tcp

import (
	"context"
	"fmt"
	"github.com/panjf2000/gnet/v2"
	"go-net-demo/http"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"go-net-demo/logger"
)

type Server struct {
	gnet.BuiltinEventEngine

	eng          gnet.Engine
	Addr         string
	Multicore    bool
	ReUsePort    bool
	ReUseAddr    bool
	Async        bool
	KeepAlive    time.Duration
	NumEventLoop int

	workerPool *goroutine.Pool
}

func (s *Server) OnBoot(eng gnet.Engine) gnet.Action {
	s.eng = eng
	fmt.Printf(color.New(color.FgBlue).SprintFunc()("RUNNING, powered by gnet. Status of Server:\nListening\t%v\n"+
		"Reuseaddr\t%v\nReuseport\t%v\nAysnc\t\t%v\nMulticore\t%v\nNumEvent\t%d\nKeepalive\t%fs\n"),
		s.Addr, s.ReUseAddr, s.ReUsePort, s.Async, s.Multicore, s.NumEventLoop, s.KeepAlive.Seconds())
	return gnet.None
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	//now := time.Now()
	codec := c.Context().(*Codec)
	var req http.Request
	var rets [][]byte
	var err error

	defer func() {
	}()

	for {
		// TODO: Request decoding optimization
		// TODO: 1. simplified decoding (only host, Path, IP and DedeUserID) for ACL detection
		// TODO: 2. when matching host and uri in anti-cc detection, do full decoding for rule engine
		req, err = codec.Decode(c)
		if err == ErrIncompletePacket || err == ErrEmptyPackage {
			break
		}
		if err != nil {
			//log.Printf("invalid packet: %v", err)
			_, _ = c.Write([]byte(err.Error()))
			return gnet.None
		}

		code, reason, _ := http.Detect(&req)
		rets = append(rets, []byte(string(code)+";"+reason+"\n"))

	}
	if n := len(rets); n > 1 {
		_, _ = c.Writev(rets)
	} else if n == 1 {
		_, _ = c.Write(rets[0])
	} else {
		_, _ = c.Write([]byte("-1"))
	}
	return gnet.None
}

func (s *Server) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	c.SetContext(new(Codec))
	return out, gnet.None
}

func (s *Server) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	return gnet.None
}

func (s *Server) Run() {
	s.workerPool = goroutine.Default()
	err := gnet.Run(
		s,
		s.Addr,
		gnet.WithMulticore(s.Multicore),
		gnet.WithReusePort(s.ReUsePort),
		gnet.WithReuseAddr(s.ReUseAddr),
		gnet.WithTCPKeepAlive(s.KeepAlive),
		gnet.WithLockOSThread(s.Async),
		//gnet.WithLoadBalancing(gnet.LeastConnections),
		gnet.WithNumEventLoop(s.NumEventLoop),
		gnet.WithSocketRecvBuffer(2*1024*1024),
		gnet.WithWriteBufferCap(2*1024*1024),
		gnet.WithReadBufferCap(2*1024*1024),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
	)

	if err != nil {
		logger.Logf(logger.FATAL, "%v", err)
	}
}

func (s *Server) Close() {
	_ = s.eng.Stop(context.Background())
	log.Println("tcp server is stopped")
	log.Println("Waiting for 10 seconds for the remaining request detection to be completed")
	if os.Getenv("DEV_ENV") == "" {
		time.Sleep(10 * time.Second)
	}
}
