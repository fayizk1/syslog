package main
import (
	"net"
	"sync"
	"time"
	"log"
	"git-wip-us.apache.org/repos/asf/thrift.git/lib/go/thrift"
	"github.com/fayizk1/gen-go/elasticsearch" // generated code
)

const BUFFER_SIZE = 1024

type thriftClient struct {
	sync.Mutex
	client *elasticsearch.RestClient
	host, thriftPort string
}

func Connect(host string, thriftPort string) (*elasticsearch.RestClient, error) {
	binaryProtocol := thrift.NewTBinaryProtocolFactoryDefault()
	socket, err := thrift.NewTSocket(net.JoinHostPort(host, thriftPort))
	if err != nil {
		return nil, err
	}
	bufferedTransport := thrift.NewTBufferedTransport(socket, BUFFER_SIZE)
	client := elasticsearch.NewRestClientFactory(bufferedTransport, binaryProtocol)
	if err := bufferedTransport.Open(); err != nil {
		return nil, err
	}
	return client, nil
}

func NewThriftClient(host string, thriftPort string) (*thriftClient, error) {
	client, err := Connect(host, thriftPort)
	if err != nil {
		return nil, err
	}
	return &thriftClient{client:client, host:host, thriftPort:thriftPort}, nil
}

func (tc *thriftClient) Reconnect() {
	tc.Lock()
	defer tc.Unlock()
	if tc.client.Transport != nil {
		tc.client.Transport.Close()
	}
connect:
	log.Println("Reconnecting thrift client- ", tc.host, tc.thriftPort)
	client, err := Connect(tc.host, tc.thriftPort)
	if err != nil {
		log.Println("Reconnecting failed, waiting 2 Sec before next try")
		time.Sleep(2 * time.Second)
		goto connect
	}
	tc.client = client
}

func (tc *thriftClient) SendData(request elasticsearch.RestRequest) error {
	tc.Lock()
	defer tc.Unlock()
	_,err := tc.client.Execute(&request)
	if err != nil {
		go tc.Reconnect()
		return err
	}
	return nil
}
