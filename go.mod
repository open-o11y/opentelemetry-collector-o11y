module github.com/open-o11y/opentelemetry-collector-o11y

go 1.14

replace go.opentelemetry.io/collector => ./internal/opentelemetry-collector

require (
	github.com/aws/aws-sdk-go v1.31.9
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/google/addlicense v0.0.0-20200817051935-6f4cd4aacc89 // indirect
	github.com/hashicorp/consul/api v1.4.0 // indirect
	github.com/hashicorp/serf v0.9.2 // indirect
	github.com/mitchellh/go-testing-interface v1.0.3 // indirect
	github.com/open-telemetry/opentelemetry-proto v0.4.0 // indirect
	github.com/shirou/gopsutil v2.20.6+incompatible // indirect
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.9.0
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
)
