package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type DouyuLiveData struct {
	Error int    `json:"error"`
	Msg   string `json:"msg"`
	Data  struct {
		RoomID       string `json:"room_id"`
		TagName      string `json:"tag_name"`
		RoomSrc      string `json:"room_src"`
		RoomName     string `json:"room_name"`
		ShowStatus   string `json:"show_status"`
		Online       int    `json:"online"`
		Nickname     string `json:"nickname"`
		HlsURL       string `json:"hls_url"`
		IsPassPlayer int    `json:"is_pass_player"`
		IsTicket     int    `json:"is_ticket"`
		StoreLink    string `json:"storeLink"`
	} `json:"data"`
}

func GetStreamUrl(roomId int) string {
	resp, err := http.Get(fmt.Sprintf("https://m.douyu.com/html5/live?roomId=%d", roomId))
	//resp, err := http.Get("https://m.douyu.com/html5/live?roomId=156277")
	//resp, err := http.Get("https://m.douyu.com/html5/live?roomId=3258")
	if err != nil {
		log.Println(err)
		return ""
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return ""
	}

	var douyuLiveData DouyuLiveData
	if err := json.Unmarshal(data, &douyuLiveData); err != nil {
		log.Println(err)
		return ""
	}

	log.Println("douyuLiveData =", JsonStringify(douyuLiveData, true))

	if douyuLiveData.Error != 0 {
		log.Printf("douyuLiveData.error = %d, douyuLiveData.msg = %s\n", douyuLiveData.Error, douyuLiveData.Msg)
		return ""
	}

	return douyuLiveData.Data.HlsURL
}

func JsonStringify(obj interface{}, indent bool) string {
	if indent {
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return ""
		}
		return string(data)
	} else {
		data, err := json.Marshal(obj)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

var mplayerProcess *os.Process

func OpenMPlayerWithUrl(url string) {
	cmd := exec.Command("mplayer", "-vo", "null", "-cache", "20480", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	mplayerProcess = cmd.Process

	pipeReader, pipeWriter := io.Pipe()
	cmd.Stdout = pipeWriter

	if err := cmd.Start(); err != nil {
		log.Println(err)
	}

	WaitPlay(pipeReader)

	if err := cmd.Wait(); err != nil {
		log.Println(err)
	}
}

func WaitPlay(pipeReader *io.PipeReader) {
	const startPlay = "Starting playback"
	bufReader := bufio.NewReader(pipeReader)
	var totalLineData []byte
	lastIsPrefix := false
	for {
		lineData, isPrefix, err := bufReader.ReadLine()
		if err != nil {
			log.Println("error:", err)
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
				log.Println("start")
				break
			}
		}
	}
	go io.Copy(ioutil.Discard, bufReader)
}

const (
	OpenDouyuAddr        = "openbarrage.douyutv.com:8601"
	MsgTypeSend   uint16 = 689
	MsgTypeRecv   uint16 = 690
)

func sendMsg(conn net.Conn, msg string) error {
	msgLen := len(msg) + 1 + 12
	buf := make([]byte, msgLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(msgLen-4))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(msgLen-4))
	binary.LittleEndian.PutUint16(buf[8:10], MsgTypeSend)
	copy(buf[12:], []byte(msg))

	_, err := conn.Write(buf)
	return err
}

func danmukuLogin(conn net.Conn, roomId int) {
	msg := fmt.Sprintf("type@=loginreq/roomid@=%d/", roomId)
	if err := sendMsg(conn, msg); err != nil {
		log.Println(err)
	}
}

func danmukuJoin(conn net.Conn, roomId int) {
	msg := fmt.Sprintf("type@=joingroup/rid@=%d/gid@=-9999/", roomId)
	if err := sendMsg(conn, msg); err != nil {
		log.Println(err)
	}
}

func danmukuKeeplive(conn net.Conn) {
	msg := fmt.Sprintf("type@=keeplive/tick@=%d/", time.Now().Unix())
	if err := sendMsg(conn, msg); err != nil {
		log.Println(err)
	}
}

func parseMessage(message string) map[string]string {
	msg := make(map[string]string)
	regex := regexp.MustCompile(`(.*?)@=(.*?)/`)
	submatchs := regex.FindAllStringSubmatch(message, -1)

	for _, submatch := range submatchs {
		msg[submatch[1]] = submatch[2]
	}
	return msg
}

func readMessage(conn net.Conn) (string, error) {
	var (
		length      uint32
		length2     uint32
		messageType uint16
		unused      uint16
	)
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	if err := binary.Read(conn, binary.LittleEndian, &length2); err != nil {
		return "", err
	}
	if length != length2 {
		return "", fmt.Errorf("243: length(%d) != length2(%d)\n", length, length2)
	}
	if err := binary.Read(conn, binary.LittleEndian, &messageType); err != nil {
		return "", err
	}
	if messageType != MsgTypeRecv {
		return "", fmt.Errorf("249: messageData(%d) != typeRecv\n", messageType)
	}
	if err := binary.Read(conn, binary.LittleEndian, &unused); err != nil {
		return "", err
	}
	length = length - 8
	messageData := make([]byte, length)

	for i := 0; i < int(length); {
		n, err := conn.Read(messageData[i:])
		if err != nil {
			return "", err
		}
		i += n
	}

	return string(messageData), nil
}

type GiftData struct {
	Data struct {
		Gift []struct {
			Desc  string  `json:"desc"`
			Gx    float64 `json:"gx"`
			Himg  string  `json:"himg"`
			ID    string  `json:"id"`
			Intro string  `json:"intro"`
			Mimg  string  `json:"mimg"`
			Name  string  `json:"name"`
			Pc    float64 `json:"pc"`
			Type  string  `json:"type"`
		} `json:"gift"`
	} `json:"data"`
	Error int `json:"error"`
}

var gifts = make(map[string]string)

func getGiftList(roomId int) {

	resp, err := http.Get(fmt.Sprintf("http://open.douyucdn.cn/api/RoomApi/room/%d", roomId))
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	var giftData GiftData
	if err := json.Unmarshal(data, &giftData); err != nil {
		log.Println(err)
		return
	}

	log.Println("giftData =", JsonStringify(giftData, true))

	if giftData.Error != 0 {
		log.Printf("giftData.error = %d\n", giftData.Error)
		return
	}

	for _, gift := range giftData.Data.Gift {
		gifts[gift.ID] = gift.Name
	}
}

func danmukuReadAndPrint(conn net.Conn) {
	msgStr, err := readMessage(conn)
	if err != nil {
		log.Println(err)
	}
	message := parseMessage(msgStr)

	switch message["type"] {
	case "chatmsg":
		log.Printf("%s(%s): %s", message["nn"], message["uid"], message["txt"])
	case "dgb":
		hits := message["hits"]
		if hits == "" {
			hits = "1"
		}
		log.Printf("%s(%s) 送出 %s (%s 连击)", message["nn"], message["uid"], gifts[message["gfid"]], hits)
	default:
		// log.Printf("%#v", message)
	}

}

func Danmuku(roomId int) {
	getGiftList(roomId)

	conn, err := net.Dial("tcp", OpenDouyuAddr)
	if err != nil {
		log.Println(err)
		return
	}
	danmukuLogin(conn, roomId)
	danmukuJoin(conn, roomId)
	go func() {
		for {
			danmukuKeeplive(conn)
			time.Sleep(30 * time.Second)
		}
	}()

	for {
		danmukuReadAndPrint(conn)
	}
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	roomId := flag.Int("id", 156277, "room id")
	onlyDanmu := flag.Bool("d", false, "only danmu")
	flag.Parse()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		if mplayerProcess != nil {
			if err := syscall.Kill(-mplayerProcess.Pid, syscall.SIGKILL); err != nil {
				log.Println("failed to kill: ", err)
			}
		}
		os.Exit(0)
	}()

	go Danmuku(*roomId)

	for {
		if *onlyDanmu {
			time.Sleep(1 * time.Minute)
		}

		url := GetStreamUrl(*roomId)
		if url == "" {
			time.Sleep(5 * time.Second)
			continue
		}
		OpenMPlayerWithUrl(url)
	}
}
