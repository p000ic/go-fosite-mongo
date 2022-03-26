module github.com/p000ic/go-fosite-mongo/examples

go 1.17

// use the local code, rather than go'getting the module
replace github.com/p000ic/go-fosite-mongo => ../../../storage

require (
	github.com/p000ic/go-fosite-mongo v0.0.0
	github.com/ory/fosite v0.42.1
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
)
