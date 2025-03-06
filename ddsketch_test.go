package quantile

import (
	"math/rand/v2"
	"testing"
)

func TestDDKeys(tb *testing.T) {
	s := NewDDLog(0.01)

	tb.Logf("min %g  max %g  gamma %g", s.minPossible, s.maxPossible, s.gamma)

	tb.Logf("%9.4v -> key %3d", 0.5, s.key(0.5))

	for i := 0; i < 10; i++ {
		v := s.unkey(i)
		j := s.key(v)

		tb.Logf("key %3d -> %9.4v -> %3d", i, v, j)
	}

	for v := 0.01; v < 1; v *= 1.3 {
		i := s.key(v)
		b := s.unkey(i)

		tb.Logf("key %9.4f -> %3d -> %9.4f  diff %9.5f", v, i, b, b-v)
	}

	for v := 1.; v < 20; v++ {
		i := s.key(v)
		b := s.unkey(i)

		tb.Logf("key %9.4f -> %3d -> %9.4f  diff %9.5f", v, i, b, b-v)
	}
}

func TestDD(tb *testing.T) {
	const W, Acc = 16, 0.01

	e := NewExact()
	s := NewDDLog(Acc)

	assertEqual(tb, e, s, 0, 0)

	e.Insert(5)
	s.Insert(5)

	assertEqual(tb, e, s, 0, 0.06)
	assertEqual(tb, e, s, 1, 0.06)
	assertEqual(tb, e, s, 0.5, 0.06)

	if tb.Failed() {
		tb.Logf("dump\n%v", s.dump())
		return
	}

	for i := range W + 1 {
		v := float64(i)
		e.Insert(v)
		s.Insert(v)
	}

	eps := 0.4

	assertEqual(tb, e, s, 0, eps)
	assertEqual(tb, e, s, 1, eps)
	assertEqual(tb, e, s, 0.5, eps)

	if tb.Failed() {
		tb.Logf("dump\n%v", s.dump())
	}
}

func TestCompareUniformDD(tb *testing.T) {
	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	s := NewDDLog(0.01)

	testCompare(tb, r.Float64, s)
}

func TestCompareNormalDD(tb *testing.T) {
	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	s := NewDDLog(0.01)

	testCompare(tb, r.NormFloat64, s)
}
