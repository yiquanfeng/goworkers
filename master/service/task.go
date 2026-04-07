package service

import (
	"errors"
	"goworkers/config"
	"goworkers/model"
	"time"
)

func CreateTask(name, command string, owner_id uint, scheduledat time.Time) (*model.Task, error) {
	var task = model.Task{
		Name:        name,
		Command:     command,
		Status:      "pending",
		OwnerID:     owner_id,
		ScheduledAt: scheduledat,
	}
	if result := config.DB.Create(&task); result.Error != nil {
		return nil, errors.New("can not create task")
	}

	return &task, nil
}

func GetTasks(ownerID uint) ([]model.Task, error) {
	var tasks []model.Task
	if result := config.DB.Where("owner_id = ?", ownerID).Find(&tasks); result.Error != nil {
		return nil, result.Error
	}
	return tasks, nil
}

func GetTask(id, ownerID uint) (*model.Task, error) {
	var task model.Task
	if result := config.DB.Where("id = ? AND owner_id = ?", id, ownerID).First(&task); result.Error != nil {
		return nil, errors.New("task not found")
	}
	return &task, nil
}

func UpdateTask(id, ownerID uint, name, command string, scheduledAt time.Time) (*model.Task, error) {
	task, err := GetTask(id, ownerID)
	if err != nil {
		return nil, err
	}
	if result := config.DB.Model(task).Updates(model.Task{Name: name, Command: command, ScheduledAt: scheduledAt}); result.Error != nil {
		return nil, errors.New("can not update task")
	}
	return task, nil
}

func DeleteTask(id, ownerID uint) error {
	if result := config.DB.Where("id = ? AND owner_id = ?", id, ownerID).Delete(&model.Task{}); result.Error != nil {
		return errors.New("can not delete task")
	} else if result.RowsAffected == 0 {
		return errors.New("task not found")
	}
	return nil
}

func GetTaskLogs(taskID, ownerID uint) ([]model.TaskLog, error) {
	if _, err := GetTask(taskID, ownerID); err != nil {
		return nil, err
	}
	var logs []model.TaskLog
	if result := config.DB.Where("task_id = ?", taskID).Find(&logs); result.Error != nil {
		return nil, result.Error
	}
	return logs, nil
}
