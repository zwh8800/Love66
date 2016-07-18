package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	"strconv"

	"github.com/zwh8800/Love66/danmuku"
	"github.com/zwh8800/Love66/player"
	"github.com/zwh8800/Love66/room"
	"github.com/zwh8800/Love66/view"
)

var (
	isDebug       bool
	rooms         []*room.DouyuRoom
	danmukuRooms  []*danmuku.DanmukuRoom
	currentRoom   int
	mainPlayer    *player.Player
	maxLineCount  int
	quitChannel   chan bool = make(chan bool)
	changeChannel chan bool = make(chan bool)
)

func main() {
	playlistFilename := flag.String("playlist", "playlist.json", "specify a playlist with json format")
	flag.Parse()

	isDebug, rooms, danmukuRooms = parsePlaylist(*playlistFilename)
	if !isDebug {
		os.Stderr.Close()
	}
	currentRoom = 0

	mainPlayer = player.NewPlayer(rooms[currentRoom].LiveStreamUrl())

	if err := view.Init(); err != nil {
		log.Panic(err)
	}
	defer view.DeInit()
	maxLineCount = view.GetMaxLineCount()

	view.SetData(getViewData(nil, nil))
	view.OnMaxLineCountChange(func(args ...interface{}) {
		var ok bool
		maxLineCount, ok = args[0].(int)
		if !ok {
			log.Panic("cast error")
		}
		view.SetData(getViewData(nil, nil))
		view.Update()
	})
	view.OnKeyNext(func(args ...interface{}) {
		stopDanmukuRoom()
		if currentRoom >= len(rooms)-1 {
			currentRoom = 0
		} else {
			currentRoom++
		}
		startDanmukuRoom()
		playRoom()

		view.SetData(getViewData(nil, nil))
		view.Update()
		changeChannel <- true
	})
	view.OnKeyPrev(func(args ...interface{}) {
		stopDanmukuRoom()
		if currentRoom <= 0 {
			currentRoom = len(rooms) - 1
		} else {
			currentRoom--
		}
		startDanmukuRoom()
		playRoom()

		view.SetData(getViewData(nil, nil))
		view.Update()
		changeChannel <- true
	})
	view.OnKeyQuit(func(args ...interface{}) {
		close(quitChannel)
	})
	view.Update()
	go view.MainLoop()

	dataChannel := make(chan *view.Data)
	go func() {
		for {
			danmukuRoom := danmukuRooms[currentRoom]
			select {
			case <-changeChannel:
			case danmuku := <-danmukuRoom.GetDanmukuChannel():
				dataChannel <- getViewData(view.GetData(), &danmuku)
			}
		}
	}()

	startDanmukuRoom()
	playRoom()

	mainLoop(dataChannel)
}

func mainLoop(dataChannel chan *view.Data) {
	for {
		select {
		case data := <-dataChannel:
			view.SetData(data)
			view.Update()
		case <-quitChannel:
			return
		}
	}
}

func parsePlaylist(playlistFilename string) (bool, []*room.DouyuRoom, []*danmuku.DanmukuRoom) {
	playlistData, err := ioutil.ReadFile(playlistFilename)
	if err != nil {
		log.Panic(err)
	}
	playlist := struct {
		Debug    bool  `json:"debug"`
		Playlist []int `json:"playlist"`
	}{}

	if err := json.Unmarshal(playlistData, &playlist); err != nil {
		log.Panic(err)
	}
	rooms := make([]*room.DouyuRoom, 0)
	danmukuRooms := make([]*danmuku.DanmukuRoom, 0)
	for _, roomId := range playlist.Playlist {
		room, err := room.NewDouyuRoom(int(roomId))
		if err != nil {
			log.Panic(err)
		}
		rooms = append(rooms, room)
		danmukuRoom := danmuku.NewDanmukuRoom(int(roomId))
		danmukuRooms = append(danmukuRooms, danmukuRoom)
	}
	return playlist.Debug, rooms, danmukuRooms
}

func playRoom() {
	room := rooms[currentRoom]
	room.RefreshIfExpire(time.Minute * 2)
	mainPlayer.ChangeLiveStreamUrl(room.LiveStreamUrl())
	mainPlayer.Play()
}

func startDanmukuRoom() {
	room := rooms[currentRoom]
	if room.Online() {
		curRoom := danmukuRooms[currentRoom]
		curRoom.Start()
	}
}

func stopDanmukuRoom() {
	room := rooms[currentRoom]
	if room.Online() {
		prevRoom := danmukuRooms[currentRoom]
		prevRoom.Stop()
	}
}

func getViewData(prevData *view.Data, newDanmuku *danmuku.Danmuku) *view.Data {
	var danmukuData []string
	if prevData == nil {
		danmukuData = []string{
			"欢迎",
		}
	} else {
		danmukuData = append(prevData.RightLines, newDanmuku.User+": "+newDanmuku.Content)
		if len(danmukuData) > maxLineCount {
			danmukuData =
				danmukuData[len(danmukuData)-maxLineCount : len(danmukuData)]
		}
	}

	room := rooms[currentRoom]
	onlineStr := ""
	if room.Online() {
		onlineStr = "【在线】"
	} else {
		onlineStr = "【离线】"
	}
	data := view.Data{
		[]string{
			onlineStr + room.Nickname(),
			"#" + strconv.Itoa(room.RoomId()),
			room.RoomName(),
			room.GameName(),
			//			strings.Replace(room.Details(), "\n", " ", -1),
			//			room.LiveStreamUrl(),
		},
		danmukuData,
		mainPlayer.Loading(),
	}

	return &data
}
