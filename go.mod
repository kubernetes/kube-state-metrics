module k8s.io/kube-state-metrics

require (
	github.com/brancz/gojsontoyaml v0.0.0-20190425155809-e8bd32d46b3d
	github.com/campoy/embedmd v1.0.0
	github.com/dgryski/go-jump v0.0.0-20170409065014-e1f439676b57
	github.com/google/go-jsonnet v0.14.0
	github.com/jsonnet-bundler/jsonnet-bundler v0.1.1-0.20190930114713-10e24cb86976
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/robfig/cron/v3 v3.0.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/tools v0.0.0-20210106214847-113979e3529a
	k8s.io/api v0.17.5
	k8s.io/apimachinery v0.17.5
	k8s.io/autoscaler/vertical-pod-autoscaler v0.0.0-20191115143342-4cf961056038
	k8s.io/client-go v0.17.5
	k8s.io/klog v1.0.0
)

require (
	cloud.google.com/go v0.56.0 // indirect
	github.com/Azure/go-autorest/autorest v0.10.0 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.3 // indirect
	github.com/Azure/go-autorest/autorest/date v0.2.0 // indirect
	github.com/Azure/go-autorest/logger v0.1.0 // indirect
	github.com/Azure/go-autorest/tracing v0.5.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/evanphx/json-patch v4.2.0+incompatible // indirect
	github.com/fatih/color v1.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v0.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/gophercloud/gophercloud v0.10.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.11 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.21.0 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
	k8s.io/klog/v2 v2.0.0 // indirect
	k8s.io/kube-openapi v0.0.0-20200316234421-82d701f24f9d // indirect
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace (
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go v0.0.0-20210802184156-9742bd7fca1c
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20200609090129-a6600f564e3c
)

go 1.17
