SWAGGER_UI_VERSION:=v4.15.5

.PHONY: generate
generate: generate/proto generate/swagger-ui

.PHONY: generate/proto
generate/proto:
	easyp generate

.PHONY: generate/swagger-ui
generate/swagger-ui:
	SWAGGER_UI_VERSION=$(SWAGGER_UI_VERSION) ./scripts/generate-swagger-ui.sh

