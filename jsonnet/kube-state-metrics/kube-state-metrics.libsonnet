
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

  extraRecommendedLabels:: {
    'app.kubernetes.io/component': 'exporter',
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
        labels: ksm.commonLabels + ksm.extraRecommendedLabels,
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
          'serviceaccounts',
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
        apiGroups: ['discovery.k8s.io'],
        resources: [
          'endpointslices',
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
          'ingressclasses',
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
      {
        apiGroups: ['rbac.authorization.k8s.io'],
        resources: [
          'clusterrolebindings',
          'clusterroles',
          'rolebindings',
          'roles',
        ],
        verbs: ['list', 'watch'],
      },
     ];

    {
      apiVersion: 'rbac.authorization.k8s.io/v1',
      kind: 'ClusterRole',
      metadata: {
        name: ksm.name,
        labels: ksm.commonLabels + ksm.extraRecommendedLabels,
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
      securityContext: {
        runAsUser: 65534,
        allowPrivilegeEscalation: false,
        readOnlyRootFilesystem: true,
        capabilities: { drop: ['ALL'] },
      },
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
        labels: ksm.commonLabels + ksm.extraRecommendedLabels,
      },
      spec: {
        replicas: 1,
        selector: { matchLabels: ksm.podLabels },
        template: {
          metadata: {
            labels: ksm.commonLabels + ksm.extraRecommendedLabels,
          },
          spec: {
            containers: [c],
            serviceAccountName: ksm.serviceAccount.metadata.name,
            automountServiceAccountToken: true,
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
        labels: ksm.commonLabels + ksm.extraRecommendedLabels,
      },
      automountServiceAccountToken: false,
    },

  service:
    {
      apiVersion: 'v1',
      kind: 'Service',
      metadata: {
        name: ksm.name,
        namespace: ksm.namespace,
        labels: ksm.commonLabels + ksm.extraRecommendedLabels,
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
          labels: ksm.commonLabels + ksm.extraRecommendedLabels,
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
          labels: ksm.commonLabels + ksm.extraRecommendedLabels,
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
          labels: ksm.commonLabels + ksm.extraRecommendedLabels,
        },
        spec: {
          replicas: 2,
          selector: { matchLabels: ksm.podLabels },
          serviceName: ksm.service.metadata.name,
          template: {
            metadata: {
              labels: ksm.commonLabels + ksm.extraRecommendedLabels,
            },
            spec: {
              containers: [c],
              serviceAccountName: ksm.serviceAccount.metadata.name,
              automountServiceAccountToken: true,
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
  daemonsetsharding:: {
    local shardksmname = ksm.name + "-shard",
		daemonsetService: std.mergePatch(ksm.service,
       {
				 metadata: {
					 name: shardksmname,
					 labels: {'app.kubernetes.io/name': shardksmname}
				 },
			   spec: {selector: {'app.kubernetes.io/name': shardksmname}},
			 }
		),
    deployment:
      // extending the default container from above
      local c = ksm.deployment.spec.template.spec.containers[0] {
        args: [
          '--resources=certificatesigningrequests,configmaps,cronjobs,daemonsets,deployments,endpoints,horizontalpodautoscalers,ingresses,jobs,leases,limitranges,mutatingwebhookconfigurations,namespaces,networkpolicies,nodes,persistentvolumeclaims,persistentvolumes,poddisruptionbudgets,replicasets,replicationcontrollers,resourcequotas,secrets,services,statefulsets,storageclasses,validatingwebhookconfigurations,volumeattachments',
        ],
      };
      std.mergePatch(ksm.deployment,
        {
          spec: {
            template: {
              spec: {
                containers: [c],
              },
            },
          },
        },
      ),

    daemonset:
      // extending the default container from above
      local c0 = ksm.deployment.spec.template.spec.containers[0] {
        args: [
          '--resources=pods',
          '--node=$(NODE_NAME)',
        ],
        env: [
          { name: 'NODE_NAME', valueFrom: { fieldRef: { apiVersion: 'v1', fieldPath: 'spec.nodeName' } } },
        ],
      };

      local c = std.mergePatch(c0, {name: shardksmname});

      local ksmLabels =  std.mergePatch(ksm.commonLabels + ksm.extraRecommendedLabels, {'app.kubernetes.io/name': shardksmname});
      local ksmPodLabels =  std.mergePatch(ksm.podLabels, {'app.kubernetes.io/name': shardksmname});

      {
        apiVersion: 'apps/v1',
        kind: 'DaemonSet',
        metadata: {
          namespace: ksm.namespace,
          labels: ksmLabels,
					name: shardksmname,
        },
        spec: {
          selector: { matchLabels: ksmPodLabels },
          template: {
            metadata: {
              labels: ksmLabels,
            },
            spec: {
              containers: [c],
              serviceAccountName: ksm.serviceAccount.metadata.name,
              automountServiceAccountToken: true,
              nodeSelector: { 'kubernetes.io/os': 'linux' },
            },
          },
        },
      },

  } + {
    deploymentService: ksm.service,
    serviceAccount: ksm.serviceAccount,
    clusterRole: ksm.clusterRole,
    clusterRoleBinding: ksm.clusterRoleBinding,
  },
}


