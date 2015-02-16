package main

import (
	"net"
	"time"
)

type TcpConnMng struct {
	net             string
	laddr           *net.TCPAddr
	raddr           *net.TCPAddr
	tcpConnection   *net.TCPConn
}

func DialTcpMng(net string, laddr, raddr *net.TCPAddr) (*TcpConnMng, error) {
	tcpConnMng := &TcpConnMng {
		net:             net,
		laddr:           laddr,
		raddr:           raddr,
	}

	if connErr := tcpConnMng.reconnect(); nil != connErr {
		return nil, connErr
	} else {
		return tcpConnMng, nil
	}
}

func (tcpConnMng *TcpConnMng) reconnect() error {
	if tcpConnMng.tcpConnection != nil {
		tcpConnMng.tcpConnection.Close()
		tcpConnMng.tcpConnection = nil
	}

	tcpConnection, tcpErr := net.DialTCP(
		tcpConnMng.net,
		tcpConnMng.laddr,
		tcpConnMng.raddr,
	)

	if nil != tcpErr {
		return tcpErr
	}

	tcpConnMng.tcpConnection = tcpConnection

	return nil
}

func (tcpConnMng *TcpConnMng) callWithRetry(action func([]byte) (int, error), data []byte,) (int,error,) {
	if n, err := action(data); nil == err {
		return n, err
	}

	if connErr := tcpConnMng.reconnect(); nil != connErr {
		return 0, connErr
	}

	return action(data)
}

func (tcpConnMng *TcpConnMng) Read(b []byte) (n int, err error) {
	return tcpConnMng.callWithRetry(tcpConnMng.tcpConnection.Read, b)
}

func (tcpConnMng *TcpConnMng) Write(b []byte) (n int, err error) {
	return tcpConnMng.callWithRetry(tcpConnMng.tcpConnection.Write, b)
}

func (tcpConnMng *TcpConnMng) Close() error {
	return tcpConnMng.tcpConnection.Close()
}

func (tcpConnMng *TcpConnMng) LocalAddr() net.Addr {
	return tcpConnMng.tcpConnection.LocalAddr()
}

func (tcpConnMng *TcpConnMng) RemoteAddr() net.Addr {
	return tcpConnMng.tcpConnection.RemoteAddr()
}

func (tcpConnMng *TcpConnMng) SetDeadline(t time.Time) error {
	return tcpConnMng.tcpConnection.SetDeadline(t)
}

func (tcpConnMng *TcpConnMng) SetReadDeadline(t time.Time) error {
	return tcpConnMng.tcpConnection.SetReadDeadline(t)
}

func (tcpConnMng *TcpConnMng) SetWriteDeadline(t time.Time) error {
	return tcpConnMng.tcpConnection.SetWriteDeadline(t)
}
