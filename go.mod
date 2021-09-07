module github.com/aiven/aiven-kubernetes-operator

go 1.17

require (
	github.com/aiven/aiven-go-client v1.6.0
	github.com/go-logr/logr v0.4.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
	golang.org/x/tools v0.1.3 // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)
