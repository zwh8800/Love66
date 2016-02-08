package danmuku

import (
	"testing"
	"time"
)

func TestDanmuku(t *testing.T) {
	danmukuRoom := NewDanmukuRoom(301712)
	if err := danmukuRoom.Start(); err != nil {
		t.Error(err)
		return
	}
	defer danmukuRoom.Stop()

	time.Sleep(5 * time.Second)
}
