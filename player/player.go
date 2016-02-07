package player

import (
	"bytes"
	"errors"
	"time"
	"unsafe"

	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
	"github.com/giorgisio/goav/swresample"
	"github.com/veandco/go-sdl2/sdl"
)

func Init() error {
	inited := sdl.WasInit(sdl.INIT_AUDIO | sdl.INIT_TIMER)

	if inited&sdl.INIT_AUDIO != sdl.INIT_AUDIO {
		if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
			return err
		}
	}
	if inited&sdl.INIT_TIMER != sdl.INIT_TIMER {
		if err := sdl.InitSubSystem(sdl.INIT_TIMER); err != nil {
			return err
		}
	}
	avformat.AvRegisterAll()
	avformat.AvformatNetworkInit()

	return nil
}

func DeInit() {
	sdl.QuitSubSystem(sdl.INIT_AUDIO)
	sdl.QuitSubSystem(sdl.INIT_TIMER)
	avformat.AvformatNetworkDeinit()
}

type Player struct {
	notifyM2SChannel chan string
	notifyS2MChannel chan string
	errorChannel     chan error
	playing          bool
	liveStreamUrl    string
	audioBuffer      *bytes.Buffer
}

func NewPlayer(liveStreamUrl string) *Player {
	return &Player{
		make(chan string),
		make(chan string),
		make(chan error),
		false,
		liveStreamUrl,
		nil,
	}
}

func (p *Player) ChangeLiveStreamUrl(liveStreamUrl string) {
	p.liveStreamUrl = liveStreamUrl
}

func (p *Player) Playing() bool {
	return p.playing
}

func (p *Player) Play() {
	if p.playing {
		p.Stop()
	}
	if p.liveStreamUrl == "" {
		return
	}
	p.playing = true
	go p.playRoutine()
}

func (p *Player) Stop() {
	if !p.playing {
		return
	}
	p.notifyM2SChannel <- "stop"
	for <-p.notifyS2MChannel != "stoped" {
	}
	p.playing = false
}

func (p *Player) ErrorChannel() chan error {
	return p.errorChannel
}

func (p *Player) Error() error {
	select {
	case err := <-p.errorChannel:
		return err
	case <-time.After(time.Second * 1):
		return nil
	}
}

const (
	outChannelCount      = 2
	outChannelLayout     = AV_CH_LAYOUT_STEREO
	outSampleSize        = 2
	outSampleFormat      = AV_SAMPLE_FMT_S16
	outSampleRate        = 48000
	audioFrameBufferSize = 192000
)

func findAudioStream(formatContext *avformat.Context) (*avcodec.CodecContext, int) {
	n := formatContext.NbStreams()
	for i := 0; i < int(n); i++ {
		codec := formatContext.Streams(uint(i)).Codec()
		if codec.CodecType() == avutil.AVMEDIA_TYPE_AUDIO {
			return codec, i
		}
	}
	return nil, -1
}

func createSwr(codecContext *avcodec.CodecContext) *swresample.Context {
	swrContext := swresample.SwrAlloc()
	swrContext.SwrAllocSetOpts(outChannelLayout, outSampleFormat, outSampleRate,
		AvGetDefaultChannelLayout(codecContext.Channels()),
		swresample.AvSampleFormat(codecContext.SampleFmt()), codecContext.SampleRate(),
		0, 0)
	swrContext.SwrInit()

	return swrContext
}

func (p *Player) playRoutine() {
	defer func() { p.notifyS2MChannel <- "stoped" }()

	var formatContext *avformat.Context
	if errNum := avformat.AvformatOpenInput(&formatContext, p.liveStreamUrl, nil, nil); errNum != 0 {
		p.errorChannel <- AvError(errNum)
		return
	}
	defer formatContext.AvformatCloseInput()
	if errNum := formatContext.AvformatFindStreamInfo(nil); errNum < 0 {
		p.errorChannel <- AvError(errNum)
		return
	}
	formatContext.AvDumpFormat(0, p.liveStreamUrl, 0)

	codecContext, audioIndex := findAudioStream(formatContext)
	if codecContext == nil {
		p.errorChannel <- errors.New("audio stream not found")
		return
	}
	defer codecContext.AvcodecClose()
	if errNum := codecContext.AvcodecOpen2(avcodec.AvcodecFindDecoder(codecContext.CodecId()), nil); errNum < 0 {
		p.errorChannel <- errors.New("audio stream not found")
		return
	}
	p.audioBuffer = new(bytes.Buffer)
	defer func() { p.audioBuffer = nil }()
	audioBufferCap := codecContext.FrameSize() * outChannelCount * outSampleSize * 4
	p.audioBuffer.Grow(audioBufferCap)

	swrContext := createSwr(codecContext)
	defer swrContext.SwrClose()

	packet := new(avcodec.Packet)
	packet.AvInitPacket()
	utilFrame := avutil.AvFrameAlloc()
	outBuffer := [audioFrameBufferSize]uint8{}
	outBufferArray := [...]*uint8{&outBuffer[0]}

	wanted := createAudioSpec(codecContext, p.audioBuffer)
	if err := sdl.OpenAudio(wanted, nil); err != nil {
		p.errorChannel <- err
	}
	defer sdl.CloseAudio()

readPacketLoop:
	for formatContext.AvReadFrame(packet) >= 0 {
		select {
		case msg := <-p.notifyM2SChannel:
			if msg == "stop" {
				break readPacketLoop
			}
		default:
		}

		if packet.StreamIndex() == audioIndex {
			codecFrame := (*avcodec.Frame)(unsafe.Pointer(utilFrame))

			var gotFrame int
			if errNum := codecContext.AvcodecDecodeAudio4(codecFrame, &gotFrame, packet); errNum < 0 {
				p.errorChannel <- AvError(errNum)
				return
			}
			if gotFrame > 0 {
				frame := (*Frame)(unsafe.Pointer(codecFrame))
				n := swrContext.SwrConvert(&outBufferArray[0], audioFrameBufferSize, frame.Data(), frame.NbSamples())

				len := outChannelCount * outSampleSize * n

				for p.audioBuffer.Len() >= audioBufferCap {
					sdl.Delay(1)
				}
				if _, err := p.audioBuffer.Write(outBuffer[:len]); err != nil {
					p.errorChannel <- err
					return
				}

				sdl.PauseAudio(false)
			}
		}
		packet.AvFreePacket()
	}
}
