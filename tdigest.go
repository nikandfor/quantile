package quantile

// Base on clickhouse implementation by Alexei Borzenkov (https://github.com/snaury).
// https://github.com/ClickHouse/ClickHouse/blob/master/src/AggregateFunctions/QuantileTDigest.h

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type (
	TDigest struct {
		tdigest

		Decay float32

		Compressions      int
		BruteCompressions int
	}

	tdigest struct {
		v []float64
		w []float32

		i    int
		size int

		eps float32

		sorted bool
	}
)

const epsSize = 8 * 1024

func TDigestSize(eps float32) int {
	size := int(eps) * epsSize
	return size &^ 1
}

func TDigestEpsilon(size int) float32 {
	return float32(size) / epsSize
}

// NewTDigest creates a new tdigest stream.
// 512 is a good size to start with.
// TDigestEpsilon(size) is a good epsilon to start with.
func NewTDigest(size int, eps float32) *TDigest {
	if size%2 != 0 {
		panic(size)
	}

	return &TDigest{
		tdigest: tdigest{
			v: make([]float64, size),
			w: make([]float32, size),

			size: size,
			eps:  eps,
		},

		Decay: 1,
	}
}

func (s *TDigest) Query(q float64) float64 {
	var buf [1]float64

	s.QueryMulti([]float64{q}, buf[:])

	return buf[0]
}

// QueryMulti make multiple queries at once.
// qs is a list of queries (quantiles).
// res is a buffer for results, res[i] = Query(qs[i]).
func (s *TDigest) QueryMulti(qs, res []float64) {
	if s.i == 0 || len(qs) == 0 {
		for i := range qs {
			res[i] = 0
		}

		return
	}
	if s.i == 1 {
		for i := range qs {
			res[i] = s.v[0]
		}

		return
	}

	if !s.sorted {
		s.sort()
	}

	copy(res, qs)
	sort.Float64s(res[:len(qs)])

	var total, sum, prev float32

	for _, w := range s.w[:s.i] {
		total += w
	}

	qi := 0

	for qi < len(qs) && res[qi] <= 0 {
		res[qi] = s.v[0]
		qi++
	}

	for qi < len(qs) && res[qi] >= 1 {
		res[qi] = s.v[s.i-1]
		qi++
	}

	if qi == len(qs) {
		return
	}

	target := float32(res[qi]) * total
	prevV := s.v[0]

	for i := 0; i < s.i; {
		cur := sum + 0.5*s.w[i]

		//	log.Printf("query %.2f  i %2d  cur %.3f / %.3f  v %.2f", q, i, cur, target, s.v[i])

		if cur >= target {
			l := prev
			r := cur

			switch {
			case target <= l:
				res[qi] = prevV
			case target >= r:
				res[qi] = s.v[i]
			default:
				res[qi] = s.interpolate(cur, l, r, prevV, s.v[i])
			}

			qi++

			if qi == len(qs) {
				break
			}

			continue
		}

		sum += s.w[i]

		prev = cur
		prevV = s.v[i]

		i++
	}

	for qi < len(qs) {
		res[qi] = s.v[s.i-1]
		qi++
	}
}

func (s *TDigest) interpolate(x, x1, x2 float32, y1, y2 float64) float64 {
	k := float64(x-x1) / float64(x2-x1)

	return y1*(1-k) + y2*k
}

func (s *TDigest) Insert(v float64) {
	if math.IsNaN(v) {
		return
	}

	if s.i == s.size {
		s.compress()
	}

	s.v[s.i] = v
	s.w[s.i] = 1

	s.sorted = s.i == 0 || s.sorted && v > s.v[s.i-1]
	s.i++
}

func (s *TDigest) compress() {
	if !s.sorted {
		s.sort()
	}

	//	log.Printf("compress\nv: %5.2f\nw: %5.2f\n", s.v[:s.i], s.w[:s.i])

	s.compress0()
	s.Compressions++

	if s.i != s.size {
		//		log.Printf("light\nv: %5.2f\nw: %5.2f\n", s.v[:s.i], s.w[:s.i])
		return
	}

	s.compressBrute()
	s.BruteCompressions++

	// log.Printf("brute\nv: %5.2f\nw: %5.2f\n", s.v[:s.i], s.w[:s.i])

	if s.Decay != 1 {
		for i := range s.i {
			s.w[i] *= s.Decay
		}
	}
}

func (s *TDigest) compress0() {
	var total float32

	for _, w := range s.w[:s.i] {
		total += w
	}

	totalEpsilon4 := total * s.eps * 4

	l, r := 0, 1

	var sum float32

	//	log.Printf("compress\nv: %5.2f\nw: %5.2f\n", s.v[:s.i], s.w[:s.i])
	//	log.Printf("total %d  %v  eps %v", s.i, total, s.eps)

	for r < s.i {
		ql := (sum + s.w[l]*0.5) / total
		qr := (sum + s.w[l] + s.w[r]*0.5) / total

		err := ql * (1 - ql)
		if err2 := qr * (1 - qr); err2 < err {
			err = err2
		}

		k := err * totalEpsilon4

		//	log.Printf("pair  l %3v r %3v  w %5.2f %5.2f  q %.2f %.2f  err %.2f %.2f  k %.3f  merge %v", l, r, s.w[l], s.w[r], ql, qr, ql*(1-ql), qr*(1-qr), k, s.w[l]+s.w[r] <= k)

		if s.w[l]+s.w[r] <= k && s.canBeMerged(s.v[l], s.v[r]) {
			if s.v[l] != s.v[r] { // Handling infinities of the same sign well.
				s.v[l] = (s.v[l]*float64(s.w[l]) + s.v[r]*float64(s.w[r])) / float64(s.w[l]+s.w[r])
			}

			s.w[l] += s.w[r]
		} else {
			sum += s.w[l]
			l++

			if l != r {
				s.v[l] = s.v[r]
				s.w[l] = s.w[r]
			}
		}

		r++
	}

	s.i = l + 1

	//	log.Printf("light\nv: %5.2f\nw: %5.2f\ntotal %.0f -> %.0f", s.v[:s.i], s.w[:s.i], total, sum+s.w[l])
}

func (s *TDigest) compressBrute() {
	if s.i%2 != 0 {
		panic(s.i)
	}

	for i := range s.i / 2 {
		n := i * 2
		ww := s.w[n] + s.w[n+1]

		s.v[i] = (s.v[n]*float64(s.w[n]) + s.v[n+1]*float64(s.w[n+1])) / float64(ww)
		s.w[i] = ww
	}

	s.i /= 2
}

func (s *TDigest) canBeMerged(l, r float64) bool {
	return !math.IsInf(l, 0) && !math.IsInf(r, 0) || l == r
}

func (s *TDigest) dump() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%.2f\n", s.v[:s.i])
	fmt.Fprintf(&b, "%.2f\n", s.w[:s.i])

	return b.String()
}

func (s *TDigest) sort() {
	sort.Sort(&s.tdigest)
	s.sorted = true
}

func (s *tdigest) Len() int           { return s.i }
func (s *tdigest) Less(i, j int) bool { return s.v[i] < s.v[j] }
func (s *tdigest) Swap(i, j int) {
	s.v[i], s.v[j] = s.v[j], s.v[i]
	s.w[i], s.w[j] = s.w[j], s.w[i]
}
