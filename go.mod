module github.com/johnsushant/pod-crash-controller

go 1.13

require (
	github.com/ashwanthkumar/slack-go-webhook v0.0.0-20200209025033-430dd4e66960
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.0
	github.com/nlopes/slack v0.6.0
	github.com/parnurzeal/gorequest v0.2.16 // indirect
	go.uber.org/zap v1.9.1
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	moul.io/http2curl v1.0.0 // indirect
	sigs.k8s.io/controller-runtime v0.4.0
)
