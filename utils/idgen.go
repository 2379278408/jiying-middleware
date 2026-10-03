package utils

import (
	"sync"
	"sync/atomic"
)

type PositionGenerator struct {
	posMax int64
	posMin int64
}

const halfMaxPos int64 = 4500000000000000

func NewPositionGenerator() *PositionGenerator {
	return &PositionGenerator{posMax: halfMaxPos, posMin: -halfMaxPos}
}

func (g *PositionGenerator) GetFirstPosition() int64 { return g.posMin + 1 }

func (g *PositionGenerator) GetLastPosition() int64 { return g.posMax - 1 }

func (g *PositionGenerator) GetBeforePosition(pos int64) int64 {
	if pos < 0 {
		return -halfMaxPos - 1
	}
	half := pos / 2
	if half > 0 {
		half = -half
	}
	return half - 1
}

func (g *PositionGenerator) GetAfterPosition(pos int64) int64 {
	if pos >= 0 {
		return halfMaxPos + 1
	}
	half := pos / 2
	if half < 0 {
		half = -half
	}
	return half + 1
}

type IDGenerator struct {
	seq atomic.Int64
	mu  sync.Mutex
}

func NewIDGenerator(start int64) *IDGenerator {
	g := &IDGenerator{}
	if start > 0 {
		g.seq.Store(start)
	}
	return g
}

func (g *IDGenerator) NewID() int64 {
	return g.seq.Add(1)
}

func (g *IDGenerator) GetCurrentID() int64 {
	return g.seq.Load()
}

func (g *IDGenerator) SetMaxId(id int64) {
	for {
		cur := g.seq.Load()
		if id <= cur {
			return
		}
		if g.seq.CompareAndSwap(cur, id) {
			return
		}
	}
}
