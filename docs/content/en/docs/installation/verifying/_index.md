---
title: "Verifying the installation"
linkTitle: "Verifying the installation"
weight: 30
---

Use the following commands to ensure your installation was successful.

* Verify that all the operator Pods are `READY`, and their `STATUS` is `Running`.
```bash
$ kubectl get pod -n aiven-operator-system 

NAME                                                            READY   STATUS    RESTARTS   AGE
aiven-operator-controller-manager-576d944499-ggttj   1/1     Running   0          12m
```

* Verify that the `cert-manager` Pods are running:
```bash
$ kubectl get pod --namespace cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-85cpv              1/1     Running   0          76s
cert-manager-cainjector-64c949654c-n2z8l   1/1     Running   0          77s
cert-manager-webhook-6bdffc7c9d-47w6z      1/1     Running   0          76
```

* Verify the operator startup logs. These should look like the code below (output trimmed):
```bash
$ kubectl logs -n aiven-operator-system -l control-plane=controller-manager -f

2021-06-16T17:05:39.661Z	INFO	controller-runtime.metrics	metrics server is starting to listen	{"addr": ":8080"}
2021-06-16T17:05:39.699Z	INFO	controller-runtime.builder	Registering a mutating webhook	{"GVK": "aiven.io/v1alpha1, Kind=Project", "path": "/mutate-aiven-io-v1alpha1-project"}
2021-06-16T17:05:39.699Z	INFO	controller-runtime.webhook	registering webhook	{"path": "/mutate-aiven-io-v1alpha1-project"}
[...] 

2021-06-16T17:05:39.901Z	INFO	setup	starting manager
I0616 17:05:39.902448       1 leaderelection.go:243] attempting to acquire leader lease aiven-operator-system/00272a53.aiven.io...
2021-06-16T17:05:39.902Z	INFO	controller-runtime.webhook.webhooks	starting webhook server
2021-06-16T17:05:39.903Z	INFO	controller-runtime.certwatcher	Updated current TLS certificate
2021-06-16T17:05:39.903Z	INFO	controller-runtime.webhook	serving webhook server	{"host": "", "port": 9443}
2021-06-16T17:05:39.903Z	INFO	controller-runtime.certwatcher	Starting certificate watcher
2021-06-16T17:05:39.902Z	INFO	controller-runtime.manager	starting metrics server	{"path": "/metrics"}
I0616 17:05:40.017691       1 leaderelection.go:253] successfully acquired lease aiven-operator-system/00272a53.aiven.io
2021-06-16T17:05:40.018Z	DEBUG	controller-runtime.manager.events	Normal	{"object": {"kind":"ConfigMap","namespace":"aiven-operator-system","name":"00272a53.aiven.io","uid":"63bc05df-3df5-44d1-8344-41b05a6a3a4f","apiVersion":"v1","resourceVersion":"2073"}, "reason": "LeaderElection", "message": "aiven-operator-controller-manager-6d68c4d6d7-5s5kj_64fd1a06-44b6-4690-95e1-17ec8adf7ffe became leader"}
2021-06-16T17:05:40.098Z	INFO	controller-runtime.manager.controller.kafkatopic	Starting EventSource	{"reconciler group": "aiven.io", "reconciler kind": "KafkaTopic", "source": "kind source: /, Kind="}
2021-06-16T17:05:40.018Z	INFO	controller-runtime.manager.controller.projectvpc	Starting EventSource	{"reconciler group": "aiven.io", "reconciler kind": "ProjectVPC", "source": "kind source: /, Kind="}
[...]

2021-06-16T17:05:40.097Z	DEBUG	controller-runtime.manager.events	Normal	{"object": {"kind":"Lease","namespace":"aiven-operator-system","name":"00272a53.aiven.io","uid":"a81127e0-7293-4b0f-b4f9-8fc78adba8e0","apiVersion":"coordination.k8s.io/v1","resourceVersion":"2074"}, "reason": "LeaderElection", "message": "aiven-operator-controller-manager-6d68c4d6d7-5s5kj_64fd1a06-44b6-4690-95e1-17ec8adf7ffe became leader"}
2021-06-16T17:05:40.998Z	INFO	controller-runtime.manager.controller.kafka	Starting workers	{"reconciler group": "aiven.io", "reconciler kind": "Kafka", "worker count": 1}
2021-06-16T17:05:40.999Z	INFO	controller-runtime.manager.controller.serviceuser	Starting workers	{"reconciler group": "aiven.io", "reconciler kind": "ServiceUser", "worker count": 1}
2021-06-16T17:05:40.999Z	INFO	controller-runtime.manager.controller.connectionpool	Starting workers	{"reconciler group": "aiven.io", "reconciler kind": "ConnectionPool", "worker count": 1}
2021-06-16T17:05:40.999Z	INFO	controller-runtime.manager.controller.kafkaacl	Starting workers	{"reconciler group": "aiven.io", "reconciler kind": "KafkaACL", "worker count": 1}
```

You have now verified that your installation was successful.
