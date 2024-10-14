# Export missing block header fields toolkit

A toolkit for exporting and transforming missing block header fields of Scroll before {{upgrade_name}} TODO: replace when upgrade is clear.

## Context
We are using the [Clique consensus](https://eips.ethereum.org/EIPS/eip-225) in Scroll L2. Amongst others, it requires the following header fields:
- `extraData`
- `difficulty`

However, before {{upgrade_name}}, these fields were not stored on L1/DA.
In order for nodes to be able to reconstruct the correct block hashes when only reading data from L1, 
we need to provide the historical values of these fields to these nodes through a separate file.

This toolkit provides commands to export the missing fields, deduplicate the data and create a file 
with the missing fields that can be used to reconstruct the correct block hashes when only reading data from L1.

The toolkit provides the following commands:
- `fetch` - Fetch missing block header fields from a running Scroll L2 node and store in a file
- `dedup` - Deduplicate the headers file, print unique values and create a new file with the deduplicated headers 

## Binary layout deduplicated missing header fields file
The deduplicated header file binary layout is as follows:

```plaintext
<unique_vanity_count:uint8><unique_vanity_1:[32]byte>...<unique_vanity_n:[32]byte><header_1:header>...<header_n:header>

Where:
- unique_vanity_count: number of unique vanities n
- unique_vanity_i: unique vanity i
- header_i: block header i
- header: 
    <flags:uint8><seal:[65|85]byte>
    - flags: bitmask, lsb first
	    - bit 0-5: index of the vanity in the sorted vanities list
	    - bit 6: 0 if difficulty is 2, 1 if difficulty is 1
	    - bit 7: 0 if seal length is 65, 1 if seal length is 85
```

## How to run
Each of the commands has its own set of flags and options. To display the help message run with `--help` flag.

1. Fetch the missing block header fields from a running Scroll L2 node via RPC and store in a file (approx 40min for 5.5M blocks).
2. Deduplicate the headers file, print unique values and create a new file with the deduplicated headers

```bash
go run main.go fetch --rpc=http://localhost:8545 --start=0 --end=100 --batch=10 --parallelism=10 --output=headers.bin --humanOutput=true
go run main.go dedup --input=headers.bin --output=headers-dedup.bin
```


### With Docker
To run the toolkit with Docker, build the Docker image and run the commands inside the container.

```bash  
docker build -t export-headers-toolkit .

# depending on the Docker config maybe finding the RPC container's IP with docker inspect is necessary. Potentially host IP works: http://172.17.0.1:8545
docker run --rm -v "$(pwd)":/app/result export-headers-toolkit fetch --rpc=<address> --start=0 --end=5422047 --batch=10000 --parallelism=10 --output=/app/result/headers.bin --humanOutput=/app/result/headers.csv
docker run --rm -v "$(pwd)":/app/result export-headers-toolkit dedup --input=/app/result/headers.bin --output=/app/result/headers-dedup.bin 
```



