package syslog

import (
	"fmt"
	"net"
	"strings"
	"bytes"
	"time"
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

func (m *Message) Gelf() string {
	var buffer bytes.Buffer
	timeLayout := "2006-01-02 15:04:05"
        timestampLayout := "01-02 15:04:05"
	buffer.WriteString(`{"version": "1.1","host":"`)
	buffer.WriteString(fmt.Sprintf(`%s", "short_message":"%s", `, m.Hostname, m.Content))
	buffer.WriteString(fmt.Sprintf(`"timestamp":%d, "level":%d, `, m.Time.Unix(), m.Severity))
	buffer.WriteString(fmt.Sprintf(`"_tag":"%s"}`, m.Tag))
	return buffer.String()
}
