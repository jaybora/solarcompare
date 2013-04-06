// +build !appengine

package httpclient

import (
	"net/http"
	"crypto/tls"
	"net/url"
	"net"
	"time"
)

type Jar struct {
	cookies []*http.Cookie
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		jar.cookies = append(jar.cookies, cookie)
	}
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

// Creates a new client with cookie jar support, and no TLS check, and deadline set
func NewClient() *http.Client {
	jar := new(Jar)
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: func(netw, addr string) (net.Conn, error) {
			// Specify timeout/deadline
			deadline := time.Now().Add(60 * time.Second)
			c, err := net.DialTimeout(netw, addr, 60 * time.Second)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		}}
	return &http.Client{Transport: transport, Jar: jar}

}

