package tcp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go-net-demo/http"
	"go-net-demo/logger"
	"strconv"

	"github.com/cloudwego/netpoll"
	"github.com/panjf2000/gnet/v2"
)

var (
	FlagStringBytes     = []byte("WAF0 ")
	FlagStringSize      = len(FlagStringBytes)
	ErrIncompletePacket = errors.New("incomplete packet")
	ErrEmptyPackage     = errors.New("empty package")
	ErrBadSplit         = errors.New("cannot find \\r\\n")
	ErrInvalidFlag      = errors.New("flag string invalid")
	ErrGetSize          = errors.New("failed to get request size")
)

type Codec struct{}

func (codec *Codec) DecodeBytes(rawIn, buffer []byte) (left []byte, request []byte, err error) {
	var buf []byte
	split := []byte("\r\n")
	splitLen := len(split)
	bufferLen := len(buffer)
	// GET / HTTP/2, 至少12长度2
	requestOffset := FlagStringSize + splitLen + 2

	// 获取 "FLAG size\r\n"
	// FLAG 1024\r\nREQUEST\r\FLAG 1024\r\n.....
	for {
		if bufferLen == 0 {
			return nil, nil, ErrEmptyPackage
		}

		if bufferLen < requestOffset {
			return nil, nil, ErrIncompletePacket
		}
		buf = buffer[:requestOffset]

		if bytes.HasSuffix(buf, split) {
			break
		}
		if requestOffset > FlagStringSize+splitLen+10 {
			logger.Logf(logger.VERBOSE, "2, cannot find \\r\\n: whole msg: %s", buffer)
			logger.Logf(logger.VERBOSE, "2.1, whole raw msg: %s", rawIn)
			return nil, nil, errors.New("cannot find \\r\\n")
		}
		requestOffset++
	}

	// `FLAG ` must exist
	if !bytes.Equal(FlagStringBytes, buf[:FlagStringSize]) {
		logger.Logf(logger.VERBOSE, "3, flag should be: %s, but is: %s", FlagStringBytes, buf[:FlagStringSize])
		return nil, nil, errors.New("flag string invalid")
	}
	// parse the size of http package
	requestSize, err := strconv.Atoi(string(buf[FlagStringSize : requestOffset-splitLen]))
	if err != nil {
		logger.Logf(logger.VERBOSE, "4, request size is: %s, buf is: %s", string(buf[FlagStringSize:requestOffset-splitLen]), buf)
		return nil, nil, errors.New("failed to get request size")
	}
	// 检查整包实际长度是否符合
	msgLen := requestOffset + requestSize
	if bufferLen < msgLen {
		logger.Logf(logger.VERBOSE, "8, whole packet too short: %d, %d", bufferLen, msgLen)
		logger.Logf(logger.VERBOSE, "8。1, whole raw packet: %s", rawIn)
		return nil, nil, ErrIncompletePacket
	}
	// 取出对应长度的包内容
	request = buffer[requestOffset:msgLen]
	// discard old data
	return buffer[msgLen:], request, err
}

// Decode 拆包
func (codec *Codec) Decode(c gnet.Conn) (req http.Request, err error) {
	var buf []byte
	split := []byte("\r\n")
	splitLen := len(split)
	// GET / HTTP/2, 至少12位, 所以 +2
	requestOffset := FlagStringSize + splitLen + 2

	// get "FLAG size\r\n"
	// FLAG 1024\r\nHEADER\r\n\r\nBODYFLAG 1024\r\nHEADER\r\n\r\nBODY.....
	for {
		if c.InboundBuffered() == 0 {
			return req, ErrEmptyPackage
		}

		buf, err = c.Peek(requestOffset)
		if len(buf) < requestOffset {
			//buf, _ = c.Next(-1)
			logger.Logf(logger.VERBOSE, "0, len of msg too short, msg is: %s, len of msg is: %d", buf, c.InboundBuffered())
			return req, ErrIncompletePacket
		}
		if err != nil {
			logger.Logf(logger.VERBOSE, "1, failed to peek, offset: %d, msglen: %d", requestOffset, c.InboundBuffered())
			return req, err
		}
		if bytes.HasSuffix(buf, split) {
			break
		}
		if requestOffset > FlagStringSize+splitLen+10 {
			b, e := c.Peek(c.InboundBuffered())
			logger.Logf(logger.VERBOSE, "2, cannot find \\r\\n: whole msg: %s, peek error: %v", b, e)
			return req, errors.New("cannot find \\r\\n")
		}
		requestOffset++
	}

	// `FLAG ` must exist
	if !bytes.Equal(FlagStringBytes, buf[:FlagStringSize]) {
		logger.Logf(logger.VERBOSE, "3, flag should be: %s, but is: %s", FlagStringBytes, buf[:FlagStringSize])
		return req, errors.New("flag string invalid")
	}
	// parse the size of http package
	requestSize, err := strconv.Atoi(string(buf[FlagStringSize : requestOffset-splitLen]))
	if err != nil {
		logger.Logf(logger.VERBOSE, "4, request size is: %s, buf is: %s", string(buf[FlagStringSize:requestOffset-splitLen]), buf)
		return req, errors.New("failed to get request size")
	}
	//logger.Logf(logger.VERBOSE, "5, size of request: %s, %d", buf[FlagStringSize:requestOffset-splitLen], requestSize)
	// 检查整包实际长度是否符合
	msgLen := requestOffset + requestSize
	//if c.InboundBuffered() < msgLen {
	//	logger.Logf(logger.VERBOSE, "6, whole packet too short: %d, %d, %d", c.InboundBuffered(), len(buf), msgLen)
	//	b, e := c.Peek(c.InboundBuffered())
	//	logger.Logf(logger.VERBOSE, "7, whole packet is: %s, %v", b, e)
	//	return req, ErrIncompletePacket
	//}
	// 取出对应长度的包内容
	buf, err = c.Peek(msgLen)
	if err != nil {
		logger.Logf(logger.VERBOSE, "8, failed to peek %d from %d: %v", msgLen, c.InboundBuffered(), err)
		return req, ErrIncompletePacket
	}
	//logger.Logf(logger.VERBOSE, "9, requestOffset is %d, msg len is %d, packet length is %d, %d, %d",
	//	requestOffset, msgLen, c.InboundBuffered(), len(buf), cap(buf))
	raw := buf[requestOffset:msgLen]
	// serialize request
	req, err = http.NewRequest(string(raw))
	//req, err = string2xttp(string(raw))
	if err != nil {
		return req, err
	}
	// discard old data
	_, _ = c.Discard(msgLen)
	return req, err
}

func (codec *Codec) DecodeNetPoll(conn netpoll.Connection) (request []byte, err error) {
	var buf []byte
	split := []byte("\r\n")
	splitLen := len(split)
	c := conn.Reader()
	// GET / HTTP/2, 至少12位, 所以 +2
	requestOffset := FlagStringSize + splitLen + 2

	// get "FLAG size\r\n"
	// FLAG 1024\r\nHEADER\r\n\r\nBODYFLAG 1024\r\nHEADER\r\n\r\nBODY.....
	for {
		if c.Len() == 0 {
			return request, ErrEmptyPackage
		}

		buf, err = c.Peek(requestOffset)
		if len(buf) < requestOffset {
			logger.Logf(logger.VERBOSE, "0, ip: %s, len of msg too short, msg is: %s, len of msg is: %d",
				conn.RemoteAddr().String(), buf, c.Len())
			return request, ErrIncompletePacket
		}
		if err != nil {
			logger.Logf(logger.VERBOSE, "1, ip: %s, failed to peek, offset: %d, msglen: %d",
				conn.RemoteAddr().String(), requestOffset, c.Len())
			return request, err
		}
		if bytes.HasSuffix(buf, split) {
			break
		}
		if requestOffset > FlagStringSize+splitLen+10 {
			b, e := c.Peek(c.Len())
			logger.Logf(logger.VERBOSE, "2, ip: %s, cannot find \\r\\n: whole msg: %s, peek error: %v",
				conn.RemoteAddr().String(), b, e)
			return request, ErrBadSplit
		}
		requestOffset++
	}

	// `FLAG ` must exist
	if !bytes.Equal(FlagStringBytes, buf[:FlagStringSize]) {
		logger.Logf(logger.VERBOSE, "3, ip: %s, flag should be: %s, but is: %s",
			conn.RemoteAddr().String(), FlagStringBytes, buf[:FlagStringSize])
		return request, ErrInvalidFlag
	}
	// parse the size of http package
	requestSize, err := strconv.Atoi(string(buf[FlagStringSize : requestOffset-splitLen]))
	if err != nil {
		logger.Logf(logger.VERBOSE, "4, ip: %s, request size is: %s, buf is: %s",
			conn.RemoteAddr().String(), string(buf[FlagStringSize:requestOffset-splitLen]), buf)
		return request, ErrGetSize
	}
	//logger.Logf(logger.VERBOSE, "5, size of request: %s, %d", buf[FlagStringSize:requestOffset-splitLen], requestSize)
	// 检查整包实际长度是否符合
	msgLen := requestOffset + requestSize
	//if c.InboundBuffered() < msgLen {
	//	logger.Logf(logger.VERBOSE, "6, whole packet too short: %d, %d, %d", c.InboundBuffered(), len(buf), msgLen)
	//	b, e := c.Peek(c.InboundBuffered())
	//	logger.Logf(logger.VERBOSE, "7, whole packet is: %s, %v", b, e)
	//	return req, ErrIncompletePacket
	//}
	// 取出对应长度的包内容
	buf, err = c.Peek(msgLen)
	if err != nil {
		logger.Logf(logger.VERBOSE, "8, ip: %s, failed to peek %d from %d: %v",
			conn.RemoteAddr().String(), msgLen, c.Len(), err)
		return request, ErrIncompletePacket
	}
	//logger.Logf(logger.VERBOSE, "9, requestOffset is %d, msg len is %d, packet length is %d, %d, %d",
	//	requestOffset, msgLen, c.InboundBuffered(), len(buf), cap(buf))
	request = buf[requestOffset:msgLen]

	// discard old data
	var pkg netpoll.Reader
	if pkg, err = c.Slice(msgLen); err != nil {
		logger.Logf(logger.ERROR, "failed to slice connection: %v", err)
		return request, fmt.Errorf("failed to slice connection: %v", err)
	}
	if err = pkg.Release(); err != nil {
		logger.Logf(logger.ERROR, "failed to release package: %v", err)
		return request, fmt.Errorf("failed to release package: %v", err)
	}

	return request, err
}

func (codec *Codec) DecodeNet(remote string, reader *bufio.Reader) (request []byte, err error) {
	var buf []byte
	split := []byte("\r\n")
	splitLen := len(split)
	// GET / HTTP/2, 至少12位, 所以 +2
	requestOffset := FlagStringSize + splitLen + 2

	// get "FLAG size\r\n"
	// FLAG 1024\r\nHEADER\r\n\r\nBODYFLAG 1024\r\nHEADER\r\n\r\nBODY.....
	for {
		if reader.Size() == 0 {
			return request, ErrEmptyPackage
		}

		buf, err = reader.Peek(requestOffset)
		if len(buf) < requestOffset {
			logger.Logf(logger.VERBOSE, "0, ip: %s, len of msg too short, msg is: %s, len of msg is: %d",
				remote, buf, reader.Size())
			return request, ErrIncompletePacket
		}
		if err != nil {
			logger.Logf(logger.VERBOSE, "1, ip: %s, failed to peek, offset: %d, msglen: %d",
				remote, requestOffset, reader.Size())
			return request, err
		}
		if bytes.HasSuffix(buf, split) {
			break
		}
		if requestOffset > FlagStringSize+splitLen+10 {
			b, e := reader.Peek(reader.Size())
			logger.Logf(logger.VERBOSE, "2, ip: %s, cannot find \\r\\n: whole msg: %s, peek error: %v",
				remote, b, e)
			return request, ErrBadSplit
		}
		requestOffset++
	}

	// `FLAG ` must exist
	if !bytes.Equal(FlagStringBytes, buf[:FlagStringSize]) {
		logger.Logf(logger.VERBOSE, "3, ip: %s, flag should be: %s, but is: %s",
			remote, FlagStringBytes, buf[:FlagStringSize])
		return request, ErrInvalidFlag
	}
	// parse the size of http package
	requestSize, err := strconv.Atoi(string(buf[FlagStringSize : requestOffset-splitLen]))
	if err != nil {
		logger.Logf(logger.VERBOSE, "4, ip: %s, request size is: %s, buf is: %s",
			remote, string(buf[FlagStringSize:requestOffset-splitLen]), buf)
		return request, ErrGetSize
	}
	//logger.Logf(logger.VERBOSE, "5, size of request: %s, %d", buf[FlagStringSize:requestOffset-splitLen], requestSize)
	// 检查整包实际长度是否符合
	msgLen := requestOffset + requestSize
	//if c.InboundBuffered() < msgLen {
	//	logger.Logf(logger.VERBOSE, "6, whole packet too short: %d, %d, %d", c.InboundBuffered(), len(buf), msgLen)
	//	b, e := c.Peek(c.InboundBuffered())
	//	logger.Logf(logger.VERBOSE, "7, whole packet is: %s, %v", b, e)
	//	return req, ErrIncompletePacket
	//}
	// 取出对应长度的包内容
	buf, err = reader.Peek(msgLen)
	if err != nil {
		logger.Logf(logger.VERBOSE, "8, ip: %s, failed to peek %d from %d: %v",
			remote, msgLen, reader.Size(), err)
		return request, ErrIncompletePacket
	}
	//logger.Logf(logger.VERBOSE, "9, requestOffset is %d, msg len is %d, packet length is %d, %d, %d",
	//	requestOffset, msgLen, c.InboundBuffered(), len(buf), cap(buf))
	request = buf[requestOffset:msgLen]

	// discard old data
	if _, err = reader.Discard(msgLen); err != nil {
		logger.Logf(logger.ERROR, "failed to discard package: %v", err)
		return request, fmt.Errorf("failed to release package: %v", err)
	}

	return request, err
}
