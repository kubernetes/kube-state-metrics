package stable

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp/cmpopts"
)

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
