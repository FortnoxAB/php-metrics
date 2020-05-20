package main

import (
	"crypto/sha256"
	"sort"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type metrics struct {
	list map[[sha256.Size]byte]prometheus.Collector
	sync.RWMutex
}

func newMetrics() *metrics {
	return &metrics{
		list: make(map[[sha256.Size]byte]prometheus.Collector),
	}
}

func (m *metrics) Get(s string, labels prometheus.Labels) (prometheus.Collector, [sha256.Size]byte, []string) {
	labelNames := make([]string, len(labels))
	i := 0
	for k := range labels {
		labelNames[i] = k
		i++
	}

	sort.Strings(labelNames)
	hash := sha256.Sum256([]byte(s + strings.Join(labelNames, "")))

	m.RLock()
	defer m.RUnlock()
	if val, ok := m.list[hash]; ok {
		return val, hash, labelNames
	}
	return nil, hash, labelNames
}

func (m *metrics) GetCounter(s string, labels prometheus.Labels) *prometheus.CounterVec {
	val, hash, labelNames := m.Get(s, labels)

	if val, ok := val.(*prometheus.CounterVec); ok {
		return val
	}

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: s,
		},
		labelNames,
	)
	m.list[hash] = counter

	err := prometheus.Register(counter)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	return counter
}

func (m *metrics) GetSummary(s string, labels prometheus.Labels) *prometheus.SummaryVec {
	val, hash, labelNames := m.Get(s, labels)

	if val, ok := val.(*prometheus.SummaryVec); ok {
		return val
	}

	counter := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: s,
		},
		labelNames,
	)
	m.list[hash] = counter

	err := prometheus.Register(counter)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	return counter
}
