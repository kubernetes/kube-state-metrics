module k8s.io/kube-state-metrics/v2

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/brancz/gojsontoyaml v0.0.0-20201216083616-202f76bf8c1f
	github.com/campoy/embedmd v1.0.0
	github.com/dgryski/go-jump v0.0.0-20170409065014-e1f439676b57
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-jsonnet v0.17.0
	github.com/jsonnet-bundler/jsonnet-bundler v0.4.1-0.20200708074244-ada055a225fa
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/oklog/run v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.19.0
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20201208171446-5f87f3452ae9 // indirect
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777 // indirect
	golang.org/x/oauth2 v0.0.0-20210210192628-66670185b0cd // indirect
	golang.org/x/text v0.3.5 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	golang.org/x/tools v0.1.0
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/autoscaler/vertical-pod-autoscaler v0.9.2
	k8s.io/client-go v0.20.4
	k8s.io/klog/v2 v2.8.0
)

replace (
	k8s.io/api v0.18.3 => k8s.io/api v0.20.4
	k8s.io/apimachinery v0.18.3 => k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.18.3 => k8s.io/client-go v0.20.4
)

go 1.16
