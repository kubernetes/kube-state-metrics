package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/kube-state-metrics/pkg/options"
)

func TestFiltererdGatherer(t *testing.T) {
	r := prometheus.NewRegistry()
	c1 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test1",
			Help: "test1 help",
		},
	)
	c2 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test2",
			Help: "test2 help",
		},
	)
	c1.Inc()
	c1.Inc()
	c2.Inc()
	r.MustRegister(c1)
	r.MustRegister(c2)

	res, err := FilteredGatherer(r, nil, nil).Gather()
	if err != nil {
		t.Fatal(err)
	}

	found1 := false
	found2 := false
	for _, mf := range res {
		if *mf.Name == "test1" {
			found1 = true
		}
		if *mf.Name == "test2" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Fatal("No results expected to be filtered, but results were filtered.")
	}
}

func TestFiltererdGathererWhitelist(t *testing.T) {
	r := prometheus.NewRegistry()
	c1 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test1",
			Help: "test1 help",
		},
	)
	c2 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test2",
			Help: "test2 help",
		},
	)
	c1.Inc()
	c1.Inc()
	c2.Inc()
	r.MustRegister(c1)
	r.MustRegister(c2)

	whitelist := options.MetricSet{}
	whitelist.Set("test1")

	res, err := FilteredGatherer(r, whitelist, nil).Gather()
	if err != nil {
		t.Fatal(err)
	}

	found1 := false
	found2 := false
	for _, mf := range res {
		if *mf.Name == "test1" {
			found1 = true
		}
		if *mf.Name == "test2" {
			found2 = true
		}
	}

	if !found1 || found2 {
		t.Fatalf("Expected `test2` to be filtered and `test1` not. `test1`: %t ; `test2`: %t.", found1, found2)
	}
}

func TestFiltererdGathererBlacklist(t *testing.T) {
	r := prometheus.NewRegistry()
	c1 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test1",
			Help: "test1 help",
		},
	)
	c2 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test2",
			Help: "test2 help",
		},
	)
	c1.Inc()
	c1.Inc()
	c2.Inc()
	r.MustRegister(c1)
	r.MustRegister(c2)

	blacklist := options.MetricSet{}
	blacklist.Set("test1")

	res, err := FilteredGatherer(r, nil, blacklist).Gather()
	if err != nil {
		t.Fatal(err)
	}

	found1 := false
	found2 := false
	for _, mf := range res {
		if *mf.Name == "test1" {
			found1 = true
		}
		if *mf.Name == "test2" {
			found2 = true
		}
	}

	if found1 || !found2 {
		t.Fatalf("Expected `test1` to be filtered and `test2` not. `test1`: %t ; `test2`: %t.", found1, found2)
	}
}
