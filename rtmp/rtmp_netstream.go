package rtmp

import (
	"babylon/util"
	"bufio"
	"errors"
	log "github.com/cihub/seelog"
	"net"
	"strings"
	"sync"
	"time"
)

//"NetStream.Buffer.Empty"	"status"	数据的接收速度不足以填充缓冲区。 数据流将在缓冲区重新填充前中断，此时将发送 NetStream.Buffer.Full 消息，并且该流将重新开始播放。
//"NetStream.Buffer.Full"	"status"	缓冲区已满并且流将开始播放。
//"NetStream.Buffer.Flush"	"status"	数据已完成流式处理，剩余的缓冲区将被清空。
//"NetStream.Failed"	"error"	仅限 Flash Media Server。 发生了错误，在其它事件代码中没有列出此错误的原因。
//"NetStream.Publish.Start"	"status"	已经成功发布。
//"NetStream.Publish.BadName"	"error"	试图发布已经被他人发布的流。
//"NetStream.Publish.Idle"	"status"	流发布者空闲而没有在传输数据。
//"NetStream.Unpublish.Success"	"status"	已成功执行取消发布操作。
//"NetStream.Play.Start"	"status"	播放已开始。
//"NetStream.Play.Stop"	"status"	播放已结束。
//"NetStream.Play.Failed"	"error"	出于此表中列出的原因之外的某一原因（例如订阅者没有读取权限），播放发生了错误。
//"NetStream.Play.StreamNotFound"	"error"	无法找到传递给 play() 方法的 FLV。
//"NetStream.Play.Reset"	"status"	由播放列表重置导致。
//"NetStream.Play.PublishNotify"	"status"	到流的初始发布被发送到所有的订阅者。
//"NetStream.Play.UnpublishNotify"	"status"	从流取消的发布被发送到所有的订阅者。
//"NetStream.Play.InsufficientBW"	"warning"	仅限 Flash Media Server。 客户端没有足够的带宽，无法以正常速度播放数据。
//"NetStream.Pause.Notify"	"status"	流已暂停。
//"NetStream.Unpause.Notify"	"status"	流已恢复。
//"NetStream.Record.Start"	"status"	录制已开始。
//"NetStream.Record.NoAccess"	"error"	试图录制仍处于播放状态的流或客户端没有访问权限的流。
//"NetStream.Record.Stop"	"status"	录制已停止。
//"NetStream.Record.Failed"	"error"	尝试录制流失败。
//"NetStream.Seek.Failed"	"error"	搜索失败，如果流处于不可搜索状态，则会发生搜索失败。
//"NetStream.Seek.InvalidTime"	"error"	对于使用渐进式下载方式下载的视频，用户已尝试跳过到目前为止已下载的视频数据的结尾或在整个文件已下载后跳过视频的结尾进行搜寻或播放。 message.details 属性包含一个时间代码，该代码指出用户可以搜寻的最后一个有效位置。
//"NetStream.Seek.Notify"	"status"	搜寻操作完成。
//"NetConnection.Call.BadVersion"	"error"	以不能识别的格式编码的数据包。
//"NetConnection.Call.Failed"	"error"	NetConnection.call 方法无法调用服务器端的方法或命令。
//"NetConnection.Call.Prohibited"	"error"	Action Message Format (AMF) 操作因安全原因而被阻止。 或者是 AMF URL 与 SWF 不在同一个域，或者是 AMF 服务器没有信任 SWF 文件的域的策略文件。
//"NetConnection.Connect.Closed"	"status"	成功关闭连接。
//"NetConnection.Connect.Failed"	"error"	连接尝试失败。
//"NetConnection.Connect.Success"	"status"	连接尝试成功。
//"NetConnection.Connect.Rejected"	"error"	连接尝试没有访问应用程序的权限。
//"NetConnection.Connect.AppShutdown"	"error"	正在关闭指定的应用程序。
//"NetConnection.Connect.InvalidApp"	"error"	连接时指定的应用程序名无效。
//"SharedObject.Flush.Success"	"status"	“待定”状态已解析并且 SharedObject.flush() 调用成功。
//"SharedObject.Flush.Failed"	"error"	“待定”状态已解析，但 SharedObject.flush() 失败。
//"SharedObject.BadPersistence"	"error"	使用永久性标志对共享对象进行了请求，但请求无法被批准，因为已经使用其它标记创建了该对象。
//"SharedObject.UriMismatch"	"error"	试图连接到拥有与共享对象不同的 URI (URL) 的 NetConnection 对象。

const (
	MODE_PRODUCER = 1
	MODE_CONSUMER = 2
	MODE_PROXY    = 3

	Level_Status                      = "status"
	Level_Error                       = "error"
	Level_Warning                     = "warning"
	NetStatus_OnStatus                = "onStatus"
	NetStatus_Result                  = "_result"
	NetStatus_Error                   = "_error"
	NetConnection_Call_BadVersion     = "NetConnection.Call.BadVersion"     //	"error"	以不能识别的格式编码的数据包。
	NetConnection_Call_Failed         = "NetConnection.Call.Failed"         //	"error"	NetConnection.call 方法无法调用服务器端的方法或命令。
	NetConnection_Call_Prohibited     = "NetConnection.Call.Prohibited"     //	"error"	Action Message Format (AMF) 操作因安全原因而被阻止。 或者是 AMF URL 与 SWF 不在同一个域，或者是 AMF 服务器没有信任 SWF 文件的域的策略文件。
	NetConnection_Connect_AppShutdown = "NetConnection.Connect.AppShutdown" //	"error"	正在关闭指定的应用程序。
	NetConnection_Connect_InvalidApp  = "NetConnection.Connect.InvalidApp"  //	"error"	连接时指定的应用程序名无效。
	NetConnection_Connect_Success     = "NetConnection.Connect.Success"     //"status"	连接尝试成功。
	NetConnection_Connect_Closed      = "NetConnection.Connect.Closed"      //"status"	成功关闭连接。
	NetConnection_Connect_Failed      = "NetConnection.Connect.Failed"      //"error"	连接尝试失败。
	NetConnection_Connect_Rejected    = "NetConnection.Connect.Rejected"

	NetStream_Play_Reset          = "NetStream.Play.Reset"
	NetStream_Play_Start          = "NetStream.Play.Start"
	NetStream_Play_StreamNotFound = "NetStream.Play.StreamNotFound"
	NetStream_Play_Stop           = "NetStream.Play.Stop"
	NetStream_Play_Failed         = "NetStream.Play.Failed"

	NetStream_Play_Switch   = "NetStream.Play.Switch"
	NetStream_Play_Complete = "NetStream.Play.Switch"

	NetStream_Data_Start = "NetStream.Data.Start"

	NetStream_Publish_Start     = "NetStream.Publish.Start"     //"status"	已经成功发布。
	NetStream_Publish_BadName   = "NetStream.Publish.BadName"   //"error"	试图发布已经被他人发布的流。
	NetStream_Publish_Idle      = "NetStream.Publish.Idle"      //"status"	流发布者空闲而没有在传输数据。
	NetStream_Unpublish_Success = "NetStream.Unpublish.Success" //"status"	已成功执行取消发布操作。

	NetStream_Buffer_Empty   = "NetStream.Buffer.Empty"
	NetStream_Buffer_Full    = "NetStream.Buffer.Full"
	NetStream_Buffe_Flush    = "NetStream.Buffer.Flush"
	NetStream_Pause_Notify   = "NetStream.Pause.Notify"
	NetStream_Unpause_Notify = "NetStream.Unpause.Notify"

	NetStream_Record_Start    = "NetStream.Record.Start"    //	"status"	录制已开始。
	NetStream_Record_NoAccess = "NetStream.Record.NoAccess" //	"error"	试图录制仍处于播放状态的流或客户端没有访问权限的流。
	NetStream_Record_Stop     = "NetStream.Record.Stop"     //	"status"	录制已停止。
	NetStream_Record_Failed   = "NetStream.Record.Failed"   //	"error"	尝试录制流失败。

	NetStream_Seek_Failed      = "NetStream.Seek.Failed"      //	"error"	搜索失败，如果流处于不可搜索状态，则会发生搜索失败。
	NetStream_Seek_InvalidTime = "NetStream.Seek.InvalidTime" //"error"	对于使用渐进式下载方式下载的视频，用户已尝试跳过到目前为止已下载的视频数据的结尾或在整个文件已下载后跳过视频的结尾进行搜寻或播放。 message.details 属性包含一个时间代码，该代码指出用户可以搜寻的最后一个有效位置。
	NetStream_Seek_Notify      = "NetStream.Seek.Notify"      //	"status"	搜寻操作完成。

)

type Infomation map[string]interface{}
type Args interface{}

type VideoChannel chan *StreamPacket
type AudioChannel chan *StreamPacket

type NetConnection interface {
	Connect(command string, args ...Args) error
	Call(command string, args ...Args) error
	Connected() bool
	Close()
	URL() string
}

type NetStream interface {
	AttachVideo(channel VideoChannel)
	AttachAudio(channel AudioChannel)
	Publish(streamName string, streamType string) error
	Play(streamName string, args ...Args) error //streamName string, start uint64,length uint64,reset bool
	Pause()
	Resume()
	TogglePause()
	Seek(offset uint64)
	Send(handlerName string, args ...Args)
	Close()
	ReceiveAudio(flag bool)
	ReceiveVideo(flag bool)
	NetConnection() NetConnection
	BufferTime() time.Duration
	BytesLoaded() uint64
	BufferLength() uint64
}
type SharedObject interface {
}

type RtmpNetConnection struct {
	remoteAddr        string
	url               string
	app               string
	createTime        time.Time
	readChunkSize     int
	writeChunkSize    int
	readTimeout       time.Duration
	writeTimeout      time.Duration
	bandwidth         uint32
	limitType         byte
	wirtesequencenum  uint32
	sequencenum       uint32
	totalreadbytes    uint32
	totalwritebytes   uint32
	server            *Server           // the Server on which the connection arrived
	conn              net.Conn          // i/o connection
	buf               *bufio.ReadWriter // buffered(lr,rwc), reading from bufio->limitReader->sr->rwc
	br                *bufio.Reader
	bw                *bufio.Writer
	lock              *sync.Mutex // guards the following
	incompletePackets map[uint32]Payload
	lastReadHeaders   map[uint32]*RtmpHeader
	lastWriteHeaders  map[uint32]*RtmpHeader
	lastRtmpHeader    *RtmpHeader
	connected         bool
	nextStreamId      func(chunkid uint32) uint32
	//nslistener        NetStatusListener
	//errlistener       IoErrorListener
	streamid       uint32
	objectEncoding int
}
type RtmpNetStream struct {
	conn               *RtmpNetConnection
	metaData           *StreamPacket
	firstVideoKeyFrame *StreamPacket
	firstAudioKeyFrame *StreamPacket
	lastVideoKeyFrame  *StreamPacket
	videochan          VideoChannel
	audiochan          AudioChannel
	path               string
	bufferTime         time.Duration
	bufferLength       uint64
	bufferLoad         uint64
	lock               *sync.Mutex // guards the following
	sh                 ServerHandler
	ch                 ClientHandler
	mode               int
	vkfsended          bool
	akfsended          bool
	dispatcher         *Broadcast
	mrev_duration      uint32
	vsend_time         uint32
	asend_time         uint32
	closed             bool
}

func newNetStream(conn *RtmpNetConnection, sh ServerHandler, ch ClientHandler) (s *RtmpNetStream) {
	s = new(RtmpNetStream)
	s.conn = conn
	s.lock = new(sync.Mutex)
	s.sh = sh
	s.ch = ch
	s.vkfsended = false
	s.akfsended = false
	return
}

func (c *RtmpNetConnection) Connect(command string, args ...Args) error {
	//rtmp://host:port/app
	log.Debug(command)
	p := strings.Split(command, "/")
	address := p[2]
	log.Debug(address)
	app := p[3]
	log.Debug(app)
	host := strings.Split(address, ":")
	if len(host) == 1 {
		address += ":1935"
	}
	log.Debug(address)
	conn, err := net.DialTimeout("tcp", address, time.Second*30)
	if err != nil {
		return err
	}
	c.conn = conn
	c.app = app
	c.remoteAddr = conn.RemoteAddr().String()
	c.br = bufio.NewReader(conn)
	c.bw = bufio.NewWriter(conn)
	c.buf = bufio.NewReadWriter(c.br, c.bw)
	if !client_simple_handshake(c.buf) {
		return errors.New("Client Handshake Fail")
	}
	err = sendConnect(c, app, "", "", command)
	if err != nil {
		return err
	}
	for {
		msg, err := readMessage(c)
		if err != nil {
			c.Close()
			return err
		}
		log.Debug(msg)
		if _, ok := msg.(*ReplyMessage); ok {
			dec := newDecoder(msg.Body())
			reply := new(ReplyConnectMessage)
			reply.RtmpHeader = msg.Header()
			reply.Command = readString(dec)
			reply.TransactionId = readNumber(dec)
			//dec.readNull()
			reply.Properties = readObject(dec)
			reply.Infomation = readObject(dec)
			log.Debug(reply)
			if NetConnection_Connect_Success == getString(reply.Infomation, "code") {
				c.connected = true
				return nil
			} else {
				return errors.New(getString(reply.Infomation, "code"))
			}
		}
	}
	return nil
}

func getString(obj interface{}, key string) string {
	return obj.(Map)[key].(string)
}

func (c *RtmpNetConnection) Call(command string, args ...Args) error {
	return nil
}
func (c *RtmpNetConnection) Connected() bool {
	return c.connected
}
func (c *RtmpNetConnection) Close() {
	if c.conn == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.conn.Close()
	c.connected = false
}
func (c *RtmpNetConnection) URL() string {
	return c.url
}

func (s *RtmpNetStream) AttachVideo(channel VideoChannel) {
	s.videochan = channel
}
func (s *RtmpNetStream) AttachAudio(channel AudioChannel) {
	s.audiochan = channel
}
func (s *RtmpNetStream) Publish(streamName string, streamType string) error {
	return nil
}
func (s *RtmpNetStream) Play(streamName string, args ...Args) error {
	conn := s.conn
	s.mode = MODE_PRODUCER
	sendCreateStream(conn)
	for {
		msg, err := readMessage(conn)
		if err != nil {
			return err
		}

		if m, ok := msg.(*UnknowCommandMessage); ok {
			log.Debug(m)
			continue
		}
		reply := new(ReplyCreateStreamMessage)
		reply.Decode0(msg.Header(), msg.Body())
		log.Debug(reply)
		conn.streamid = reply.StreamId
		break
	}
	sendPlay(conn, streamName, 0, 0, false)
	for {
		msg, err := readMessage(conn)
		if err != nil {
			return err
		}
		if m, ok := msg.(*UnknowCommandMessage); ok {
			log.Debug(m)
			continue
		}
		result := new(ReplyPlayMessage)
		result.Decode0(msg.Header(), msg.Body())
		log.Debug(result)
		code := getString(result.Object, "code")
		if code == NetStream_Play_Reset {
			continue
		} else if code == NetStream_Play_Start {
			break
		} else {
			return errors.New(code)
		}
	}
	sendSetBufferMessage(conn)
	if strings.HasSuffix(conn.app, "/") {
		s.path = conn.app + strings.Split(streamName, "?")[0]
	} else {
		s.path = conn.app + "/" + strings.Split(streamName, "?")[0]
	}

	err := notifyPlaying(s)
	if err != nil {
		return err
	}
	go s.cserve()
	return nil
}
func (s *RtmpNetStream) Pause() {

}
func (s *RtmpNetStream) Resume() {

}
func (s *RtmpNetStream) TogglePause() {

}
func (s *RtmpNetStream) Seek(offset uint64) {

}
func (s *RtmpNetStream) Send(handlerName string, args ...Args) {

}
func (s *RtmpNetStream) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return
	}
	s.conn.Close()
	s.closed = true
	notifyClosed(s)
}
func (s *RtmpNetStream) ReceiveAudio(flag bool) {

}
func (s *RtmpNetStream) ReceiveVideo(flag bool) {

}
func (s *RtmpNetStream) NetConnection() NetConnection {
	return s.conn
}

func (s *RtmpNetStream) BufferTime() time.Duration {
	return s.bufferTime
}
func (s *RtmpNetStream) BytesLoaded() uint64 {
	return s.bufferLoad
}
func (s *RtmpNetStream) BufferLength() uint64 {
	return s.bufferLength
}

func (s *RtmpNetStream) SendVideo(video *StreamPacket) error {
	log.Debug(video)
	if s.vkfsended {
		video.Timestamp -= s.vsend_time - uint32(s.bufferTime)
		s.vsend_time += video.Timestamp
		return sendVideo(s.conn, video)
	}
	if !video.isKeyFrame() {
		//log.Info("No Video Key Frame,Ignore Video ", video)
		//video = s.dispatcher.producer.lastVideoKeyFrame
		return nil
	}
	fkf := s.dispatcher.producer.firstVideoKeyFrame
	if fkf == nil {
		log.Info("No Video Configurate Record,Ignore Video ", video)
		return nil
	}
	fkf.Timestamp = 0
	log.Info("Send Video Configurate Record ", fkf)
	//log.Infof(" Payload %02X", fkf.Payload)
	ver := fkf.Payload[4+1]
	avcPfofile := fkf.Payload[4+2]
	profileCompatibility := fkf.Payload[4+3]
	avcLevel := fkf.Payload[4+4]
	reserved := fkf.Payload[4+5] >> 2
	lengthSizeMinusOne := fkf.Payload[4+5] & 0x03
	reserved2 := fkf.Payload[4+6] >> 5
	numOfSPS := fkf.Payload[4+6] & 31
	spsLength := util.BigEndian.Uint16(fkf.Payload[4+7:])
	sps := fkf.Payload[4+9 : 4+9+int(spsLength)]
	numOfPPS := fkf.Payload[4+9+int(spsLength)]
	ppsLength := util.BigEndian.Uint16(fkf.Payload[4+9+int(spsLength)+1:])
	pps := fkf.Payload[4+9+int(spsLength)+1+2:]
	log.Infof("  cfgVersion(%v) | avcProfile(%v) | profileCompatibility(%v) |avcLevel(%v) | reserved(%v) | lengthSizeMinusOne(%v) | reserved(%v) | numOfSPS(%v) |spsLength(%v) | sps(%02X) | numOfPPS(%v) | ppsLength(%v) | pps(%02X) ",
		ver,
		avcPfofile,
		profileCompatibility,
		avcLevel,
		reserved,
		lengthSizeMinusOne,
		reserved2,
		numOfSPS,
		spsLength,
		sps,
		numOfPPS,
		ppsLength,
		pps)
	err := sendFullVideo(s.conn, fkf)
	if err != nil {
		return err
	}
	s.vkfsended = true
	s.vsend_time = video.Timestamp
	video.Timestamp = 0
	log.Info("Send I Frame ", video)
	log.Infof(" Payload %v/%v", video.Payload[9]&0x1f, video.Payload[10])

	return sendFullVideo(s.conn, video)
}

func (s *RtmpNetStream) SendAudio(audio *StreamPacket) error {
	if !s.vkfsended {
		//log.Debug("No Video Configurate Record,Ignore Audio ", audio)
		return nil
	}
	if s.akfsended {
		audio.Timestamp -= s.asend_time - uint32(s.bufferTime)
		s.asend_time += audio.Timestamp
		return sendAudio(s.conn, audio)
	}
	fakf := s.dispatcher.producer.firstAudioKeyFrame
	fakf.Timestamp = 0
	log.Info("Send Audio Configurate Record ", fakf)
	err := sendFullAudio(s.conn, fakf)
	if err != nil {
		return err
	}
	s.akfsended = true
	s.asend_time = audio.Timestamp
	audio.Timestamp = 0
	return sendFullAudio(s.conn, audio)
}

func (s *RtmpNetStream) SendMetaData(data *StreamPacket) error {
	return sendMetaData(s.conn, data)
}

func (s *RtmpNetStream) loop() {
	log.Info("RtmpNetStream " + s.conn.remoteAddr + " " + s.path + " loop started")
	conn := s.conn
	for {
		msg, err := readMessage(conn)
		if err != nil {
			log.Error(err)
			notifyError(s, err)
			break
		}
		//log.Debug("<<<<< ", msg)
		if msg.Header().MessageLength <= 0 {
			continue
		}
		if am, ok := msg.(*AudioMessage); ok {
			sp := new(StreamPacket)
			if am.RtmpHeader.Timestamp == 0xffffff {
				s.mrev_duration += am.RtmpHeader.ExtendTimestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			} else {
				s.mrev_duration += am.RtmpHeader.Timestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			}
			sp.Type = am.RtmpHeader.MessageType
			sp.Payload = am.Payload
			one := sp.Payload[0]
			sp.AudioFormat = one >> 4
			sp.SamplingRate = (one & 0x0c) >> 2
			sp.SampleLength = (one & 0x02) >> 1
			sp.AudioType = one & 0x01
			if s.firstAudioKeyFrame == nil {
				s.firstAudioKeyFrame = sp
			} else {
				s.audiochan <- sp
			}
		} else if vm, ok := msg.(*VideoMessage); ok {
			sp := new(StreamPacket)
			if vm.RtmpHeader.Timestamp == 0xffffff {
				s.mrev_duration += vm.RtmpHeader.ExtendTimestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			} else {
				s.mrev_duration += vm.RtmpHeader.Timestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			}
			sp.Type = vm.RtmpHeader.MessageType
			sp.Payload = vm.Payload
			one := sp.Payload[0]
			sp.VideoFrameType = one >> 4
			sp.VideoCodecID = one & 0x0f
			if s.firstVideoKeyFrame == nil {
				s.firstVideoKeyFrame = sp
			} else {
				if sp.isKeyFrame() {
					s.lastVideoKeyFrame = sp
				}
				s.videochan <- sp
			}
			//debug("IN", " <<<<<<< ", vm)
		} else if mm, ok := msg.(*MetadataMessage); ok {
			sp := new(StreamPacket)
			sp.Timestamp = mm.RtmpHeader.Timestamp
			if mm.RtmpHeader.Timestamp == 0xffffff {
				sp.Timestamp = mm.RtmpHeader.ExtendTimestamp
			}
			sp.Type = mm.RtmpHeader.MessageType
			sp.Payload = mm.Payload
			if s.metaData == nil {
				s.metaData = sp
			}
			//s.videochan <- sp
			log.Debug("IN", " <<<<<<< ", mm)
		} else {
			log.Warn("IN", " <<<<<<< ", mm)
			s.Close()
			break
		}
	}
	log.Info("RtmpNetStream " + s.conn.remoteAddr + " " + s.path + " loop stopped")
}

//客户端模式
func (s *RtmpNetStream) cserve() {
	s.loop()
	//for {

	//	//msg, err := readMessage(conn)
	//	//if err != nil {
	//	//	notifyError(s, err)
	//	//	return
	//	//}
	//	//log.Debug(msg, err)
	//}
}

/**
服务端模式
**/
func (s *RtmpNetStream) serve() {
	conn := s.conn
	for {
		msg, err := readMessage(conn)
		if err != nil {
			notifyError(s, err)
			break
		}
		//log.Debug("<<<<< ", msg)
		if msg.Header().MessageLength <= 0 {
			continue
		}
		if am, ok := msg.(*AudioMessage); ok {
			sp := new(StreamPacket)
			if am.RtmpHeader.Timestamp == 0xffffff {
				s.mrev_duration += am.RtmpHeader.ExtendTimestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			} else {
				s.mrev_duration += am.RtmpHeader.Timestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			}
			sp.Type = am.RtmpHeader.MessageType
			sp.Payload = am.Payload
			one := sp.Payload[0]
			sp.AudioFormat = one >> 4
			sp.SamplingRate = (one & 0x0c) >> 2
			sp.SampleLength = (one & 0x02) >> 1
			sp.AudioType = one & 0x01
			if s.firstAudioKeyFrame == nil {
				s.firstAudioKeyFrame = sp
			} else {
				s.audiochan <- sp
			}
		} else if vm, ok := msg.(*VideoMessage); ok {
			sp := new(StreamPacket)
			if vm.RtmpHeader.Timestamp == 0xffffff {
				s.mrev_duration += vm.RtmpHeader.ExtendTimestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			} else {
				s.mrev_duration += vm.RtmpHeader.Timestamp
				sp.Timestamp = s.mrev_duration
				//log.Debug("Media Total Duration ", s.mrev_duration)
			}
			sp.Type = vm.RtmpHeader.MessageType
			sp.Payload = vm.Payload
			one := sp.Payload[0]
			sp.VideoFrameType = one >> 4
			sp.VideoCodecID = one & 0x0f
			if s.firstVideoKeyFrame == nil {
				s.firstVideoKeyFrame = sp
			} else {
				if sp.VideoFrameType == 1 {
					s.lastVideoKeyFrame = sp
				}
				s.videochan <- sp
			}
			//debug("IN", " <<<<<<< ", vm)
		} else if mm, ok := msg.(*MetadataMessage); ok {
			sp := new(StreamPacket)
			sp.Timestamp = mm.RtmpHeader.Timestamp
			if mm.RtmpHeader.Timestamp == 0xffffff {
				sp.Timestamp = mm.RtmpHeader.ExtendTimestamp
			}
			sp.Type = mm.RtmpHeader.MessageType
			sp.Payload = mm.Payload
			if s.metaData == nil {
				s.metaData = sp
			}
			//s.videochan <- sp
			log.Debug("IN", " <<<<<<< ", mm)
		} else if csm, ok := msg.(*CreateStreamMessage); ok {
			log.Debug("IN", " <<<<<<< ", csm)
			conn.streamid = conn.nextStreamId(csm.RtmpHeader.ChunkId)
			err := sendCreateStreamResult(conn, csm.TransactionId)
			if err != nil {
				notifyError(s, err)
				return
			}
		} else if m, ok := msg.(*PublishMessage); ok {
			log.Debug("IN", " <<<<<<< ", m)
			if strings.HasSuffix(conn.app, "/") {
				s.path = conn.app + strings.Split(m.PublishName, "?")[0]
			} else {
				s.path = conn.app + "/" + strings.Split(m.PublishName, "?")[0]
			}
			if err := notifyPublishing(s); err != nil {
				err = sendPublishResult(conn, "error", err.Error())
				if err != nil {
					notifyError(s, err)
				}
				return
			}
			err := sendStreamBegin(conn)
			if err != nil {
				notifyError(s, err)
				return
			}
			err = sendPublishStart(conn)
			if err != nil {
				notifyError(s, err)
				return
			}
			if s.mode == 0 {
				s.mode = MODE_PRODUCER
			} else {
				s.mode = s.mode | MODE_PRODUCER
			}
		} else if m, ok := msg.(*PlayMessage); ok {
			log.Debug("IN", " <<<<<<< ", m)
			if strings.HasSuffix(s.conn.app, "/") {
				s.path = s.conn.app + strings.Split(m.StreamName, "?")[0]
			} else {
				s.path = s.conn.app + "/" + strings.Split(m.StreamName, "?")[0]
			}
			conn.writeChunkSize = 512 //RTMP_MAX_CHUNK_SIZE
			err = notifyPlaying(s)
			if err != nil {
				err = sendPlayResult(conn, "error", err.Error())
				if err != nil {
					notifyError(s, err)
				}
				return
			}
			err := sendChunkSize(conn, uint32(conn.writeChunkSize))
			if err != nil {
				notifyError(s, err)
				return
			}
			err = sendStreamRecorded(conn)
			if err != nil {
				notifyError(s, err)
				return
			}
			err = sendStreamBegin(conn)
			if err != nil {
				notifyError(s, err)
				return
			}
			err = sendPlayReset(conn)
			if err != nil {
				notifyError(s, err)
				return
			}
			err = sendPlayStart(conn)
			if err != nil {
				notifyError(s, err)
				return
			}
			if s.mode == 0 {
				s.mode = MODE_CONSUMER
			} else {
				s.mode = s.mode | MODE_CONSUMER
			}
		} else if m, ok := msg.(*CloseStreamMessage); ok {
			log.Debug("IN", " <<<<<<< ", m)
			s.Close()
		} else {
			log.Debug("IN Why?", " <<<<<<< ", msg)
		}
	}
}
