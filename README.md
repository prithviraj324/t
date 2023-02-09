# Cassandra Cluster Replication: Simulation and Analysis


This project provides a simulation framework for the replication strategies used in Apache Cassandra, a highly scalable and distributed NoSQL database system.

## Features

- Simulates different replication strategies over a P2P network established across data nodes.
- Allows for configurable cluster settings, such as number of nodes and replication factor (ALL by default).
- Dataset stored as a []string in memory across all nodes.

## Usage

To run the simulation, execute the following command:

```bash
go run main.go -secio -l 10000
```

## Tech

- [Golang]
- [go-libp2p]


[//]: # (These are reference links used in the body of this note and get stripped out when the markdown processor does its job. There is no need to format nicely because it shouldn't be seen. Thanks SO - http://stackoverflow.com/questions/4823468/store-comments-in-markdown-syntax)

   [go-libp2p]: <https://pkg.go.dev/github.com/libp2p/go-libp2p>
   [Golang]: <https://go.dev/>
