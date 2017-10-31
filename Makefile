default: all

GO_PACKAGES = $$(go list ./... | grep -v vendor)
GO_FILES = $$(find . -name "*.go" | grep -v vendor | uniq)

do-build:
	go build -o kibosh ./main.go

unit-test:
	@go test ${GO_PACKAGES}

fmt:
	gofmt -s -l -w $(GO_FILES)

vet:
	@go vet ${GO_PACKAGES}

test: unit-test vet

run:
	SERVICE_ID=123 \
	ADMIN_USERNAME=admin \
	ADMIN_PASSWORD=pass \
	go run main.go

all: fmt test do-build
