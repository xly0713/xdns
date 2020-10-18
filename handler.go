package main

import (
	"errors"
	"log"
	"net"

	"github.com/miekg/dns"
)

type xdnsHandler struct {
	//cache ...
}

func newHandler() *xdnsHandler {
	return &xdnsHandler{}
}

func (h *xdnsHandler) DoUDP(w dns.ResponseWriter, req *dns.Msg) {
	h.Do("udp", w, req)
}

func (h *xdnsHandler) DoTCP(w dns.ResponseWriter, req *dns.Msg) {
	h.Do("tcp", w, req)
}

func (h *xdnsHandler) Do(Net string, w dns.ResponseWriter, req *dns.Msg) {
	var remoteAddr net.IP
	if Net == "udp" {
		remoteAddr = w.RemoteAddr().(*net.UDPAddr).IP
	} else if Net == "tcp" {
		remoteAddr = w.RemoteAddr().(*net.TCPAddr).IP
	}

	var eRemoteAddr net.IP
	opt := req.IsEdns0()
	if opt != nil && len(opt.Option) > 0 {
		edns0 := opt.Option[0]
		if e, ok := edns0.(*dns.EDNS0_SUBNET); ok {
			eRemoteAddr = e.Address
		}
	}

	q := req.Question[0]
	log.Printf("Net: %s, client: %s, edge client: %s, lookup %s\n", Net, remoteAddr, eRemoteAddr, q.String()[1:])

	if q.Qclass != dns.ClassINET || (q.Qtype != dns.TypeA && q.Qtype != dns.TypeAAAA) {
		m := new(dns.Msg)
		m.SetRcode(req, dns.RcodeNotImplemented)
		_ = w.WriteMsg(m)
		return
	}

	rrs, ttl, err := h.getRRset(&q, &remoteAddr, &eRemoteAddr)
	if err != nil {
		dns.HandleFailed(w, req) // server failure
		return
	}

	m := new(dns.Msg)
	//m.RecursionAvailable = true  //fake允许递归
	m.SetReply(req)

	rr_header := &dns.RR_Header{
		Name:   q.Name,
		Rrtype: q.Qtype,
		Class:  q.Qclass,
		Ttl:    ttl,
	}

	answer, err := encodeRRset(q.Qtype, rr_header, rrs)
	if err != nil {
		dns.HandleFailed(w, req)
		return
	}
	m.Answer = answer

	if err := w.WriteMsg(m); err != nil {
		log.Printf("dns response error: %v", err)
	}
}

func (h *xdnsHandler) getRRset(q *dns.Question, remoteAddr, eRemoteAddr *net.IP) ([]string, uint32, error) {
	geoData, err := h.queryGeo(remoteAddr, eRemoteAddr)
	if err != nil {
		return nil, 0, err
	}

	ips, ttl, err := h.queryRRSet(q, geoData)
	if err != nil {
		return nil, 0, err
	}

	return ips, ttl, nil
}

func (h *xdnsHandler) queryGeo(remoteAddr, eRemoteAddr *net.IP) (*geoInfo, error) {
	//TODO: query mmdb ip database
	return &geoInfo{
		isp:      "dx",
		country:  "CN",
		province: "Shanghai",
		city:     "Shanghai",
	}, nil
}

func (h *xdnsHandler) queryRRSet(q *dns.Question, geoData *geoInfo) ([]string, uint32, error) {
	//TODO: query database, such as: redis
	//key := fmt.Sprintf("%s_%s", q.Name, q.Qtype)

	if q.Qtype == dns.TypeA {
		return []string{"1.1.1.1", "1.1.2.2"}, 3600, nil
	} else if q.Qtype == dns.TypeAAAA {
		return []string{"2001:db8::68"}, 300, nil
	}

	return nil, 0, errors.New("no rrset")
}

func encodeRRset(qtype uint16, rr_header *dns.RR_Header, rrs []string) (answer []dns.RR, err error) {
	switch qtype {
	case dns.TypeA:
		for _, rr := range rrs {
			if ip := net.ParseIP(rr); ip != nil {
				a := &dns.A{Hdr: *rr_header, A: ip}
				answer = append(answer, a)
			}
		}
	case dns.TypeAAAA:
		for _, rr := range rrs {
			if ip := net.ParseIP(rr); ip != nil {
				a := &dns.AAAA{Hdr: *rr_header, AAAA: ip}
				answer = append(answer, a)
			}
		}
	}

	return answer, err
}

type geoInfo struct {
	isp      string
	country  string //country code: https://zh.wikipedia.org/zh-hans/ISO_3166-1
	province string
	city     string
}
