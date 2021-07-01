---
title: "Troubleshooting"
linkTitle: "Troubleshooting"
weight: 85
---
## Verifying operator status
Use the following checks to help you troubleshoot the Aiven Kubernetes Operator.

### Checking the Pods
Verify that all the operator pods are `READY`, and the `STATUS` is `Running`.
```bash
$ kubectl get pod -n aiven-kubernetes-operator-system 

NAME                                                            READY   STATUS    RESTARTS   AGE
aiven-kubernetes-operator-controller-manager-576d944499-ggttj   1/1     Running   0          12m
```

Verify that the `cert-manager` pods are also running.
```bash
$ kubectl get pod -n cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-85cpv              1/1     Running   0          76s
cert-manager-cainjector-64c949654c-n2z8l   1/1     Running   0          77s
cert-manager-webhook-6bdffc7c9d-47w6z      1/1     Running   0          76s
```

### Visualizing the operator logs
Use the following command to visualize all the logs from the operator.
```bash
$ kubectl logs -n aiven-kubernetes-operator-system -l control-plane=controller-manager
```

### Verifing the operator version
```bash
$ kubectl get pod -n aiven-kubernetes-operator-system -l control-plane=controller-manager -o jsonpath="{.items[0].spec.containers[0].image}"
```

## Known issues and limitations
We're always working to resolve problems that pop up in Aiven products. 
If your problem is listed below, we know about it and are working to fix it. If your problem isn't listed below, report it as an issue.

### cert-manager
#### Issue
The following event appears on the operator pod:
```bash
MountVolume.SetUp failed for volume "cert" : secret "webhook-server-cert" not found
```
#### Impact
You cannot run the operator. 

#### Solution
Make sure that cert-manager is up and running.
```bash
$ kubectl get pod -n cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-85cpv              1/1     Running   0          76s
cert-manager-cainjector-64c949654c-n2z8l   1/1     Running   0          77s
cert-manager-webhook-6bdffc7c9d-47w6z      1/1     Running   0          76s
```
