package amf

import (
	"encoding/binary"
	"io"
)

var (
	/* basic types */
	NGX_RTMP_AMF_NUMBER      = 0x00
	NGX_RTMP_AMF_BOOLEAN     = 0x01
	NGX_RTMP_AMF_STRING      = 0x02
	NGX_RTMP_AMF_OBJECT      = 0x03
	NGX_RTMP_AMF_NULL        = 0x05
	NGX_RTMP_AMF_ARRAY_NULL  = 0x06
	NGX_RTMP_AMF_MIXED_ARRAY = 0x08
	NGX_RTMP_AMF_END         = 0x09
	NGX_RTMP_AMF_ARRAY       = 0x0a

	/* extended types */
	NGX_RTMP_AMF_INT8     = 0x0100
	NGX_RTMP_AMF_INT16    = 0x0101
	NGX_RTMP_AMF_INT32    = 0x0102
	NGX_RTMP_AMF_VARIANT_ = 0x0103

	/* r/w flags */
	NGX_RTMP_AMF_OPTIONAL = 0x1000
	NGX_RTMP_AMF_TYPELESS = 0x2000
	NGX_RTMP_AMF_CONTEXT  = 0x4000

	NGX_RTMP_AMF_VARIANT = NGX_RTMP_AMF_VARIANT_ | NGX_RTMP_AMF_TYPELESS
)

type amf_elt struct {
	atype  int
	name   string
	data   interface{}
	length int
}

//typedef ngx_chain_t * (*ngx_rtmp_amf_alloc_pt)(void *arg);

type amf_ctx struct {
	//ngx_chain_t                        *link, *first;
	offset int
	// ngx_rtmp_amf_alloc_pt               alloc;
	arg interface{}
	// ngx_log_t                          *log;
}

///* reading AMF */
//ngx_int_t ngx_rtmp_amf_read(ngx_rtmp_amf_ctx_t *ctx,
//        ngx_rtmp_amf_elt_t *elts, size_t nelts);

///* writing AMF */
//ngx_int_t ngx_rtmp_amf_write(ngx_rtmp_amf_ctx_t *ctx,
//        ngx_rtmp_amf_elt_t *elts, size_t nelts);

//NetConnection.Call.Failed
//NetConnection.Call.BadVersion
//NetConnection.Connect.AppShutdown
//NetConnection.Connect.Closed
//NetConnection.Connect.Rejected
//NetConnection.Connect.Success
//NetStream.Clear.Success
//NetStream.Clear.Failed
//NetStream.Publish.Start
//NetStream.Publish.BadName
//NetStream.Failed
//NetStream.Unpublish.Success
//NetStream.Record.Start
//NetStream.Record.NoAccess
//NetStream.Record.Stop
//NetStream.Record.Failed
//NetStream.Play.InsufficientBW
//NetStream.Play.Start
//NetStream.Play.StreamNotFound
//NetStream.Play.Stop
//NetStream.Play.Failed
//NetStream.Play.Reset
//NetStream.Play.PublishNotify
//NetStream.Play.UnpublishNotify
//NetStream.Data.Start
//Application.Script.Error
//Application.Script.Warning
//Application.Resource.LowMemory
//Application.Shutdown
//Application.GC
//Play
//Pause
//demoService.getListOfAvailableFLVs
//getStreamLength
//connect
//app
//flashVer
//swfUrl
//tcUrl
//fpad
//capabilities
//audioCodecs
//audioCodecs
//videoCodecs
//videoFunction
//pageUrl
//createStream
//deleteStream
//duration
//framerate
//audiocodecid
//audiodatarate
//videocodecid
//videodatarate
//height
//width\\package rtmp

var (
	AMF_NUMBER      = 0x00
	AMF_BOOLEAN     = 0x01
	AMF_STRING      = 0x02
	AMF_OBJECT      = 0x03
	AMF_NULL        = 0x05
	AMF_ARRAY_NULL  = 0x06
	AMF_MIXED_ARRAY = 0x08
	AMF_END         = 0x09
	AMF_ARRAY       = 0x0a

	AMF_INT8     = 0x0100
	AMF_INT16    = 0x0101
	AMF_INT32    = 0x0102
	AMF_VARIANT_ = 0x0103
)

type AMFMessage struct {
	Type         byte
	Command      string
	MsgId        int
	Object       interface{}
	OptionObject interface{}
}
type AMFObj struct {
	atype int
	str   string
	i     int
	buf   []byte
	obj   map[string]AMFObj
	f64   float64
}

func ReadAMF(r io.Reader) (a AMFObj) {
	a.atype = ReadInt(r, 1)
	switch a.atype {
	case AMF_STRING:
		n := ReadInt(r, 2)
		b := ReadBuf(r, n)
		a.str = string(b)
	case AMF_NUMBER:
		binary.Read(r, binary.BigEndian, &a.f64)
	case AMF_BOOLEAN:
		a.i = ReadInt(r, 1)
	case AMF_MIXED_ARRAY:
		ReadInt(r, 4)
		fallthrough
	case AMF_OBJECT:
		a.obj = map[string]AMFObj{}
		for {
			n := ReadInt(r, 2)
			if n == 0 {
				break
			}
			name := string(ReadBuf(r, n))
			a.obj[name] = ReadAMF(r)
		}
	case AMF_ARRAY, AMF_VARIANT_:
		panic("amf: read: unsupported array or variant")
	case AMF_INT8:
		a.i = ReadInt(r, 1)
	case AMF_INT16:
		a.i = ReadInt(r, 2)
	case AMF_INT32:
		a.i = ReadInt(r, 4)
	}
	return
}

func WriteAMF(r io.Writer, a AMFObj) {
	WriteInt(r, a.atype, 1)
	switch a.atype {
	case AMF_STRING:
		WriteInt(r, len(a.str), 2)
		r.Write([]byte(a.str))
	case AMF_NUMBER:
		binary.Write(r, binary.BigEndian, a.f64)
	case AMF_BOOLEAN:
		WriteInt(r, a.i, 1)
	case AMF_MIXED_ARRAY:
		r.Write(a.buf[:4])
	case AMF_OBJECT:
		for name, val := range a.obj {
			WriteInt(r, len(name), 2)
			r.Write([]byte(name))
			WriteAMF(r, val)
		}
		WriteInt(r, 9, 3)
	case AMF_ARRAY, AMF_VARIANT_:
		panic("amf: write unsupported array, var")
	case AMF_INT8:
		WriteInt(r, a.i, 1)
	case AMF_INT16:
		WriteInt(r, a.i, 2)
	case AMF_INT32:
		WriteInt(r, a.i, 4)
	}
}

func ReadBuf(r io.Reader, n int) (b []byte) {
	b = make([]byte, n)
	r.Read(b)
	return
}

func ReadInt(r io.Reader, n int) (ret int) {
	b := ReadBuf(r, n)
	for i := 0; i < n; i++ {
		ret <<= 8
		ret += int(b[i])
	}
	return
}

func ReadIntLE(r io.Reader, n int) (ret int) {
	b := ReadBuf(r, n)
	for i := 0; i < n; i++ {
		ret <<= 8
		ret += int(b[n-i-1])
	}
	return
}

func WriteBuf(w io.Writer, buf []byte) {
	w.Write(buf)
}

func WriteInt(w io.Writer, v int, n int) {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[n-i-1] = byte(v & 0xff)
		v >>= 8
	}
	WriteBuf(w, b)
}

func WriteIntLE(w io.Writer, v int, n int) {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte(v & 0xff)
		v >>= 8
	}
	WriteBuf(w, b)
}
