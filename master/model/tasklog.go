package model

import (
	"gorm.io/gorm"
)

// 任务日志
type TaskLog struct {
	gorm.Model
	TaskID   uint   `json:"task_id"`
	WorkerID uint   `json:"worker_id"`
	Level    string `json:"level"` // INFO / WARN / ERROR
	Message  string `json:"message"`
}
