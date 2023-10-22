package tcp

import (
	"fmt"
	"github.com/lesismal/nbio"
	"go-net-demo/http"
	"go-net-demo/logger"
)

type NBIO struct {
	Port int

	eng *nbio.Engine
}

func (n *NBIO) Serve() {
	n.eng = nbio.NewEngine(nbio.Config{
		Network:            "tcp",
		Addrs:              []string{fmt.Sprintf(":%d", n.Port)},
		MaxWriteBufferSize: 6 * 1024 * 1024,
	})

	n.eng.OnData(n.process)

	err := n.eng.Start()
	if err != nil {
		fmt.Printf("nbio.Start failed: %v\n", err)
		return
	}
}
func (n *NBIO) Stop() {
	n.eng.Stop()
}

func (n *NBIO) process(conn *nbio.Conn, data []byte) {
	var err error
	var request []byte
	var req http.Request

	//now := time.Now()
	codec := Codec{}
	remote := conn.RemoteAddr().String()

	defer func() {
		conn.Close()
	}()

	buf := data
	for {
		buf, request, err = codec.DecodeBytes(data, buf)
		if err == ErrEmptyPackage {
			break
		}

		if err == ErrIncompletePacket {
			logger.Logf(logger.VERBOSE, "incomplete packet from `%s`, close this connection", remote)
			if _, err = conn.Write([]byte("OK;" + err.Error() + "\n")); err != nil {
			}
			_ = conn.Close()
		}

		if err != nil {
			logger.Logf(logger.VERBOSE, "failed to decode tcp from `%s`: %v", remote, err)
			conn.Close()
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

		code = "NO"
		reason = "test"

		if _, err = conn.Write([]byte(string(code) + ";" + reason + "\n")); err != nil {
			logger.Logf(logger.ERROR, "failed to write to conn `%s`: %v", conn.RemoteAddr().String(), err)
		}

	}
}
