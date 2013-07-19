babylon
=======

Another live streaming media server, written by go. 
support rtmp
How to use
  runtime.GOMAXPROCS(runtime.NumCPU())
  l := ":1935"
  err := rtmp.ListenAndServe(l)
	if err != nil {
		panic(err)
	}
  select {}
  
roadmap
   rtmp reverse proxy and forward           2013.10.20
   rtsp(h264/aac),hls(h264/aac) 2013.12.20
   rtsp(h264/aac) forward      2014.3.20
