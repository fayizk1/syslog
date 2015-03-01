package syslog

import (
	"fmt"
	"net"
	"strings"
	"time"
	"encoding/json"
)

type Message struct {
	Time   time.Time
	Source net.Addr
	Facility
	Severity
	Timestamp time.Time // optional
	Hostname  string    // optional
	Tag       string // message tag as defined in RFC 3164
	Content   string // message content as defined in RFC 3164
	Tag1      string // alternate message tag (white rune as separator)
	Content1  string // alternate message content (white rune as separator)
}

type index struct {
	Index string `json:"_index"`
	Type string `json:"_type"`
	Id string `json:"_id"`
} 

type OperationHeader struct {
	Index index `json:"index"`
}

type Gelf struct {
	Version string `json:"version"`
        Host  string `json:"host"`
	ShortMessage  string `json:"short_message"`
	Timestamp int64`json:"timestamp"`
	Level int  `json:"level"`
	Tag string `json:"_tag"`
	Source string `json:"_source"`
	LogType string `json:"_log_type"`
	Id string `json:"_id"`
	Gl2SourceInput string `json:"gl2_source_input"`
	Gl2SourceNode string `json:"gl2_source_node"`
}

// NetSrc only network part of Source as string (IP for UDP or Name for UDS)
func (m *Message) NetSrc() string {
	switch a := m.Source.(type) {
	case *net.UDPAddr:
		return a.IP.String()
	case *net.UnixAddr:
		return a.Name
	case *net.TCPAddr:
		return a.IP.String()
	}
	// Unknown type
	return m.Source.String()
}

func (m *Message) String() string {
	timeLayout := "2006-01-02 15:04:05"
	timestampLayout := "01-02 15:04:05"
	var h []string
	if !m.Timestamp.IsZero() {
		h = append(h, m.Timestamp.Format(timestampLayout))
	}
	if m.Hostname != "" {
		h = append(h, m.Hostname)
	}
	var header string
	if len(h) > 0 {
		header += " " + strings.Join(h, " ")
	}
	return fmt.Sprintf(
		"%s %s <%s,%s>%s %s%s",
		m.Time.Format(timeLayout), m.Source,
		m.Facility, m.Severity,
		header,
		m.Tag, m.Content,
	)
}

func (m *Message) Gelf(current_index , id, gl2_source_input, gl2_source_node string, callback func([]byte, string,string)([]byte, error)) ([]byte, error) {
	request := &OperationHeader{Index:index{Index:current_index, Type: "message", Id:id}}
	gelf := &Gelf{Version : "1.1", Host : m.Hostname, ShortMessage:m.Content, Timestamp:m.Time.Unix(), Level: int(m.Severity), 
		Tag: m.Tag, Source: m.NetSrc(), LogType: "syslog", Id:id, Gl2SourceInput:gl2_source_input, Gl2SourceNode:gl2_source_node}
	requestJ, err := json.Marshal(request)
	if err != nil {
                return nil, err
        }
	baseJ, err := json.Marshal(gelf)
	if err != nil {
		return nil, err
	}
	parseJ, err := callback(baseJ, m.Tag, m.Content)
	if err != nil {
		val := append(requestJ, []byte("\n")...)
		val = append(val, baseJ...)
		val = append(val, []byte("\n")...)
		return val, nil
	}
	val := append(requestJ, []byte("\n")...) 
	val = append(val, parseJ...)
	val = append(val, []byte("\n")...)
	return val, nil
}
