package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"time"

	"go-net-demo/tcp"
)

func main() {

	var network, uds string
	var keepalive int64
	var port, numEventLoop, metricsPort, httpPort, mode int
	var multicore, reuseport, reuseaddr, async, test, debug, save bool

	flag.StringVar(&network, "network", "tcp", "--network=tcp")
	flag.StringVar(&uds, "uds", "*", "--uds=/var/acl.sock")
	flag.IntVar(&metricsPort, "metrics", 8081, "--metrics=8888")
	flag.IntVar(&port, "port", 9002, "--port=9002")
	flag.IntVar(&numEventLoop, "event", 64, "--event=64, numbers of event loop")
	flag.IntVar(&httpPort, "http", 8090, "--http=8090")
	flag.IntVar(&mode, "mode", 0, "--mode=0")
	flag.BoolVar(&multicore, "multicore", false, "--multicore=true")
	flag.BoolVar(&reuseport, "reuseport", true, "--reuseport=true")
	flag.BoolVar(&reuseaddr, "reuseaddr", true, "--reuseaddr=true")
	flag.BoolVar(&async, "async", false, "--async=true")
	flag.Int64Var(&keepalive, "keepalive", 600, "--keepalive=60")
	flag.BoolVar(&test, "test", false, "--test=true")
	flag.BoolVar(&debug, "pprof", false, "--pprof=true")
	flag.BoolVar(&save, "save", false, "--save=true")
	flag.Parse()

	// tcp server
	switch mode {
	case 0:
		server := &tcp.Server{
			Multicore:    multicore,
			ReUsePort:    reuseport,
			ReUseAddr:    reuseaddr,
			Async:        async,
			NumEventLoop: numEventLoop,
			KeepAlive:    time.Duration(keepalive) * time.Second,
		}
		if port != 0 {
			server.Addr = fmt.Sprintf("%s://:%d", network, port)
		} else if uds != "*" {
			server.Addr = fmt.Sprintf("unix://%s", uds)
		} else {
			return
		}
		go server.Run()
		defer server.Close()
	case 1:
		//evio := tcp.Evio{
		//	Port:      port,
		//	Reuseport: true,
		//}
		//go evio.Serve()
	case 2:
		netpoll := tcp.NetPoll{Port: port}
		go netpoll.Serve()
		defer netpoll.Stop()
	case 3:
		std := tcp.Standard{Port: port}
		go std.Serve()
	//defer std.Close()
	case 4:
		n := tcp.NBIO{Port: port}
		go n.Serve()
		defer n.Stop()
	}
	ch := make(chan int, 1)
	<-ch
}
