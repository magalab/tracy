module github.com/panda/tracy/tests/cozeloop-e2e

go 1.26

require github.com/coze-dev/cozeloop-go v0.1.10

// v0.1.10 declares an older spec module that is missing prompt symbols used by
// the SDK package. Pin the matching spec revision until the upstream module is
// released with a corrected dependency graph.
replace github.com/coze-dev/cozeloop-go/spec => github.com/coze-dev/cozeloop-go/spec v0.1.4-0.20250829072213-3812ddbfb735

require (
	github.com/bluele/gcache v0.0.2 // indirect
	github.com/coze-dev/cozeloop-go/spec v0.1.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nikolalohinski/gonja/v2 v2.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/exp v0.0.0-20240404231335-c0f41cb1a7a0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
