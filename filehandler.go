package syslog

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// FileHandler saves messages into text file. It properly handles logrotate
// HUP signal (closes a file and tries to open/create new one).
type FileHandler struct {
	*BaseHandler
	filename string
	f        *os.File
	l        *log.Logger
}

// NewFileHandler accepts all arguments expected by NewBaseHandler plus
// filename which is the path to the log file. It returns implementation of
// Handler interface that saves syslog messages into a text file.
func NewFileHandler(filename string, qlen int, filter func(*Message) bool,
	ft bool) *FileHandler {

	h := &FileHandler{
		BaseHandler: NewBaseHandler(qlen, filter, ft),
		filename:    filename,
	}
	go h.mainLoop()
	return h
}

// SetLogger changes internal logger used to log I/O errors. If l == nil
// (default value for internal logger) it uses functions from log package
func (h *FileHandler) SetLogger(l *log.Logger) {
	h.l = l
}

func (h *FileHandler) mainLoop() {
	defer h.BaseHandler.End()

	mq := h.BaseHandler.Queue()
	sq := make(chan os.Signal, 1)
	signal.Notify(sq, syscall.SIGHUP)

	for {
		select {
		case <-sq: // SIGHUP probably from logrotate
			h.checkErr(h.f.Close())
			h.f = nil
		case m, ok := <-mq: // message to save
			if !ok {
				if h.f != nil {
					h.checkErr(h.f.Close())
				}
				return
			}
			h.saveMessage(m)
		}
	}
}

func (h *FileHandler) saveMessage(m *Message) {
	var err error
	if h.f == nil {
		h.f, err = os.OpenFile(
			h.filename,
			os.O_WRONLY|os.O_APPEND|os.O_CREATE,
			0620,
		)
		if h.checkErr(err) {
			return
		}
	}
	_, err = h.f.WriteString(m.String() + "\n")
	h.checkErr(err)
}

func (h *FileHandler) checkErr(err error) bool {
	if err == nil {
		return false
	}
	if h.l == nil {
		log.Print(h.filename, ": ", err)
	} else {
		h.l.Print(h.filename, ": ", err)
	}
	return true
}