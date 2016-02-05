package player

// #cgo pkg-config: libavutil
// #include <libavutil/error.h>
// #include <libavutil/channel_layout.h>
// #include <libswresample/swresample.h>
//extern void fillAudio(void *userdata, uint8_t *stream, int len);
import "C"
import (
	"bytes"
	"errors"
	"log"
	"unsafe"

	"github.com/giorgisio/goav/avcodec"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	AV_CH_LAYOUT_STEREO = C.AV_CH_LAYOUT_STEREO
	AV_SAMPLE_FMT_S16   = C.AV_SAMPLE_FMT_S16
)

func AvGetDefaultChannelLayout(channels int) int64 {
	return int64(C.av_get_default_channel_layout(C.int(channels)))
}

func AvError(errNum int) error {
	buf := make([]byte, 64)

	C.av_strerror(C.int(errNum), (*C.char)(unsafe.Pointer(&buf[0])), 64)
	return errors.New(string(buf))
}

type Frame C.struct_AVFrame

func (p *Frame) Data() **uint8 {
	return (**uint8)(unsafe.Pointer(&p.data[0]))
}

func (p *Frame) NbSamples() int {
	return (int)(p.nb_samples)
}

//export fillAudio
func fillAudio(uData unsafe.Pointer, stream *C.uint8_t, len C.int) {
	C.memset(unsafe.Pointer(stream), 0, C.size_t(len))
	length := uint32(len)
	audioBuffer := (*bytes.Buffer)(uData)
	if audioBuffer.Len() < int(length) {
		return
	}

	audioData := make([]byte, int(length))
	if _, err := audioBuffer.Read(audioData); err != nil {
		log.Fatal(err)
	}

	sdl.MixAudio((*uint8)(stream), &audioData[0], length, sdl.MIX_MAXVOLUME)
}

func createAudioSpec(codecContext *avcodec.CodecContext, audioBuffer *bytes.Buffer) *sdl.AudioSpec {
	var spec sdl.AudioSpec
	spec.Freq = int32(outSampleRate)
	spec.Format = sdl.AUDIO_S16SYS
	spec.Channels = outChannelCount
	spec.Samples = uint16(codecContext.FrameSize())
	spec.Callback = sdl.AudioCallback(C.fillAudio)
	spec.UserData = unsafe.Pointer(audioBuffer)
	return &spec
}
