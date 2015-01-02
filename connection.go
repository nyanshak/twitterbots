package twitterbots

import (
	"github.com/ChimeraCoder/anaconda"
	"net"
	"io"
	"encoding/json"
	"net/http"
	"time"
)

type Connection struct {
	decoder *json.Decoder
	Conn    net.Conn
	Client  *http.Client
	closer  io.Closer
	timeout time.Duration
}

func (c *Connection) Close() error {
	// Have to close the raw connection, since closing the response body reader
	// will make Go try to read the request, which goes on forever.
	if err := c.Conn.Close(); err != nil {
		c.closer.Close()
		return err
	}
	return c.closer.Close()
}

func (c *Connection) Next() (*anaconda.Tweet, error) {
	var tweet anaconda.Tweet
	if c.timeout != 0 {
		c.Conn.SetReadDeadline(time.Now().Add(c.timeout))
	}
	if err := c.decoder.Decode(&tweet); err != nil {
		return nil, err
	}
	return &tweet, nil
}

func (c *Connection) Setup(rc io.ReadCloser) {
	c.closer = rc
	c.decoder = json.NewDecoder(rc)
}

func NewConnection(timeout time.Duration) *Connection {
	Conn := &Connection{timeout: timeout}
	dialer := func(netw, addr string) (net.Conn, error) {
		netc, err := net.DialTimeout(netw, addr, 5 * time.Second)
		if err != nil {
			return nil, err
		}
		Conn.Conn = netc
		return netc, nil
	}

	Conn.Client = &http.Client{
		Transport: &http.Transport{
			Dial: dialer,
		},
	}

	return Conn
}
