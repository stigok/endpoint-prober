package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type ProbeResult struct {
	URL        string
	StatusCode int
	Body       []byte `json:"-"`
	Error      error
}

type Prober interface {
	Start()
	Close()
	C() <-chan ProbeResult
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	svc := NewProbeService(ctx, []string{
		"https://blog.stigok.com/",
		"https://stigok.com/",
		"https://www.nrk.no/",
		"https://stigok.com/not/found?q=3",
	})

	go func() {
		log.Println("exiting upon request...")
		<-ctx.Done()
		svc.Close()
		stop()
	}()

	log.Println("starting probes...")
	go svc.Start()

	enc := json.NewEncoder(os.Stdout)
	for r := range svc.C() {
		if err := enc.Encode(r); err != nil {
			log.Printf("error: json encode: %v", err)
		}
	}
}

type ProbeService struct {
	Endpoints     []string
	probeInterval time.Duration
	c             chan ProbeResult
	client        *http.Client
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewProbeService(ctx context.Context, endpoints []string) Prober {
	ctx, cancel := context.WithCancel(ctx)
	svc := &ProbeService{
		Endpoints:     endpoints,
		probeInterval: 3000 * time.Millisecond,
		c:             make(chan ProbeResult),
		client:        http.DefaultClient,
		ctx:           ctx,
		cancel:        cancel,
	}
	return svc
}

func (svc *ProbeService) Start() {
	for svc.ctx.Err() == nil {
		rctx, cancel := context.WithTimeout(svc.ctx, svc.probeInterval)
		defer cancel()

		for _, url := range svc.Endpoints {
			go func(url string) {
				svc.c <- svc.probeURL(rctx, url)
			}(url)
		}

		time.Sleep(svc.probeInterval)
	}
}

func (svc *ProbeService) C() <-chan ProbeResult {
	return svc.c
}

func (svc *ProbeService) Close() {
	svc.cancel()
	svc.client.CloseIdleConnections()
	time.Sleep(time.Second)
	close(svc.c)
}

func (svc *ProbeService) probeURL(ctx context.Context, url string) ProbeResult {
	result := ProbeResult{
		URL: url,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Error = err
		return result
	}

	resp, err := svc.client.Do(req)
	if err != nil {
		result.Error = err
		return result
	}

	result.StatusCode = resp.StatusCode

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		return result
	}

	result.Body = b
	return result
}
