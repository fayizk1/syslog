package main

import (
	"net"
	"log"
	"fmt"
	"sync"
	"time"
	"errors"
	"strconv"
)

const TCP_NETWORK = "tcp"

var MESSAGE_SEPARATOR = []byte{0}
type MessageData []byte

type TcpClient struct {
	sync.Mutex
	ServerAddr *net.TCPAddr
	connection  *TcpConnMng
}

func NewTcpClient(host string, port uint16) (*TcpClient, error) {
	hostWithPort := net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10))

	ipAddr, resolveErr := net.ResolveTCPAddr(TCP_NETWORK, hostWithPort)
	if nil != resolveErr {
		return nil, resolveErr
	}

	connection, dialErr := DialTcpMng(TCP_NETWORK, nil, ipAddr)
	if nil != dialErr {
		return nil, dialErr
	}

	return &TcpClient {
		ServerAddr: ipAddr,
		connection: connection,
	}, nil

}

func MustTcpClient(host string, port uint16) *TcpClient {
	tcpClient, err := NewTcpClient(host, port)
	if nil != err {
		panic(err.Error())
	}
	return tcpClient
}

func (tcpClient *TcpClient) SendMessageData(message MessageData) (err error) {
	tcpClient.Lock()
	defer tcpClient.Unlock()
	defer func() {
		pe := recover()
		if pe != nil {
			err = errors.New(fmt.Sprintf("Recovers from panic %v, " , pe))
		}
	}()
     	messageWithSeparator := append(message, MESSAGE_SEPARATOR...)
	if _, err := tcpClient.connection.Write(messageWithSeparator); nil != err {
		go tcpClient.Reconnect()
		return err
	}
	return nil
}

func (tcpClient *TcpClient) Reconnect() {
	tcpClient.Lock()
	defer tcpClient.Unlock()
	defer recover()
connect:
	log.Println("Reconnecting.......")
	err := tcpClient.connection.Reconnect()
	if err != nil {
		log.Println("Failed to reconnect ....., Waiting 2 sec...")
		time.Sleep(2 * time.Second)
		goto connect
	}
	log.Println("Successfully reconnected.")
}
