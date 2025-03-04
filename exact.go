package quantile

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type (
	Exact struct {
		v []float64

		sorted bool
	}
)

func NewExact() *Exact {
	return &Exact{}
}

func (s *Exact) Query(q float64) float64 {
	if len(s.v) == 0 {
		return 0
	}

	if !s.sorted {
		s.sort()
	}

	if q <= 0 {
		return s.v[0]
	}
	if q >= 1 {
		return s.v[len(s.v)-1]
	}

	i := int(q * float64(len(s.v)))
	if i == len(s.v) {
		i--
	}

	return s.v[i]
}

func (s *Exact) Insert(v float64) {
	if math.IsNaN(v) {
		return
	}

	s.sorted = len(s.v) == 0 || s.sorted && v > s.v[len(s.v)-1]

	s.v = append(s.v, v)
}

func (s *Exact) sort() {
	sort.Float64s(s.v)
	s.sorted = true
}

func (s *Exact) dump() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%.2f\n", s.v)

	return b.String()
}

var _ = (*Exact)(nil).dump
