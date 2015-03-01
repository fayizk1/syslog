package syslog

import (
	"log"
	"sync"
	"time"
	"math/rand"
	"io/ioutil"
	"net/http"
	"net/url"
	"encoding/json"
)

type Handler interface {
	Handle(*Message) *Message
}

type BaseHandler struct {
	sync.RWMutex
	queue  chan []byte
	end    chan struct{}
	filter func(*Message) bool
	parse  func([]byte, string,string)([]byte, error)
	ft     bool
	inputId string
	graylog2NodeId string
	graylog2Index string
	graylog2_username string
	graylog2_password string
	graylog2_uri string
}

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}


func ReadClusterStatus(uri, userid, password string) (string, string, error){
	var res_json1, res_json2 map[string]interface{}
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://" + uri +"/system/deflector", nil)
	if err != nil {
		return "", "", err
        }
	req.SetBasicAuth(userid, password)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
        }
	if resp.StatusCode != 200 {
		return "", "", err
	}
	content, err := ioutil.ReadAll(resp.Body)
        if err != nil {
		return "", "", err
        }
	resp.Body.Close()
	err = json.Unmarshal(content, &res_json1)
	if err != nil {
		return "", "", err
	}
	index, ok := res_json1["current_target"].(string)
	if !ok {
		return "", "", err
	}
	req.URL, _ = url.Parse("http://" + uri +"/system")
	resp, err = client.Do(req)
	if err != nil {
		return "", "", err
        }
	if resp.StatusCode != 200 {
		return "", "", err
        }
	content, err = ioutil.ReadAll(resp.Body)
        if err != nil {
		return "", "", err
        }
        resp.Body.Close()
	err = json.Unmarshal(content, &res_json2)
	if err != nil {
		return "", "", err
        }
	serverid, ok := res_json2["server_id"].(string)
	if !ok {
                return "", "", err
	}
	return index, serverid, nil
}

func NewBaseHandler(qlen int, filter func(*Message) bool,parse func([]byte, string,string)([]byte, error), graylog2_username, graylog2_password, graylog2_uri string,ft bool) *BaseHandler {
	index, nodeid, err := ReadClusterStatus(graylog2_uri, graylog2_username, graylog2_password)
	if err != nil {
		panic(err)
	}
	return &BaseHandler{
		queue:  make(chan []byte, qlen),
		end:    make(chan struct{}),
		filter: filter,
		parse: parse,
		ft:     ft,
		inputId : randSeq(10),
		graylog2NodeId : nodeid,
		graylog2Index : index,
		graylog2_username : graylog2_username,
		graylog2_password : graylog2_username,
		graylog2_uri : graylog2_uri,
	}
}


func (h *BaseHandler) Handle(m *Message) *Message {
	if m == nil {
		close(h.queue) // signal that ther is no more messages for processing
		<-h.end        // wait for handler shutdown
		return nil
	}
	if h.filter != nil && !h.filter(m) {
		// m doesn't match the filter
		return m
	}
	h.RLock()
	message,err := m.Gelf(h.graylog2Index,randSeq(32), h.inputId, h.graylog2NodeId, h.parse)
	h.RUnlock()
	if err != nil {
		log.Println("Parse error,", err)
		return m
	}
	// Try queue m
	select {
	case h.queue <- message:
	default:
	}
	if h.ft {
		return m
	}
	return nil

}

func (h *BaseHandler) Get() []byte {
	m, ok := <-h.queue
	if ok {
		return m
	}
	return nil
}

func (h *BaseHandler) ValueUpdater(interval int) {
	for {
		time.Sleep(time.Duration(interval) * time.Second)
		go func() {
			index, nodeid, err := ReadClusterStatus(h.graylog2_uri, h.graylog2_username, h.graylog2_password)
			if err != nil {
				return
			}
			h.Lock()
			defer h.Unlock()
			h.graylog2NodeId = nodeid
			h.graylog2Index = index
		}()
	}
}

func (h *BaseHandler) Queue() <-chan []byte {
	return h.queue
}

func (h *BaseHandler) End() {
	close(h.end)
}

