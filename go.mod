module github.com/cloud104/kube-db

go 1.12

require (
	github.com/aws/aws-sdk-go v1.19.47
	github.com/aws/aws-sdk-go-v2 v0.9.0
	github.com/cloud104/k8s-rds v1.0.0-master
	github.com/go-logr/logr v0.1.0
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/onsi/ginkgo v1.6.0
	github.com/onsi/gomega v1.4.2
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8 // indirect
	golang.org/x/net v0.0.0-20190611141213-3f473d35a33a
	golang.org/x/sys v0.0.0-20190610200419-93c9922d18ae // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/tools v0.0.0-20190611222205-d73e1c7e250b // indirect
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/cluster-api v0.0.0-20190604211153-54593075a7a1
	sigs.k8s.io/controller-runtime v0.2.0-beta.1
)
