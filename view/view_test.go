package view

import (
	"log"
	"os"
	"os/signal"
	"testing"
)

func TestView(t *testing.T) {
	c := make(chan os.Signal)
	if err := Init(); err != nil {
		log.Fatal(err)
	}
	defer DeInit()

	data := Data{
		[]string{
			"hello world",
		},
		[]string{
			"hello world",
		},
		false,
	}
	maxLineCount := GetMaxLineCount()
	SetData(&data)
	OnKeyNext(func(args ...interface{}) {
		data.LeftLines[0] = "Next Press"
		data.RightLines = append(data.RightLines, "Next Press")
		if len(data.RightLines) > maxLineCount {
			data.RightLines =
				data.RightLines[len(data.RightLines)-maxLineCount : len(data.RightLines)]
		}
		Update()
	})
	OnKeyPrev(func(args ...interface{}) {
		data.Loading = !data.Loading

		data.LeftLines[0] = "Prev Press"
		data.RightLines = append(data.RightLines, "Prev Press Prev Press Prev Press Prev Press Prev Press Prev Press Prev Press Prev Press Prev Press Prev Press ")
		if len(data.RightLines) > maxLineCount {
			data.RightLines =
				data.RightLines[len(data.RightLines)-maxLineCount : len(data.RightLines)]
		}
		Update()
	})
	OnKeyQuit(func(args ...interface{}) {
		close(c)
		Update()
	})
	OnMaxLineCountChange(func(args ...interface{}) {
		maxLineCount, ok := args[0].(int)
		if !ok {
			log.Fatal("cast error")
		}
		if len(data.RightLines) > maxLineCount {
			data.RightLines =
				data.RightLines[len(data.RightLines)-maxLineCount : len(data.RightLines)]
		}
		Update()
	})

	Update()
	go MainLoop()

	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
