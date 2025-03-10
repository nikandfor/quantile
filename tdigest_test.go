package quantile

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

var (
	tdbenchW = []int{128, 512, 2048}
	tdbenchE = []float32{0.1, 0.03, 0.01}
)

func TestTDigest(tb *testing.T) {
	const W, Eps = 16, 0.51

	e := NewExact()
	s := NewTDExtremesBiased(0.1, W)

	assertEqual(tb, e, s, 0, Eps)

	e.Insert(5)
	s.Insert(5)

	assertEqual(tb, e, s, 0, 0)
	assertEqual(tb, e, s, 1, 0)
	assertEqual(tb, e, s, 0.5, 0)

	for i := range W + 1 {
		v := float64(i)
		e.Insert(v)
		s.Insert(v)
	}

	assertEqual(tb, e, s, 0, Eps)
	assertEqual(tb, e, s, 1, Eps)
	assertEqual(tb, e, s, 0.5, Eps)

	if tb.Failed() {
		tb.Logf("dump\n%v", s.dump())
	}
}

func TestTDigest10(tb *testing.T) {
	e := NewExact()
	s := NewTDExtremesBiased(0.01, 16)

	for _, v := range []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1} {
		e.Insert(v)
		s.Insert(v)
	}

	qs := []float64{0., 0.01, 0.1, 0.5, 0.9, 0.99, 1}

	for _, q := range qs {
		assertEqual(tb, e, s, q, 0.02)
	}
}

func TestCompareUniformTD(tb *testing.T) {
	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	s := NewTDExtremesBiased(0.01, 1024)

	testCompare(tb, r.Float64, s)
}

func TestCompareNormalTD(tb *testing.T) {
	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	s := NewTDExtremesBiased(0.01, 1024)

	testCompare(tb, r.NormFloat64, s)
}

func BenchmarkInsertTD(tb *testing.B) {
	for _, W := range tdbenchW {
		for _, E := range tdbenchE {
			tb.Run(fmt.Sprintf("W%d_E%.3f", W, E), func(tb *testing.B) {
				s := NewTDExtremesBiased(E, W)

				benchInsert(tb, s)

				tb.Logf("stats: compressions %v / %v,  average reduction %v / %v", s.Compressions, s.BruteCompressions, s.ElementsReduced, s.size)

				if tb.Failed() {
					tb.Logf("dump\n%v", s.dump())
				}
			})
		}
	}
}

func BenchmarkQueryTD(tb *testing.B) {
	for _, W := range tdbenchW {
		tb.Run(fmt.Sprintf("W%d", W), func(tb *testing.B) {
			s := NewTDExtremesBiased(0.01, W)

			benchQuery(tb, s)

			if tb.Failed() {
				tb.Logf("dump\n%v", s.dump())
			}
		})
	}
}

func assertEqual(tb testing.TB, e, s Stream, q, eps float64) {
	tb.Helper()

	ext := e.Query(q)
	v := s.Query(q)

	if math.Abs(v-ext) > eps {
		tb.Errorf("q %.2f => %7.3f  wanted %7.3f  diff %7.4f", q, v, ext, v-ext)
	}
}
