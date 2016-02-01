package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"unsafe"
	"flag"

	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
	"strconv"
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
		log.Panic("json.Unmarshal error: ", err)
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

func main() {
	log.Printf("start\n")
	flag.Parse()
	roomId, err := strconv.ParseInt(flag.Arg(0), 10, 32)
	if err != nil {
		roomId = 156277
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
		codecContext = (*avcodec.CodecContext)(unsafe.Pointer(&codec))
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
