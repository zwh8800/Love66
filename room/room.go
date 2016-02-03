package room

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type DouyuRoom struct {
	roomId   int
	roomInfo *douyuRoomInfoJson
}

func NewDouyuRoom(roomId int) (*DouyuRoom, error) {
	url := resolveApiUrl(roomId)
	roomInfo, err := getRoomInfo(url)
	if err != nil {
		return nil, err
	}
	return &DouyuRoom{roomId, roomInfo}, nil
}

func (r *DouyuRoom) Refresh() error {
	url := resolveApiUrl(r.roomId)
	roomInfo, err := getRoomInfo(url)
	if err != nil {
		return err
	}
	r.roomInfo = roomInfo
	return nil
}

func (r *DouyuRoom) Online() bool {
	return r.roomInfo.Data.ShowStatus == "1"
}

func (r *DouyuRoom) RoomId() int {
	id, err := strconv.ParseInt(r.roomInfo.Data.RoomID, 10, 32)
	if err != nil {
		return r.roomId
	}
	return int(id)
}

func (r *DouyuRoom) RoomName() string {
	return r.roomInfo.Data.RoomName
}

func (r *DouyuRoom) Nickname() string {
	return r.roomInfo.Data.Nickname
}

func (r *DouyuRoom) GameName() string {
	return r.roomInfo.Data.GameName
}

func (r *DouyuRoom) Details() string {
	return r.roomInfo.Data.ShowDetails
}

func (r *DouyuRoom) LiveStreamUrl() string {
	if r.Online() {
		return r.roomInfo.Data.RtmpURL + "/" + r.roomInfo.Data.RtmpLive
	} else {
		return ""
	}
}

func resolveApiUrl(roomId int) string {
	suffix := fmt.Sprintf("room/%d?aid=android&client_sys=android&time=%d", roomId, time.Now().Unix())
	sumArr := md5.Sum([]byte(suffix + "1231"))
	sum := sumArr[:]
	sign := hex.EncodeToString(sum)
	url := fmt.Sprintf("http://www.douyutv.com/api/v1/%s&auth=%s", suffix, sign)
	return url
}

func getRoomInfo(url string) (*douyuRoomInfoJson, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var info douyuRoomInfoJson

	if err := json.Unmarshal(respBodyData, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// 不关注的字段统统用interface{}, 防止parse出错
type douyuRoomInfoJson struct {
	Error int `json:"error"`
	Data  struct {
		RoomID           string      `json:"room_id"`
		RoomSrc          string      `json:"room_src"`
		CateID           string      `json:"cate_id"`
		RoomName         string      `json:"room_name"`
		VodQuality       string      `json:"vod_quality"`
		ShowStatus       string      `json:"show_status"`
		ShowTime         string      `json:"show_time"`
		OwnerUID         string      `json:"owner_uid"`
		SpecificCatalog  string      `json:"specific_catalog"`
		SpecificStatus   string      `json:"specific_status"`
		Online           int         `json:"online"`
		Nickname         string      `json:"nickname"`
		URL              string      `json:"url"`
		GameURL          string      `json:"game_url"`
		GameName         string      `json:"game_name"`
		GameIconURL      string      `json:"game_icon_url"`
		RtmpURL          string      `json:"rtmp_url"`
		RtmpLive         string      `json:"rtmp_live"`
		RtmpCdn          string      `json:"rtmp_cdn"`
		RtmpMultiBitrate interface{} `json:"rtmp_multi_bitrate"`
		HlsURL           string      `json:"hls_url"`
		Servers          []struct {
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
