module github.com/aiven/aiven-k8s-operator

go 1.13

require (
	github.com/aiven/aiven-go-client v1.5.12-0.20210126101622-c6cb3f424339
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.3
)
