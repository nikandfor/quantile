package quantile

import (
	"math"
	"math/rand/v2"
	"sort"
	"testing"
)

type Stream interface {
	Insert(v float64)
	Query(q float64) float64
}

func testCompare(tb *testing.T, rand func() float64, ss ...Stream) {
	const N = 100000

	ext := NewExact()

	for i := 0; i < N; i++ {
		v := rand()

		ext.Insert(v)

		for _, s := range ss {
			s.Insert(v)
		}
	}

	qs := []float64{1, 0, 0.5}

	for q := 0.01; q < 0.5; q += q * 2 / 3 {
		qs = append(qs, q, 1-q)
	}

	sort.Float64s(qs)

	vv := make([]float64, len(ss))
	dd := make([]float64, len(ss))
	ds := make([]float64, len(ss))
	cnt := 0

	for _, q := range qs {
		e := ext.Query(q)

		for j, s := range ss {
			v := s.Query(q)

			vv[j] = v
			dd[j] = e - v
			ds[j] += math.Abs(dd[j] * dd[j])
		}

		tb.Logf("quantile %7.3f: exact %9.4f  v %9.4f  d %9.4f", q, e, vv, dd)

		cnt++
	}

	a := ext.Query(1)

	for j, d := range ds {
		dv := math.Sqrt(d / float64(cnt))

		tb.Logf("stddev abs %7.4f (rel %7.4f)  %T", dv, dv/a, ss[j])
	}
}

func benchInsert(tb *testing.B, s Stream) {
	tb.ReportAllocs()

	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	vs := make([]float64, tb.N)

	for i := 0; i < tb.N; i++ {
		vs[i] = r.Float64()
	}

	tb.ResetTimer()

	for i := 0; i < tb.N; i++ {
		s.Insert(vs[i])
	}
}

func benchQuery(tb *testing.B, s Stream) {
	tb.ReportAllocs()

	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	for range int(1e6) {
		v := r.Float64()
		s.Insert(v)
	}

	vs := make([]float64, tb.N)

	for i := 0; i < tb.N; i++ {
		vs[i] = r.Float64()
	}

	tb.ResetTimer()

	for i := 0; i < tb.N; i++ {
		_ = s.Query(vs[i])
	}
}
