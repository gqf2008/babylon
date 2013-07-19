package rtmp

import (
	"babylon/util"
	"errors"
	log "github.com/cihub/seelog"
	"io"
)

func notifyPublishing(s *RtmpNetStream) error {
	if s.sh != nil {
		return s.sh.OnPublishing(s)
	}
	if s.ch != nil {
		return s.ch.OnPublishStart(s)
	}
	return errors.New("Handler Not Found")
}

func notifyPlaying(s *RtmpNetStream) error {
	if s.sh != nil {
		return s.sh.OnPlaying(s)
	}
	if s.ch != nil {
		return s.ch.OnPlayStart(s)
	}
	return errors.New("Handler Not Found")
}

func notifyError(s *RtmpNetStream, err error) {
	if s.sh != nil {
		s.sh.OnError(s, err)
	}
	if s.ch != nil {
		s.ch.OnError(s, err)
	}
}

func notifyClosed(s *RtmpNetStream) {
	if s.sh != nil {
		s.sh.OnClosed(s)
	}
	if s.ch != nil {
		s.ch.OnClosed(s)
	}
}

func writeMessage(p *RtmpNetConnection, msg RtmpMessage) (err error) {
	if p.wirtesequencenum > p.bandwidth {
		p.totalwritebytes += p.wirtesequencenum
		p.wirtesequencenum = 0
		sendAck(p, p.totalwritebytes)
		sendPing(p)
	}
	log.Debug(p.remoteAddr, " >>>>> ", msg)
	chunk, reset, err := encodeChunk12(msg.Header(), msg.Body(), p.writeChunkSize)
	if err != nil {
		return
	}
	_, err = p.bw.Write(chunk)
	if err != nil {
		return
	}
	err = p.bw.Flush()
	if err != nil {
		return
	}
	p.wirtesequencenum += uint32(len(chunk))
	//log.Debug(">>>>> chunk ", len(chunk), " reset ", len(reset))
	for reset != nil && len(reset) > 0 {
		chunk, reset, err = encodeChunk1(msg.Header(), reset, p.writeChunkSize)
		if err != nil {
			return
		}
		_, err = p.bw.Write(chunk)
		if err != nil {
			return
		}
		err = p.bw.Flush()
		if err != nil {
			return
		}
		p.wirtesequencenum += uint32(len(chunk))
		//log.Debug(">>>>> chunk ", len(chunk), " reset ", len(reset))
	}

	return
}
func readMessage(conn *RtmpNetConnection) (msg RtmpMessage, err error) {
	if conn.sequencenum >= conn.bandwidth {
		conn.totalreadbytes += conn.sequencenum
		conn.sequencenum = 0
		sendAck(conn, conn.totalreadbytes)
	}
	msg, err = readMessage0(conn)
	if err != nil {
		return nil, err
	}
	switch msg.Header().MessageType {
	case RTMP_MSG_CHUNK_SIZE:
		log.Info(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*ChunkSizeMessage)
		conn.readChunkSize = int(m.ChunkSize)
		return readMessage(conn)
	case RTMP_MSG_ABORT:
		log.Info(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*AbortMessage)
		delete(conn.incompletePackets, m.ChunkId)
		return readMessage(conn)
	case RTMP_MSG_ACK:
		log.Info(conn.remoteAddr, " <<<<< ", msg)
		return readMessage(conn)
	case RTMP_MSG_USER:
		log.Info(conn.remoteAddr, " <<<<< ", msg)
		if _, ok := msg.(*PingMessage); ok {
			sendPong(conn)
		}
		return readMessage(conn)
	case RTMP_MSG_ACK_SIZE:
		log.Info(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*AckWinSizeMessage)
		conn.bandwidth = m.AckWinsize
		return readMessage(conn)
	case RTMP_MSG_BANDWIDTH:
		log.Info(conn.remoteAddr, " <<<<< ", msg)
		m := msg.(*SetPeerBandwidthMessage)
		conn.bandwidth = m.AckWinsize
		//sendAckWinsize(conn, m.AckWinsize)
		return readMessage(conn)
	case RTMP_MSG_EDGE:
		log.Info(conn.remoteAddr, " <<<<< ", msg)
		return readMessage(conn)
	}
	return
}
func readMessage0(p *RtmpNetConnection) (msg RtmpMessage, err error) {
	//0 500 1000 1500 2000
	chunkhead, err := p.br.ReadByte()
	p.sequencenum += 1
	if err != nil {
		return nil, err
	}

	csid := uint32((chunkhead & 0x3f))
	chunktype := (chunkhead & 0xc0) >> 6
	//log.Debug("csid ", csid, " chunktype ", chunktype)
	switch csid {
	case 0:
		u8, err := p.br.ReadByte()
		p.sequencenum += 1
		if err != nil {
			return nil, err
		}
		csid = 64 + uint32(u8)
	case 1:
		u16 := make([]byte, 2)
		if _, err = io.ReadFull(p.br, u16); err != nil {
			return
		}
		p.sequencenum += 2
		csid = 64 + uint32(u16[0]) + 256*uint32(u16[1])
	}
	if p.lastReadHeaders[csid] == nil {
		p.lastReadHeaders[csid] = &RtmpHeader{ChunkType: chunktype, ChunkId: csid}
	}
	//if p.lastRtmpHeader == nil {
	//	p.lastRtmpHeader = p.lastReadHeaders[csid].Clone()
	//}
	h := p.lastReadHeaders[csid]
	if chunktype != 3 && p.incompletePackets[csid] != nil {
		err = errors.New("incomplete message droped")
		return
	}

	switch chunktype {
	case 0: //11字节头
		b := make([]byte, 3)
		if _, err = io.ReadFull(p.br, b); err != nil {
			return
		}
		p.sequencenum += 3
		h.Timestamp = util.BigEndian.Uint24(b) //type=0的时间戳为绝对时间，其他的都为相对时间
		//p.lastRtmpHeader.Timestamp = h.Timestamp
		if _, err = io.ReadFull(p.br, b); err != nil {
			return
		}
		p.sequencenum += 3
		h.MessageLength = util.BigEndian.Uint24(b)
		//p.lastRtmpHeader.MessageLength = h.MessageLength
		v, err := p.br.ReadByte()
		if err != nil {
			return nil, err
		}
		p.sequencenum += 1
		h.MessageType = v
		//p.lastRtmpHeader.MessageType = h.MessageType
		bb := make([]byte, 4)
		if _, err = io.ReadFull(p.br, bb); err != nil {
			return nil, err
		}
		p.sequencenum += 4
		h.StreamId = util.LittleEndian.Uint32(bb)
		//p.lastRtmpHeader.StreamId = h.StreamId
		if h.Timestamp == 0xffffff {
			if _, err = io.ReadFull(p.br, bb); err != nil {
				return nil, err
			}
			p.sequencenum += 4
			h.ExtendTimestamp = util.BigEndian.Uint32(bb)
			//p.lastRtmpHeader.Timestamp = h.Timestamp
		}
	case 1: //7字节头
		b := make([]byte, 3)
		if _, err = io.ReadFull(p.br, b); err != nil {
			return
		}
		p.sequencenum += 3
		h.ChunkType = chunktype
		h.Timestamp = util.BigEndian.Uint24(b)
		//p.lastRtmpHeader.Timestamp += h.Timestamp
		if _, err = io.ReadFull(p.br, b); err != nil {
			return
		}
		p.sequencenum += 3
		h.MessageLength = util.BigEndian.Uint24(b)
		//p.lastRtmpHeader.MessageLength = h.MessageLength
		v, err := p.br.ReadByte()
		if err != nil {
			return nil, err
		}
		p.sequencenum += 1
		h.MessageType = v
		//p.lastRtmpHeader.MessageType = h.MessageType
		if h.Timestamp == 0xffffff {
			bb := make([]byte, 4)
			if _, err := io.ReadFull(p.br, bb); err != nil {
				return nil, err
			}
			p.sequencenum += 4
			h.ExtendTimestamp = util.BigEndian.Uint32(bb)
			//p.lastRtmpHeader.Timestamp = h.Timestamp
		}
	case 2: //3字节头
		b := make([]byte, 3)
		if _, err = io.ReadFull(p.br, b); err != nil {
			return
		}
		h.ChunkType = chunktype
		p.sequencenum += 3
		h.Timestamp = util.BigEndian.Uint24(b)
		//p.lastRtmpHeader.Timestamp += h.Timestamp
		if h.Timestamp == 0xffffff {
			bb := make([]byte, 4)
			if _, err := io.ReadFull(p.br, bb); err != nil {
				return nil, err
			}
			p.sequencenum += 4
			h.ExtendTimestamp = util.BigEndian.Uint32(bb)
			//p.lastRtmpHeader.Timestamp += h.Timestamp
		}
	case 3: //0字节头
		h.ChunkType = chunktype

	}
	if p.incompletePackets[csid] == nil {
		p.incompletePackets[csid] = make(Payload, 0)
	}
	nRead := uint32(len(p.incompletePackets[csid]))
	size := h.MessageLength - nRead
	if size > uint32(p.readChunkSize) {
		size = uint32(p.readChunkSize)
	}
	buf := make(Payload, size)
	i, err := io.ReadFull(p.br, buf)
	if err != nil {
		return
	}
	p.sequencenum += uint32(i)
	buf = append(p.incompletePackets[csid], buf...)
	p.incompletePackets[csid] = buf
	//log.Debug("   << ", p.lastReadHeaders[csid].String(), size, len(buf))
	if uint32(len(p.incompletePackets[csid])) == h.MessageLength {
		chunkheader := h.Clone()
		//p.lastRtmpHeader = nil
		//log.Debugf("%02X", p.incompletePackets[csid])
		msg = decodeRtmpMessage(chunkheader, p.incompletePackets[csid])
		delete(p.incompletePackets, csid)
		//log.Debug("<<<<< ", msg)
		return
	}

	return readMessage0(p)
}

func sendChunkSize(conn *RtmpNetConnection, size uint32) error {
	msg := new(ChunkSizeMessage)
	msg.ChunkSize = size
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_CHUNK_SIZE, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}
func sendAck(conn *RtmpNetConnection, num uint32) error {
	msg := new(AckMessage)
	msg.SequenceNumber = num
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_ACK, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}
func sendAckWinsize(conn *RtmpNetConnection, size uint32) error {
	msg := new(AckWinSizeMessage)
	msg.AckWinsize = size
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_ACK_SIZE, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendPeerBandwidth(conn *RtmpNetConnection, size uint32) error {
	msg := new(SetPeerBandwidthMessage)
	msg.AckWinsize = size
	msg.LimitType = byte(2)
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_BANDWIDTH, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendStreamBegin(conn *RtmpNetConnection) error {
	msg := new(StreamBeginMessage)
	msg.EventType = RTMP_USER_STREAM_BEGIN
	msg.StreamId = conn.streamid
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendStreamRecorded(conn *RtmpNetConnection) error {
	msg := new(RecordedMessage)
	msg.EventType = RTMP_USER_RECORDED
	msg.StreamId = conn.streamid
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendPing(conn *RtmpNetConnection) error {
	msg := new(PingMessage)
	msg.EventType = RTMP_USER_PING
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendPong(conn *RtmpNetConnection) error {
	msg := new(PongMessage)
	msg.EventType = RTMP_USER_PONG
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendSetBufferMessage(conn *RtmpNetConnection) error {
	msg := new(SetBufferMessage)
	msg.EventType = RTMP_USER_SET_BUFLEN
	msg.StreamId = conn.streamid
	msg.Millisecond = 100
	msg.Encode()
	head := newRtmpHeader(RTMP_CHANNEL_CONTROL, 0, len(msg.Payload), RTMP_MSG_USER, 0, 0)
	msg.RtmpHeader = head
	return writeMessage(conn, msg)
}

func sendConnect(conn *RtmpNetConnection, app, pageUrl, swfUrl, tcUrl string) error {
	result := new(ConnectMessage)
	result.Command = "connect"
	result.TransactionId = 1
	//pro := newObject()
	//putString(pro, "app", app)
	//putNumber(pro, "audioCodecs", 3575)
	//putNumber(pro, "capabilities", 239)
	//putString(pro, "flashVer", "MAC 11,7,700,203")
	//putBool(pro, "fpad", false)
	//putNumber(pro, "objectEncoding", 0)
	//putString(pro, "pageUrl", pageUrl)
	//putString(pro, "swfUrl", swfUrl)
	//putString(pro, "tcUrl", tcUrl)
	//putNumber(pro, "videoCodecs", 252)
	//putNumber(pro, "videoFunction", 1)
	obj := newMap()
	obj["app"] = app
	obj["audioCodecs"] = 3575
	obj["capabilities"] = 239
	obj["flashVer"] = "MAC 11,7,700,203"
	obj["fpad"] = false
	obj["objectEncoding"] = 0
	obj["pageUrl"] = pageUrl
	obj["swfUrl"] = swfUrl
	obj["tcUrl"] = tcUrl
	obj["videoCodecs"] = 252
	obj["videoFunction"] = 1
	result.Object = obj
	info := newMap()
	result.Optional = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendCreateStream(conn *RtmpNetConnection) error {
	m := new(CreateStreamMessage)
	m.Command = "createStream"
	m.TransactionId = 1
	m.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(m.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	m.RtmpHeader = head
	return writeMessage(conn, m)
}

func sendPlay(conn *RtmpNetConnection, name string, start, duration int, rest bool) error {
	m := new(PlayMessage)
	m.Command = "play"
	m.TransactionId = 1
	m.StreamName = name
	m.Start = uint64(start)
	m.Duration = uint64(duration)
	m.Rest = rest
	m.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(m.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	m.RtmpHeader = head
	return writeMessage(conn, m)
}

func sendConnectResult(conn *RtmpNetConnection, level, code string) error {
	result := new(ReplyConnectMessage)
	result.Command = NetStatus_Result
	result.TransactionId = 1
	pro := newMap()
	pro["fmsVer"] = SERVER_NAME + "/" + VERSION
	pro["capabilities"] = 31
	pro["mode"] = 1
	pro["Author"] = "G.Q.F/gao.qingfeng@gmail.com"
	result.Properties = pro
	info := newMap()
	info["level"] = level
	info["code"] = NetConnection_Connect_Success
	info["objectEncoding"] = uint64(conn.objectEncoding)
	result.Infomation = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendConnectSuccess(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Status, NetConnection_Connect_Success)
}

func sendConnectFailed(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_Failed)
}
func sendConnectRejected(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_Rejected)
}
func sendConnectInvalidApp(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_InvalidApp)
}
func sendConnectClose(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Status, NetConnection_Connect_Closed)
}
func sendConnectAppShutdown(conn *RtmpNetConnection) error {
	return sendConnectResult(conn, Level_Error, NetConnection_Connect_AppShutdown)
}

func sendCreateStreamResult(conn *RtmpNetConnection, tid uint64) error {
	result := new(ReplyCreateStreamMessage)
	result.Command = NetStatus_Result
	result.TransactionId = tid
	result.StreamId = conn.streamid
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, 0, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendPlayResult(conn *RtmpNetConnection, level, code string) error {
	result := new(ReplyPlayMessage)
	result.Command = NetStatus_OnStatus
	result.TransactionId = 0
	info := newMap()
	info["level"] = level
	info["code"] = code
	//putString(info, "details", details)
	//putString(info, "description", "OK")
	info["clientid"] = 1
	result.Object = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, conn.streamid, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendPlayReset(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Status, NetStream_Play_Reset)
}
func sendPlayStart(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Status, NetStream_Play_Start)
}
func sendPlayStop(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Status, NetStream_Play_Stop)
}
func sendPlayFailed(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Error, NetStream_Play_Failed)
}
func sendPlayNotFound(conn *RtmpNetConnection) error {
	return sendPlayResult(conn, Level_Error, NetStream_Play_StreamNotFound)
}

func sendPublishResult(conn *RtmpNetConnection, level, code string) error {
	result := new(ReplyPublishMessage)
	result.Command = NetStatus_OnStatus
	result.TransactionId = 0
	info := newMap()
	info["level"] = level
	info["code"] = code
	info["clientid"] = 1
	result.Infomation = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_CMD, conn.streamid, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendPublishStart(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Status, NetStream_Publish_Start)
}
func sendPublishIdle(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Status, NetStream_Publish_Idle)
}
func sendPublishBadName(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Error, NetStream_Publish_BadName)
}
func sendUnpublishSuccess(conn *RtmpNetConnection) error {
	return sendPublishResult(conn, Level_Status, NetStream_Unpublish_Success)
}

func sendStreamDataStart(conn *RtmpNetConnection) error {
	result := new(ReplyPlayMessage)
	result.Command = NetStatus_OnStatus
	result.TransactionId = 1
	info := newMap()
	info["level"] = Level_Status
	info["code"] = NetStream_Data_Start
	info["clientid"] = 1
	result.Object = info
	result.Encode0()
	head := newRtmpHeader(RTMP_CHANNEL_COMMAND, 0, len(result.Payload), RTMP_MSG_AMF_META, conn.streamid, 0)
	result.RtmpHeader = head
	return writeMessage(conn, result)
}

func sendSampleAccess(conn *RtmpNetConnection) error {
	return nil
}

func sendMetaData(conn *RtmpNetConnection, data *StreamPacket) error {
	head := newRtmpHeader(RTMP_CHANNEL_DATA, 0, len(data.Payload), RTMP_MSG_AMF_META, conn.streamid, 0)
	msg := new(MetadataMessage)
	msg.RtmpHeader = head
	msg.Payload = data.Payload
	return writeMessage(conn, msg)
}

func sendFullVideo(conn *RtmpNetConnection, video *StreamPacket) (err error) {
	log.Debug(">>>>> ", video)
	if conn.wirtesequencenum > conn.bandwidth {
		conn.totalwritebytes += conn.wirtesequencenum
		conn.wirtesequencenum = 0
		sendAck(conn, conn.totalwritebytes)
		sendPing(conn)
	}
	h := newRtmpHeader(RTMP_CHANNEL_VIDEO, video.Timestamp, len(video.Payload), RTMP_MSG_VIDEO, conn.streamid, 0)
	chunk, reset, err := encodeChunk12(h, video.Payload, conn.writeChunkSize)
	if err != nil {
		return
	}
	_, err = conn.bw.Write(chunk)
	if err != nil {
		return
	}
	err = conn.bw.Flush()
	if err != nil {
		return
	}
	conn.wirtesequencenum += uint32(len(chunk))
	for reset != nil && len(reset) > 0 {
		chunk, reset, err = encodeChunk1(h, reset, conn.writeChunkSize)
		if err != nil {
			return
		}
		_, err = conn.bw.Write(chunk)
		if err != nil {
			return
		}
		err = conn.bw.Flush()
		if err != nil {
			return
		}
		conn.wirtesequencenum += uint32(len(chunk))
	}
	err = conn.bw.Flush()
	return
}

func sendFullAudio(conn *RtmpNetConnection, audio *StreamPacket) (err error) {
	log.Debug(">>>>> ", audio)
	if conn.wirtesequencenum > conn.bandwidth {
		conn.totalwritebytes += conn.wirtesequencenum
		conn.wirtesequencenum = 0
		sendAck(conn, conn.totalwritebytes)
		sendPing(conn)
	}
	h := newRtmpHeader(RTMP_CHANNEL_AUDIO, audio.Timestamp, len(audio.Payload), RTMP_MSG_AUDIO, conn.streamid, 0)
	chunk, reset, err := encodeChunk12(h, audio.Payload, conn.writeChunkSize)
	if err != nil {
		return
	}
	_, err = conn.bw.Write(chunk)
	if err != nil {
		return
	}
	err = conn.bw.Flush()
	if err != nil {
		return
	}
	conn.wirtesequencenum += uint32(len(chunk))
	for reset != nil && len(reset) > 0 {
		chunk, reset, err = encodeChunk1(h, reset, conn.writeChunkSize)
		if err != nil {
			return
		}
		_, err = conn.bw.Write(chunk)
		if err != nil {
			return
		}
		err = conn.bw.Flush()
		if err != nil {
			return
		}
		conn.wirtesequencenum += uint32(len(chunk))
	}
	return
}

func sendVideo(conn *RtmpNetConnection, video *StreamPacket) (err error) {
	log.Debug(">>>>> ", video)
	if conn.wirtesequencenum > conn.bandwidth {
		conn.totalwritebytes += conn.wirtesequencenum
		conn.wirtesequencenum = 0
		sendAck(conn, conn.totalwritebytes)
		sendPing(conn)
	}
	h := newRtmpHeader(RTMP_CHANNEL_VIDEO, video.Timestamp, len(video.Payload), RTMP_MSG_VIDEO, conn.streamid, 0)
	chunk, reset, err := encodeChunk8(h, video.Payload, conn.writeChunkSize)
	if err != nil {
		return
	}
	_, err = conn.bw.Write(chunk)
	if err != nil {
		return
	}
	err = conn.bw.Flush()
	if err != nil {
		return
	}
	conn.wirtesequencenum += uint32(len(chunk))
	for reset != nil && len(reset) > 0 {
		chunk, reset, err = encodeChunk1(h, reset, conn.writeChunkSize)
		if err != nil {
			return
		}
		_, err = conn.bw.Write(chunk)
		if err != nil {
			return
		}
		err = conn.bw.Flush()
		if err != nil {
			return
		}
		conn.wirtesequencenum += uint32(len(chunk))
	}
	return
}

func sendAudio(conn *RtmpNetConnection, audio *StreamPacket) (err error) {
	//log.Debug(">>>>> ", audio)
	if conn.wirtesequencenum > conn.bandwidth {
		conn.totalwritebytes += conn.wirtesequencenum
		conn.wirtesequencenum = 0
		sendAck(conn, conn.totalwritebytes)
		sendPing(conn)
	}
	h := newRtmpHeader(RTMP_CHANNEL_AUDIO, audio.Timestamp, len(audio.Payload), RTMP_MSG_AUDIO, conn.streamid, 0)
	chunk, reset, err := encodeChunk8(h, audio.Payload, conn.writeChunkSize)
	if err != nil {
		return
	}
	_, err = conn.bw.Write(chunk)
	if err != nil {
		return
	}
	err = conn.bw.Flush()
	if err != nil {
		return
	}
	conn.wirtesequencenum += uint32(len(chunk))
	for reset != nil && len(reset) > 0 {
		chunk, reset, err = encodeChunk1(h, reset, conn.writeChunkSize)
		if err != nil {
			return
		}
		_, err = conn.bw.Write(chunk)
		if err != nil {
			return
		}
		err = conn.bw.Flush()
		if err != nil {
			return
		}
		conn.wirtesequencenum += uint32(len(chunk))
	}

	return
}
