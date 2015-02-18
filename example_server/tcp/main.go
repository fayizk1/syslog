package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"github.com/fayizk1/syslog"
	"code.google.com/p/gcfg"
	"github.com/bennyscetbun/jsongo"
)

type handler struct {
	*syslog.BaseHandler
}

type Server struct {
	Uri string
	Grayloghostname string
	Graylogport uint16
}

type Filter struct {
	Tag []string
	Message []string
}

type Rule struct {
	Keys []string
	Keywords string
}

type RuleKeywords map[string]*Rule{}

type ConfigReader struct {
        Server struct {
                Uri string
		Grayloghostname string
		Graylogport uint16
        }
	Filter struct {
		Tag []string
		Message []string
	}
	Parser struct {
		Rules []string
	}
}

var tcpclt *TcpClient
var rulekeywords RuleKeywords
var filter Filter
var server Server 

func init() {
	var cfgrd ConfigReader
	rulekeywords = make(RuleKeywords)
	err := gcfg.ReadFileInto(&cfgrd, "server.gcfg")
	if err == nil {
		server.Uri = cfgrd.Server.Uri
		server.Grayloghostname = cfgrd.Server.Grayloghostname
		server.Graylogport = cfgrd.Server.Graylogport
		filter.Tag = cfgrd.Filter.Tag
		filter.Message = cfgrd.Filter.Message
	} else {
		server = Server{"0.0.0.0:514", "127.0.0.1", 12201}
//		filter =  Filter{[], []}
		fmt.Println("Config Error:", err)
	}
	fmt.Println(filter.Message, filter.Tag)
	tcpclt = MustTcpClient(server.Grayloghostname, server.Graylogport)
}

func rulesvalidator() {
	for i:= range Parser.Rules {
		temprulestr := strings.Split(Parser.Rules[i], "~~~")
		if len(temprulestr) == 3{
			if temprulestr[1] == "" {
				continue
			}
			tempkeys := temprulestr[1].Split(",")
			temprule := &Rule{Keys:tempkeys, Keywords : temprulestr[2]}
			rulekeywords[temprulestr[0]] = temprule
		}
	}
}

func filterfn(m *syslog.Message) bool {
	for i := range filter.Tag {
		//fmt.Println(filter.Tag[i])
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
	if (len(result) - 1) != len(result) {
		return nil,  errors.New("No match.")
	}
	root := jsongo.JSONNode{}
	err := json.Unmarshal(baseJ, &root)
	if err != nil {
		return nil, err
	}
	for i:= range rulekeyword.Keys {
		root.Map(rulekeyword.Keys[i]).Val(result[i+1])
	}
	parseJ, err := json.Marshal(&root)
	return parseJ, err
}

func newHandler() *handler {
	h := handler{syslog.NewBaseHandler(512, filterfn, false)}
	go h.mainLoop() 
	return &h
}

func (h *handler) mainLoop() {
/*	g := gelf.New(gelf.Config{
		GraylogPort:     server.Graylogport,
		GraylogHostname: server.Grayloghostname,
		Connection:      "wan",
	})*/
	for {
		m := h.Get()
		if m == nil {
			break
		}
		message, err := m.Gelf()
		if err != nil {
		   fmt.Println("Error", message)
		   continue
		}
		tcpclt.SendMessageData(message)
//		g.Log(m.Gelf())
	//	fmt.Println(m)
		//fmt.Println(m.Gelf())
	}
	fmt.Println("Exit handler")
	h.End()
}

func main() {
	s := syslog.NewServer()
	s.AddHandler(newHandler())
	s.Listen(server.Uri)

	sc := make(chan os.Signal, 2)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc

	fmt.Println("Shutdown the server...")
	s.Shutdown()
	fmt.Println("Server is down")
}
