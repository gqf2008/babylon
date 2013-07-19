// mss project main.go
package main

import (
	"babylon/rtmp"
	_ "flag"
	log "github.com/cihub/seelog"
	_ "net"
	_ "net/rtp"
	_ "path/filepath"
	"runtime"
)

//UINT GetUeValue(BYTE *pBuff, UINT nLen, UINT &nStartBit) {
// //计算0bit的个数
// UINT nZeroNum = 0;
//while (nStartBit < nLen * 8)
//{
// if (pBuff[nStartBit / 8] & (0x80 >> (nStartBit % 8)))
//  {
//  break;
//    }
//	  nZeroNum++;
//	     nStartBit++;
//}
//		 nStartBit ++;
//		 //计算结果
//		 DWORD dwRet = 0;
//		   for (UINT i=0; i<nZeroNum; i++)
//		    {
//			dwRet <<= 1;
//			      if (pBuff[nStartBit / 8] & (0x80 >> (nStartBit % 8)))
//				        {
//						     dwRet += 1;
//							       }

//								   nStartBit++;
//								  }

//								return (1 << nZeroNum) - 1 + dwRet;

//								 }

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//filePath, _ := filepath.Abs(os.Args[0])
	//file := filepath.Dir(filePath) + "../etc/mss.conf"
	//c := flag.String("c", file, "config file path")
	//daemon := flag.Bool("d", false, "daemon mode")
	////s := flag.String("s", "", "test|reload|nodes|static|table|shutdown")
	//flag.Parse()
	//if *daemon && os.Getppid() != 1 {
	//	filePath, _ := filepath.Abs(os.Args[0])
	//	args := append([]string{filePath}, os.Args[1:]...)
	//	os.StartProcess(filePath, args, &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
	//	return
	//}
	//e := config_init(*c)
	//if e != nil {
	//	panic(e)
	//}
	//log_init()

	l := ":1935" //GetString("rtmp", "listen", ":1935")
	err := rtmp.ListenAndServe(l)
	if err != nil {
		panic(err)
	}
	log.Info("Babylon Rtmp Server Listen At " + l)

	log.Info("Babylon Media Streaming Server Running")

	select {}
}
