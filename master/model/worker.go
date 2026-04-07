package model

import (
	"gorm.io/gorm"
	"time"
)

// Worker 节点
type Worker struct {
	gorm.Model
	Name     string    `json:"name"`      // worker 名称
	Status   string    `json:"status"`    // online / offline
	LastSeen time.Time `json:"last_seen"` // 最后心跳时间
}
