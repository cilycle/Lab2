package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// 1. define Task types
type TaskType int

const (
	TaskMap    TaskType = 0
	TaskReduce TaskType = 1
	TaskWait   TaskType = 2
	TaskExit   TaskType = 3
)

// 2. the arguments Worker use to request a task, should be empty
type GetTaskArgs struct{}

// 3. the reply coordinator to worker
type GetTaskReply struct {
	Type     TaskType
	Filename string //for map task, the filename
	TaskID   int
	NReduce  int //how many reduce task we have
}

type ReportTaskArgs struct {
	Type   TaskType //which type
	TaskID int      // which one
}
type ReportTaskReply struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
