package danmuku

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/satori/go.uuid"
)

const (
	typeSend int32 = 0x000002b1
	typeRecv int32 = 0x000002b2
)

const danmukuServer = "danmu.douyutv.com:8601"

type Danmuku struct {
	User    string
	Content string
}

type DanmukuRoom struct {
	roomId         int
	conn           net.Conn
	gidConn        net.Conn
	danmukuChannel chan Danmuku
	stopChannel    chan bool
}

func NewDanmukuRoom(roomId int) *DanmukuRoom {
	return &DanmukuRoom{
		roomId,
		nil,
		nil,
		nil,
		nil,
	}
}

func (r *DanmukuRoom) Start() error {
	r.danmukuChannel = make(chan Danmuku)
	r.stopChannel = make(chan bool)
	roomHtml, err := r.getHtml()
	if err != nil {
		return err
	}
	sc, err := parseServerConfig(roomHtml)
	if err != nil {
		return err
	}
	gidConn, err := net.Dial("tcp", sc[0].IP+":"+sc[0].Port)
	if err != nil {
		return err
	}
	r.gidConn = gidConn
	defer r.gidConn.Close()
	gid, err := r.getGid()
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", danmukuServer)
	if err != nil {
		return err
	}
	r.conn = conn

	loginReq := formatMessage(map[string]string{
		"type":     "loginreq",
		"username": "auto_KRLJbE8mZM",
		"password": "1234567890123456",
		"roomid":   strconv.Itoa(r.roomId),
	})

	if err := writeMessage(r.conn, loginReq); err != nil {
		return err
	}

	joinGroup := formatMessage(map[string]string{
		"type": "joingroup",
		"rid":  strconv.Itoa(r.roomId),
		"gid":  strconv.Itoa(gid),
	})
	if err := writeMessage(r.conn, joinGroup); err != nil {
		return err
	}

	go r.readRoutine()
	go r.keepAliveRoutine()

	return nil
}

func (r *DanmukuRoom) Stop() {
	close(r.stopChannel)
	r.conn.Close()
}

func (r *DanmukuRoom) PeekDanmuku() *Danmuku {
	danmuku := <-r.danmukuChannel
	return &danmuku
}

func (r *DanmukuRoom) GetDanmukuChannel() chan Danmuku {
	return r.danmukuChannel
}

func formatMessage(msg map[string]string) string {
	message := make([]string, 0)
	for k, v := range msg {
		message = append(message, k+"@="+v+"/")
	}
	return strings.Join(message, "")
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

type serverConfig []struct {
	IP   string `json:"ip"`
	Port string `json:"port"`
}

func parseServerConfig(html string) (serverConfig, error) {
	regex := regexp.MustCompile(`"server_config":"(.*?)"`)
	submatch := regex.FindStringSubmatch(html)

	jsonData, err := url.QueryUnescape(submatch[1])
	if err != nil {
		return nil, err
	}

	var sc serverConfig
	if err := json.Unmarshal([]byte(jsonData), &sc); err != nil {
		return nil, err
	}
	return sc, nil
}

func (r *DanmukuRoom) getGid() (int, error) {
	devId := strings.ToUpper(strings.Replace(uuid.NewV4().String(), "-", "", -1))
	rt := strconv.Itoa(int(time.Now().Unix()))
	magic := "7oE9nPEG9xXV69phU31FYCLUagKeYtsF"
	sumArr := md5.Sum([]byte(rt + magic + devId))
	sum := sumArr[:]
	vk := hex.EncodeToString(sum)

	loginReq := formatMessage(map[string]string{
		"type":     "loginreq",
		"username": "",
		"password": "",
		"roomid":   strconv.Itoa(r.roomId),
		"ct":       "0",
		"devid":    devId,
		"rt":       rt,
		"vk":       vk,
		"ver":      "20150929",
	})

	if err := writeMessage(r.gidConn, loginReq); err != nil {
		return 0, err
	}

	for {
		message, err := readMessage(r.gidConn)
		if err != nil {
			return 0, err
		}
		msg := parseMessage(message)
		if msg["type"] == "setmsggroup" {
			gid, err := strconv.ParseInt(msg["gid"], 10, 32)
			if err != nil {
				return 0, nil
			}
			return int(gid), nil
		}
	}
}

func (r *DanmukuRoom) getHtml() (string, error) {
	resp, err := http.Get("http://www.douyutv.com/" + strconv.Itoa(r.roomId))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeMessage(conn net.Conn, message string) error {
	buf := new(bytes.Buffer)

	length := int32(len(message) + 1 + 8)
	frame := []interface{}{
		int32(length),
		int32(length),
		int32(typeSend),
	}
	for _, item := range frame {
		if err := binary.Write(buf, binary.LittleEndian, item); err != nil {
			return err
		}
	}
	buf.Write(append([]byte(message), 0))
	conn.Write(buf.Bytes())

	return nil
}

func readMessage(conn net.Conn) (string, error) {
	var (
		length      int32
		length2     int32
		messageType int32
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
	if messageType != typeRecv {
		return "", fmt.Errorf("249: messageData(%d) != typeRecv\n", messageType)
	}
	messageData := make([]byte, length-8)

	for i := 0; i < int(length-8); {
		n, err := conn.Read(messageData[i:])
		if err != nil {
			return "", err
		}
		i += n
	}

	return string(messageData), nil
}

func (r *DanmukuRoom) readRoutine() {
	for {
		select {
		case <-r.stopChannel:
			return
		default:
		}
		message, err := readMessage(r.conn)
		if err != nil {
			log.Println("272:", err)
		}
		msg := parseMessage(message)
		if msg["type"] == "chatmessage" {
			r.danmukuChannel <- Danmuku{
				msg["snick"],
				msg["content"],
			}
		}
	}
}

func (r *DanmukuRoom) keepAliveRoutine() {
	for {
		select {
		case <-r.stopChannel:
			return
		default:
		}
		keepAlive := formatMessage(map[string]string{
			"type": "keeplive",
			"tick": strconv.Itoa(int(time.Now().Unix())),
		})
		if err := writeMessage(r.conn, keepAlive); err != nil {
			log.Println("297:", err)
		}
		time.Sleep(40 * time.Second)
	}
}
