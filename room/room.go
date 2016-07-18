package room

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type DouyuRoom struct {
	roomId      int
	roomInfo    *douyuRoomInfoJson
	lastRefresh time.Time
}

func NewDouyuRoom(roomId int) (*DouyuRoom, error) {
	url := resolveApiUrl(roomId)
	roomInfo, err := getRoomInfo(url)
	if err != nil {
		return nil, err
	}
	return &DouyuRoom{roomId, roomInfo, time.Now()}, nil
}

func (r *DouyuRoom) Refresh() error {
	url := resolveApiUrl(r.roomId)
	roomInfo, err := getRoomInfo(url)
	if err != nil {
		return err
	}
	r.roomInfo = roomInfo
	r.lastRefresh = time.Now()
	return nil
}

func (r *DouyuRoom) RefreshIfExpire(expire time.Duration) {
	if time.Now().Sub(r.lastRefresh) > expire {
		r.Refresh()
	}
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
	return r.roomInfo.Data.TagName
}

func (r *DouyuRoom) LiveStreamUrl() string {
	if r.Online() {
		return r.roomInfo.Data.HlsURL
	} else {
		return ""
	}
}

func resolveApiUrl(roomId int) string {
	u, err := url.Parse("http://m.douyu.com/html5/live")
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("roomId", strconv.Itoa(roomId))
	u.RawQuery = q.Encode()

	return u.String()
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
	Error int    `json:"error"`
	Msg   string `json:"msg"`
	Data  struct {
		RoomID       string `json:"room_id"`
		TagName      string `json:"tag_name"`
		RoomSrc      string `json:"room_src"`
		RoomName     string `json:"room_name"`
		ShowStatus   string `json:"show_status"`
		Online       int    `json:"online"`
		Nickname     string `json:"nickname"`
		HlsURL       string `json:"hls_url"`
		IsPassPlayer int    `json:"is_pass_player"`
		IsTicket     int    `json:"is_ticket"`
		StoreLink    string `json:"storeLink"`
	} `json:"data"`
}
