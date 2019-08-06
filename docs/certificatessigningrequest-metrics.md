# CertificateSigningRequest Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_certificatesigningrequest_created| Gauge | `certificatesigningrequest`=&lt;certificatesigningrequest-name&gt;| STABLE |
| kube_certificatesigningrequest_condition | Gauge | `certificatesigningrequest`=&lt;certificatesigningrequest-name&gt; <br> `condition`=&lt;approved\|denied&gt; | STABLE |
| kube_certificatesigningrequest_labels | Gauge | `certificatesigningrequest`=&lt;certificatesigningrequest-name&gt;| STABLE |
| kube_certificatesigningrequest_cert_length | Gauge | `certificatesigningrequest`=&lt;certificatesigningrequest-name&gt;| STABLE |
