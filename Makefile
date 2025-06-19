COVER_FILE := test.cover

test:
	go test ./... -count=1 -coverprofile $(COVER_FILE)

test-coverpkg:
	go test ./... \
		-coverprofile $(COVER_FILE) \
		-covermode=atomic \
		-coverpkg $$(go list ./... | paste -sd "," -)

test-coverage-view:
	go tool cover -html $(COVER_FILE)

bench:
	go test ./tests/... -benchmem -run='^$$' -bench=.
