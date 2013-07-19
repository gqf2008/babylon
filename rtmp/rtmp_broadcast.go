package rtmp

import (
	log "github.com/cihub/seelog"
	"sync"
	"time"
)

var (
	broadcasts = make(map[string]*Broadcast)
)

type Terminal chan bool

func newTerminal() Terminal {
	return make(Terminal, 1)
}

func start_broadcast(producer *RtmpNetStream, vl, al int) {
	in := newChannel(producer.conn.remoteAddr, vl, al)
	producer.AttachAudio(in.audioChannel)
	producer.AttachVideo(in.videoChannel)
	d := &Broadcast{path: producer.path,
		lock:      new(sync.Mutex),
		consumers: make(map[string]*RtmpNetStream, 0),
		producer:  producer,
		control:   make(chan interface{}, 10)}
	broadcasts[producer.path] = d
	d.start()
}

func find_broadcast(path string) (*Broadcast, bool) {
	v, ok := broadcasts[path]
	return v, ok
}

func newChannel(id string, vl, al int) Channel {
	return Channel{id: id,
		videoChannel: make(VideoChannel, vl),
		audioChannel: make(AudioChannel, al)}
}

type Channel struct {
	id           string
	videoChannel VideoChannel //StreamChannel
	audioChannel AudioChannel //StreamChannel
}

type Broadcast struct {
	lock      *sync.Mutex
	path      string
	producer  *RtmpNetStream
	out       Channel
	consumers map[string]*RtmpNetStream
	control   chan interface{}
	//terminal  Terminal
}

func (p *Broadcast) stop() {
	//p.terminal <- true
	delete(broadcasts, p.path)
	p.control <- "stop"
}

func (p *Broadcast) start() {
	//p.terminal = newTerminal()
	go func(p *Broadcast) {
		defer func() {
			if e := recover(); e != nil {
				log.Critical(e)
			}
			log.Info("Broadcast " + p.path + " stopped")
		}()
		log.Info("Broadcast " + p.path + " started")
		for {
			select {
			case amsg := <-p.producer.audiochan:
				for _, s := range p.consumers {
					err := s.SendAudio(amsg.Clone())
					if err != nil {
						notifyError(s, err)
					}
				}
			case vmsg := <-p.producer.videochan:
				for _, s := range p.consumers {
					err := s.SendVideo(vmsg.Clone())
					if err != nil {
						notifyError(s, err)
					}
				}
			case obj := <-p.control:
				if c, ok := obj.(*RtmpNetStream); ok {
					if c.closed {
						delete(p.consumers, c.conn.remoteAddr)
						log.Debugf("Broadcast %v consumers %v", p.path, len(p.consumers))
					} else {
						p.consumers[c.conn.remoteAddr] = c
						log.Debugf("Broadcast %v consumers %v", p.path, len(p.consumers))
					}
				} else if v, ok := obj.(string); ok && "stop" == v {
					for k, ss := range p.consumers {
						delete(p.consumers, k)
						ss.Close()
					}
					return
				}
			case <-time.After(time.Second * 90):
				log.Warn("Broadcast " + p.path + " Video | Audio Buffer Empty,Timeout 30s")
				p.stop()
				p.producer.Close()
				return
			}
		}
	}(p)

}

func (p *Broadcast) addConsumer(s *RtmpNetStream) {
	s.dispatcher = p
	p.control <- s
}
func (p *Broadcast) removeConsumer(s *RtmpNetStream) {
	s.closed = true
	p.control <- s
}
