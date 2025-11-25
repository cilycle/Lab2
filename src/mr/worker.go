package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for {
		args := GetTaskArgs{}
		reply := GetTaskReply{}

		ok := call("Coordinator.GetTask", &args, &reply)

		// if connection fail
		if !ok {
			fmt.Println("Worker: unable to connect to Coordinator，quit")
			return
		}

		switch reply.Type {
		case TaskMap:
			fmt.Printf("Worker: start process Map task %d -> filename: %s\n", reply.TaskID, reply.Filename)

			// 1. read input file
			content, err := os.ReadFile(reply.Filename)
			if err != nil {
				log.Fatalf("unable to read file %v", reply.Filename)
			}

			// 2.
			kva := mapf(reply.Filename, string(content))

			// 3.
			files := make([]*os.File, reply.NReduce)
			encoders := make([]*json.Encoder, reply.NReduce)

			for i := 0; i < reply.NReduce; i++ {
				filename := fmt.Sprintf("mr-%d-%d", reply.TaskID, i)

				file, err := os.Create(filename)
				if err != nil {
					log.Fatalf("unable to creat file %v", filename)
				}

				files[i] = file
				encoders[i] = json.NewEncoder(file)
			}

			// 4. Partition & Write
			for _, kv := range kva {

				bucket := ihash(kv.Key) % reply.NReduce

				err := encoders[bucket].Encode(&kv)
				if err != nil {
					log.Fatalf("write KeyValue fail: %v", err)
				}
			}

			// 5. close
			for _, file := range files {
				file.Close()
			}

			fmt.Printf("Worker: Map task %d finish, report back...\n", reply.TaskID)

			// 6. report
			reportArgs := ReportTaskArgs{
				Type:   TaskMap,
				TaskID: reply.TaskID,
			}
			reportReply := ReportTaskReply{}

			call("Coordinator.ReportTask", &reportArgs, &reportReply)
		case TaskReduce:
			fmt.Printf("Worker: 收到 Reduce 任务 %d\n", reply.TaskID)

			// 1. collect all related files
			pattern := fmt.Sprintf("mr-*-%d", reply.TaskID)
			matches, err := filepath.Glob(pattern)
			if err != nil || len(matches) == 0 {
				log.Fatalf("unable to find Reduce file: %v", pattern)
			}

			// 2.
			intermediate := []KeyValue{}
			for _, filename := range matches {
				file, err := os.Open(filename)
				if err != nil {
					log.Fatalf("unable to open %v", filename)
				}
				dec := json.NewDecoder(file)
				for {
					var kv KeyValue
					if err := dec.Decode(&kv); err != nil {
						break
					}
					intermediate = append(intermediate, kv)
				}
				file.Close()
			}

			// 3.
			sort.Sort(ByKey(intermediate))

			// 4.
			oname := fmt.Sprintf("mr-out-%d", reply.TaskID)
			ofile, _ := os.Create(oname)

			// 5.
			i := 0
			for i < len(intermediate) {
				j := i + 1

				for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
					j++
				}

				values := []string{}
				for k := i; k < j; k++ {
					values = append(values, intermediate[k].Value)
				}

				output := reducef(intermediate[i].Key, values)

				fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

				i = j
			}

			ofile.Close()
			fmt.Printf("Worker: Reduce task %d finish\n", reply.TaskID)

			// 6. report
			reportArgs := ReportTaskArgs{
				Type:   TaskReduce,
				TaskID: reply.TaskID,
			}
			reportReply := ReportTaskReply{}
			call("Coordinator.ReportTask", &reportArgs, &reportReply)

		case TaskWait:
			fmt.Println("Worker: no task assigned,wait for 1 second...")
			time.Sleep(time.Second)

		case TaskExit:
			fmt.Println("Worker: receive exit task, quit")
			return

		default:
			fmt.Println("Worker: unknown task type")
		}
	}
	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
