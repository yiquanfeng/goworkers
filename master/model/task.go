package model

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	Name        string    `json:"name"`
	Command     string    `json:"command"`      // 执行的命令，比如 "echo hello"
	Status      string    `json:"status"`       // pending / running / done / failed
	OwnerID     uint      `json:"owner_id"`     // 创建者
	AssignedTo  uint      `json:"assigned_to"`  // 分配给哪个 worker
	ScheduledAt time.Time `json:"scheduled_at"` // 计划执行时间
}
