package proxy

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"gitlab.com/klikapim/lura/v2/config"
	"gitlab.com/klikapim/lura/v2/proxy"

	gologging "github.com/op/go-logging"
	gcb "gitlab.com/klikapim/circuitbreaker/v2/gobreaker"
)

func BenchmarkNewCircuitBreakerMiddleware_ok(b *testing.B) {
	p := NewMiddleware(&cfg, gologging.MustGetLogger("proxy_test"))(dummyProxy(&proxy.Response{}, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p(context.Background(), &proxy.Request{
			Path: "/tupu",
		})
	}
}

func BenchmarkNewCircuitBreakerMiddleware_ko(b *testing.B) {
	p := NewMiddleware(&cfg, gologging.MustGetLogger("proxy_test"))(dummyProxy(nil, errors.New("sample error")))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p(context.Background(), &proxy.Request{
			Path: "/tupu",
		})
	}
}

func BenchmarkNewCircuitBreakerMiddleware_burst(b *testing.B) {
	err := errors.New("sample error")
	p := NewMiddleware(&cfg, gologging.MustGetLogger("proxy_test"))(burstProxy(&proxy.Response{}, err, 100, 6))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p(context.Background(), &proxy.Request{
			Path: "/tupu",
		})
	}
}

var cfg = config.Backend{
	ExtraConfig: map[string]interface{}{
		gcb.Namespace: map[string]interface{}{
			"interval":   100.0,
			"timeout":    100.0,
			"max_errors": 1.0,
		},
	},
}

func dummyProxy(r *proxy.Response, err error) proxy.Proxy {
	return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		return r, err
	}
}

func burstProxy(r *proxy.Response, err error, ok, ko int) proxy.Proxy {
	tmp := make([]bool, ok+ko)
	for i := 0; i < ok+ko; i++ {
		tmp[i] = i < ok
	}
	calls := uint64(0)
	return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		total := atomic.AddUint64(&calls, 1) - 1
		if tmp[total%uint64(len(tmp))] {
			return r, nil
		}
		return nil, err
	}
}
