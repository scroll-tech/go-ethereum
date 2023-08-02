.PHONY: clean libzkp

clean:
	rm -f *.a *.so
	cargo clean

libzkp:
	cargo build --release
	sudo cp $(PWD)/target/release/libzkp.so /usr/local/lib/
	find $(PWD)/target | grep libzktrie.so | xargs -I{} sudo cp {} /usr/local/lib/
