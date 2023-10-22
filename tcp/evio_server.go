package tcp

import (
	"github.com/tidwall/evio"
)

type Evio struct {
	Port      int
	Reuseport bool

	events evio.Events
}

//func (ev *Evio) Serve() {
//	ev.events = evio.Events{
//		NumLoops: runtime.GOMAXPROCS(0),
//
//		Data: func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
//			var req http.Request
//			var err error
//
//			now := time.Now()
//			codec := Codec{}
//			buf := in
//
//			for {
//				buf, req, err = codec.DecodeBytes(in, buf)
//				if err == ErrIncompletePacket || err == ErrEmptyPackage {
//					break
//				}
//				if err != nil {
//					atomic.AddInt64(&detector.Count5, 1)
//					out = []byte(err.Error())
//					return
//				}
//
//				code, reason, comment := http.Detect(&req)
//				out = append(out, []byte(string(code)+";"+reason+"\n")...)
//
//				// debug
//				if strings.HasPrefix(req.Path, "/announcement/test/") {
//					logger.Logf(logger.DEBUG, "path: %s, cost: %f, code: %s, comment: %s", req.Path, time.Since(now).Seconds(), code, comment)
//				}
//			}
//
//			return out, evio.None
//		},
//	}
//
//	logger.Logf(logger.FATAL, "%v", evio.Serve(ev.events, fmt.Sprintf("tcp://:%d?reuseport=true", ev.Port)))
//}
