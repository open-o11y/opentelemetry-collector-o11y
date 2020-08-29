module github.com/open-o11y/opentelemetry-collector-o11y

go 1.14

replace go.opentelemetry.io/collector => ./internal/opentelemetry-collector

require (
	github.com/aws/aws-sdk-go v1.31.9
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.9.0
)
