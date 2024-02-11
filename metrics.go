package main

type Metric struct {
	Count int
	Sum   float64
}

func (m *Metric) Add(v float64) {
	m.Count++
	m.Sum += v
}

func (m *Metric) Avg() float64 {
	if m.Count == 0 {
		return 0
	}
	return m.Sum / float64(m.Count)
}
