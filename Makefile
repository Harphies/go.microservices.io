TAG := v0.1.5

.PHONY: push/changes
push/changes:
	echo 'Push changes'
	git add .
	git commit -m 'new release for ${TAG}' --allow-empty
	git push

.PHONY: release/tag/gitub
release/tag/gitub: push/changes
	echo 'Release New Module Tag to github'
	git tag ${TAG}
	git push origin ${TAG}

release/module/go:
	echo "Release New tag to go module registry"
	GOPROXY=proxy.golang.org go list -m github.com/harphies/go.microservices.io@${TAG}