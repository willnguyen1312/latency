package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/bloom42/astroflow-go"
	"github.com/bloom42/astroflow-go/log"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"
	"time"
)

const Port = "8080"

const Version = "1.0.0"

// Result stores httpstat info.
type Result struct {
	// The following are duration for each phase
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	contentTransfer  time.Duration

	// The followings are timeline of request
	NameLookup    time.Duration
	Connect       time.Duration
	Pretransfer   time.Duration
	StartTransfer time.Duration
	total         time.Duration

	t0 time.Time
	t1 time.Time
	t2 time.Time
	t3 time.Time
	t4 time.Time
	t5 time.Time // need to be provided from outside

	dnsStart      time.Time
	dnsDone       time.Time
	tcpStart      time.Time
	tcpDone       time.Time
	tlsStart      time.Time
	tlsDone       time.Time
	serverStart   time.Time
	serverDone    time.Time
	transferStart time.Time
	trasferDone   time.Time // need to be provided from outside

	// isTLS is true when connection seems to use TLS
	isTLS bool

	// isReused is true when connection is reused (keep-alive)
	isReused bool
}

type EndResult struct {
	DNSLookup        int64 `json:"dns_lookup"`
	TCPConnection    int64 `json:"tcp_connection"`
	TLSHandshake     int64 `json:"tls_handshake"`
	ServerProcessing int64 `json:"server_processing"`
	ContentTransfer  int64 `json:"content_transfer"`
	Total            int64 `json:"total"`
}

type Error struct {
	Error string `json:"error"`
}

func SendError(c *gin.Context) {
	c.JSON(500, Error{"unkown error"})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = Port
	}

	log.Config(
		astroflow.SetFormatter(astroflow.NewConsoleFormatter()),
	)

	r := gin.Default()
	r.Use(cors.Default())

	r.GET("*url", func(c *gin.Context) {
		u := c.Param("url")
		if u[0] == '/' {
			u = u[1:]
		}
		_, err := url.ParseRequestURI(u)
		if err != nil {
			c.JSON(400, Error{"url is not valid"})
			return
		}

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			log.Error(err.Error())
			SendError(c)
			return
		}

		var result Result
		ctx := WithHTTPStat(req.Context(), &result)
		req = req.WithContext(ctx)

		client := http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		}
		res, err := client.Do(req)
		if err != nil {
			log.Error(err.Error())
			SendError(c)
			return
		}

		if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
			log.Error(err.Error())
			SendError(c)
			return
		}
		res.Body.Close()
		result.End(time.Now())

		c.JSON(200, gin.H{
			"data": result.ToEndResult(),
		})
	})
	r.Run(fmt.Sprintf(":%s", port))
	/*
		args := os.Args
		if len(args) < 2 {
			log.Fatalf("Usage: go run main.go URL")
		}
		req, err := http.NewRequest("GET", args[1], nil)
		if err != nil {
			log.Fatal(err)
		}

		var result Result
		ctx := WithHTTPStat(req.Context(), &result)
		req = req.WithContext(ctx)

		client := http.DefaultClient
		res, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
			log.Fatal(err)
		}
		res.Body.Close()
		result.End(time.Now())

		fmt.Printf("%+v", result)
	*/
}

func (r *Result) durations() map[string]time.Duration {
	return map[string]time.Duration{
		"DNSLookup":        r.DNSLookup,
		"TCPConnection":    r.TCPConnection,
		"TLSHandshake":     r.TLSHandshake,
		"ServerProcessing": r.ServerProcessing,
		"ContentTransfer":  r.contentTransfer,

		"NameLookup":    r.NameLookup,
		"Connect":       r.Connect,
		"Pretransfer":   r.Connect,
		"StartTransfer": r.StartTransfer,
		"Total":         r.total,
	}
}

func (r Result) ToEndResult() EndResult {
	var contentTransfer int64
	var total int64

	if r.total > 0 {
		contentTransfer = int64(r.contentTransfer / time.Millisecond)
		total = int64(r.total / time.Millisecond)
	} else {
		contentTransfer = 0
		total = 0
	}
	ret := EndResult{
		DNSLookup:        int64(r.DNSLookup / time.Millisecond),
		TCPConnection:    int64(r.TCPConnection / time.Millisecond),
		TLSHandshake:     int64(r.TLSHandshake / time.Millisecond),
		ServerProcessing: int64(r.ServerProcessing / time.Millisecond),
		ContentTransfer:  contentTransfer,
		Total:            total,
	}
	return ret
}

// Format formats stats result.
func (r Result) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			var buf bytes.Buffer
			fmt.Fprintf(&buf, "DNS lookup:        %4d ms\n",
				int(r.DNSLookup/time.Millisecond))
			fmt.Fprintf(&buf, "TCP connection:    %4d ms\n",
				int(r.TCPConnection/time.Millisecond))
			fmt.Fprintf(&buf, "TLS handshake:     %4d ms\n",
				int(r.TLSHandshake/time.Millisecond))
			fmt.Fprintf(&buf, "Server processing: %4d ms\n",
				int(r.ServerProcessing/time.Millisecond))

			if r.total > 0 {
				fmt.Fprintf(&buf, "Content transfer:  %4d ms\n\n",
					int(r.contentTransfer/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Content transfer:  %4s ms\n\n", "-")
			}

			/*fmt.Fprintf(&buf, "Name Lookup:    %4d ms\n",
				int(r.NameLookup/time.Millisecond))
			fmt.Fprintf(&buf, "Connect:        %4d ms\n",
				int(r.Connect/time.Millisecond))
			fmt.Fprintf(&buf, "Pre Transfer:   %4d ms\n",
				int(r.Pretransfer/time.Millisecond))
			fmt.Fprintf(&buf, "Start Transfer: %4d ms\n",
				int(r.StartTransfer/time.Millisecond))
			*/
			if r.total > 0 {
				fmt.Fprintf(&buf, "Total:          %4d ms\n",
					int(r.total/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Total:          %4s ms\n", "-")
			}
			io.WriteString(s, buf.String())
			return
		}

		fallthrough
	case 's', 'q':
		d := r.durations()
		list := make([]string, 0, len(d))
		for k, v := range d {
			// Handle when End function is not called
			if (k == "ContentTransfer" || k == "Total") && r.t5.IsZero() {
				list = append(list, fmt.Sprintf("%s: - ms", k))
				continue
			}
			list = append(list, fmt.Sprintf("%s: %d ms", k, v/time.Millisecond))
		}
		io.WriteString(s, strings.Join(list, ", "))
	}

}

// WithHTTPStat is a wrapper of httptrace.WithClientTrace. It records the
// time of each httptrace hooks.
func WithHTTPStat(ctx context.Context, r *Result) context.Context {
	return withClientTrace(ctx, r)
}

// End sets the time when reading response is done.
// This must be called after reading response body.
func (r *Result) End(t time.Time) {
	r.trasferDone = t

	// This means result is empty (it does nothing).
	// Skip setting value(contentTransfer and total will be zero).
	if r.dnsStart.IsZero() {
		return
	}

	r.contentTransfer = r.trasferDone.Sub(r.transferStart)
	r.total = r.trasferDone.Sub(r.dnsStart)
}

// ContentTransfer returns the duration of content transfer time.
// It is from first response byte to the given time. The time must
// be time after read body (go-httpstat can not detect that time).
func (r *Result) ContentTransfer(t time.Time) time.Duration {
	return t.Sub(r.serverDone)
}

// Total returns the duration of total http request.
// It is from dns lookup start time to the given time. The
// time must be time after read body (go-httpstat can not detect that time).
func (r *Result) Total(t time.Time) time.Duration {
	return t.Sub(r.dnsStart)
}

func withClientTrace(ctx context.Context, r *Result) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.dnsStart = time.Now()
		},

		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.dnsDone = time.Now()

			r.DNSLookup = r.dnsDone.Sub(r.dnsStart)
			r.NameLookup = r.dnsDone.Sub(r.dnsStart)
		},

		ConnectStart: func(_, _ string) {
			r.tcpStart = time.Now()

			// When connecting to IP (When no DNS lookup)
			if r.dnsStart.IsZero() {
				r.dnsStart = r.tcpStart
				r.dnsDone = r.tcpStart
			}
		},

		ConnectDone: func(network, addr string, err error) {
			r.tcpDone = time.Now()

			r.TCPConnection = r.tcpDone.Sub(r.tcpStart)
			r.Connect = r.tcpDone.Sub(r.dnsStart)
		},

		TLSHandshakeStart: func() {
			r.isTLS = true
			r.tlsStart = time.Now()
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.tlsDone = time.Now()

			r.TLSHandshake = r.tlsDone.Sub(r.tlsStart)
			r.Pretransfer = r.tlsDone.Sub(r.dnsStart)
		},

		GotConn: func(i httptrace.GotConnInfo) {
			// Handle when keep alive is used and connection is reused.
			// DNSStart(Done) and ConnectStart(Done) is skipped
			if i.Reused {
				r.isReused = true
			}
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.serverStart = time.Now()

			// When client doesn't use DialContext or using old (before go1.7) `net`
			// pakcage, DNS/TCP/TLS hook is not called.
			if r.dnsStart.IsZero() && r.tcpStart.IsZero() {
				now := r.serverStart

				r.dnsStart = now
				r.dnsDone = now
				r.tcpStart = now
				r.tcpDone = now
			}

			// When connection is re-used, DNS/TCP/TLS hook is not called.
			if r.isReused {
				now := r.serverStart

				r.dnsStart = now
				r.dnsDone = now
				r.tcpStart = now
				r.tcpDone = now
				r.tlsStart = now
				r.tlsDone = now
			}

			if r.isTLS {
				return
			}

			r.TLSHandshake = r.tcpDone.Sub(r.tcpDone)
			r.Pretransfer = r.Connect
		},

		GotFirstResponseByte: func() {
			r.serverDone = time.Now()

			r.ServerProcessing = r.serverDone.Sub(r.serverStart)
			r.StartTransfer = r.serverDone.Sub(r.dnsStart)

			r.transferStart = r.serverDone
		},
	})
}
