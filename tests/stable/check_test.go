package stable

import (
	"flag"
	"fmt"
	"testing"

	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/google/go-cmp/cmp"
	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp/cmpopts" )

var promText string
var stableYaml string

var skipStableMetrics = []string{
	"kube_job_owner",
}

var unCaughtStableMetrics = []string{
	"kube_cronjob_spec_starting_deadline_seconds",
	"kube_cronjob_status_last_schedule_time",
	"kube_ingress_tls",
	"kube_job_complete",
	"kube_job_failed",
	"kube_job_spec_active_deadline_seconds",
	"kube_job_status_completion_time",
	"kube_node_spec_taint",
	"kube_persistentvolume_claim_ref",
	"kube_pod_completion_time",
	"kube_pod_init_container_info",
	"kube_pod_init_container_status_ready",
	"kube_pod_init_container_status_restarts_total",
	"kube_pod_init_container_status_running",
	"kube_pod_init_container_status_terminated",
	"kube_pod_init_container_status_waiting",
	"kube_pod_spec_volumes_persistentvolumeclaims_info",
	"kube_pod_spec_volumes_persistentvolumeclaims_readonly",
	"kube_pod_status_unschedulable",
	"kube_service_spec_external_ip",
	"kube_service_status_load_balancer_ingress",
}

type Metric struct {
	Name   string `yaml:"name"`
	Help   string `yaml:"help"`
	Type   string
	Labels []string
	// Histogram type
	Buckets []float64 `yaml:"buckets,omitempty"`
}

func TestMain(m *testing.M) {
	flag.StringVar(&promText, "collectedMetricsFile", "", "input prometheus metrics text file, text format")
	flag.StringVar(&stableYaml, "stableMetricsFile", "", "expected stable metrics yaml file, yaml format")
	flag.Parse()
	m.Run()
}


func TestStableMetrics(t *testing.T) {
	mf, err := parsePromText(promText)
	fatal(err)
	collectedStableMetrics := extractStableMetrics(mf)
	printMetric(collectedStableMetrics)

	expectedStableMetrics, err := readYaml(stableYaml)
	if err != nil {
		t.Fatalf("Can't read stable metrics from file. err = %v", err)
	}

	err = compare(collectedStableMetrics, *expectedStableMetrics, skipStableMetrics)
	if err != nil {
		t.Fatalf("Stable metrics changed: err = %v", err)
	} else {
		fmt.Println("## passed")
		t.Fatalf("Passed")
	}

}


func fatal(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func printMetric(metrics []Metric) {
	 yamlData, err := yaml.Marshal(metrics)
    if err != nil {
        fmt.Printf("Error while Marshaling. %v", err)
    }
    fmt.Println("---begin YAML file---")
    fmt.Println(string(yamlData))
    fmt.Println("---end YAML file---")
}

func parsePromText(path string) (map[string]*prommodel.MetricFamily, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, err
	}
	return mf, nil
}

func getBuckets(v *prommodel.MetricFamily) []float64 {
	buckets := []float64{}
	if v.GetType() == prommodel.MetricType_HISTOGRAM {
		for _, bucket := range v.Metric[0].GetHistogram().GetBucket() {
			buckets = append(buckets, *bucket.UpperBound)
		}
	} else {
		buckets = nil
	}
	return buckets
}

func extractStableMetrics(mf map[string]*prommodel.MetricFamily) []Metric {
	metrics := []Metric{}
	for _, v := range mf {
		if !strings.Contains(*(v.Help), "[STABLE]") {
			continue
		}

		m := Metric{
			Name:    *(v.Name),
			Help:    *(v.Help),
			Type:    (v.Type).String(),
			Buckets: getBuckets(v),
		}
		labels := []string{}
		for _, y := range v.Metric[0].Label {
			labels = append(labels, y.GetName())
		}
		m.Labels = labels
		metrics = append(metrics, m)
	}
	return metrics
}

func readYaml(filename string) (*[]Metric, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	c := &[]Metric{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("error %q: %w", filename, err)
	}
	return c, err
}

func compare(collectedStableMetrics []Metric, expectedStableMetrics []Metric, skipStableMetrics []string) error {
	var ok bool
	name2Metric := map[string]Metric{}
	skipMap := map[string]int{}
	expectedMetricMap := map[string]int{}
	for _, v := range collectedStableMetrics {
		name2Metric[v.Name] = v
	}
	for _, v := range skipStableMetrics {
		skipMap[v] = 1
	}

	for _, v := range expectedStableMetrics {
		expectedMetricMap[v.Name] = 1
		if _, ok = skipMap[v.Name]; ok {
			fmt.Printf("## skip, metric %s is in skip list\n", v.Name)
			continue
		}
		var val Metric
		if val, ok = name2Metric[v.Name]; !ok {
			return fmt.Errorf("Not found stable metric %s \n", v.Name)
		}
		if diff := cmp.Diff(v, val, cmpopts.IgnoreFields(Metric{}, "Help")); diff != "" {
			return fmt.Errorf("Stable metric %s mismatch (-want +got):\n%s", v.Name, diff)
		}
	}
	for _, v := range collectedStableMetrics {
		if _, ok = skipMap[v.Name]; ok {
			fmt.Printf("## skip, metric %s is in skip list\n", v.Name)
			continue
		}
		if _, ok = expectedMetricMap[v.Name]; !ok {
			return fmt.Errorf("Detected new stable metric %s which isn't in testdata ", v.Name)
		}
	}
	return nil
}
