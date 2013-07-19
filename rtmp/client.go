package rtmp

import (
	log "github.com/cihub/seelog"
	"strings"
	"sync"
	"time"
)

var clientHandler ClientHandler = new(DefaultClientHandler)

func HandleClientRTMP(h ClientHandler) {
	clientHandler = h
}

func Connect(url string) (s *RtmpNetStream, err error) {
	//rtmp://host:port/xxx/xxxx
	ss := strings.Split(url, "/")
	addr := ss[0] + "//" + ss[2] + "/" + ss[3]
	log.Debug(addr)
	file := strings.Join(ss[4:], "/")
	log.Debug("file: ", file)
	conn := newNetConnection()

	err = conn.Connect(addr)
	if err != nil {
		return
	}
	s = newNetStream(conn, nil, clientHandler)
	s.Play(file, "live")
	return
}
func newNetConnection() (c *RtmpNetConnection) {
	c = new(RtmpNetConnection)
	c.readChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.writeChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	c.createTime = time.Now()
	c.bandwidth = 512 << 10
	c.lock = new(sync.Mutex)
	//c.csid_chunk = make(map[uint32]uint32)
	c.lastReadHeaders = make(map[uint32]*RtmpHeader)
	c.lastWriteHeaders = make(map[uint32]*RtmpHeader)
	c.incompletePackets = make(map[uint32]Payload)
	c.nextStreamId = gen_next_stream_id
	c.objectEncoding = 0
	return
}
