package rtp

import (
	"babylon/util"
	"bytes"
)

type Payload []byte
type RtpHeader struct {
	V         byte   //2bit，版本号，0x2
	P         byte   //1bit，填充标志，值为0
	X         byte   //1bit，扩充标识，值为0
	CC        byte   //4bit,CSRC计数，值为0
	M         byte   //1bit，标识位
	PT        byte   //载荷类型 7bit
	Seq       uint16 //序列号
	Timestamp uint32 //时间戳
	SSRC      uint32 //同步源标识符
	//CSRC      uint32
}

func (h *RtpHeader) Encode() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(0x80)
	buf.WriteByte(h.PT)
	buf.Write(util.BigEndian.ToUint16(h.Seq))
	buf.Write(util.BigEndian.ToUint32(h.Timestamp))
	buf.Write(util.BigEndian.ToUint32(h.SSRC))
	return buf.Bytes()
}

type RtpPacket struct {
	Header        RtpHeader
	PayloadType   byte
	PayloadLength uint32
	Packet        Payload
	PacketLength  uint32
}

func NewRtpHeader(ssrc uint32, seq uint16, pt byte, timestamp uint32) *RtpHeader {
	h := new(RtpHeader)
	h.SSRC = ssrc
	h.Seq = seq
	h.Timestamp = timestamp
	h.PT = pt
	return h
}

func PacketAVC(h264 Payload, timestamp uint32) *RtpPacket {
	//head := MakeRtpHeader(1, 1)
	//10000000 00000000 0000000000000000
	return nil
}

func (rtp *RtpPacket) Fragment() ([]Payload, error) {
	return nil, nil
}

const (
	RTCP_SR   = 200
	RTCP_RR   = 201
	RTCP_SDES = 202
	RTCP_BYE  = 203
	RTCP_APP  = 204
)

type RtcpHeader struct {
	V      byte   //2bit，版本号，0x2
	P      byte   //1bit，填充标志，值为0
	RC     byte   //接收报告计数，5bit
	PT     byte   //包类型，8bit
	Length uint32 //包长度
}

type RtcpRRPacket struct {
}

type RtcpSRPacket struct {
}
