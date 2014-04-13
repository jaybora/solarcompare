// +build !appengine

package httpclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	//	"net/http/cookiejar"
	"net/url"
	"time"
)

type Jar struct {
	cookies []*http.Cookie
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie != nil && len(cookie.Value) > 0 {
			jar.cookies = append(jar.cookies, cookie)
		}
	}
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

func noRedirects(req *http.Request, via []*http.Request) error {
	return fmt.Errorf("Dont follow redirects")
}

// Creates a new client with cookie jar support, and no TLS check, and deadline set
func NewClient() *http.Client {
	//jar, _ := cookiejar.New(nil)
	jar := new(Jar)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: func(netw, addr string) (net.Conn, error) {
			// Specify timeout/deadline
			deadline := time.Now().Add(30 * time.Second)
			c, err := net.DialTimeout(netw, addr, 5*time.Second)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		}}
	return &http.Client{Transport: transport,
		Jar:           jar,
		CheckRedirect: noRedirects}

}
