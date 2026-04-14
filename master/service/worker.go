package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"goworkers/config"
	"goworkers/model"
	"time"
)

const heartbeatTTL = 30 * time.Second

func workerInfoKey(id uint) string      { return fmt.Sprintf("worker:info:%d", id) }
func workerHeartbeatKey(id uint) string { return fmt.Sprintf("worker:heartbeat:%d", id) }

func RegisterWorker(name string) (*model.Worker, error) {
	worker := model.Worker{
		Name:     name,
		Status:   "online",
		LastSeen: time.Now(),
	}
	if result := config.DB.Create(&worker); result.Error != nil {
		return nil, errors.New("can not register worker")
	}

	ctx := context.Background()
	data, _ := json.Marshal(worker)
	config.RDB.Set(ctx, workerInfoKey(worker.ID), data, 0)
	config.RDB.Set(ctx, workerHeartbeatKey(worker.ID), 1, heartbeatTTL)

	return &worker, nil
}

func Heartbeat(workerID uint) (*model.Worker, error) {
	ctx := context.Background()

	// 刷新心跳 TTL
	refreshed, err := config.RDB.Expire(ctx, workerHeartbeatKey(workerID), heartbeatTTL).Result()
	if err != nil || !refreshed {
		// key 不存在说明 worker 从未注册或太久未心跳，回查 DB
		var worker model.Worker
		if result := config.DB.First(&worker, workerID); result.Error != nil {
			return nil, errors.New("worker not found")
		}
		data, _ := json.Marshal(worker)
		config.RDB.Set(ctx, workerInfoKey(workerID), data, 0)
		config.RDB.Set(ctx, workerHeartbeatKey(workerID), 1, heartbeatTTL)
		return &worker, nil
	}

	// 从 Redis 读取 worker 信息，避免查 DB
	data, err := config.RDB.Get(ctx, workerInfoKey(workerID)).Bytes()
	if err != nil {
		var worker model.Worker
		if result := config.DB.First(&worker, workerID); result.Error != nil {
			return nil, errors.New("worker not found")
		}
		return &worker, nil
	}

	var worker model.Worker
	json.Unmarshal(data, &worker)
	return &worker, nil
}

// IsWorkerOnline 通过 Redis TTL 判断 worker 是否在线
func IsWorkerOnline(workerID uint) bool {
	return config.RDB.Exists(context.Background(), workerHeartbeatKey(workerID)).Val() == 1
}

// SyncWorkerStatus 将 Redis 在线状态同步回 DB，供后台定期调用
func SyncWorkerStatus() {
	var workers []model.Worker
	if config.DB.Find(&workers).Error != nil {
		return
	}
	for _, w := range workers {
		newStatus := "offline"
		if IsWorkerOnline(w.ID) {
			newStatus = "online"
		}
		if w.Status != newStatus {
			config.DB.Model(&w).Updates(map[string]any{
				"status":    newStatus,
				"last_seen": time.Now(),
			})
		}
	}
}

func GetNextTask(workerID uint) (*model.Task, error) {
	var task model.Task
	result := config.DB.Where("status = ? AND assigned_to = 0", "pending").
		Order("scheduled_at asc").
		First(&task)
	if result.Error != nil {
		return nil, errors.New("no pending task")
	}
	if result := config.DB.Model(&task).Updates(map[string]any{
		"status":      "running",
		"assigned_to": workerID,
	}); result.Error != nil {
		return nil, errors.New("can not assign task")
	}
	return &task, nil
}

func SubmitLog(taskID, workerID uint, level, message string) (*model.TaskLog, error) {
	log := model.TaskLog{
		TaskID:   taskID,
		WorkerID: workerID,
		Level:    level,
		Message:  message,
	}
	if result := config.DB.Create(&log); result.Error != nil {
		return nil, errors.New("can not submit log")
	}
	return &log, nil
}

func CompleteTask(taskID, workerID uint) (*model.Task, error) {
	var task model.Task
	if result := config.DB.Where("id = ? AND assigned_to = ? AND status = ?", taskID, workerID, "running").First(&task); result.Error != nil {
		return nil, errors.New("task not found or not assigned to this worker")
	}
	if result := config.DB.Model(&task).Update("status", "done"); result.Error != nil {
		return nil, errors.New("can not complete task")
	}
	return &task, nil
}

func FailTask(taskID, workerID uint) (*model.Task, error) {
	var task model.Task
	if result := config.DB.Where("id = ? AND assigned_to = ? AND status = ?", taskID, workerID, "running").First(&task); result.Error != nil {
		return nil, errors.New("task not found or not assigned to this worker")
	}
	if result := config.DB.Model(&task).Update("status", "failed"); result.Error != nil {
		return nil, errors.New("can not fail task")
	}
	return &task, nil
}
