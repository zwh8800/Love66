package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/zwh8800/Love66/player"
	"github.com/zwh8800/Love66/room"
)

func showRoomInfo(room *room.DouyuRoom) {
	var onlineStr string
	if room.Online() {
		onlineStr = "is online"
	} else {
		onlineStr = "is offline"
	}
	log.Printf("room #%d: ", room.RoomId())
	log.Printf("\troom %s", onlineStr)
	log.Printf("\troom name: %s", room.RoomName())
	log.Printf("\thost nickname: %s", room.Nickname())
	log.Printf("\tgame name: %s", room.GameName())
	log.Printf("\tdetail: %s", string([]rune(room.Details())[:10]))
	log.Printf("\tlive stream url: %s", room.LiveStreamUrl())
}

func main() {
	flag.Parse()
	roomId, err := strconv.ParseInt(flag.Arg(0), 10, 32)
	if err != nil {
		roomId = 156277
	}

	room, err := room.NewDouyuRoom(int(roomId))
	if err != nil {
		log.Panic(err)
	}
	showRoomInfo(room)
	if room.Online() != true {
		log.Println("主播不在线")
		return
	}

	if err := player.Init(); err != nil {
		log.Panic(err)
	}
	defer player.DeInit()

	player := player.NewPlayer(room.LiveStreamUrl())
	player.Play()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	s := <-sigChan
	log.Println("Got signal:", s)
	player.Stop()
}
