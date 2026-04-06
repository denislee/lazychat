BINARY = lazychat

.PHONY: build run clean

build:
	go build -o $(BINARY) ./cmd/lazychat

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
