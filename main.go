package main

import (
	"flag"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/jonaz/gograce"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
)

var metricsCache = newMetrics()

func main() {
	var httpPort string
	var tcpPort string
	flag.StringVar(&tcpPort, "tcp-port", "8080", "port to listen tcp connections on")
	flag.StringVar(&httpPort, "http-port", "8081", "port to listen to http /metrics on")
	flag.Parse()
	ln, err := net.Listen("tcp", ":"+tcpPort)
	if err != nil {
		logrus.Errorf("cannot start socket listener on port %s: %s", tcpPort, err)
		return
		// handle error
	}
	srv, shutdown := gograce.NewServerWithTimeout(10 * time.Second)
	srv.Handler = http.DefaultServeMux
	srv.Addr = ":" + httpPort
	go func() {
		logrus.Error(srv.ListenAndServe())
	}()
	http.Handle("/metrics", promhttp.Handler())
	var wg sync.WaitGroup

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				logrus.Error(err)
				return
			}
			wg.Add(1)
			go handleConnection(&wg, conn)
		}
	}()

	<-shutdown
	ln.Close()
	logrus.Info("shutdown asfd")

	wg.Wait()
}

func handleConnection(wg *sync.WaitGroup, c net.Conn) {
	logrus.Debugf("Serving %s", c.RemoteAddr().String())
	defer wg.Done()
	defer c.Close()

	err := c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		logrus.Error(err)
		return
	}

	var parser expfmt.TextParser
	inFamilies, err := parser.TextToMetricFamilies(c)
	if err != nil {
		logrus.Error(err)
		return
	}

	for _, fam := range inFamilies {
		for _, met := range fam.GetMetric() {
			labels := make(prometheus.Labels)
			for _, v := range met.GetLabel() {
				labels[v.GetName()] = v.GetValue()
			}
			switch fam.GetType() {
			case dto.MetricType_COUNTER:

				if m := metricsCache.GetCounter(fam.GetName(), labels); m != nil {
					m.With(labels).Add(met.GetCounter().GetValue())
				}
			case dto.MetricType_SUMMARY:
				if m := metricsCache.GetSummary(fam.GetName(), labels); m != nil {
					m.With(labels).Observe(met.GetSummary().GetSampleSum())
				}
			}
		}
	}
}
