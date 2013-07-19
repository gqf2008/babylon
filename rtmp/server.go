package rtmp

import (
	"bufio"
	log "github.com/cihub/seelog"
	"net"
	"runtime"
	"sync"
	"time"
)

var shandler ServerHandler = new(DefaultServerHandler)

func ListenAndServe(addr string) error {
	srv := &Server{
		Addr:         addr,
		ReadTimeout:  time.Duration(time.Second * 30),
		WriteTimeout: time.Duration(time.Second * 30),
		Lock:         new(sync.Mutex)}
	return srv.ListenAndServe()
}

type Server struct {
	Addr         string        //监听地址
	ReadTimeout  time.Duration //读超时
	WriteTimeout time.Duration //写超时
	Lock         *sync.Mutex
}

var gstreamid = uint32(64)

func gen_next_stream_id(chunkid uint32) uint32 {
	gstreamid += 1
	return gstreamid
}

func newconn(conn net.Conn, srv *Server) (c *RtmpNetConnection) {
	c = new(RtmpNetConnection)
	c.remoteAddr = conn.RemoteAddr().String()
	c.server = srv
	c.readChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.writeChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.createTime = time.Now()
	c.bandwidth = 512 << 10
	c.conn = conn
	c.br = bufio.NewReader(conn)
	c.bw = bufio.NewWriter(conn)
	c.buf = bufio.NewReadWriter(c.br, c.bw)
	c.lock = new(sync.Mutex)
	//c.csid_chunk = make(map[uint32]uint32)
	c.lastReadHeaders = make(map[uint32]*RtmpHeader)
	c.lastWriteHeaders = make(map[uint32]*RtmpHeader)
	c.incompletePackets = make(map[uint32]Payload)
	c.nextStreamId = gen_next_stream_id
	c.objectEncoding = 0
	return
}

func (p *Server) ListenAndServe() error {
	addr := p.Addr
	if addr == "" {
		addr = ":1935"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		go p.loop(l)
	}
	return nil
}

func (srv *Server) loop(l net.Listener) error {
	defer l.Close()
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		grw, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Errorf("rtmp: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		go serve(srv, grw)
	}
}

func serve(srv *Server, con net.Conn) {
	conn := newconn(con, srv)
	if !handshake1(conn.buf) {
		conn.Close()
		return
	}
	log.Debug("readMessage")
	msg, err := readMessage(conn)
	if err != nil {
		log.Error("NetConnecton read error", err)
		conn.Close()
		return
	}
	cmd, ok := msg.(*ConnectMessage)
	if !ok || cmd.Command != "connect" {
		log.Error("NetConnecton Received Invalid ConnectMessage ", msg)
		conn.Close()
		return
	}
	conn.app = getString(cmd.Object, "app")
	conn.objectEncoding = int(getNumber(cmd.Object, "objectEncoding"))
	log.Debug(cmd)
	err = sendAckWinsize(conn, 512<<10)
	if err != nil {
		log.Error("NetConnecton sendAckWinsize error", err)
		conn.Close()
		return
	}
	err = sendPeerBandwidth(conn, 512<<10)
	if err != nil {
		log.Error("NetConnecton sendPeerBandwidth error", err)
		conn.Close()
		return
	}
	err = sendStreamBegin(conn)
	if err != nil {
		log.Error("NetConnecton sendStreamBegin error", err)
		conn.Close()
		return
	}
	err = sendConnectSuccess(conn)
	if err != nil {
		log.Error("NetConnecton sendConnectSuccess error", err)
		conn.Close()
		return
	}
	conn.connected = true
	newNetStream(conn, shandler, nil).serve()
}

func getNumber(obj interface{}, key string) float64 {
	if v, exist := obj.(Map)[key]; exist {
		return v.(float64)
	}
	return 0.0
}
