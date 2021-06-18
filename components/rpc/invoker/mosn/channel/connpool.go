package channel

import (
	"container/list"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"mosn.io/pkg/buffer"
	"mosn.io/pkg/log"
	"mosn.io/pkg/utils"
)

var (
	connpoolTimeout = errors.New("connection pool timeout")
)

type wrapConn struct {
	net.Conn
	buf   buffer.IoBuffer
	state interface{}

	closed    int32
	closeChan chan struct{}
}

func (w *wrapConn) isClose() bool {
	return atomic.LoadInt32(&w.closed) == 1
}

func (w *wrapConn) close() error {
	var err error
	if atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		err = w.Conn.Close()
		close(w.closeChan)
	}
	return err
}

// im-memory fake conn pool
func newConnPool(
	maxActive int,
	dialFunc func() (net.Conn, error),
	stateFunc func() interface{},
	onDataFunc func(*wrapConn) error) *connPool {

	p := &connPool{
		maxActive:  maxActive,
		dialFunc:   dialFunc,
		stateFunc:  stateFunc,
		onDataFunc: onDataFunc,
		sema:       make(chan struct{}, maxActive),
		free:       list.New(),
	}
	return p
}

type connPool struct {
	maxActive  int
	dialFunc   func() (net.Conn, error)
	stateFunc  func() interface{}
	onDataFunc func(*wrapConn) error

	sema chan struct{}
	mu   sync.Mutex
	free *list.List
}

func (p *connPool) Get(ctx context.Context) (*wrapConn, error) {
	if err := p.waitTurn(ctx); err != nil {
		return nil, err
	}

	p.mu.Lock()
	// get free conn
	if ele := p.free.Front(); ele != nil {
		p.free.Remove(ele)
		p.mu.Unlock()
		wc := ele.Value.(*wrapConn)
		if !wc.isClose() {
			return wc, nil
		}
	} else {
		p.mu.Unlock()
	}

	// create new conn
	c, err := p.dialFunc()
	if err != nil {
		p.freeTurn()
		return nil, err
	}
	wc := &wrapConn{Conn: c, closeChan: make(chan struct{})}
	if p.stateFunc != nil {
		wc.state = p.stateFunc()
	}
	if p.onDataFunc != nil {
		utils.GoWithRecover(func() {
			p.readloop(wc)
		}, nil)
	}
	return wc, nil
}

func (p *connPool) Put(c *wrapConn, close bool) {
	if close {
		c.close()
		p.freeTurn()
		return
	}

	p.mu.Lock()
	if p.free.Len() < p.maxActive {
		p.free.PushBack(c)
		p.mu.Unlock()
	} else {
		p.mu.Unlock()
		c.close()
	}
	p.freeTurn()
}

func (p *connPool) readloop(c *wrapConn) {
	defer c.close()

	c.buf = buffer.NewIoBuffer(16 * 1024)
	for {
		_, err := c.buf.ReadOnce(c)
		if err != nil {
			if err == io.EOF {
				log.DefaultLogger.Debugf("[runtime][rpc]connpool readloop err: %s", err.Error())
			} else {
				log.DefaultLogger.Errorf("[runtime][rpc]connpool readloop err: %s", err.Error())
			}
			return
		}
		if err = p.onDataFunc(c); err != nil {
			log.DefaultLogger.Errorf("[runtime][rpc]connpool onData err: %s", err.Error())
			return
		}
	}
}

func (p *connPool) waitTurn(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return connpoolTimeout
	case p.sema <- struct{}{}:
		return nil
	}
}

func (p *connPool) freeTurn() {
	<-p.sema
}
