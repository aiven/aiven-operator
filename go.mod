module github.com/aiven/aiven-k8s-operator

go 1.13

require (
	github.com/aiven/aiven-go-client v1.5.12-0.20210401064156-3594908646c8
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.10.2
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.3
)
