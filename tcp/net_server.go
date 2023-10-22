package tcp

import (
	"bufio"
	"fmt"
	"go-net-demo/http"
	"go-net-demo/logger"
	"net"
)

type Standard struct {
	Port int
}

// TCP Server端测试
// 处理函数
func process(conn net.Conn) {
	var err error
	var request []byte
	var req http.Request

	//now := time.Now()
	codec := Codec{}
	remote := conn.RemoteAddr().String()

	defer func() {
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	for {
		request, err = codec.DecodeNet(remote, reader)
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

func (std *Standard) Serve() {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", std.Port))
	if err != nil {
		fmt.Println("Listen() failed, err: ", err)
		return
	}
	for {
		conn, err := listen.Accept() // 监听客户端的连接请求
		if err != nil {
			fmt.Println("Accept() failed, err: ", err)
			continue
		}
		go process(conn) // 启动一个goroutine来处理客户端的连接请求
	}
}

func (std *Standard) Close() {

}
