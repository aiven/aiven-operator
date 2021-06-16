---
title: "Troubleshooting"
linkTitle: "Troubleshooting"
weight: 85
---
Finding yourself stuck with the Operator? Here are a few tips to find out what is happening.

## Check the Pods
Verify if all the Operator Pods are `READY` and the `STATUS` is `Running`.
```bash
$ kubectl get pod -n aiven-kubernetes-operator-system 

NAME                                                            READY   STATUS    RESTARTS   AGE
aiven-kubernetes-operator-controller-manager-576d944499-ggttj   1/1     Running   0          12m
```

Verify if the cert-manager Pods are also running:
```bash
$ kubectl get pod -n cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-85cpv              1/1     Running   0          76s
cert-manager-cainjector-64c949654c-n2z8l   1/1     Running   0          77s
cert-manager-webhook-6bdffc7c9d-47w6z      1/1     Running   0          76s
```

## Visualize the Operator Logs
With the following command you can visualize all the logs from the Operator:
```bash
$ kubectl logs -n aiven-kubernetes-operator-system -l control-plane=controller-manager
```

## Verify the Operator Version
```bash
$ kubectl get pod -n aiven-kubernetes-operator-system -l control-plane=controller-manager -o jsonpath="{.items[0].spec.containers[0].image}"
```

## Known Errors
Here are some errors you might while deploying the Operator.

### cert-manager
If you see the following event on the Operator Pod:
```bash
MountVolume.SetUp failed for volume "cert" : secret "webhook-server-cert" not found
```

Make sure that cert-manager is up and running:
```bash
$ kubectl get pod -n cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-85cpv              1/1     Running   0          76s
cert-manager-cainjector-64c949654c-n2z8l   1/1     Running   0          77s
cert-manager-webhook-6bdffc7c9d-47w6z      1/1     Running   0          76s
```
