TAG := v0.2.11
.PHONY: push/changes
push/changes:
	echo 'Push changes'
	git add .
	git commit -m 'new release for ${TAG}' --allow-empty
	git push

.PHONY: release/tag/github
release/tag/github: push/changes
	echo 'Release New Module Tag to github'
	git tag ${TAG}
	git push origin ${TAG}

.PHONY: release/module/go
release/module/go: release/tag/github
	echo "Release New tag to go module registry"
	GOPROXY=proxy.golang.org go list -m github.com/harphies/go.microservices.io@${TAG}

#go get -u ./... && go mod tidy