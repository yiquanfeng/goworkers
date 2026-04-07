package service

import (
	"errors"
	"goworkers/config"
	"goworkers/model"
	"time"
)

func RegisterWorker(name string) (*model.Worker, error) {
	worker := model.Worker{
		Name:     name,
		Status:   "online",
		LastSeen: time.Now(),
	}
	if result := config.DB.Create(&worker); result.Error != nil {
		return nil, errors.New("can not register worker")
	}
	return &worker, nil
}

func Heartbeat(workerID uint) (*model.Worker, error) {
	var worker model.Worker
	if result := config.DB.First(&worker, workerID); result.Error != nil {
		return nil, errors.New("worker not found")
	}
	if result := config.DB.Model(&worker).Updates(map[string]any{
		"status":    "online",
		"last_seen": time.Now(),
	}); result.Error != nil {
		return nil, errors.New("can not update heartbeat")
	}
	return &worker, nil
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
