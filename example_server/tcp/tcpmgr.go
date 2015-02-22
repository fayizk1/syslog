package main

import (
	"net"
	"time"
	"errors"
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

	if connErr := tcpConnMng.Reconnect(); nil != connErr {
		return nil, connErr
	} else {
		return tcpConnMng, nil
	}
}

func (tcpConnMng *TcpConnMng) Reconnect() error {
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

func (tcpConnMng *TcpConnMng) Read(b []byte) (n int, err error) {
	if tcpConnMng.tcpConnection == nil {
		return 0, errors.New("Cant read , Disconnected")
	}
	return tcpConnMng.tcpConnection.Read(b)
}

func (tcpConnMng *TcpConnMng) Write(b []byte) (n int, err error) {
	if tcpConnMng.tcpConnection == nil {
		return 0, errors.New("Cant Write , Disconnected")
        }
	return tcpConnMng.tcpConnection.Write(b)
}

func (tcpConnMng *TcpConnMng) Close() error {
	if tcpConnMng.tcpConnection == nil {
		return nil
	}
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
