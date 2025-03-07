package quantile

// Based on clickhouse implementation by Alexei Borzenkov (https://github.com/snaury).
//
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

		Invariant Invariant
		Decay     float32

		ElementsReduced   float32
		Compressions      int
		BruteCompressions int
	}

	tdigest struct {
		v []float64
		w []float32

		i    int
		size int

		j int // for multi-streams query

		sorted bool
	}

	sorter tdigest

	Invariant interface {
		Inv(q float32) float32
	}

	HighBias     float32
	LowBias      float32
	ExtremesBias float32

	InvariantFunc func(q float32) float32
)

func NewTDHighBiased(eps float32, size int) *TDigest {
	return NewTD(HighBias(eps), size)
}

func NewTDLowBiased(eps float32, size int) *TDigest {
	return NewTD(LowBias(eps), size)
}

func NewTDExtremesBiased(eps float32, size int) *TDigest {
	return NewTD(ExtremesBias(eps), size)
}

// NewTDigest creates a new tdigest stream.
// 512 is a good size to start with.
func NewTD(inv Invariant, size int) *TDigest {
	if size%2 != 0 {
		panic(size)
	}

	return &TDigest{
		tdigest: tdigest{
			v: make([]float64, size),
			w: make([]float32, size),

			size: size,
		},

		Invariant: inv,
		Decay:     1,
	}
}

func (s *TDigest) Reset() {
	s.i = 0
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

	var total, sum, prev float64

	for _, w := range s.w[:s.i] {
		total += float64(w)
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

	target := res[qi] * total
	prevV := s.v[0]

	for i := 0; i < s.i; {
		cur := sum + 0.5*float64(s.w[i])

		//	log.Printf("query %.2f  i %2d  cur %.3f / %.3f  v %.2f  w %.1f", res[qi], i, cur, target, s.v[i], s.w[i])

		if cur >= target {
			l := prev
			r := cur

			switch {
			case target <= l:
				res[qi] = prevV
			case target >= r:
				res[qi] = s.v[i]
			default:
				res[qi] = s.interpolate(target, l, r, prevV, s.v[i])
			}

			qi++

			if qi == len(qs) {
				break
			}

			target = res[qi] * total

			continue
		}

		sum += float64(s.w[i])

		prev = cur
		prevV = s.v[i]

		i++
	}

	for qi < len(qs) {
		res[qi] = s.v[s.i-1]
		qi++
	}
}

func (s *TDigest) interpolate(x, x1, x2 float64, y1, y2 float64) float64 {
	k := float64(x-x1) / float64(x2-x1)

	return y1*(1-k) + y2*k
}

func (s *TDigest) Insert(v float64) {
	s.InsertWeighted(v, 1)
}

func (s *TDigest) InsertWeighted(v float64, w float32) {
	if math.IsNaN(v) {
		return
	}

	if s.i >= s.size {
		s.compress()
	}

	s.v[s.i] = v
	s.w[s.i] = w

	s.sorted = s.i == 0 || s.sorted && v > s.v[s.i-1]
	s.i++
}

func (s *TDigest) Merge(s1 *TDigest) {
	s.MergeWeighted(s1, 1, 1)
}

func (s *TDigest) MergeWeighted(s1 *TDigest, w0, w1 float32) {
	s.mergeWeighted(s1, w0, w1)
}

func (s *TDigest) mergeWeighted(s1 *TDigest, w0, w1 float32) {
	s.AdjustWeights(w0)

	i := s.i

	if w0 <= 0 {
		i = 0
	}

	s.v = append(s.v[:i], s1.v[:s1.i]...)
	s.w = append(s.w[:i], s1.w[:s1.i]...)

	s.i = i + s1.i

	s.adjustWeights(w1, i, s.i)
}

func (s *TDigest) AdjustWeights(multiply float32) {
	s.adjustWeights(multiply, 0, s.i)
}

func (s *TDigest) adjustWeights(multiply float32, st, end int) {
	if multiply == 1 {
		return
	}

	_ = s.w[st:end]

	for i := st; i < end; i++ {
		s.w[i] *= multiply
	}
}

func (s *TDigest) compress() {
	if !s.sorted {
		s.sort()
	}

	//	log.Printf("compress\nv: %5.2f\nw: %5.2f\n", s.v[:s.i], s.w[:s.i])

	s.compress0()
	s.Compressions++

	const a = 0.97

	s.ElementsReduced = s.ElementsReduced*a + float32(s.size-s.i)*(1-a)

	if s.i < s.size {
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

	l, r := 0, 1

	var sum float32

	//	log.Printf("compress\nv: %5.2f\nw: %5.2f\n", s.v[:s.i], s.w[:s.i])
	//	log.Printf("total %d  %v  eps %v", s.i, total, s.eps)

	for r < s.i {
		ql := (sum + s.w[l]*0.5) / total
		qr := (sum + s.w[l] + s.w[r]*0.5) / total

		k := min(s.Invariant.Inv(ql), s.Invariant.Inv(qr)) * total

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
	sort.Sort((*sorter)(&s.tdigest))

	s.sorted = true
}

func (s *sorter) Len() int           { return s.i }
func (s *sorter) Less(i, j int) bool { return s.v[i] < s.v[j] }
func (s *sorter) Swap(i, j int) {
	s.v[i], s.v[j] = s.v[j], s.v[i]
	s.w[i], s.w[j] = s.w[j], s.w[i]
}

func (eps HighBias) Inv(q float32) float32 {
	return (1 - q*q) * float32(eps)
}

func (eps LowBias) Inv(q float32) float32 {
	q = 1 - q

	return (1 - q*q) * float32(eps)
}

func (eps ExtremesBias) Inv(q float32) float32 {
	return 4 * q * (1 - q) * float32(eps)
}

func (f InvariantFunc) Inv(q float32) float32 {
	return f(q)
}
