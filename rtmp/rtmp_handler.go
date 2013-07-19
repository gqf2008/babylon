package rtmp

import (
	"errors"
	log "github.com/cihub/seelog"
)

type ServerHandler interface {
	OnPublishing(s *RtmpNetStream) error
	OnPlaying(s *RtmpNetStream) error
	OnClosed(s *RtmpNetStream)
	OnError(s *RtmpNetStream, err error)
}

type ClientHandler interface {
	OnPublishStart(s *RtmpNetStream) error
	OnPlayStart(s *RtmpNetStream) error
	OnClosed(s *RtmpNetStream)
	OnError(s *RtmpNetStream, err error)
}

type DefaultClientHandler struct {
}

func (this *DefaultClientHandler) OnPublishStart(s *RtmpNetStream) error {
	return nil
}
func (this *DefaultClientHandler) OnPlayStart(s *RtmpNetStream) error {
	if _, ok := find_broadcast(s.path); ok {
		return errors.New("NetStream.Play.BadName")
	}
	start_broadcast(s, 5, 5)
	return nil
}
func (this *DefaultClientHandler) OnClosed(s *RtmpNetStream) {
	log.Infof("RtmpNetStream %s %s closed", s.conn.remoteAddr, s.path)
	if d, ok := find_broadcast(s.path); ok {
		if s.mode == MODE_PRODUCER {
			d.stop()
		} else if s.mode == MODE_CONSUMER {
			d.removeConsumer(s)
		}
	}
}
func (this *DefaultClientHandler) OnError(s *RtmpNetStream, err error) {
	log.Errorf("RtmpNetStream %s %s %+v", s.conn.remoteAddr, s.path, err)
	s.Close()
}

type DefaultServerHandler struct {
}

func (p *DefaultServerHandler) OnPublishing(s *RtmpNetStream) error {
	if _, ok := find_broadcast(s.path); ok {
		return errors.New("NetStream.Publish.BadName")
	}
	start_broadcast(s, 5, 5)
	return nil
}

func (p *DefaultServerHandler) OnPlaying(s *RtmpNetStream) error {
	if d, ok := find_broadcast(s.path); ok {
		d.addConsumer(s)
		return nil
	}
	return errors.New("NetStream.Play.StreamNotFound")

}

func (p *DefaultServerHandler) OnClosed(s *RtmpNetStream) {
	mode := "UNKNOWN"
	if s.mode == MODE_CONSUMER {
		mode = "CONSUMER"
	} else if s.mode == MODE_PROXY {
		mode = "PROXY"
	} else if s.mode == MODE_CONSUMER|MODE_PRODUCER {
		mode = "PRODUCER|CONSUMER"
	}
	log.Infof("RtmpNetStream %v %s %s closed", mode, s.conn.remoteAddr, s.path)
	if d, ok := find_broadcast(s.path); ok {
		if s.mode == MODE_PRODUCER {
			d.stop()
		} else if s.mode == MODE_CONSUMER {
			d.removeConsumer(s)
		} else if s.mode == MODE_CONSUMER|MODE_PRODUCER {
			d.removeConsumer(s)
			d.stop()
		}
	}
}

func (p *DefaultServerHandler) OnError(s *RtmpNetStream, err error) {

	log.Errorf("RtmpNetStream %s %s %+v", s.conn.remoteAddr, s.path, err)
	s.Close()
}
