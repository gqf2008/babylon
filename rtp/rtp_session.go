package rtp

import (
	"net"
	"time"
)

type RtpSession struct {
	conn       *net.UDPConn
	seq        int
	sent       int
	recv       int
	createTime time.Time
}

func NewRtpSession(conn *net.UDPConn) *RtpSession {
	session := new(RtpSession)
	session.conn = conn
	session.createTime = time.Now()
	return session
}

func (s *RtpSession) Send(packet *RtpPacket) error {
	frames, err := packet.Fragment()
	if err != nil {
		return err
	}
	for _, frame := range frames {
		to := s.conn.RemoteAddr()
		_, err = s.conn.WriteToUDP(frame, to.(*net.UDPAddr))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *RtpSession) loop() {

}
