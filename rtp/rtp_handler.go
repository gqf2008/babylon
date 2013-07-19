package rtp

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
)

func handle_rtp_connection(sock *net.TCPConn, conn *net.UDPConn) {
	file, err := os.Open("N7HARD.h264")
	if err != nil {
		panic(err)
	}
	r := bufio.NewReader(file)
	var reset []byte
	for {
		b, reset := readNalu(r, reset)
		if b == nil && (reset == nil || len(reset) == 0) {
			break
		}
		if b != nil {
			fmt.Printf("NALU %02X\n", b)
			fmt.Printf("RESET %02X\n", reset)
			err := writeH264(conn, b, 45)
			if err != nil {
				panic(err)
				break
			}
		}
	}
}

func readNalu(r *bufio.Reader, reset []byte) (Payload, []byte) {
	b := readBuffer(r, 4096)
	if (reset == nil || len(reset) == 0) && len(b) == 0 {
		return nil, nil
	}
	buf := new(bytes.Buffer)
	if reset != nil {
		buf.Write(reset)
	}
	if len(b) > 0 {
		buf.Write(b)
	} else {
		if b[0] == 0x00 && b[1] == 0x00 && b[2] == 0x01 {
			return b[3:], nil
		} else if b[0] == 0x00 && b[1] == 0x00 && b[2] == 0x00 && b[3] == 0x01 {
			return b[4:], nil
		}
	}
	return parserH264(buf.Bytes())
}

func parserH264(buf []byte) (Payload, []byte) {
	begin := 0
	end := 0
	length := len(buf)
	if buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x01 {
		begin = 3
		end = 3
	} else if buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x00 && buf[3] == 0x01 {
		begin = 4
		end = 4
	}
	for {
		if end == length {
			fmt.Printf("None %v/%v/%v\n", begin, end, length)
			return nil, buf
		}
		if buf[end] == 0x00 && buf[end+1] == 0x00 && buf[end+2] == 0x01 {
			fmt.Printf("000001 %v/%v/%v\n", begin, end, length)
			packet := buf[begin:end]
			return packet, buf[end:]
		} else if buf[end] == 0x00 && buf[end+1] == 0x00 && buf[end+2] == 0x00 && buf[end+3] == 0x01 {
			fmt.Printf("00000001 %v/%v/%v\n", begin, end, length)
			packet := buf[begin:end]
			return packet, buf[end:]
		} else {
			end += 1
		}
	}
}

func readBuffer(r *bufio.Reader, size int) []byte {
	b := make([]byte, size)
	i, err := r.Read(b)
	if err != nil {
		panic(err)
	}
	if i > 0 {
		return b[:i]
	}
	return b[0:0]
}

func unreadBuffer(r *bufio.Reader, size int) {
	for i := 0; i < size; i++ {
		r.UnreadByte()
	}
}

func writeH264(conn *net.UDPConn, frame []byte, timestamp uint32) error {
	return nil
}

func handle_rtcp_connection(sock *net.TCPConn, conn *net.UDPConn) {

}
