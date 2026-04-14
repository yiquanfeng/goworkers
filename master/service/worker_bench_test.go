package service

import (
	"context"
	"goworkers/config"
	"goworkers/model"
	"testing"
	"time"
)

// heartbeatDB 还原旧实现，仅用于对照基准测试
func heartbeatDB(workerID uint) (*model.Worker, error) {
	var worker model.Worker
	if result := config.DB.First(&worker, workerID); result.Error != nil {
		return nil, result.Error
	}
	if result := config.DB.Model(&worker).Updates(map[string]any{
		"status":    "online",
		"last_seen": time.Now(),
	}); result.Error != nil {
		return nil, result.Error
	}
	return &worker, nil
}

func setupBench(b *testing.B) (dbWorkerID, redisWorkerID uint) {
	b.Helper()
	config.InitDB()
	config.InitRedis()

	// 旧实现测试用 worker（直接写 DB，不经过 Redis）
	dbWorker := model.Worker{Name: "bench-db", Status: "online", LastSeen: time.Now()}
	config.DB.Create(&dbWorker)

	// 新实现测试用 worker（经过 RegisterWorker 写入 Redis）
	redisWorker, _ := RegisterWorker("bench-redis")

	b.Cleanup(func() {
		config.DB.Unscoped().Delete(&dbWorker)
		config.DB.Unscoped().Delete(redisWorker)
		config.RDB.Del(context.Background(),
			workerInfoKey(redisWorker.ID),
			workerHeartbeatKey(redisWorker.ID),
		)
	})

	return dbWorker.ID, redisWorker.ID
}

// BenchmarkHeartbeat_DB 旧实现：每次心跳 SELECT + UPDATE PostgreSQL
func BenchmarkHeartbeat_DB(b *testing.B) {
	dbWorkerID, _ := setupBench(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		heartbeatDB(dbWorkerID)
	}
}

// BenchmarkHeartbeat_Redis 新实现：每次心跳 EXPIRE Redis key
func BenchmarkHeartbeat_Redis(b *testing.B) {
	_, redisWorkerID := setupBench(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Heartbeat(redisWorkerID)
	}
}

// BenchmarkHeartbeat_DB_Parallel 并发压测旧实现（模拟多 worker 同时心跳）
func BenchmarkHeartbeat_DB_Parallel(b *testing.B) {
	dbWorkerID, _ := setupBench(b)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			heartbeatDB(dbWorkerID)
		}
	})
}

// BenchmarkHeartbeat_Redis_Parallel 并发压测新实现
func BenchmarkHeartbeat_Redis_Parallel(b *testing.B) {
	_, redisWorkerID := setupBench(b)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Heartbeat(redisWorkerID)
		}
	})
}
