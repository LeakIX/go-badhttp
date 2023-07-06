package go_badhttp

import (
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
)

type Request struct {
	Address     net.TCPAddr
	Url         *url.URL
	Method      string
	HttpVersion string
	Headers     map[string][]string
	Body        io.ReadCloser
}

func (req *Request) AddHeader(name, value string) {
	if currentHeader, found := req.Headers[name]; found {
		currentHeader = append(currentHeader, value)
	} else {
		req.Headers[name] = []string{value}
	}
}

func NewRawRequest(address net.TCPAddr, method, reqUrl string, body io.ReadCloser) (*Request, error) {
	parsedUrl, err := url.Parse(reqUrl)
	if err != nil {
		return nil, err
	}
	r := &Request{
		Address:     address,
		Method:      method,
		Url:         parsedUrl,
		HttpVersion: "HTTP/1.1",
		Headers:     make(map[string][]string),
		Body:        body,
	}
	r.AddHeader("Host", parsedUrl.Host)
	return r, nil
}

func NewRequest(method, reqUrl string, body io.ReadCloser) (*Request, error) {
	parsedUrl, err := url.Parse(reqUrl)
	if err != nil {
		return nil, err
	}
	r := &Request{
		Method:      method,
		Url:         parsedUrl,
		HttpVersion: "HTTP/1.1",
		Headers:     make(map[string][]string),
		Body:        body,
	}
	parsedPort := parsedUrl.Port()
	if parsedPort == "" && parsedUrl.Scheme == "http" {
		parsedPort = "80"
	}
	if parsedPort == "" && parsedUrl.Scheme == "https" {
		parsedPort = "443"
	}
	port, err := strconv.Atoi(parsedPort)
	if err != nil {
		return nil, err
	}
	var ip net.IP
	if ip = net.ParseIP(parsedUrl.Host); ip == nil {
		ips, err := net.LookupIP(parsedUrl.Hostname())
		if err != nil {
			return nil, err
		}
		if len(ips) < 1 {
			return nil, ErrDnsError
		}
		ip = ips[0]
	}
	r.Address = net.TCPAddr{IP: ip, Port: port}
	r.AddHeader("Host", parsedUrl.Host)
	return r, nil
}

var ErrDnsError = errors.New("error resolving dns")
