babylon
=======
[bbllive](https://github.com/gqf2008/bbllive) is newest

Another live streaming media server, written by go. support rtmp
### How to use

```
package main

import (
	"babylon/rtmp"
	log "github.com/cihub/seelog"
	"runtime"
)


func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  l := ":1935"
  err := rtmp.ListenAndServe(l)
  if err != nil {	
     panic(err)		
  }
  select {}
}
```

### How to publish/play streaming

* ffmpeg -i xxxx.mp4 -c:a aac -ar 44100 -ab 128k -ac 2 -strict -2 -c:v libx264 -vb 500k -r 30 -s 640x480 -ss 00.000 -f flv rtmp://127.0.0.1/live/xxxx
* ffplay -i rtmp://127.0.0.1/live/xxxx

   

