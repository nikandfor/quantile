// This is my implementation of quantile inspired by github.com/beorn7/perks/quantile,
// consulting paper
// A Fast Algorithm for Approximate Quantiles in High Speed Data Streams
// by
// Qi Zhang and Wei Wang
// University of North Carolina, Chapel Hill
// Department of Computer Science
// Chapel Hill,
// 2007.
//
// It is not mathematically identical to any of the papers or implementations but still close to them.

package quantile

import (
	"fmt"
	"io"
	"math"
	"sort"
)

type (
	Stream struct {
		v        []Sample
		i, b     int
		total    int
		sum, add float64
		e        float64
		sorted   bool
	}

	Sample struct {
		Value, Weight float64
	}
)

const oneWeight = 1

func New(e float64) *Stream {
	b := size(e)

	return &Stream{
		v: make([]Sample, b*2),
		b: b,
		e: e,
	}
}

func (s *Stream) Query(q float64) float64 {
	if !s.sorted && s.total < s.b*2 {
		s.compress()
	}

	e := s.e
	if e > 0.01 {
		e = 0.01
	}

	n := s.b
	if s.total < s.b {
		n = s.total
	}

	if n == 0 {
		return 0
	}

	t := q*s.sum + e

	var cum float64
	for _, e := range s.v[:n] {
		cum += e.Weight
		if cum > t {
			return e.Value
		}
	}

	return s.v[n-1].Value
}

func (s *Stream) Insert(v float64) {
	s.v[s.i] = Sample{
		Value:  v,
		Weight: oneWeight,
	}

	s.i++
	s.total++
	s.add += oneWeight
	s.sorted = false

	if s.i == len(s.v) {
		s.compress()
	}
}

func (s *Stream) Samples() []Sample {
	if !s.sorted {
		s.compress()
	}

	return s.v[:s.i]
}

func (s *Stream) Merge(ss []Sample) {
	s.v = append(s.v[:s.i], ss...)

	s.i += len(ss)
	s.total += len(ss)
	s.sorted = false

	for _, ss1 := range ss {
		s.add += ss1.Weight
	}

	s.compress()

	s.v = s.v[:cap(s.v)]
}

func (s *Stream) compress() {
	n := s.i
	if s.total < s.i {
		n = s.total
	}
	sort.Slice(s.v[:n], func(i, j int) bool {
		return s.v[i].Value < s.v[j].Value
	})

	if n <= s.b {
		s.sum += s.add
		s.add = 0
		s.i = n
		s.sorted = true

		return
	}

	var t, cum float64
	totsum := s.sum + s.add
	step := totsum / float64(s.b)

	//	fmt.Fprintf(os.Stderr, "compress  sum %.4f  step %.4f  total %v   i %v  : %v\n", totsum, step, s.total, s.i, s.v[:s.i])

	r := 0
	for w := 0; w < s.b && w < s.i; w++ {
		t += step

		//		or := r

		var e Sample
		for cum < t && r < s.i {
			if abs(cum-t) < abs(cum+s.v[r].Weight-t) {
				break
			}

			e.Value += s.v[r].Value * s.v[r].Weight
			e.Weight += s.v[r].Weight

			cum += s.v[r].Weight

			r++
		}

		if e.Weight != 0 {
			e.Value /= e.Weight
		}

		//		fmt.Fprintf(os.Stderr, "step %10.4f  cum %10.4f (q %.4f)  el_w %4d: %8.4f  el_r %4d: %8.4f\n", t, cum, cum/totsum, w, e, r, s.v[or:r])

		s.v[w] = e
	}

	//	if abs(cum-totsum) > 0.001 {
	//		fmt.Fprintf(os.Stderr, "sum MISMATCH: %v <- %v\n", cum, totsum)
	//	}

	s.add = 0
	s.sum = cum

	s.i = s.b
	s.sorted = true
}

func (s *Stream) Size() int {
	return s.b
}

func (s *Stream) BufLen() int {
	return len(s.v)
}

func (s *Stream) dump(w io.Writer) {
	fmt.Fprintf(w, "Stream %d/%d  sum %v\n", s.i-s.b, s.b, s.sum)

	var cum float64
	for i, v := range s.v[:s.b] {
		cum += v.Weight
		fmt.Fprintf(w, "  %4d: %.4f weight %.4f  q %.4f\n", i, v.Value, v.Weight, cum/s.sum)
	}
}

func size(e float64) (n int) {
	n = int(math.Ceil(math.Log1p(e*200000) / e))

	return
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}

	return a
}
