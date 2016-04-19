package cache

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/service"
)

func TestCache(t *testing.T) {
	var (
		svc   = service.Fixed{}
		ca    = make(closer)
		cb    = make(closer)
		c     = map[string]io.Closer{"a": ca, "b": cb}
		f     = func(instance string) (service.Service, io.Closer, error) { return svc, c[instance], nil }
		cache = New(f, log.NewNopLogger())
	)

	// Populate
	cache.Update([]string{"a", "b"})
	select {
	case <-ca:
		t.Errorf("service a closed, not good")
	case <-cb:
		t.Errorf("service b closed, not good")
	case <-time.After(time.Millisecond):
		t.Logf("no closures yet, good")
	}
	services, err := cache.Services()
	if err != nil {
		t.Error(err)
	}
	if want, have := 2, len(services); want != have {
		t.Errorf("want %d, have %d", want, have)
	}

	// Duplicate, should be no-op
	cache.Update([]string{"a", "b"})
	select {
	case <-ca:
		t.Errorf("service a closed, not good")
	case <-cb:
		t.Errorf("service b closed, not good")
	case <-time.After(time.Millisecond):
		t.Logf("no closures yet, good")
	}
	services, err = cache.Services()
	if err != nil {
		t.Error(err)
	}
	if want, have := 2, len(services); want != have {
		t.Errorf("want %d, have %d", want, have)
	}

	// Delete b
	go cache.Update([]string{"a"})
	select {
	case <-ca:
		t.Errorf("service a closed, not good")
	case <-cb:
		t.Logf("service b closed, good")
	case <-time.After(100 * time.Millisecond):
		t.Errorf("didn't close the deleted instance in time")
	}
	services, err = cache.Services()
	if err != nil {
		t.Error(err)
	}
	if want, have := 1, len(services); want != have {
		t.Errorf("want %d, have %d", want, have)
	}

	// Delete a
	go cache.Update([]string{})
	select {
	// case <-cb: will succeed, as it's closed
	case <-ca:
		t.Logf("service a closed, good")
	case <-time.After(100 * time.Millisecond):
		t.Errorf("didn't close the deleted instance in time")
	}
	services, err = cache.Services()
	if err != nil {
		t.Error(err)
	}
	if want, have := 0, len(services); want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestBadFactory(t *testing.T) {
	cache := New(func(string) (service.Service, io.Closer, error) {
		return nil, nil, errors.New("bad factory")
	}, log.NewNopLogger())

	cache.Update([]string{"foo:1234", "bar:5678"})
	services, err := cache.Services()
	if err != nil {
		t.Error(err)
	}
	if want, have := 0, len(services); want != have {
		t.Errorf("want %d, have %d", want, have)
	}

}

type closer chan struct{}

func (c closer) Close() error { close(c); return nil }
