# Building Microservices and Distributed System Applications with Go

This package is an abstraction to make building microservices and distributed systems application a breeze using Go. 

Most of the code will be coming from infrastructure perspective. What needs to be considered at application layer and what needs to be delegated to Infrastructure layer.

I have deployed/release over 500+ microservices on Multi-tenant Kubernetes Clusters. I believe I have some experience to make a judgement on what I see that's missing and how things can be improved going forward.

## Pillars

## Observability

Focuses on :

* Structured logs emitted to standard output (stdout) on the process for a logging agent such as [FluentBit](https://docs.fluentbit.io/manual/pipeline/inputs), [Splunk Forwarder](https://docs.splunk.com/Documentation/AddOns/released/Kubernetes/Install) to pick up in the `/var/log*` directory and persist to a logging backend such as Splunk, AWS CloudWatch, ElasticSearch etc.

* Microservices Prometheus Metrics Instrumentation with [prometheus go client](https://github.com/prometheus/client_golang)

* Distributed trace span collection with [Opentelemetry SDK](https://github.com/open-telemetry/opentelemetry-go) forwarded to [Jaeger Collector](https://www.jaegertracing.io/docs/1.49/getting-started/#instrumentation) and eventually persisted to trace backend such as [ElasticSearch](https://www.jaegertracing.io/docs/1.49/faq/#what-is-the-recommended-storage-backend)

## Storage 


## Security

## Networking


## Server 

Make it easy starting up a new `HTTP/GRPC` over `TCP/UDP` server for a process.


## Utils

## The DSA 

The DSA Pillar is a collection of data structures and algorithms or my data to day operations and anyone that I come across that's useful.

## Usage

```shell
go mod init github.com/harphies/go.microservices.io
GOPROXY=proxy.golang.org go list -m github.com/harphies/go.microservices.io@v0.0.1

# Usage
go get github.com/harphies/go.microservices.io@<tag> 
```

## References

- https://github.com/alessiosavi/GoGPUtils
- https://github.com/kubernetes-sigs/aws-load-balancer-controller/tree/main/pkg/algorithm
- https://pkg.go.dev/github.com/go-ozzo/ozzo-validation/v4
- https://github.com/xanzy/go-gitlab/blob/main/strings.go#L28
- https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/s3/actions/bucket_basics.go
- [Working with Cassandra](https://medium.com/@timothy-urista/an-easy-guide-to-implementing-pagination-in-cassandra-using-go-e7d13cfc804a)