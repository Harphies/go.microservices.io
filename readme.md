# Microservices and Distributed System Applications

This project focus on making building a microservices and distruted systems application a breeze with go

## Pillars

## Observability

Focuses on :

* Structured logging emitted to standard out on the progress for a logging agent such as [FluentBit](https://docs.fluentbit.io/manual/pipeline/inputs), [Splunk Forward](https://docs.splunk.com/Documentation/AddOns/released/Kubernetes/Install) to pick up in the `/var/log*` directory and persist to a logging backend such as Splunk, AWS CloudWatch, ElasticSearch etc.

* Microservices Prometheus Metrics Instrumentation with [prometheus go client](https://github.com/prometheus/client_golang)

* Distributed trace span collection with [opentelemetry](https://github.com/open-telemetry/opentelemetry-go)

## Storage 


## Security

## Networking


## Usage

```shell
go mod init github.com/harphies/go.microservices.io
GOPROXY=proxy.golang.org go list -m github.com/harphies/go.microservices.io@v0.0.1

# Usage
go install  github.com/harphies/go.microservices.io  
go list -m -versions github.com/harphies/go.microservices.io
```