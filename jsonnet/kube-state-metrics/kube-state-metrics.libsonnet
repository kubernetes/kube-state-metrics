local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

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
    local clusterRoleBinding = k.rbac.v1.clusterRoleBinding;

    clusterRoleBinding.new() +
    clusterRoleBinding.mixin.metadata.withName(ksm.name) +
    clusterRoleBinding.mixin.metadata.withLabels(ksm.commonLabels) +
    clusterRoleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
    clusterRoleBinding.mixin.roleRef.withName(ksm.name) +
    clusterRoleBinding.mixin.roleRef.mixinInstance({ kind: 'ClusterRole' }) +
    clusterRoleBinding.withSubjects([{ kind: 'ServiceAccount', name: ksm.name, namespace: ksm.namespace }]),

  clusterRole:
    local clusterRole = k.rbac.v1.clusterRole;
    local rulesType = clusterRole.rulesType;

    local rules = [
      rulesType.new() +
      rulesType.withApiGroups(['']) +
      rulesType.withResources([
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
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['extensions']) +
      rulesType.withResources([
        'daemonsets',
        'deployments',
        'replicasets',
        'ingresses',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['apps']) +
      rulesType.withResources([
        'statefulsets',
        'daemonsets',
        'deployments',
        'replicasets',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['batch']) +
      rulesType.withResources([
        'cronjobs',
        'jobs',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['autoscaling']) +
      rulesType.withResources([
        'horizontalpodautoscalers',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['authentication.k8s.io']) +
      rulesType.withResources([
        'tokenreviews',
      ]) +
      rulesType.withVerbs(['create']),

      rulesType.new() +
      rulesType.withApiGroups(['authorization.k8s.io']) +
      rulesType.withResources([
        'subjectaccessreviews',
      ]) +
      rulesType.withVerbs(['create']),

      rulesType.new() +
      rulesType.withApiGroups(['policy']) +
      rulesType.withResources([
        'poddisruptionbudgets',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['certificates.k8s.io']) +
      rulesType.withResources([
        'certificatesigningrequests',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['storage.k8s.io']) +
      rulesType.withResources([
        'storageclasses',
        'volumeattachments',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['admissionregistration.k8s.io']) +
      rulesType.withResources([
        'mutatingwebhookconfigurations',
        'validatingwebhookconfigurations',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['networking.k8s.io']) +
      rulesType.withResources([
        'networkpolicies',
      ]) +
      rulesType.withVerbs(['list', 'watch']),

      rulesType.new() +
      rulesType.withApiGroups(['coordination.k8s.io']) +
      rulesType.withResources([
        'leases',
      ]) +
      rulesType.withVerbs(['list', 'watch']),
    ];

    clusterRole.new() +
    clusterRole.mixin.metadata.withName(ksm.name) +
    clusterRole.mixin.metadata.withLabels(ksm.commonLabels) +
    clusterRole.withRules(rules),
  deployment:
    local deployment = k.apps.v1.deployment;
    local container = deployment.mixin.spec.template.spec.containersType;
    local volume = deployment.mixin.spec.template.spec.volumesType;
    local containerPort = container.portsType;
    local containerVolumeMount = container.volumeMountsType;
    local podSelector = deployment.mixin.spec.template.spec.selectorType;

    local c =
      container.new('kube-state-metrics', ksm.image) +
      container.withPorts([
        containerPort.newNamed(8080, 'http-metrics'),
        containerPort.newNamed(8081, 'telemetry'),
      ]) +
      container.mixin.livenessProbe.httpGet.withPath('/healthz') +
      container.mixin.livenessProbe.httpGet.withPort(8080) +
      container.mixin.livenessProbe.withInitialDelaySeconds(5) +
      container.mixin.livenessProbe.withTimeoutSeconds(5) +
      container.mixin.readinessProbe.httpGet.withPath('/') +
      container.mixin.readinessProbe.httpGet.withPort(8081) +
      container.mixin.readinessProbe.withInitialDelaySeconds(5) +
      container.mixin.readinessProbe.withTimeoutSeconds(5) +
      container.mixin.securityContext.withRunAsUser(65534);

    deployment.new(ksm.name, 1, c, ksm.commonLabels) +
    deployment.mixin.metadata.withNamespace(ksm.namespace) +
    deployment.mixin.metadata.withLabels(ksm.commonLabels) +
    deployment.mixin.spec.selector.withMatchLabels(ksm.podLabels) +
    deployment.mixin.spec.template.spec.withNodeSelector({ 'kubernetes.io/os': 'linux' }) +
    deployment.mixin.spec.template.spec.withServiceAccountName(ksm.name),

  serviceAccount:
    local serviceAccount = k.core.v1.serviceAccount;

    serviceAccount.new(ksm.name) +
    serviceAccount.mixin.metadata.withNamespace(ksm.namespace) +
    serviceAccount.mixin.metadata.withLabels(ksm.commonLabels),

  service:
    local service = k.core.v1.service;
    local servicePort = service.mixin.spec.portsType;

    local ksmServicePortMain = servicePort.newNamed('http-metrics', 8080, 'http-metrics');
    local ksmServicePortSelf = servicePort.newNamed('telemetry', 8081, 'telemetry');

    service.new(ksm.name, ksm.podLabels, [ksmServicePortMain, ksmServicePortSelf]) +
    service.mixin.metadata.withNamespace(ksm.namespace) +
    service.mixin.metadata.withLabels(ksm.commonLabels) +
    service.mixin.spec.withClusterIp('None'),

  autosharding:: {
    role:
      local role = k.rbac.v1.role;
      local rulesType = role.rulesType;

      local rules = [
        rulesType.new() +
        rulesType.withApiGroups(['']) +
        rulesType.withResources(['pods']) +
        rulesType.withVerbs(['get']),
        rulesType.new() +
        rulesType.withApiGroups(['apps']) +
        rulesType.withResources(['statefulsets']) +
        rulesType.withResourceNames([ksm.name]) +
        rulesType.withVerbs(['get']),
      ];

      role.new() +
      role.mixin.metadata.withName(ksm.name) +
      role.mixin.metadata.withNamespace(ksm.namespace) +
      role.mixin.metadata.withLabels(ksm.commonLabels) +
      role.withRules(rules),

    roleBinding:
      local roleBinding = k.rbac.v1.roleBinding;

      roleBinding.new() +
      roleBinding.mixin.metadata.withName(ksm.name) +
      roleBinding.mixin.metadata.withLabels(ksm.commonLabels) +
      roleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
      roleBinding.mixin.roleRef.withName(ksm.name) +
      roleBinding.mixin.roleRef.mixinInstance({ kind: 'Role' }) +
      roleBinding.withSubjects([{ kind: 'ServiceAccount', name: ksm.name }]),

    statefulset:
      local statefulset = k.apps.v1.statefulSet;
      local container = statefulset.mixin.spec.template.spec.containersType;
      local containerEnv = container.envType;

      local c = ksm.deployment.spec.template.spec.containers[0] +
                container.withArgs([
                  '--pod=$(POD_NAME)',
                  '--pod-namespace=$(POD_NAMESPACE)',
                ]) +
                container.mixin.securityContext.withRunAsUser(65534) +
                container.withEnv([
                  containerEnv.new('POD_NAME') +
                  containerEnv.mixin.valueFrom.fieldRef.withFieldPath('metadata.name'),
                  containerEnv.new('POD_NAMESPACE') +
                  containerEnv.mixin.valueFrom.fieldRef.withFieldPath('metadata.namespace'),
                ]);

      statefulset.new(ksm.name, 2, c, [], ksm.commonLabels) +
      statefulset.mixin.metadata.withNamespace(ksm.namespace) +
      statefulset.mixin.metadata.withLabels(ksm.commonLabels) +
      statefulset.mixin.spec.withServiceName(ksm.service.metadata.name) +
      statefulset.mixin.spec.selector.withMatchLabels(ksm.podLabels) +
      statefulset.mixin.spec.template.spec.withNodeSelector({ 'kubernetes.io/os': 'linux' }) +
      statefulset.mixin.spec.template.spec.withServiceAccountName(ksm.name),
  } + {
    service: ksm.service,
    serviceAccount: ksm.serviceAccount,
    clusterRole: ksm.clusterRole,
    clusterRoleBinding: ksm.clusterRoleBinding,
  },
}
