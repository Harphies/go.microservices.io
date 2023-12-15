# CHANGELOG

## v0.1.25-27 -  14/12/2023

FEATURES:

* Messaging with AWS SQS
* Experimental dynamodb
* Bump AWS SDK versions
* Increase http server read and write timeout


## v0.1.24 -  14/12/2023

FEATURES:

* Experimental Dynamodb as persistent datastore
* Centralise utilities helper and data structures and algorithms

## v0.1.22-23 -  12/12/2023

FEATURES:

* Simplified Events Publishing with MSK Kafka Proxy for services
* Experiment kgo kafka client for producer

## v0.1.20/21 -  12/12/2023

FEATURES:

* Fixed closing kafka client

## v0.1.19 -  12/12/2023

FEATURES:

* Fixed typo on AWS OpenSearch Connection message
* Refactored messaging with Kafka to dynamically pass topicName down

## v0.1.18 -  09/12/2023

FEATURES:

* Support for MSK client via IAM 
* Defer closing kafka producer client
* MSK Connection error handling

## v0.1.16/17 -  08/12/2023

FEATURES:

* Defer closing kafka producer client
* MSK Connection error handling
* Refactored MSK event types and event payloads naming conventions and expected arguments

## v0.1.15 -  6/12/2023

FEATURES:

* Refactored ElasticSearch Connection functions
* Added redis client connection to redis cluster early failure

## v0.1.10 -  16/10/2023

FEATURES:

* Updated HTTP utils Authorization Token

## v0.1.9 -  10/10/2023

FEATURES:

* Added Authentication supports via OAuth2.0 to `Security` pillar
* HTTP utils includes set and get cookies features.
* Separate capability to generate Access Token for Client credentials from authorization code flow grant type


## v0.1.0 05/10/2023

FEATURES:

* Created the repo
* Added observability section
