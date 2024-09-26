package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"github.com/Mina218/FileSharingNetwork/fileshare"
	"github.com/Mina218/FileSharingNetwork/stream"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"

	"fmt"
	"github.com/Mina218/FileSharingNetwork/p2pnet"
	"github.com/libp2p/go-libp2p"

	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func SourceNode() host.Host {
	node, err := libp2p.New()
	if err != nil {
		panic(err)
	}

	return node
}
func NewDh(ctx context.Context, host host.Host, Peers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	var options []dht.Option

	if len(Peers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
	}

	thisdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, err
	}
	if err = thisdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	for _, peerAddr := range Peers {
		peerinformations, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			return nil, err
		}
		wg.Add(1)

		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinformations); err != nil {
				log.Printf("Error while connecting to node %q: %-v", peerinformations, err)
			} else {
				log.Printf("Connection established with bootstrap node: %q", *peerinformations)
			}
		}()
	}
	wg.Wait()

	return thisdht, nil
}
func DestinationNode() host.Host {

	listenAddr := "/ip4/172.17.0.1/tcp/9090"
	node, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr))
	if err != nil {
		panic(err)
	}

	return node
}
func connectToNodeFromSource(sourceNode host.Host, targetNode host.Host) {
	targetNodeAddressInfo := host.InfoFromHost(targetNode)
	err := sourceNode.Connect(context.Background(), *targetNodeAddressInfo)
	if err != nil {
		panic(err)
	}
}
func countSourceNodePeers(sourceNode host.Host) int {
	return len(sourceNode.Network().Peers())
}
func printNodeID(host host.Host) {
	println(fmt.Sprintf("ID: %s", host.ID().String()))
}

func printNodeAddresses(host host.Host) {
	addressesString := make([]string, 0)
	for _, address := range host.Addrs() {
		addressesString = append(addressesString, address.String())
	}

	println(fmt.Sprintf("Multiaddresses: %s", strings.Join(addressesString, ", ")))
}
func createNodeWithMultiaddr(ctx context.Context, listenAddress multiaddr.Multiaddr) (host.Host, error) {
	// Create a new libp2p node specifying the listen address
	node, err := libp2p.New(libp2p.ListenAddrStrings(listenAddress.String()))
	if err != nil {
		return nil, err
	}
	return node, nil
}

func main() {
	var pid string = "/pid/file/share"
	const topic string = "/home/eya/Desktop/myFile.txt"
	file := OpenFileStatusLog()
	defer file.Close()

	ctx, h := EstablishP2P()
	h.SetStreamHandler(protocol.ID(pid), HandleInputStream)
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	//// Set up a DHT for peer discovery
	kad_dht := initDHT(ctx, h)
	bootstrapDHT(ctx, h, kad_dht)
	// Use the p2pnet package's DiscoverPeers function
	serviceTag := "myService"
	sub, top := HandlePubSub(ctx, h, topic)
	peerID := h.ID()
	fmt.Println("Peer ID:", peerID)
	p2pnet.DiscoverPeers(ctx, h, serviceTag, kad_dht)
	ResolveAll(ctx, h, top)
	HandlePubSubMessages(ctx, h, sub, top)
	go func() {
		err := SendFile(ctx, h, peerID, "/home/eya/Desktop/myFile.txt")
		if err != nil {
			fmt.Println("Error sending file:", err)
		}
	}()

	// Receiver side: handle incoming streams
	HandleIncomingStreams(ctx, h, file)

	// Wait for shutdown signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Shutting down...")
}
func multiaddrString(addr string) multiaddr.Multiaddr {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		panic(err)
	}
	return maddr
}
func initDHT(ctx context.Context, host host.Host) *dht.IpfsDHT {
	kad_dht, err := dht.New(ctx, host)
	if err != nil {
		fmt.Println("Error while creating the new DHT")
	} else {
		fmt.Println("Successfully created the new DHT")
	}
	return kad_dht
}

func bootstrapDHT(ctx context.Context, host host.Host, kad_dht *dht.IpfsDHT) {
	err := kad_dht.Bootstrap(ctx)
	if err != nil {
		fmt.Println("Error while setting the DHT in bootstrap mode")
	} else {
		fmt.Println("Successfully set the DHT in bootstrap mode")
	}
	for _, bootstrapPeers := range dht.GetDefaultBootstrapPeerAddrInfos() {
		err := host.Connect(ctx, bootstrapPeers)
		if err != nil {
			fmt.Println("Error while connecting to :", bootstrapPeers.ID)
		} else {
			fmt.Println("Successfully connected to :", bootstrapPeers.ID)
		}
	}
	fmt.Println("Done with all connections to DHT Bootstrap peers")
}
func EstablishP2P() (context.Context, host.Host) {

	privkey := GenerateKey()
	ctx := context.Background()

	tcpTransport := tcp.NewTCPTransport

	transport, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Identity(privkey),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Transport(tcpTransport),
	)
	if err != nil {
		fmt.Println("Error while creating a new node:", err)
		return ctx, nil
	}

	fmt.Println("Successfully created a new node")
	return ctx, transport
}

func GenerateKey() crypto.PrivKey {
	privkey, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		fmt.Println("Error while generating the key pair:", err)
		return nil
	}

	keyval, _ := peer.IDFromPrivateKey(privkey)
	fmt.Println("PUBLIC KEY:", keyval)
	return privkey
}
func HandleInputStream(stream network.Stream) {
	fmt.Println("New incoming stream detected")
	go sendToStream(stream)
}

var filename_send string = "/home/eya/Desktop/myFile.txt "

func sendToStream(str network.Stream) {
	fmt.Println("Sending file to ", str.Conn().RemotePeer())
	filesize, err := getByteSize(filename_send)
	if err != nil {
		fmt.Println("Error while walking file")
	}
	fmt.Println("Sending file :", filename_send, "Of size-", filesize)
	buffersize := filesize / 10000
	sendBytes := make([]byte, 10000)
	file, err := os.Open(filename_send)
	if err != nil {
		fmt.Println("Error while opening the sending file")
	} else {
		bufstr := bufio.NewWriter(str)
		for i := 0; i < buffersize; i++ {
			_, err = file.Read(sendBytes)
			if err == io.EOF {
				fmt.Println("Send the file completely")
				break
			}
			if err != nil {
				fmt.Println("Error while reading from the file")
			}
			_, err = bufstr.Write(sendBytes)
			if err != nil {
				fmt.Println("Error while sending to the stream")
			}
		}
		leftByte := filesize % 10000
		leftbytebuffer := make([]byte, leftByte)
		_, err = file.Read(leftbytebuffer)
		if err == io.EOF {
			fmt.Println("Send the file completely")
		}
		if err != nil {
			fmt.Println("Error while reading from the file")
		}
		_, err = bufstr.Write(leftbytebuffer)
		if err != nil {
			fmt.Println("Error while sending to the stream")
		}
		fmt.Println("Closing the stream")
		str.Close()
	}

}
func getByteSize(filename string) (int, error) {
	file, err := os.Stat(filename)
	if err != nil {
		fmt.Println("Error while walking the file - file doesn't exist")
		return 0, err
	}
	return int(file.Size()), nil
}

//func ResolveAll(ctx context.Context, host host.Host, top *pubsub.Topic) {
//	mode := flag.Int("mode", 0, " 0 - for normal mode | 1- for mentor mode")
//	flag.Parse()
//
//	if *mode == 1 {
//		str.IsBroadcaster = true
//		go msgpass.BroadCastMentorDetails(ctx, host, top)
//	}
//
//}

var pid string = "/pid/file/share"
var IsBroadcaster bool = false
var isAlreadyRequested bool = false

type Chatmessage struct {
	Messagecontent string
	Messagefrom    peer.ID
	Authorname     string
}

type BroadcastMsg struct {
	MentorNode peer.ID
}

type Packet struct {
	Type         string
	InnerContent []byte
}

type BroadcastRely struct {
	To     peer.ID
	From   peer.ID
	status string
}

type filereceivereq struct {
	Filename string
	Type     string
	From     peer.ID
	Size     int
}

func (frq filereceivereq) newFileRecieveRequest(ctx context.Context, topic *pubsub.Topic) {
	frqbytes, err := json.Marshal(frq)
	if err != nil {
		fmt.Println("Error while marshalling the file recieve request")
	} else {
		pktbytes, err := json.Marshal(Packet{
			Type:         "frq",
			InnerContent: frqbytes,
		})
		if err != nil {
			fmt.Println("Error while marshalling the frq packet")
		} else {
			topic.Publish(ctx, pktbytes)
		}
	}

}

func composeMessage(msg string, host host.Host) *Chatmessage {
	return &Chatmessage{
		Messagecontent: msg,
		Messagefrom:    host.ID(),
		Authorname:     host.ID().String()[len(host.ID().String())-6:],
	}
}

func broadCastReply(ctx context.Context, host host.Host, topic *pubsub.Topic, brdpacket BroadcastMsg) {
	mentorPeerId := brdpacket.MentorNode
	replyPacket := BroadcastRely{
		To:     mentorPeerId,
		From:   host.ID(),
		status: "ready",
	}
	rplypacketbytes, err := json.Marshal(replyPacket)
	if err != nil {
		fmt.Println("Error while marhsalling the brd rply packet")
	} else {
		packet := Packet{
			Type:         "rpl",
			InnerContent: rplypacketbytes,
		}

		packetByte, err := json.Marshal(packet)
		if err != nil {
			fmt.Println("Error while marshalling rplypacket")
		} else {
			topic.Publish(ctx, packetByte)
		}
	}
}

func handleInputFromSubscription(ctx context.Context, host host.Host, sub *pubsub.Subscription, topic *pubsub.Topic) {
	inputPacket := &Packet{}
	for {
		inputMsg, err := sub.Next(ctx)

		if err != nil {
			fmt.Println("Error while getting message from subscription")
		} else {
			err := json.Unmarshal(inputMsg.Data, inputPacket)
			if err != nil {
				fmt.Println("Error while unmarshaling the inputMsg from subscription")
			} else {
				if string(inputPacket.Type) == "brd" {
					if !IsBroadcaster {
						brdpacket := &BroadcastMsg{}
						err := json.Unmarshal(inputPacket.InnerContent, brdpacket)
						if err != nil {
							fmt.Println("Error while unmarshalling brd packet")
						} else {
							fmt.Println("Mentor >", brdpacket.MentorNode)
							broadCastReply(ctx, host, topic, *brdpacket)

						}
					}
				} else if string(inputPacket.Type) == "msg" {
					chatMsg := &Chatmessage{}
					err := json.Unmarshal(inputPacket.InnerContent, chatMsg)
					if err != nil {
						fmt.Println("Error while unmarshalling msg packet")
					} else {
						fmt.Println("[", "BY >", inputMsg.ReceivedFrom.String()[len(inputMsg.ReceivedFrom.String())-6:], "FRM >", chatMsg.Authorname, "]", chatMsg.Messagecontent[:len(chatMsg.Messagecontent)-1])
					}
				} else if string(inputPacket.Type) == "rpl" {
					rplpacket := &BroadcastRely{}
					err := json.Unmarshal(inputPacket.InnerContent, rplpacket)
					if err != nil {
						fmt.Println("Error while unmarshalling rpl packet")
					} else {
						fmt.Println("broadcast reply [", rplpacket.To, "]", "[", rplpacket.From, "]", "[", rplpacket.status, "]")
					}
				} else if string(inputPacket.Type) == "frq" {
					filercvrq := &filereceivereq{}
					err := json.Unmarshal(inputPacket.InnerContent, filercvrq)
					if filercvrq.From != host.ID() {
						fmt.Println("Recieved file recieve request")
						if err != nil {
							fmt.Println("Error while unmarshalling frq packet")
						} else {
							go requestFile(ctx, host, filercvrq.Filename, filercvrq.Type, filercvrq.Size, filercvrq.From, protocol.ID(pid))
						}
					}
				}
			}
		}
	}
}

func requestFile(ctx context.Context, host host.Host, filename string, filetype string, size int, mentr peer.ID, proto protocol.ID) {
	file := fileshare.OpenFileStatusLog()
	time.Sleep(30 * time.Second)
	stream, err := host.NewStream(ctx, mentr, proto)
	if err != nil {
		fmt.Fprintln(file, "Error while handling file request stream")
	}
	ReceivedFromStream(stream, filename, filetype, file, size)
}

func writeToSubscription(ctx context.Context, host host.Host, pubSubTopic *pubsub.Topic) {
	reader := bufio.NewReader(os.Stdin)
	for {
		messg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error while reading from standard input")
		} else {
			fmt.Println(messg[:3])
			if messg[:3] == "<s>" {
				filename := messg[3:]
				fmt.Println("send file ", filename[:len(filename)-1])
				filename_send = filename[:len(filename)-1]
				filename_sep := strings.Split(filename[:len(filename)-1], ".")
				size, _ := getByteSize(filename_sep[0] + "." + filename_sep[1])
				newFrq := filereceivereq{
					Filename: filename_sep[0],
					Type:     filename_sep[1],
					From:     host.ID(),
					Size:     size,
				}
				newFrq.newFileRecieveRequest(ctx, pubSubTopic)
				continue

			}

			chatMsg := composeMessage(messg, host)
			inputCnt, err := json.Marshal(*chatMsg)
			if err != nil {
				fmt.Println("Error while marshaling the chat message")
			}

			pktMsg, err := json.Marshal(Packet{
				Type:         "msg",
				InnerContent: inputCnt,
			})
			if err != nil {
				fmt.Println("Error while marshalling the paket")
			} else {
				pubSubTopic.Publish(ctx, pktMsg)
			}
		}
	}
}

func HandlePubSubMessages(ctx context.Context, host host.Host, sub *pubsub.Subscription, top *pubsub.Topic) {
	go handleInputFromSubscription(ctx, host, sub, top)
	writeToSubscription(ctx, host, top)
}

func ReceivedFromStream(str network.Stream, filename string, filetype string, logfile *os.File, filesize int) {
	fullfilename := filename + "." + filetype
	file, err := os.Create(fullfilename)
	fmt.Fprintln(logfile, "Recieving file from stream to :", str.Conn().RemotePeer())
	buffersize := filesize / 10000
	readBytes := make([]byte, 10000)
	if err != nil {
		fmt.Fprintln(logfile, "Error while creating the recieving file")
	} else {
		bufstr := bufio.NewReader(str)
		for i := 0; i < buffersize; i++ {
			_, err := bufstr.Read(readBytes)
			if err == io.EOF {
				fmt.Fprintln(logfile, "End of file reached")
				break
			}
			if err != nil {
				fmt.Fprintln(logfile, "Error while reading from stream")
				break
			}
			fmt.Print(readBytes)
			_, err = file.Write(readBytes)
			if err != nil {
				fmt.Fprintln(logfile, "Error while writing to the stream")
			}

		}
		leftByte := filesize % 10000
		leftbytebuffer := make([]byte, leftByte)
		_, err := bufstr.Read(leftbytebuffer)
		if err == io.EOF {
			fmt.Fprintln(logfile, "End of file reached after leftbyte read")
		}
		if err != nil {
			fmt.Fprintln(logfile, "Error while reading from stream after loop")
		}
		fmt.Print(leftbytebuffer)
		_, err = file.Write(leftbytebuffer)
		if err != nil {
			fmt.Fprintln(logfile, "Error while writing to the file after loop")
		}
		fmt.Fprintln(logfile, "Completed reading from the stream")
		fmt.Println("Completed reading from the stream")
	}
	file.Close()
	time.Sleep(1 * time.Minute)

}
func ResolveAll(ctx context.Context, host host.Host, top *pubsub.Topic) {
	mode := flag.Int("mode", 0, " 0 - for normal mode | 1- for mentor mode")
	flag.Parse()

	if *mode == 1 {
		IsBroadcaster = true
		go BroadCastMentorDetails(ctx, host, top)
	}

}
func BroadCastMentorDetails(ctx context.Context, host host.Host, topic *pubsub.Topic) {
	broadcastmsg := stream.BroadcastMsg{
		MentorNode: host.ID(),
	}
	boradcastbytes, err := json.Marshal(broadcastmsg)
	if err != nil {
		fmt.Println("Error while marshalling the broadcast message")
	}
	packetMsg := stream.Packet{
		Type:         "brd",
		InnerContent: boradcastbytes,
	}
	packetBytes, err := json.Marshal(packetMsg)
	if err != nil {
		fmt.Println("Error while marshalling the broadcast message")
	}
	for {
		time.Sleep(30 * time.Second)
		err = topic.Publish(ctx, packetBytes)
		if err != nil {
			fmt.Println("Error while publishing the broadcast message")
		}
	}
}
func HandlePubSub(cxt context.Context, host host.Host, topic string) (*pubsub.Subscription, *pubsub.Topic) {
	pubSubServ := newPubSub(cxt, host, topic)
	pubSubTopic, err := pubSubServ.Join(topic)
	if err != nil {
		fmt.Println("Error while joining [", topic, "]")
	} else {
		fmt.Println("Successfull in joining [", topic, "]")
	}
	pubSubSubscription, err := pubSubServ.Subscribe(topic)
	if err != nil {
		fmt.Println("Error while subscribing to")
	}
	return pubSubSubscription, pubSubTopic
}
func newPubSub(cxt context.Context, host host.Host, topic string) *pubsub.PubSub {
	pubSubService, err := pubsub.NewGossipSub(cxt, host)
	if err != nil {
		fmt.Println("Error while establishing a pubsub service")
	} else {
		fmt.Println("Successfully established a new pubsubservice")
	}
	return pubSubService
}

// Sender side function to share a file
func SendFile(ctx context.Context, host host.Host, peerID peer.ID, filename string) error {
	// Open the file you want to share
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Create a new stream to the target peer
	stream, err := host.NewStream(ctx, peerID, protocol.ID("/pid/file/share"))
	if err != nil {
		return fmt.Errorf("error opening stream to peer %s: %w", peerID, err)
	}
	defer stream.Close()

	// Write the file contents to the stream
	_, err = io.Copy(stream, file)
	if err != nil {
		return fmt.Errorf("error sending file data: %w", err)
	}

	fmt.Println("File sent successfully")
	return nil
}

// Receiver side function to handle incoming streams and receive file data
func HandleIncomingStreams(ctx context.Context, host host.Host, logfile *os.File) {
	// Set up stream handler
	host.SetStreamHandler(protocol.ID("/pid/file/share"), func(stream network.Stream) {
		defer stream.Close()

		// Extract filename and other information from stream metadata if needed
		filename := "received_file"
		filetype := "txt"
		filesize := 1024 // example file size

		// Call function to receive file data
		ReceivedFromStream(stream, filename, filetype, logfile, filesize)
	})
}
func OpenFileStatusLog() *os.File {
	var filename_fileshare string = "log/filesharelog"
	var filename_const string = "log/connectionlog"
	i := 0
	for {
		_, err := os.Stat(filename_fileshare + fmt.Sprintf("%d", i) + ".txt")
		if err != nil {
			break
		} else {
			i++
		}
	}
	file, err := os.Create(filename_fileshare + fmt.Sprintf("%d", i) + ".txt")
	if err != nil {
		fmt.Println("Error while opening the file", filename_const)
	}
	fmt.Println("Using [", filename_fileshare, "] for connection status log")
	return file
}
