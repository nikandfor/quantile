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
//
// I tried to optimize it as much as I can.

package quantile

import (
	"fmt"
	"io"
	"math"
	"sort"
)

type (
	Stream struct {
		v        []sample
		i, b     int
		total    int
		sum, add float64
		e        float64
		sorted   bool
	}

	sample struct {
		value, width float64
	}
)

func New(e float64) *Stream {
	b := size(e)

	return &Stream{
		v: make([]sample, b*2),
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

	t := q*s.sum + e

	var cum float64
	for _, e := range s.v[:s.b] {
		cum += e.width
		if cum >= t {
			return e.value
		}
	}

	return s.v[s.b-1].value
}

func (s *Stream) Insert(v float64) {
	s.v[s.i] = sample{
		value: v,
		width: 1,
	}
	s.i++
	s.total++
	s.add++
	s.sorted = false

	if s.i == len(s.v) {
		s.compress()
	}
}

func (s *Stream) compress() {
	sort.Slice(s.v[:s.i], func(i, j int) bool {
		return s.v[i].value < s.v[j].value
	})

	var t, cum, div float64
	totsum := s.sum + s.add
	step := totsum / float64(s.b)

	if step > 3 {
		div = float64(s.b) / totsum
	}

	//	fmt.Fprintf(os.Stderr, "compress  sum %.4f  step %.4f  total %v   i %v  div %.5f: %v\n", totsum, step, s.total, s.i, div, s.v[:s.i])

	r := 0
	for w := 0; w < s.b && w < s.i; w++ {
		t += step

		//		or := r

		var e sample
		for cum < t && r < s.i {
			if abs(cum-t) < abs(cum+s.v[r].width-t) {
				break
			}

			e.value += s.v[r].value * s.v[r].width
			e.width += s.v[r].width

			cum += s.v[r].width

			r++
		}

		if e.width != 0 {
			e.value /= e.width
		}

		if div != 0 {
			e.width *= div
		}

		//		fmt.Fprintf(os.Stderr, "step %10.4f  cum %10.4f (q %.4f)  el_w %4d: %8.4f  el_r %4d: %8.4f\n", t, cum, cum/totsum, w, e, r, s.v[or:r])

		s.v[w] = e
	}

	//	if abs(cum-totsum) > 0.001 {
	//		fmt.Printf("sum MISMATCH: %v <- %v\n", cum, totsum)
	//	}

	s.add = 0
	s.sum = cum
	if div != 0 {
		s.sum *= div
	}

	s.i = s.b
	s.sorted = true
}

func (s *Stream) Size() int {
	return s.b
}

func (s *Stream) dump(w io.Writer) {
	fmt.Fprintf(w, "Stream %d/%d  sum %v\n", s.i-s.b, s.b, s.sum)

	var cum float64
	for i, v := range s.v[:s.b] {
		cum += v.width
		fmt.Fprintf(w, "  %4d: %.4f width %.4f  q %.4f\n", i, v.value, v.width, cum/s.sum)
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
