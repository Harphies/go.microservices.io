# CHANGELOG

## Release Processs

```sh
# Prerequisite: Push your changes first and check all the available tags before cutting a new tag to avoid duplicate etc.
git tag # this will list all available tags in the upstream
# Step 1: Cut a new tag and create release
git tag <tag>
git push origin <tag>
# Step 2: The create a release from the tag created above on github UI

# Step 3: Push to Go proxy
GOPROXY=proxy.golang.org go list -m github.com/harphies/go.microservices.io@<tag>

# Check the list of available published module versions
go list -m -versions github.com/harphies/go.microservices.io
```

4 weeks release cycle/cadense. Every 2 weeks, new tag is cut and released.

## v0.1.0 05/10/2023

FEATURES:

* Created the repo
* Added observability section
