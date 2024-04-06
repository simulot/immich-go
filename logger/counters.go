package logger

import (
	"cmp"
	"maps"
	"sync"
	"time"
)

type Measure interface {
	cmp.Ordered
	String() string
}

// Counters implements a bunch of Measures
type Counters[M Measure] struct {
	l        sync.RWMutex
	counters map[M]int
	canary   int64
}

func NewCounters[M Measure]() *Counters[M] {
	c := Counters[M]{
		counters: map[M]int{},
		canary:   time.Now().UnixMilli(),
	}
	return &c
}

func (c *Counters[M]) Add(m M) {
	c.l.Lock()
	c.counters[m] = c.counters[m] + 1
	c.l.Unlock()
}

func (c *Counters[M]) GetCounters() map[M]int {
	if c == nil {
		return nil
	}
	c.l.RLock()
	defer c.l.RUnlock()

	r := map[M]int{}
	maps.Copy(r, c.counters)
	return r
}
