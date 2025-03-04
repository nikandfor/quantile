package kll

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type (
	// KLL is kll-like streaming quantile algorithm.
	KLL struct {
		v []float64
		l []int
		s []bool

		width, depth int
	}
)

func New(width, depth int) *KLL {
	if width%2 != 0 {
		panic(width)
	}

	return &KLL{
		v: make([]float64, depth*width), // values
		l: make([]int, depth),           // level length
		s: make([]bool, depth),          // sorted

		width: width,
		depth: depth,
	}
}

func (s *KLL) Query(q float64) float64 {
	lo, hi, n := s.preQuery()
	if q <= 0 {
		return lo
	}
	if q >= 1 {
		return hi
	}

	r := int(q * float64(n))

	for i := 0; i < 20; i++ {
		v := lo + (hi-lo)*q
		m, x := s.rank(v, hi)

		//	log.Printf("query step %2x  q %.3f: v %6.3f  [%6.3f %6.3f]  =>  %6.3f %d -> %d of %d", i, q, v, lo, hi, x, m, r, n)

		if m == r {
			return x
		}

		if m < r {
			lo = v
		} else {
			hi = v
		}
	}

	return lo + (hi-lo)*q
}

func (s *KLL) rank(v, hi float64) (r int, x float64) {
	var xok bool

	for l := 0; l < s.depth; l++ {
		if s.l[l] == 0 {
			break
		}

		st, end := s.startEnd(l)

		lr := sort.SearchFloat64s(s.v[st:end], v)
		r += lr

		if st+lr < end {
			n := s.v[st+lr]
			if !xok || n < x {
				x = n
				xok = true
			}
		}
	}

	if !xok {
		x = hi
	}

	return r, x
}

func (s *KLL) preQuery() (lo, hi float64, n int) {
	for l := 0; l < s.depth; l++ {
		if s.l[l] == 0 {
			break
		}

		n += s.l[l]

		if s.s[l] {
			continue
		}

		st, end := s.startEnd(l)

		s.sort(st, end)
		s.s[l] = true
	}

	lo = s.v[0]
	hi = lo

	for l := 0; l < s.depth; l++ {
		if s.l[l] == 0 {
			break
		}

		st, end := s.startEnd(l)

		if s.v[st] < lo {
			lo = s.v[st]
		}
		if s.v[end-1] > hi {
			hi = s.v[end-1]
		}
	}

	return lo, hi, n
}

func (s *KLL) Insert(v float64) {
	if math.IsNaN(v) {
		return
	}

	if s.l[0] == s.width {
		s.compact(0)
	}

	s.s[0] = s.l[0] == 0 || s.s[0] && v >= s.v[s.l[0]-1]

	s.v[s.l[0]] = v
	s.l[0]++
}

func (s *KLL) compact(l int) {
	if l+1 == s.depth {
		s.l[l] = 0
		return
	}

	if s.l[l+1] > s.width/2 {
		s.compact(l + 1)
	}

	st, end := s.startEnd(l)

	if !s.s[l] {
		s.sort(st, end)
	}

	next := end + s.l[l+1]
	i := 0

	for i < s.width/2 {
		s.v[next] = s.v[st+i]
		next++
		i += 2
	}

	for i < s.width {
		s.v[next] = s.v[st+i+1]
		next++
		i += 2
	}

	s.s[l+1] = s.l[l+1] == 0
	s.l[l+1] = next - end

	s.l[l] = 0
}

func (s *KLL) sort(st, end int) {
	sort.Float64s(s.v[st:end])
}

func (s *KLL) startEnd(l int) (st, end int) {
	st = l * s.width
	end = st + s.l[l]
	return st, end
}

func (s *KLL) dump() string {
	var b strings.Builder

	for l := range s.depth {
		st, end := s.startEnd(l)

		fmt.Fprintf(&b, "dump l %2x: %.2f\n", l, s.v[st:end])
	}

	return b.String()
}
