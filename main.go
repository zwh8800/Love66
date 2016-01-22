package main

import (
	"log"
	"unsafe"

	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avcodec"
	"fmt"
	"github.com/giorgisio/goav/avutil"
)

const (
	AVMEDIA_TYPE_UNKNOWN = iota - 1
	AVMEDIA_TYPE_VIDEO
	AVMEDIA_TYPE_AUDIO
	AVMEDIA_TYPE_DATA
	AVMEDIA_TYPE_SUBTITLE
	AVMEDIA_TYPE_ATTACHMENT
	AVMEDIA_TYPE_NB
)

func main() {
	fmt.Println(AVMEDIA_TYPE_VIDEO)

	log.Printf("start\n")
	filename := "http://58.55.123.85/hdl1a.douyutv.com/live/319598rGAlaPK6ib_550.flv?wsSecret=36f7bd10b930f61e002e41f81b6797cc&wsTime=1453443814&wshc_tag=0&wsts_tag=56a1caea&wsid_tag=6a273ab2&wsiphost=ipdbm"
	avformat.AvRegisterAll()
	avformat.AvformatNetworkInit()
	var formatContext *avformat.Context

	if avformat.AvformatOpenInput(&formatContext, filename, nil, nil) != 0 {
		log.Println("Error: Couldn't open file.")
		return
	}
	if formatContext.AvformatFindStreamInfo(nil) < 0 {
		log.Println("Error: Couldn't find stream information.")
		return
	}

	formatContext.AvDumpFormat(0, filename, 0)

	var codecContext *avcodec.Context
	n := formatContext.NbStreams()
	log.Printf("number of streams: %d\n", n)

	audioFrame := -1
	s := formatContext.Streams()

	for i := 0; i < int(n); i++ {
		log.Println("Stream Number:", i)
		codec := (*avformat.Stream)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) + uintptr(i))).Codec()
		codecContext = (*avcodec.Context)(unsafe.Pointer(&codec))
		if codecContext.CodecType() == AVMEDIA_TYPE_AUDIO {
			audioFrame = i
			break
		}
	}

	log.Println("Bit Rate:", codecContext.BitRate())
	log.Println("Channels:", codecContext.Channels())
	log.Println("Coded_height:", codecContext.CodedHeight())
	log.Println("Coded_width:", codecContext.CodedWidth())
	log.Println("Coder_type:", codecContext.CoderType())
	log.Println("Height:", codecContext.Height())
	log.Println("Profile:", codecContext.Profile())
	log.Println("Width:", codecContext.Width())
	log.Println("Codec ID:", codecContext.CodecId())

	codecId := codecContext.CodecId()
	audioCodec := avcodec.AvcodecFindDecoder(codecId)

	if codecContext.AvcodecOpen2(audioCodec, nil) < 0 {
		log.Println("Error: Couldn't open codec.")
		return
	}

	var packet *avcodec.Packet
	frame := avutil.AvFrameAlloc()

	for formatContext.AvReadFrame(packet) >= 0 {
		var gotFrame int
		if packet.StreamIndex() == audioFrame {
			if codecContext.AvcodecDecodeAudio4((*avcodec.Frame)(unsafe.Pointer(frame)), &gotFrame, packet) < 0 {
				log.Println("Error in decoding audio frame.")
				return
			}
			if gotFrame > 0 {



			} else {
				log.Println("finish")
			}
		}
	}

}
