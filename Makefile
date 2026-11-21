# Test layers. The backend packages are listed explicitly: pkg/ui needs Wails/GTK and a built frontend.
PKGS := ./internal/... ./pkg/cli/... ./tests/...
FUZZTIME ?= 20s

.PHONY: test-frontend test-fast test-integration test race fuzz bench netem lint vuln test-ui

test-fast:        ## unit + protocol tests, no real peers (seconds)
	go test -short -count=1 $(PKGS)

test-integration: ## real peers over loopback QUIC/UDP, includes impaired-network tests (about 1 min)
	go test -count=1 $(PKGS)

test: test-integration

race:             ## everything under the race detector
	go test -race -count=1 $(PKGS)

fuzz:             ## each fuzz target for $(FUZZTIME)
	go test ./internal/protocol  -run '^$$' -fuzz '^FuzzRequestRoundTrip$$'      -fuzztime $(FUZZTIME)
	go test ./internal/protocol  -run '^$$' -fuzz '^FuzzControlFrameDecoder$$'   -fuzztime $(FUZZTIME)
	go test ./internal/discovery -run '^$$' -fuzz '^FuzzGetPeer$$'               -fuzztime $(FUZZTIME)
	go test ./internal/discovery -run '^$$' -fuzz '^FuzzParseXORMappedAddress$$' -fuzztime $(FUZZTIME)
	go test ./internal/discovery -run '^$$' -fuzz '^FuzzParseToken$$'            -fuzztime $(FUZZTIME)
	go test ./internal/discovery -run '^$$' -fuzz '^FuzzParseBindingResponse$$'  -fuzztime $(FUZZTIME)
	go test ./internal/transfer  -run '^$$' -fuzz '^FuzzSanitizeFileName$$'      -fuzztime $(FUZZTIME)

bench:            ## benchmarks only (no correctness tests)
	go test -run '^$$' -bench . -benchmem $(PKGS)

test-frontend:    ## frontend unit tests (Vitest): phase state machine and formatters
	cd frontend && npm ci && npm test

test-ui:          ## UI end-to-end: real UI + real backends + real QUIC (needs Node, see tests/ui/README.md)
	cd tests/ui && npm ci && npx playwright install chromium && npm test

netem:            ## kernel-level impairment; needs root + tc (see docs/TESTING.md)
	sudo -E env "PATH=$$PATH" go test -tags netem -count=1 -run Netem -v ./internal/e2e/

lint:
	gofmt -l internal pkg/cli | (! grep .)
	go vet $(PKGS)

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest $(PKGS)
