{
  local ksm = self,
  name:: error 'must set namespace',
  namespace:: error 'must set namespace',
  version:: error 'must set version',
  image:: error 'must set image',

  commonLabels:: {
    'app.kubernetes.io/name': 'kube-state-metrics',
    'app.kubernetes.io/version': ksm.version,
  },

  podLabels:: {
    [labelName]: ksm.commonLabels[labelName]
    for labelName in std.objectFields(ksm.commonLabels)
    if !std.setMember(labelName, ['app.kubernetes.io/version'])
  },

  clusterRoleBinding:
    {
      apiVersion: 'rbac.authorization.k8s.io/v1',
      kind: 'ClusterRoleBinding',
      metadata: {
        name: ksm.name,
        labels: ksm.commonLabels,
      },
      roleRef: {
        apiGroup: 'rbac.authorization.k8s.io',
        kind: 'ClusterRole',
        name: ksm.name,
      },
      subjects: [{
        kind: 'ServiceAccount',
        name: ksm.name,
        namespace: ksm.namespace,
      }],
    },

  clusterRole:
    local rules = [
      {
        apiGroups: [''],
        resources: [
          'configmaps',
          'secrets',
          'nodes',
          'pods',
          'services',
          'resourcequotas',
          'replicationcontrollers',
          'limitranges',
          'persistentvolumeclaims',
          'persistentvolumes',
          'namespaces',
          'endpoints',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['apps'],
        resources: [
          'statefulsets',
          'daemonsets',
          'deployments',
          'replicasets',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['batch'],
        resources: [
          'cronjobs',
          'jobs',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['autoscaling'],
        resources: [
          'horizontalpodautoscalers',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['authentication.k8s.io'],
        resources: [
          'tokenreviews',
        ],
        verbs: ['create'],
      },
      {
        apiGroups: ['authorization.k8s.io'],
        resources: [
          'subjectaccessreviews',
        ],
        verbs: ['create'],
      },
      {
        apiGroups: ['policy'],
        resources: [
          'poddisruptionbudgets',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['certificates.k8s.io'],
        resources: [
          'certificatesigningrequests',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['storage.k8s.io'],
        resources: [
          'storageclasses',
          'volumeattachments',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['admissionregistration.k8s.io'],
        resources: [
          'mutatingwebhookconfigurations',
          'validatingwebhookconfigurations',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['networking.k8s.io'],
        resources: [
          'networkpolicies',
          'ingresses',
        ],
        verbs: ['list', 'watch'],
      },
      {
        apiGroups: ['coordination.k8s.io'],
        resources: [
          'leases',
        ],
        verbs: ['list', 'watch'],
      },
    ];

    {
      apiVersion: 'rbac.authorization.k8s.io/v1',
      kind: 'ClusterRole',
      metadata: {
        name: ksm.name,
        labels: ksm.commonLabels,
      },
      rules: rules,
    },
  deployment:
    local c = {
      name: 'kube-state-metrics',
      image: ksm.image,
      ports: [
        { name: 'http-metrics', containerPort: 8080 },
        { name: 'telemetry', containerPort: 8081 },
      ],
      securityContext: { runAsUser: 65534 },
      livenessProbe: { timeoutSeconds: 5, initialDelaySeconds: 5, httpGet: {
        port: 8080,
        path: '/healthz',
      } },
      readinessProbe: { timeoutSeconds: 5, initialDelaySeconds: 5, httpGet: {
        port: 8081,
        path: '/',
      } },
    };

    {
      apiVersion: 'apps/v1',
      kind: 'Deployment',
      metadata: {
        name: ksm.name,
        namespace: ksm.namespace,
        labels: ksm.commonLabels,
      },
      spec: {
        replicas: 1,
        selector: { matchLabels: ksm.podLabels },
        template: {
          metadata: {
            labels: ksm.commonLabels,
          },
          spec: {
            containers: [c],
            serviceAccountName: ksm.serviceAccount.metadata.name,
            nodeSelector: { 'kubernetes.io/os': 'linux' },
          },
        },
      },
    },

  serviceAccount:
    {
      apiVersion: 'v1',
      kind: 'ServiceAccount',
      metadata: {
        name: ksm.name,
        namespace: ksm.namespace,
        labels: ksm.commonLabels,
      },
    },

  service:
    {
      apiVersion: 'v1',
      kind: 'Service',
      metadata: {
        name: ksm.name,
        namespace: ksm.namespace,
        labels: ksm.commonLabels,
      },
      spec: {
        clusterIP: 'None',
        selector: ksm.podLabels,
        ports: [
          { name: 'http-metrics', port: 8080, targetPort: 'http-metrics' },
          { name: 'telemetry', port: 8081, targetPort: 'telemetry' },
        ],
      },
    },

  autosharding:: {
    role:
      {
        apiVersion: 'rbac.authorization.k8s.io/v1',
        kind: 'Role',
        metadata: {
          name: ksm.name,
          namespace: ksm.namespace,
          labels: ksm.commonLabels,
        },
        rules: [{
          apiGroups: [''],
          resources: ['pods'],
          verbs: ['get'],
        }, {
          apiGroups: ['apps'],
          resourceNames: ['kube-state-metrics'],
          resources: ['statefulsets'],
          verbs: ['get'],
        }],
      },

    roleBinding:
      {
        apiVersion: 'rbac.authorization.k8s.io/v1',
        kind: 'RoleBinding',
        metadata: {
          name: ksm.name,
          namespace: ksm.namespace,
          labels: ksm.commonLabels,
        },
        roleRef: {
          apiGroup: 'rbac.authorization.k8s.io',
          kind: 'Role',
          name: 'kube-state-metrics',
        },
        subjects: [{
          kind: 'ServiceAccount',
          name: ksm.serviceAccount.metadata.name,
        }],
      },

    statefulset:
      // extending the default container from above
      local c = ksm.deployment.spec.template.spec.containers[0] {
        args: [
          '--pod=$(POD_NAME)',
          '--pod-namespace=$(POD_NAMESPACE)',
        ],
        env: [
          { name: 'POD_NAME', valueFrom: { fieldRef: { fieldPath: 'metadata.name' } } },
          { name: 'POD_NAMESPACE', valueFrom: { fieldRef: { fieldPath: 'metadata.namespace' } } },
        ],
      };

      {
        apiVersion: 'apps/v1',
        kind: 'StatefulSet',
        metadata: {
          name: ksm.name,
          namespace: ksm.namespace,
          labels: ksm.commonLabels,
        },
        spec: {
          replicas: 2,
          selector: { matchLabels: ksm.podLabels },
          serviceName: ksm.service.metadata.name,
          template: {
            metadata: {
              labels: ksm.commonLabels,
            },
            spec: {
              containers: [c],
              serviceAccountName: ksm.serviceAccount.metadata.name,
              nodeSelector: { 'kubernetes.io/os': 'linux' },
            },
          },
        },
      },
  } + {
    service: ksm.service,
    serviceAccount: ksm.serviceAccount,
    clusterRole: ksm.clusterRole,
    clusterRoleBinding: ksm.clusterRoleBinding,
  },
}
