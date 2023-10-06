# CHANGELOG

## Release Processs

```sh
# Cut a new tag and create release
git tag <tag>
git push origin <tag>
# The create a release on the github UI

# Push to Go proxy
GOPROXY=proxy.golang.org go list -m github.com/harphies/go.microservices.io@<tag>
```

4 weeks release cycle/cadense. Every 2 weeks, new tag is cut and released.

## v0.0.1 05/10/2023

FEATURES:

* Created the repo
* Added observability section
