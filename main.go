package main

//#cgo pkg-config: libavutil
//#cgo pkg-config: sdl2
//#include <stdlib.h>
//#include <libavutil/avutil.h>
//#include <libavutil/channel_layout.h>
//#include <libswresample/swresample.h>
//#include <SDL.h>
/*
extern void fillAudio(Uint8 *udata, Uint8 *stream, int len);

static void set_callback(SDL_AudioSpec* wanted) {
	wanted->callback = (SDL_AudioCallback)fillAudio;
}
*/
import "C"
import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"unsafe"

	"io"
	"os"
	"strconv"

	"bytes"
	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
	"github.com/giorgisio/goav/swresample"
	"github.com/veandco/go-sdl2/sdl"
)

const MAX_AUDIO_FRAME_SIZE = 192000

type DouyuRoomInfo struct {
	Error int `json:"error"`
	Data  struct {
		RoomID           string `json:"room_id"`
		RoomSrc          string `json:"room_src"`
		CateID           string `json:"cate_id"`
		RoomName         string `json:"room_name"`
		VodQuality       string `json:"vod_quality"`
		ShowStatus       string `json:"show_status"`
		ShowTime         string `json:"show_time"`
		OwnerUID         string `json:"owner_uid"`
		SpecificCatalog  string `json:"specific_catalog"`
		SpecificStatus   string `json:"specific_status"`
		Online           int    `json:"online"`
		Nickname         string `json:"nickname"`
		URL              string `json:"url"`
		GameURL          string `json:"game_url"`
		GameName         string `json:"game_name"`
		GameIconURL      string `json:"game_icon_url"`
		RtmpURL          string `json:"rtmp_url"`
		RtmpLive         string `json:"rtmp_live"`
		RtmpCdn          string `json:"rtmp_cdn"`
		RtmpMultiBitrate struct {
			Middle  string `json:"middle"`
			Middle2 string `json:"middle2"`
		} `json:"rtmp_multi_bitrate"`
		HlsURL  string `json:"hls_url"`
		Servers []struct {
			IP   string `json:"ip"`
			Port string `json:"port"`
		} `json:"servers"`
		UseP2P      string        `json:"use_p2p"`
		RoomDmDelay int           `json:"room_dm_delay"`
		Black       []interface{} `json:"black"`
		ShowDetails string        `json:"show_details"`
		OwnerAvatar string        `json:"owner_avatar"`
		Cdns        []string      `json:"cdns"`
		OwnerWeight string        `json:"owner_weight"`
		Fans        string        `json:"fans"`
		Gift        []struct {
			ID               string  `json:"id"`
			Name             string  `json:"name"`
			Pc               string  `json:"pc"`
			Type             string  `json:"type"`
			Gx               float64 `json:"gx"`
			Desc             string  `json:"desc"`
			Intro            string  `json:"intro"`
			Ef               int     `json:"ef"`
			Pimg             string  `json:"pimg"`
			Mimg             string  `json:"mimg"`
			Cimg             string  `json:"cimg"`
			Himg             string  `json:"himg"`
			StayTime         int     `json:"stay_time"`
			Drgb             string  `json:"drgb"`
			Urgb             string  `json:"urgb"`
			Grgb             string  `json:"grgb"`
			Brgb             string  `json:"brgb"`
			Pdbimg           string  `json:"pdbimg"`
			Pdhimg           string  `json:"pdhimg"`
			SmallEffectIcon  string  `json:"small_effect_icon"`
			BigEffectIcon    string  `json:"big_effect_icon"`
			PadBigEffectIcon string  `json:"pad_big_effect_icon"`
		} `json:"gift"`
	} `json:"data"`
}

func resolveStream(roomId int) string {
	suffix := fmt.Sprintf("room/%d?aid=android&client_sys=android&time=%d", roomId, time.Now().Unix())
	sumArr := md5.Sum([]byte(suffix + "1231"))
	sum := sumArr[:]
	sign := hex.EncodeToString(sum)
	url := fmt.Sprintf("http://www.douyutv.com/api/v1/%s&auth=%s", suffix, sign)

	log.Println("request to: ", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Panic("http error: ", err)
	}
	defer resp.Body.Close()
	respBodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic("ioutil.ReadAll error: ", err)
	}
	var douyuRoomInfo DouyuRoomInfo
	err = json.Unmarshal(respBodyData, &douyuRoomInfo)
	if err != nil {
		log.Panic("json.Unmarshal error: ", err, "主播可能未开播")
	}

	log.Println("room info: ", douyuRoomInfo)

	if douyuRoomInfo.Error != 0 {
		log.Panic("server response error: ", douyuRoomInfo.Error, "主播可能未开播")
	}
	if douyuRoomInfo.Data.ShowStatus != "1" {
		log.Panic("The live stream is not online!: ", douyuRoomInfo.Error, "主播可能未开播")
	}

	liveUrl := douyuRoomInfo.Data.RtmpURL + "/" + douyuRoomInfo.Data.RtmpLive
	log.Println("find live url: ", liveUrl)
	return liveUrl
}

func AvGetDefaultChannelLayout(channels int) int64 {
	return int64(C.av_get_default_channel_layout(C.int(channels)))
}
func AvGetChannelLayoutNbChannels(channelLayout uint64) int {
	return int(C.av_get_channel_layout_nb_channels(C.uint64_t(channelLayout)))
}

type Frame C.struct_AVFrame

func (p *Frame) Data() **uint8 {
	return (**uint8)(unsafe.Pointer(&p.data[0]))
}

func (p *Frame) NbSamples() int {
	return (int)(p.nb_samples)
}

var audioLen uint32
var audioPos *uint8

var audioBuffer bytes.Buffer

//export fillAudio
func fillAudio(uData *C.Uint8, stream *C.Uint8, len C.int) {
	log.Println("audioLen = ", audioLen, "len = ", len)

	C.memset(unsafe.Pointer(stream), 0, C.size_t(len))
	if audioLen == 0 {
		return
	}
	var length uint32 = uint32(len)
	if length > audioLen {
		length = audioLen
	}

	sdl.MixAudio((*uint8)(stream), audioPos, length, sdl.MIX_MAXVOLUME)
	audioLen -= length

	audioPos = (*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(audioPos)) + uintptr(length)))
}

func main() {
	log.Printf("start\n")
	flag.Parse()
	roomId, err := strconv.ParseInt(flag.Arg(0), 10, 32)
	if err != nil {
		roomId = 156277
	}

	var pcmFile io.Writer

	pcmFileName := flag.Arg(1)
	if pcmFileName == "" {
		pcmFileName = "./output.pcm"
		pcmFile, err = os.OpenFile(pcmFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
	} else if pcmFileName == "-" {
		pcmFile = os.Stdout
	} else {
		pcmFile, err = os.OpenFile(pcmFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	filename := resolveStream(int(roomId))
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

	var codecContext *avcodec.CodecContext
	n := formatContext.NbStreams()
	log.Printf("number of streams: %d\n", n)

	audioFrame := -1

	for i := 0; i < int(n); i++ {
		log.Println("Stream Number:", i)
		codec := formatContext.Streams(uint(i)).Codec()
		codecContext = codec
		log.Println("Stream CodecType:", codecContext.CodecType())
		log.Println("AVMEDIA_TYPE_AUDIO: ", C.AVMEDIA_TYPE_AUDIO)
		if codecContext.CodecType() == C.AVMEDIA_TYPE_AUDIO {
			audioFrame = i
			log.Println("audioFrame: ", audioFrame)
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
	log.Println("audioCodec", audioCodec)

	var packet *avcodec.Packet = new(avcodec.Packet)
	packet.AvInitPacket()
	utilFrame := avutil.AvFrameAlloc()

	outSampleRate := 48000

	swrContext := swresample.SwrAlloc()
	swrContext.SwrAllocSetOpts(C.AV_CH_LAYOUT_STEREO, C.AV_SAMPLE_FMT_S16, outSampleRate,
		AvGetDefaultChannelLayout(codecContext.Channels()),
		(swresample.AvSampleFormat)(codecContext.SampleFmt()), codecContext.SampleRate(), 0, 0)
	swrContext.SwrInit()

	if err := sdl.Init(sdl.INIT_AUDIO | sdl.INIT_TIMER); err != nil {
		log.Fatal("sdl.Init(sdl.INIT_AUDIO): ", err)
	}
	var wanted sdl.AudioSpec
	wanted.Freq = int32(outSampleRate)
	wanted.Format = sdl.AUDIO_S16SYS
	wanted.Channels = 2
	wanted.Silence = 0
	wanted.Samples = uint16(codecContext.FrameSize())
	C.set_callback((*C.SDL_AudioSpec)(unsafe.Pointer(&wanted)))
	log.Printf("var wanted sdl.AudioSpec = %#v\n", wanted)

	if err := sdl.OpenAudio(&wanted, nil); err != nil {
		log.Fatal("sdl.OpenAudio(&wanted, nil): ", err)
	}

	index := 0
	outBuffer := [MAX_AUDIO_FRAME_SIZE]uint8{}
	outBufferArray := [...]*uint8{&outBuffer[0]}
	for formatContext.AvReadFrame(packet) >= 0 {
		log.Println("--------")
		log.Println("Packet read:", packet)
		log.Println("Packet StreamIndex:", packet.StreamIndex())
		if packet.StreamIndex() == audioFrame {
			log.Println("audioFrame")

			codecFrame := (*avcodec.Frame)(unsafe.Pointer(utilFrame))

			var gotFrame int
			if codecContext.AvcodecDecodeAudio4(codecFrame, &gotFrame, packet) < 0 {
				log.Println("Error in decoding audio frame.")
				return
			}
			if gotFrame > 0 {
				log.Println("got")
				log.Printf("index:%5d\t pts:%d\t packet size:%d\n", index, packet.Pts(), packet.Size())
				frame := (*Frame)(unsafe.Pointer(codecFrame))
				n := swrContext.SwrConvert(&outBufferArray[0], MAX_AUDIO_FRAME_SIZE, frame.Data(), frame.NbSamples())

				len := 2 * 2 * n
				log.Println("n: ", n, "bytes: ", len)

				for audioLen > 0 {
					sdl.Delay(1)
				}
				audioPos = &outBuffer[0]
				audioLen = uint32(len)

				sdl.PauseAudio(false)

				_, err := pcmFile.Write(outBuffer[:len])
				if err != nil {
					log.Fatal(err)
				}
				index++
			} else {
				log.Println("finish")
			}
		}
		packet.AvFreePacket()
	}

}
