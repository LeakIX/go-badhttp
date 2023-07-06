package go_badhttp

import "io"

type Response struct {
	Status         string
	StatusCode     int
	Headers        map[string][]string
	InvalidHeaders []string
	Body           io.Reader
}

func (resp *Response) AddHeader(name, value string) {
	if currentHeader, found := resp.Headers[name]; found {
		currentHeader = append(currentHeader, value)
	} else {
		resp.Headers[name] = []string{value}
	}
}
