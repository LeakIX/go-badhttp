package go_badhttp

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type Client struct {
	InsecureSkipVerify bool
	CloseConnection    bool
}

type ClientOpt func(client *Client)

func NewClient(opts ...ClientOpt) *Client {
	client := &Client{
		InsecureSkipVerify: false,
		CloseConnection:    true,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func WithInsecureSkipVerify(skip bool) ClientOpt {
	return func(client *Client) {
		client.InsecureSkipVerify = skip
	}
}

func WithConnectionClose(close bool) ClientOpt {
	return func(client *Client) {
		client.CloseConnection = close
	}
}

func (client *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	var conn net.Conn
	var err error
	if client.CloseConnection {
		req.AddHeader("Connection", "close")
	}
	if req.Url.Scheme == "http" {
		dialer := net.Dialer{}
		conn, err = dialer.DialContext(ctx, "tcp", req.Address.String())
	} else {
		dialer := tls.Dialer{
			Config: &tls.Config{
				InsecureSkipVerify: client.InsecureSkipVerify,
				ServerName:         req.Url.Hostname(),
			},
		}
		conn, err = dialer.DialContext(ctx, "tcp", req.Address.String())
	}
	if err != nil {
		return nil, err
	}
	respChan := client.ParseResponse(conn)
	_, err = fmt.Fprintf(conn, "%s %s %s\r\n", req.Method, req.Url.RequestURI(), req.HttpVersion)
	if err != nil {
		return nil, err
	}
	for headerName, headerValues := range req.Headers {
		for _, headerValue := range headerValues {
			_, err = fmt.Fprintf(conn, "%s: %s\r\n", headerName, headerValue)
			if err != nil {
				return nil, err
			}
		}
	}
	_, err = fmt.Fprintf(conn, "\r\n\r\n")
	if err != nil {
		return nil, err
	}
	if req.Body != nil {
		_, err = io.Copy(conn, req.Body)
		if err != nil {
			return nil, err
		}
		_, err = fmt.Fprintf(conn, "\r\n\r\n")
		if err != nil {
			return nil, err
		}
	}
	resp, valid := <-respChan
	if !valid {
		conn.Close()
		return nil, ErrInvalidResponse
	}
	return &resp, nil
}

func (client *Client) ParseResponse(reader io.Reader) chan Response {
	respChan := make(chan Response)
	go func() {
		resp := Response{
			Headers: make(map[string][]string),
		}
		defer close(respChan)
		buffer := bufio.NewReader(reader)
		line, err := buffer.ReadString('\n')
		if err != nil {
			return
		}
		resp.Status = strings.TrimSpace(line)
		statusPart := strings.Split(resp.Status, " ")
		if len(statusPart) > 1 {
			resp.StatusCode, _ = strconv.Atoi(statusPart[1])
		}
		for {
			rawLine, err := buffer.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(rawLine)
			if line == "" {
				break
			}
			headerParts := strings.SplitN(line, ":", 2)
			if len(headerParts) == 2 {
				resp.AddHeader(strings.TrimSpace(headerParts[0]), strings.TrimSpace(headerParts[1]))
			} else {
				resp.InvalidHeaders = append(resp.InvalidHeaders, line)
			}
		}
		resp.Body = buffer
		respChan <- resp
	}()
	return respChan
}

var ErrInvalidResponse = errors.New("invalid response")
