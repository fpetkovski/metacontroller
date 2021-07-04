module metacontroller/library

go 1.16

require (
	k8s.io/apimachinery v0.21.2
	k8s.io/klog/v2 v2.9.0
	metacontroller v0.0.0-00010101000000-000000000000
)

replace metacontroller => ./../../../..
