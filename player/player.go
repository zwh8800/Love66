package player

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"strings"
)

type Player struct {
	notifyM2SChannel chan interface{}
	notifyS2MChannel chan interface{}

	playing bool
	loading bool
	closing bool

	liveStreamUrl string
}

type startMessage struct {
	liveStreamUrl string
}
type startedMessage struct {
}
type stopMessage struct {
}
type stoppedMessage struct {
}

func NewPlayer(liveStreamUrl string) *Player {
	p := &Player{
		notifyM2SChannel: make(chan interface{}),
		notifyS2MChannel: make(chan interface{}),

		playing: false,
		loading: false,
		closing: false,

		liveStreamUrl: liveStreamUrl,
	}

	go dispatcher(p)
	go playRoutine(p.notifyM2SChannel, p.notifyS2MChannel)
	return p
}

func dispatcher(p *Player) {
	for msg := range p.notifyS2MChannel {
		switch msg := msg.(type) {
		case *startedMessage:
			p.loading = false
		case *stoppedMessage:
			p.closing = false
			p.playing = false
			p.loading = false

		case error:
			log.Println(msg)
		}
	}
}

func playRoutine(notifyM2SChannel, notifyS2MChannel chan interface{}) {
	var cmd *exec.Cmd
	for msg := range notifyM2SChannel {
		switch msg := msg.(type) {
		case *startMessage:
			var err error
			cmd, err = startPlay(msg.liveStreamUrl)
			if err != nil {
				notifyS2MChannel <- err
				break
			}
			notifyS2MChannel <- &startedMessage{}
		case *stopMessage:
			if cmd != nil {
				if err := stopPlay(cmd); err != nil {
					notifyS2MChannel <- err
					break
				}
			}
			notifyS2MChannel <- &stoppedMessage{}
		}
	}
}

func startPlay(liveStreamUrl string) (*exec.Cmd, error) {
	cmd := exec.Command("mplayer", "-vo", "null", "-cache", "20480", liveStreamUrl)

	pipeReader, pipeWriter := io.Pipe()
	cmd.Stdout = pipeWriter

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	waitPlay(pipeReader)
	return cmd, nil
}

func waitPlay(pipeReader *io.PipeReader) {
	const startPlay = "Starting playback"
	bufReader := bufio.NewReader(pipeReader)
	var totalLineData []byte
	lastIsPrefix := false
	for {
		lineData, isPrefix, err := bufReader.ReadLine()
		if err != nil {
			break
		}
		if lastIsPrefix {
			totalLineData = append(totalLineData, lineData...)
		} else {
			totalLineData = lineData
		}
		lastIsPrefix = isPrefix
		if !isPrefix {
			line := string(totalLineData)
			i := strings.Index(line, startPlay)
			if i != -1 {
				break
			}
		}
	}
}

func stopPlay(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

func (p *Player) Playing() bool {
	return p.playing
}

func (p *Player) Loading() bool {
	return p.loading
}

func (p *Player) Play() {
	if p.playing {
		p.Stop()
	}
	if p.liveStreamUrl == "" {
		return
	}
	p.playing = true
	p.loading = true
	p.notifyM2SChannel <- &startMessage{
		liveStreamUrl: p.liveStreamUrl,
	}
}

func (p *Player) Stop() {
	if !p.playing {
		return
	}
	p.closing = true
	p.notifyM2SChannel <- &stopMessage{}
}

func (p *Player) ChangeLiveStreamUrl(liveStreamUrl string) {
	p.liveStreamUrl = liveStreamUrl
}
