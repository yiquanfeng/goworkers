package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var (
	masterURL string
	workerID  uint
	taskBuf   = make(chan Task, 10)
)

// Task mirrors master 的 model.Task，只取需要的字段
type Task struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

type workerResp struct {
	ID uint `json:"id"`
}

func post(path string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return http.Post(masterURL+path, "application/json", bytes.NewReader(data))
}

func Register() {
	hostname, _ := os.Hostname()
	resp, err := post("/api/workers", map[string]string{"name": hostname})
	if err != nil {
		fmt.Println("[worker] register failed:", err)
		return
	}
	defer resp.Body.Close()

	var w workerResp
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		fmt.Println("[worker] register decode failed:", err)
		return
	}
	workerID = w.ID
	fmt.Printf("[worker] registered, id=%d\n", workerID)
}

func Heartbeat() {
	url := fmt.Sprintf("%s/api/workers/%d/heartbeat", masterURL, workerID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Println("[worker] heartbeat failed:", err)
		return
	}
	resp.Body.Close()
}

func GetNewTask() {
	url := fmt.Sprintf("%s/api/workers/%d/next-task", masterURL, workerID)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return
	}
	// 非阻塞写入缓冲
	select {
	case taskBuf <- task:
		fmt.Printf("[worker] task queued: id=%d name=%s\n", task.ID, task.Name)
	default:
		fmt.Println("[worker] task buffer full, skip")
	}
}

func submitLog(taskID uint, level, message string) {
	url := fmt.Sprintf("%s/api/workers/%d/logs", masterURL, workerID)
	post(url, map[string]any{
		"task_id": taskID,
		"level":   level,
		"message": message,
	})
}

func Execute(task Task) {
	fmt.Printf("[worker] executing task id=%d: %s\n", task.ID, task.Command)

	cmd := exec.Command("sh", "-c", task.Command)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		submitLog(task.ID, "ERROR", "start failed: "+err.Error())
		failTask(task.ID)
		return
	}

	// 收集 stdout/stderr 并上报
	drain := func(r io.Reader, level string) {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				submitLog(task.ID, level, string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}
	go drain(stdout, "INFO")
	go drain(stderr, "WARN")

	if err := cmd.Wait(); err != nil {
		submitLog(task.ID, "ERROR", "exit error: "+err.Error())
		failTask(task.ID)
		return
	}

	completeTask(task.ID)
	fmt.Printf("[worker] task id=%d done\n", task.ID)
}

func completeTask(taskID uint) {
	url := fmt.Sprintf("%s/api/workers/%d/tasks/%d/complete", masterURL, workerID, taskID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Println("[worker] complete task failed:", err)
		return
	}
	resp.Body.Close()
}

func failTask(taskID uint) {
	url := fmt.Sprintf("%s/api/workers/%d/tasks/%d/fail", masterURL, workerID, taskID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Println("[worker] fail task failed:", err)
		return
	}
	resp.Body.Close()
}

func Setup(master string) {
	masterURL = master

	// 1. 注册，拿到 workerID
	Register()

	// 2. 每 10s 发送心跳
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			Heartbeat()
		}
	}()

	// 3. 每 5s 拉取一个新任务放入缓冲
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			GetNewTask()
		}
	}()

	// 4. 从缓冲中取任务并执行
	for task := range taskBuf {
		go Execute(task)
	}
}
