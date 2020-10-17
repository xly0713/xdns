package main

import (
	"log"
	"net"
	"strconv"

	"github.com/miekg/dns"
)

type server struct {
	host    string
	port    int
	handler *xdnsHandler
	//readTimeout  time.Duration
	//writeTimeout time.Duration
}

func (s *server) addr() string {
	return net.JoinHostPort(s.host, strconv.Itoa(s.port))
}

func (s *server) run() {
	s.handler = newHandler()

	s.runUpdSrv()
	s.runTcpSrv()
}

func (s *server) runUpdSrv() {
	udpMux := dns.NewServeMux()
	udpMux.HandleFunc(".", s.handler.DoUDP)

	udpServer := &dns.Server{
		Addr:    s.addr(),
		Net:     "udp",
		Handler: udpMux,
	}

	go s.start(udpServer)
}

func (s *server) runTcpSrv() {
	tcpMux := dns.NewServeMux()
	tcpMux.HandleFunc(".", s.handler.DoTCP)

	tcpServer := &dns.Server{
		Addr:    s.addr(),
		Net:     "tcp",
		Handler: tcpMux,
	}

	go s.start(tcpServer)
}

func (s *server) start(ds *dns.Server) {
	if err := ds.ListenAndServe(); err != nil {
		log.Fatalf("Start %s listener on %s failed, error: %v", ds.Net, s.addr(), err)
	}
}
