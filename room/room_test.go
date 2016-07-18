package room

import "testing"

func TestGetInfo(t *testing.T) {
	roomIds := []int{156277, 3258, 423574, 60937}
	rooms := make([]*DouyuRoom, 0)
	for _, id := range roomIds {
		room, err := NewDouyuRoom(id)
		if err != nil {
			t.Logf("Error in NewDouyuRoom(%d): %s\n", id, err)
			continue
		}
		rooms = append(rooms, room)
	}

	for _, room := range rooms {
		var onlineStr string
		if room.Online() {
			onlineStr = "is online"
		} else {
			onlineStr = "is offline"
		}
		t.Logf("room #%d: ", room.RoomId())
		t.Logf("\troom %s", onlineStr)
		t.Logf("\troom name: %s", room.RoomName())
		t.Logf("\thost nickname: %s", room.Nickname())
		t.Logf("\tgame name: %s", room.GameName())
		t.Logf("\tlive stream url: %s", room.LiveStreamUrl())
	}
}
