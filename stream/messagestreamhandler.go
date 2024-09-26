package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Mina218/FileSharingNetwork/fileshare"
	"os"
	"strings"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

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
