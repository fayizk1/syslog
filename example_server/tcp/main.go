package main

import (
	"log"
	"os"
	"time"
	"syscall"
	"strings"
	"errors"
	"regexp"
	"os/signal"
	"encoding/json"
	"github.com/fayizk1/gen-go/elasticsearch"
	"github.com/fayizk1/syslog"
	"code.google.com/p/gcfg"
)

type handler struct {
	*syslog.BaseHandler
}

type Server struct {
	Uri string
	Grayloghostname string
	Graylogport string
}

type Filter struct {
	Tag []string
	Message []string
}

type Rule struct {
	Keys []string
	Keywords string
}

type RuleKeywords map[string]Rule

type ConfigReader struct {
        Server struct {
                Uri string
		Grayloghostname string
		Graylogport string
        }
	Filter struct {
		Tag []string
		Message []string
	}
	Parser struct {
		Rules []string
	}
}


//var tcpclt *TcpClient
var thriftclt *thriftClient
var rulekeywords RuleKeywords
var filter Filter
var server Server 
var cfgrd ConfigReader

func init() {
	rulekeywords = make(RuleKeywords)
	err := gcfg.ReadFileInto(&cfgrd, "server.gcfg")
	if err == nil {
		server.Uri = cfgrd.Server.Uri
		server.Grayloghostname = cfgrd.Server.Grayloghostname
		server.Graylogport = cfgrd.Server.Graylogport
		filter.Tag = cfgrd.Filter.Tag
		filter.Message = cfgrd.Filter.Message
		rulesvalidator()
	} else {
		server = Server{"0.0.0.0:514", "127.0.0.1", "12201"}
		log.Println("Config Error:", err)
	}
	log.Println(filter.Message, filter.Tag)
	thriftclt, err = NewThriftClient(server.Grayloghostname, server.Graylogport)
	if err != nil {
		panic(err)
	}

}

func rulesvalidator() {
	log.Println(cfgrd.Parser.Rules)
	for i:= range cfgrd.Parser.Rules {
		log.Println(cfgrd.Parser.Rules[i])
		temprulestr := strings.Split(cfgrd.Parser.Rules[i], "~~~")
		if len(temprulestr) == 3{
			if temprulestr[1] == "" {
				continue
			}
			tempkeys := strings.Split(temprulestr[1], ",")
			temprule := Rule{Keys:tempkeys, Keywords : temprulestr[2]}
			rulekeywords[temprulestr[0]] = temprule
		}
	}
	log.Println(rulekeywords)
}

func filterfn(m *syslog.Message) bool {
	for i := range filter.Tag {
		if m.Tag == filter.Tag[i] {
			return false
		}
	}
	for i:= range filter.Message {
		if strings.Contains(m.Content, filter.Message[i]) {
			return false
		}
	}
	return true
}

func parserfn(baseJ []byte, tag, content string) ([]byte, error) {
	rulekeyword, ok := rulekeywords[tag]
	if !ok {
		return nil, errors.New("Tag not found.")
	}
	Regexp := regexp.MustCompile(rulekeyword.Keywords)
	result := Regexp.FindStringSubmatch(content)
	if (len(result) - 1) != len(rulekeyword.Keys) {
		return nil, errors.New("No match.")
	}
	root := make(map[string]interface{})
	d := json.NewDecoder(strings.NewReader(string(baseJ)))
	d.UseNumber()
	err := d.Decode(&root);
	if err != nil {
		return nil, err
	}
	for i:= range rulekeyword.Keys {
		root[rulekeyword.Keys[i]] = result[i+1]
	}
	parseJ, err := json.Marshal(&root)
	return parseJ, err
}

func newHandler() *handler {
	h := handler{syslog.NewBaseHandler(512, filterfn, parserfn, "admin", "yourpassword", "10.0.2.15:12900",false)}
	go h.ValueUpdater(1)
	go h.mainLoop() 
	return &h
}

func (h *handler) mainLoop() {
	end := false
	var request = elasticsearch.RestRequest{
		Method: elasticsearch.Method_POST,
		Uri:    "/_bulk",
	}
	for {
		var message []byte
		for i := 0;i<100; i++ { 
			tempmessage := h.Get()
			if tempmessage == nil {
				end = true
				break
			}
			message = append(message, tempmessage...)
		}
		request.Body = message
	send:
		err := thriftclt.SendData(request)
		if err != nil {
			log.Println("Failed to send message,", err)
			time.Sleep(100 * time.Millisecond)
			log.Println("sending data gain")
			goto send
		}
		if end {
			break
		}
	}
	log.Println("Exit handler")
	h.End()
}

func main() {
	s := syslog.NewServer()
	s.AddHandler(newHandler())
	s.Listen(server.Uri)
	sc := make(chan os.Signal, 2)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc
	log.Println("Shutdown the server...")
	s.Shutdown()
	log.Println("Server is down")
}
