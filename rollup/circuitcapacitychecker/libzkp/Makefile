.PHONY: clean libzkp

clean:
	rm -f *.a *.so
	cargo clean

libzkp:
	cargo build --release
	cp $(PWD)/target/release/libzkp.so /usr/local/lib/
	find $(PWD)/target | grep libzktrie.so | xargs -I{} cp {} /usr/local/lib/
