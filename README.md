# MIT 6.5840 Lab 2: Distributed MapReduce

这是一个基于 Go 语言实现的分布式 MapReduce 系统，完成了 MIT 6.5840 (原 6.824) 分布式系统课程的 Lab 2 作业。

该项目实现了一个简单的 MapReduce 框架，由一个 Coordinator（协调者）进程和多个 Worker（工作者）进程组成，能够并行处理大规模文本数据（如 WordCount）。

## 📂 核心文件与功能实现 (Implementation Details)

本项目主要修改了 `src/mr` 目录下的三个核心文件，具体新增功能如下：

### 1. `src/mr/rpc.go` (通信协议)
**功能描述**：定义了 Coordinator 与 Worker 之间 RPC 通信的数据结构。
* **新增 `GetTaskArgs/Reply`**：
    * 定义了请求任务的参数（空）和响应参数（包含任务类型 `TaskType`、文件名 `Filename`、任务编号 `TaskID`、Reduce 数量 `NReduce`）。
* **新增 `ReportTaskArgs/Reply`**：
    * 定义了 Worker 汇报任务完成的参数（汇报完成的任务类型和 ID）。
* **新增 `TaskType` 枚举**：
    * 区分 `TaskMap`, `TaskReduce`, `TaskWait`, `TaskExit`。

### 2. `src/mr/coordinator.go` (协调者/大脑)
**功能描述**：负责任务调度、状态管理和并发控制。
* **状态机管理**：
    * 维护 `mapTasks` 和 `reduceTasks` 两个任务列表。
    * 每个任务包含状态：`Idle` (闲置), `InProgress` (进行中), `Completed` (已完成)。
* **`GetTask` RPC 方法**：
    * **Map 阶段**：优先分配 Idle 状态的 Map 任务。
    * **Reduce 阶段**：当所有 Map 任务完成后，分配 Idle 状态的 Reduce 任务。
    * **等待机制**：如果任务都在进行中但未完成，通知 Worker 等待 (`TaskWait`)。
    * **退出机制**：所有任务完成后，通知 Worker 退出 (`TaskExit`)。
* **`ReportTask` RPC 方法**：
    * 接收 Worker 的完成汇报，将对应任务标记为 `Completed`。
    * 实时检测 Map 阶段和 Reduce 阶段是否全部结束。
* **并发安全**：使用 `sync.Mutex` 互斥锁，确保多 Worker 并发请求时数据的一致性。

### 3. `src/mr/worker.go` (工作者/执行者)
**功能描述**：向 Coordinator 索取任务并执行具体的业务逻辑。
* **Worker 主循环**：不断通过 RPC 请求任务，根据 `TaskType` 执行不同逻辑。
* **Map 任务处理**：
    * 读取输入文件内容。
    * 调用插件函数 `mapf` 生成 KeyValue 数组。
    * **Hash 分区**：使用 `ihash(key) % NReduce` 算法，将 KeyValue 分配到 `mr-X-Y` 个中间文件中。
    * **JSON 序列化**：将中间结果以 JSON 格式写入磁盘。
* **Reduce 任务处理**：
    * **文件收集**：读取所有相关的中间文件 `mr-*-Y`。
    * **排序**：在内存中按 Key 对数据进行排序。
    * **归约**：调用插件函数 `reducef` 聚合数据。
    * **结果输出**：将最终结果写入 `mr-out-Y` 文件。

---

## 🚀 快速开始 (Usage)

### 前置要求
* Linux 环境 (推荐 WSL2 或 Docker)
* Go 1.15+

### 第一步：编译插件
进入 `src/main` 目录，将 MapReduce 应用程序（如 WordCount）编译为 Go 插件：
```bash
cd src/main
go build -buildmode=plugin ../mrapps/wc.go
```

### 第二步：启动 Coordinator (协调者)
运行 Coordinator 并传入输入文件（例如 Project Gutenberg 的电子书）：
```bash
# 建议先清理旧文件
rm mr-out* mr-*-* # 启动协调者
go run mrcoordinator.go pg-*.txt
```

### 第三步：启动 Worker (工作者)
打开两个新的终端窗口，运行 Worker 加载插件：
```bash
cd src/main
go run mrworker.go wc.so
```

### 第四步：验证结果
等待 Coordinator 和 Worker 自动退出后，查看生成的输出文件：
```bash
cat mr-out-* | sort | more
```

### 测试脚本
```bash
bash test-mr.sh
```




