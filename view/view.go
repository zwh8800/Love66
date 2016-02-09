package view

import (
	"unicode"

	"github.com/nsf/termbox-go"
)

type Handler func(args ...interface{})

type Data struct {
	LeftLines  []string
	RightLines []string
	Loading    bool
}

var (
	prev            Handler
	next            Handler
	quit            Handler
	lineCountChange Handler
	mainLoopChannel chan bool

	w int
	h int

	data *Data
)

func Init() error {
	if err := termbox.Init(); err != nil {
		return err
	}
	termbox.SetInputMode(termbox.InputEsc)
	mainLoopChannel = make(chan bool)
	w, h = termbox.Size()

	return nil
}

func DeInit() {
	close(mainLoopChannel)
	termbox.Close()
}

func GetMaxLineCount() int {
	return h
}

func SetData(d *Data) {
	data = d
}

func findMaxLength(lines []string) int {
	max := 0
	for _, line := range lines {
		if len(line) > max {
			max = len(line)
		}
	}
	return max
}

func drawSplitter() {
	for x, y := w/2, 0; y < h; y++ {
		termbox.SetCell(x, y, '|', termbox.ColorDefault, termbox.ColorDefault)
	}
}

func drawLeft() {
	maxLength := findMaxLength(data.LeftLines)
	width := w/2 - 1
	x := 0
	if maxLength < width {
		x = (width - maxLength) / 2
	}
	length := len(data.LeftLines)
	y := (h - length) / 2

	for i := 0; i < length; i++ {
		y += tbPrint(x, y, width, termbox.ColorDefault, termbox.ColorDefault, data.LeftLines[i])
	}
}

func calcLines(lines []string, width int) (int, []int) {
	c := 0
	arr := make([]int, 0)
	for _, line := range lines {
		c += len(line)/width + 1
		arr = append(arr, len(line)/width+1)
	}
	return c, arr
}

func drawRight() {
	x := w/2 + 1
	y := 0
	width := w/2 - 1

	lineCount, lineArray := calcLines(data.RightLines, width)
	startLine := 0
	for h < lineCount {
		lineCount -= lineArray[startLine]
		startLine++
	}
	for i := startLine; i < len(data.RightLines); i++ {
		y += tbPrint(x, y, width, termbox.ColorDefault, termbox.ColorDefault, data.RightLines[i])
	}
}

func Update() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	drawSplitter()
	drawLeft()
	drawRight()
	termbox.Flush()
}

func OnKeyPrev(h Handler) {
	prev = h
}

func OnKeyNext(h Handler) {
	next = h
}

func OnKeyQuit(h Handler) {
	quit = h
}

func OnMaxLineCountChange(h Handler) {
	lineCountChange = h
}

func emit(h Handler, args ...interface{}) {
	if h != nil {
		h(args)
	}
}

func MainLoop() {
	for {
		select {
		case <-mainLoopChannel:
			return
		default:
		}
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Ch == 'q' {
				emit(quit)
			}
			switch ev.Key {
			case termbox.KeyEsc:
				emit(quit)
			case termbox.KeyArrowLeft:
				emit(next)
			case termbox.KeyArrowRight:
				emit(prev)
			}
		case termbox.EventResize:
			if h != ev.Height {
				emit(lineCountChange, ev.Height)
			}
			w = ev.Width
			h = ev.Height
			Update()
		}
	}
}

func isChineseChar(r rune) bool {
	if r > unicode.MaxLatin1 {
		return true
	} else {
		return false
	}
}

func tbPrint(x, y, w int, fg, bg termbox.Attribute, msg string) int {
	offset := 1
	initX := x
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		if isChineseChar(c) {
			x += 2
		} else {
			x++
		}
		if x-initX >= w {
			x = initX
			y++
			offset++
		}
	}
	return offset
}
