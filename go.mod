module github.com/aiven/aiven-k8s-operator

go 1.15

require (
	github.com/aiven/aiven-go-client v1.5.12-0.20210401064156-3594908646c8
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.3
	github.com/onsi/gomega v1.13.0
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)
