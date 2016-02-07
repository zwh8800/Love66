package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"
	"unicode"

	"os"

	"github.com/nsf/termbox-go"
	"github.com/zwh8800/Love66/player"
	"github.com/zwh8800/Love66/room"
)

func formatRoomInfo(room *room.DouyuRoom) []string {
	ret := make([]string, 0)

	var onlineStr string
	if room.Online() {
		onlineStr = "is online"
	} else {
		onlineStr = "is offline"
	}
	ret = append(ret, fmt.Sprintf("room #%d: ", room.RoomId()))
	ret = append(ret, fmt.Sprintf("    room %s", onlineStr))
	ret = append(ret, fmt.Sprintf("    room name: %s", room.RoomName()))
	ret = append(ret, fmt.Sprintf("    host nickname: %s", room.Nickname()))
	ret = append(ret, fmt.Sprintf("    game name: %s", room.GameName()))
	ret = append(ret, fmt.Sprintf("    detail: %s", string([]rune(room.Details())[:10])))
	ret = append(ret, fmt.Sprintf("    live stream url: %s", room.LiveStreamUrl()))
	return ret
}

func parsePlaylist(playlistFilename string) (bool, []*room.DouyuRoom) {
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
	for _, roomId := range playlist.Playlist {
		room, err := room.NewDouyuRoom(int(roomId))
		if err != nil {
			log.Panic(err)
		}
		rooms = append(rooms, room)
	}
	return playlist.Debug, rooms
}

func playRoom(player *player.Player, room *room.DouyuRoom) {
	room.RefreshIfExpire(time.Minute * 2)
	player.ChangeLiveStreamUrl(room.LiveStreamUrl())
	player.Play()
}

func isChineseChar(r rune) bool {
	if r > unicode.MaxLatin1 {
		return true
	} else {
		return false
	}
}

func tbPrint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		if isChineseChar(c) {
			x += 2
		} else {
			x++
		}
	}
}

func update(room *room.DouyuRoom) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	tbPrint(0, 0, termbox.ColorMagenta, termbox.ColorDefault, "Press 'q' to quit")
	roomInfo := formatRoomInfo(room)
	for i, line := range roomInfo {
		tbPrint(0, 1+i, termbox.ColorDefault, termbox.ColorDefault, line)
	}
	termbox.Flush()
}

func main() {
	playlistFilename := flag.String("playlist", "playlist.json", "specify a playlist with json format")
	flag.Parse()

	currentRoom := 0
	isDebug, rooms := parsePlaylist(*playlistFilename)
	if !isDebug {
		os.Stderr.Close()
	}

	if err := player.Init(); err != nil {
		log.Panic(err)
	}
	defer player.DeInit()

	if err := termbox.Init(); err != nil {
		log.Panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	player := player.NewPlayer(rooms[currentRoom].LiveStreamUrl())
	update(rooms[currentRoom])
	playRoom(player, rooms[currentRoom])

mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Ch == 'q' {
				break mainloop
			}

			switch ev.Key {
			case termbox.KeyEsc:
				break mainloop
			case termbox.KeyArrowUp:
			case termbox.KeyArrowLeft:
				if currentRoom <= 0 {
					currentRoom = len(rooms) - 1
				} else {
					currentRoom--
				}
				update(rooms[currentRoom])
				playRoom(player, rooms[currentRoom])
			case termbox.KeyArrowDown:
			case termbox.KeyArrowRight:
				if currentRoom >= len(rooms)-1 {
					currentRoom = 0
				} else {
					currentRoom++
				}
				update(rooms[currentRoom])
				playRoom(player, rooms[currentRoom])
			}
		}
	}
}
