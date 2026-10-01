TINYGO ?= tinygo
TAGS = -tags fat_noexfat

test:
	go vet ./...
	go test ./...

smoketest:
	$(TINYGO) build -o test.bin -size=short -target=xiao-esp32c3 ./examples/embed
	$(TINYGO) build -o test.bin -size=short -target=xiao-esp32c3 $(TAGS) ./examples/flash
	$(TINYGO) build -o test.bin -size=short -target=xiao-esp32s3 ./examples/embed
	$(TINYGO) build -o test.bin -size=short -target=xiao-esp32c6 ./examples/embed
	$(TINYGO) build -o test.uf2 -size=short -target=pico ./examples/embed
	$(TINYGO) build -o test.uf2 -size=short -target=pico $(TAGS) ./examples/flash
	$(TINYGO) build -o test.uf2 -size=short -target=pico $(TAGS) ./examples/msc
	$(TINYGO) build -o test.uf2 -size=short -target=xiao-rp2350 ./examples/embed
	$(TINYGO) build -o test.uf2 -size=short -target=xiao-rp2350 $(TAGS) ./examples/flash
	$(TINYGO) build -o test.uf2 -size=short -target=xiao-rp2350 $(TAGS) ./examples/msc
	$(TINYGO) build -o test.uf2 -size=short -target=xiao-ble ./examples/embed
	$(TINYGO) build -o test.uf2 -size=short -target=xiao-ble $(TAGS) ./examples/flash
	$(TINYGO) build -o test.uf2 -size=short -target=xiao-ble $(TAGS) ./examples/msc
	@rm -f test.bin test.uf2

.PHONY: test smoketest
