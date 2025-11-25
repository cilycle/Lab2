package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type TaskStatus int

const (
	Idle       TaskStatus = 0
	InProgress TaskStatus = 1
	Completed  TaskStatus = 2
)

type Task struct {
	Type      TaskType
	Status    TaskStatus
	StartTime time.Time
	Filename  string
}
type Coordinator struct {
	// Your definitions here.
	files   []string   //the list of file to be processed
	nReduce int        // the number of reduce tasks
	mu      sync.Mutex //lock

	mapTasks    []Task // all map tasks
	reduceTasks []Task // all reduce tasks

	mapDone    bool // are map tasks done
	reduceDone bool // are reduce tasks done
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) GetTask(args *GetTaskArgs, reply *GetTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock() //lock the coordinator, untill the function end it unlock

	if !c.mapDone {
		for i, task := range c.mapTasks {
			if task.Status == Idle {
				reply.Type = TaskMap
				reply.Filename = task.Filename
				reply.TaskID = i
				reply.NReduce = c.nReduce

				c.mapTasks[i].Status = InProgress
				c.mapTasks[i].StartTime = time.Now()
				return nil
			}
		}
		reply.Type = TaskWait
		return nil
	}

	if !c.reduceDone {
		for i, task := range c.reduceTasks {
			if task.Status == Idle {
				reply.Type = TaskReduce
				reply.TaskID = i
				reply.NReduce = c.nReduce

				c.reduceTasks[i].Status = InProgress
				c.reduceTasks[i].StartTime = time.Now()
				return nil
			}
		}
		reply.Type = TaskWait
		return nil
	}

	reply.Type = TaskWait
	return nil
}

func (c *Coordinator) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.Type == TaskMap {
		c.mapTasks[args.TaskID].Status = Completed

		allDone := true
		for _, t := range c.mapTasks {
			if t.Status != Completed {
				allDone = false
				break
			}
		}
		c.mapDone = allDone

	} else if args.Type == TaskReduce {
		c.reduceTasks[args.TaskID].Status = Completed

		allDone := true
		for _, t := range c.reduceTasks {
			if t.Status != Completed {
				allDone = false
				break
			}
		}
		c.reduceDone = allDone
	}

	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	c.mu.Lock()
	defer c.mu.Unlock()

	ret = c.reduceDone

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	c.files = files
	c.nReduce = nReduce
	c.mapDone = false
	c.reduceDone = false

	c.mapTasks = make([]Task, len(files))
	for i, file := range files {
		c.mapTasks[i] = Task{
			Type:     TaskMap,
			Status:   Idle,
			Filename: file,
		}
	}

	c.reduceTasks = make([]Task, nReduce)
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = Task{
			Type:   TaskReduce,
			Status: Idle,
		}
	}

	c.server()
	return &c
}
