package util

import (
	"sync"
	"sync/atomic"
	"time"
)

type val struct {
	data        interface{}
	expiredTime int64
}

const delChannelCap = 100

type ExpiredMap struct {
	m        map[interface{}]*val
	timeMap  map[int64][]interface{}
	lock     *sync.Mutex
	stop     chan struct{}
	needStop int32
}

func NewExpiredMap() *ExpiredMap {
	e := ExpiredMap{
		m:       make(map[interface{}]*val),
		timeMap: make(map[int64][]interface{}),
		lock:    new(sync.Mutex),
		stop:    make(chan struct{}),
	}
	atomic.StoreInt32(&e.needStop, 0)
	go e.run(time.Now().Unix())
	return &e
}

type delMsg struct {
	keys []interface{}
	t    int64
}

func (e *ExpiredMap) run(now int64) {
	t := time.NewTicker(time.Second)
	delCh := make(chan *delMsg, delChannelCap)
	go func() {
		for v := range delCh {
			if atomic.LoadInt32(&e.needStop) == 1 {
				return
			}
			e.multiDelete(v.keys, v.t)
		}
	}()
	for {
		select {
		case <-t.C:
			now++
			if keys, found := e.timeMap[now]; found {
				delCh <- &delMsg{keys: keys, t: now}
			}
		case <-e.stop:
			atomic.StoreInt32(&e.needStop, 1)
			delCh <- &delMsg{keys: []interface{}{}, t: 0}
			return
		}
	}
}

func (e *ExpiredMap) multiDelete(keys []interface{}, t int64) {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.timeMap, t)
	for _, key := range keys {
		delete(e.m, key)
	}
}

func (e *ExpiredMap) checkDeleteKey(key interface{}) bool {
	if val, found := e.m[key]; found {
		if val.expiredTime <= time.Now().Unix() {
			delete(e.m, key)
			return false
		}
		return true
	}
	return false
}

func (e *ExpiredMap) Set(key, value interface{}, expireSeconds int64) {
	if expireSeconds == 0 {
		return
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	expiredTime := time.Now().Unix() + expireSeconds
	e.m[key] = &val{
		data:        value,
		expiredTime: expiredTime,
	}
	e.timeMap[expiredTime] = append(e.timeMap[expiredTime], key)
}

func (e *ExpiredMap) Get(key interface{}) (interface{}, bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if !e.checkDeleteKey(key) {
		return nil, false
	}
	return e.m[key].data, true
}

func (e *ExpiredMap) Delete(key interface{}) {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.m, key)
}

func (e *ExpiredMap) Length() int {
	e.lock.Lock()
	defer e.lock.Unlock()
	return len(e.m)
}

func (e *ExpiredMap) TTL(key interface{}) int64 {
	e.lock.Lock()
	defer e.lock.Unlock()
	if !e.checkDeleteKey(key) {
		return -1
	}
	return e.m[key].expiredTime - time.Now().Unix()
}

func (e *ExpiredMap) Clear() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.m = make(map[interface{}]*val)
	e.timeMap = make(map[int64][]interface{})
}

func (e *ExpiredMap) Close() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.stop <- struct{}{}
}

func (e *ExpiredMap) DoForEach(handler func(interface{}, interface{})) {
	e.lock.Lock()
	defer e.lock.Unlock()
	for k, v := range e.m {
		if !e.checkDeleteKey(k) {
			continue
		}
		handler(k, v)
	}
}

func (e *ExpiredMap) DoForEachWithBreak(handler func(interface{}, interface{}) bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	for k, v := range e.m {
		if !e.checkDeleteKey(k) {
			continue
		}
		if handler(k, v) {
			break
		}
	}
}
