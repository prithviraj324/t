package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"io/ioutil"

	"encoding/json"
	"flag"
	"fmt"
	"io"

	"net/http"

	"log"
	mrand "math/rand"
	"os"

	"strings"
	"sync"
	"time"

	golog "github.com/ipfs/go-log"

	libp2p "github.com/libp2p/go-libp2p" //@v0.24.2
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	net "github.com/libp2p/go-libp2p/core/network"

	peer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	merkle "github.com/prithviraj324/p2p_go/merkle_hash"
)

type IP struct {
	Query string
}

func getip2() string {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return err.Error()
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}
	var ip IP
	json.Unmarshal(body, &ip)
	// fmt.Print(ip.Query)
	return ip.Query
}

var Data []merkle.Block

var mutex = &sync.Mutex{}

// makeBasicHost creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It will use secio if secio is true.
func makeBasicHost(listenPort int, secio bool, randseed int64) (host.Host, error) {
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	m := getip2() //get host ip address
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", m, listenPort)),
		libp2p.Identity(priv),
	}

	basicHost, err := libp2p.New(opts...) //updated .New() uses context.Background() by default
	if err != nil {
		return nil, err
	}
	//Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host by encapsulating both addresses:
	addrs := basicHost.Addrs()
	var addr ma.Multiaddr
	//select address starting with ipv4
	for _, i := range addrs {
		if strings.HasPrefix(i.String(), "/ip4") {
			addr = i
			break
		}
	}
	fullAddr := addr.Encapsulate(hostAddr)
	log.Printf("Host address: %s \n", fullAddr)

	if secio {
		log.Printf("\nRun \"go run main.go -l %d -d %s -secio\" on a different node\n", listenPort+1, fullAddr)
	} else {
		log.Printf("Run \"go run main.go -l %d -d %s\" on a different node\n", listenPort+1, fullAddr)
	}

	return basicHost, nil
}

func handleStream(s net.Stream) {

	log.Println("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	go writeData(rw)

	// stream 's' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if str == "" {
			return
		}
		//if valid input was intercepted
		if str != "\n" {
			newData := make([]merkle.Block, 0)
			if err := json.Unmarshal([]byte(str), &newData); err != nil {
				log.Fatal(err)
			}
			mutex.Lock()
			if len(newData) > len(Data) {
				Data = newData
				bytes, err := json.MarshalIndent(Data, "", " ")
				if err != nil {
					log.Fatal(err)
				}
				log.Print("Received new valid Data[]:")
				fmt.Printf("\x1b[32m %s \x1b[0m> ", string(bytes)) //sets font color to green and resets to default
			}
			mutex.Unlock()
		}
	}
}

func writeData(rw *bufio.ReadWriter) {
	//broadcast latest state of its []Data every 5seconds
	//goroutine for concurrency
	go func() {
		for {
			time.Sleep(5 * time.Second)

			mutex.Lock()
			bytes, err := json.Marshal(Data)
			if err != nil {
				log.Println(err)
			}
			mutex.Unlock()

			mutex.Lock()
			rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
			rw.Flush()
			mutex.Unlock()
		}
	}()

	//take new data input from console
	stdReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		sendData = strings.Replace(sendData, "\n", "", -1)
		//if console input was to print current state of global DataBlock
		if sendData == "--print" {
			mutex.Lock()
			for _, blc := range Data {
				fmt.Printf("\t%s\n", blc.Content)
			}
			mutex.Unlock()
			continue
		}
		//else add input to DataBlock
		newBlock := merkle.GenerateBlock(Data[len(Data)-1], sendData)
		Data = append(Data, newBlock)

		bytes, err := json.Marshal(Data)
		if err != nil {
			log.Println(err)
		}

		for _, blk := range Data {
			fmt.Printf("Index: %d,\t", blk.Index)
			fmt.Printf("Content: %s\n", blk.Content)
		}
		//spew.Dump(Data)

		mutex.Lock()
		rw.WriteString(fmt.Sprintf("%s\n", string(bytes)))
		log.Println("Writing to others")
		rw.Flush()
		mutex.Unlock()
	}
}

func main() {
	fmt.Print("Starting up...")

	//libp2p uses golog for logging msgs
	golog.SetAllLoggers(golog.LevelInfo) //logging verbosity level

	//flags to parse options provided in cli
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	secio := flag.Bool("secio", false, "enable secio")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	//making a host that listens on given multiaddress
	ha, err := makeBasicHost(*listenF, *secio, *seed)
	if err != nil {
		log.Fatal(err)
	}

	if *target == "" {
		log.Println("Listening for connections...")
		//adding genesis block because its the only node in network
		genesis := merkle.Block{Index: 1, Timestamp: time.Now().String(), Content: "---Genesis---", Hash: "", PrevHash: ""}
		genesis.Hash = merkle.CalculateHash(genesis)
		Data = append(Data, genesis)
		//set a StreamHandler
		ha.SetStreamHandler("/p2p/1.0.0", handleStream)

		select {} //hang forever
	} else { //Listener Ends
		ha.SetStreamHandler("/p2p/1.0.0", handleStream)

		//extract target's peer id from received multiaddress
		//parse *target string into ma.Multiaddr
		ipfsaddr, err := ma.NewMultiaddr(*target)
		if err != nil {
			log.Fatal(err)
		}
		// Decapsulate /ipfs/<peerID> part from the target
		// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
		pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
		if err != nil {
			log.Fatal(err)
		}

		peerid, err := peer.Decode(pid)
		if err != nil {
			log.Fatal(err)
		}

		targetPeerAddr, _ := ma.NewMultiaddr((fmt.Sprintf("/ipfs/%s", peerid.String())))
		targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

		// We have a peer ID and a targetAddr so we add it to the peerstore so LibP2P knows how to contact it
		ha.Peerstore().AddAddr(peerid, targetAddr, 5*time.Hour)
		log.Println("Opening New Stream...")

		// make a new stream from host B to host A
		// it should be handled on host A by the handler we set above because we use the same '/p2p/1.0.0' protocol
		s, err := ha.NewStream(context.Background(), peerid, "/p2p/1.0.0")
		if err != nil {
			log.Fatal(err)
		}
		// Creating a buffered stream for non-blocking reads and writes.
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		//Creating thread to read and write data
		go writeData(rw)
		go readData(rw)

		select {} //hang
	} //end else
}
