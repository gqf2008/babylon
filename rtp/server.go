package rtp

import (
	"fmt"
	"net"
)

func ListenAndServe(addr string, port int) {
	//addr_tcp, err := net.ResolveTCPAddr("tcp", l)
	//if err != nil {
	//	panic(err)
	//}
	//listen_tcp, err := net.ListenTCP("tcp", addr_tcp)
	//if err != nil {
	//	panic(err)
	//}
	//go tcp_listen(listen_tcp)
	rtp_l := fmt.Sprintf("%v:%v", addr, port)
	rtcp_l := fmt.Sprintf("%v:%v", addr, port+1)
	addr1, err := net.ResolveUDPAddr("udp", rtp_l)
	if err != nil {
		panic(err)
	}
	listen_rtp, err := net.ListenUDP("udp", addr1)
	if err != nil {
		panic(err)
	}
	go udp_rtp_listen(listen_rtp)

	addr2, err := net.ResolveUDPAddr("udp", rtcp_l)
	if err != nil {
		panic(err)
	}
	listen_rtcp, err := net.ListenUDP("udp", addr2)
	if err != nil {
		panic(err)
	}
	go udp_rtcp_listen(listen_rtcp)
}

func tcp_rtp_listen(listen *net.TCPListener) {
	for {
		sock, err := listen.AcceptTCP()
		if err != nil {
			fmt.Printf("Accept Error: %+v", err)
			continue
		}
		go handle_rtp_connection(sock, nil)

	}
}

func tcp_rtcp_listen(listen *net.TCPListener) {
	for {
		sock, err := listen.AcceptTCP()
		if err != nil {
			fmt.Printf("Accept Error: %+v", err)
			continue
		}
		go handle_rtcp_connection(sock, nil)

	}
}

func udp_rtp_listen(conn *net.UDPConn) {
	for {
		handle_rtp_connection(nil, conn)
	}
}

func udp_rtcp_listen(conn *net.UDPConn) {
	for {

		handle_rtcp_connection(nil, conn)
	}
}

//func onAccepted(sock *net.TCPConn, listen *net.TCPListener) {
//	defer sock.Close()
//	//log.Printf("Accept: %s/%s", sock.RemoteAddr(), sock.LocalAddr())
//	sock.SetKeepAlive(true)
//	sock.SetNoDelay(true)

//	data := make([]byte, 128)
//	sock.SetReadDeadline(time.Now().Add(time.Duration(30) * time.Second))
//	l, err := sock.Read(data)
//	if err != nil || l < 0 {
//		return
//	}
//	info := strings.Split(strings.Trim(string(data), string(0)), "/")
//	//log.Println(info)
//	file := fmt.Sprintf("%s%s%s%s%s", strings.Replace(info[0], " ", "", -1), strings.Replace(info[1], ".", "", -1), info[2], time.Now().Format("20060102150405"), ".mp4")
//	//log.Println(file)
//	fout, err := os.Create(file)
//	if err != nil {
//		fmt.Printf("Accept Error: %+v", err)
//		return
//	}
//	defer fout.Close()

//	for {
//		data := make([]byte, 4096)
//		sock.SetReadDeadline(time.Now().Add(time.Duration(30) * time.Second))
//		l, err := sock.Read(data)
//		if err != nil || l < 0 {
//			break
//		}
//		fout.Write(data[0 : l-1])
//	}
//	//log.Printf("Goroutine: %s/%s Exit", sock.RemoteAddr(), sock.LocalAddr())
//}
