package main

import (
	"fmt"
	"sync"
)

type LogLine struct {
	Level   string
	Message string
}

type Stats struct {
	Info  int
	Warn  int
	Error int
}

func generate(logs []LogLine) <-chan LogLine {
	ch := make(chan LogLine, 3)
	go func() {
		for _, s := range logs {
			ch <- s
		}
		close(ch)
	}()
	return ch
}

func analyze(id int, in <-chan LogLine) <-chan Stats {
	ch := make(chan Stats, 3)

	go func() {
		for s := range in {
			fmt.Printf("[analyzer %d] %s: %s\n", id, s.Level, s.Message)
			switch s.Level {
			case "INFO":
				{
					ch <- Stats{1, 0, 0}
				}
			case "WARN":
				{
					ch <- Stats{0, 1, 0}
				}
			case "ERROR":
				{
					ch <- Stats{0, 0, 1}
				}
			}
		}
		close(ch)
	}()

	return ch
}

func merge(channels ...<-chan Stats) <-chan Stats {
	channel := make(chan Stats, 3)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan Stats) {
			defer wg.Done()
			for s := range c {
				channel <- s
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(channel)
	}()

	return channel
}

func main() {
	logs := []LogLine{
		{"ERROR", "登录失败 user=alice"},
		{"INFO", "服务启动成功"},
		{"WARN", "磁盘使用率 85%"},
	}

	source := generate(logs)

	ch1 := analyze(1, source)
	ch2 := analyze(2, source)
	ch3 := analyze(3, source)

	merged := merge(ch1, ch2, ch3)

	var total Stats
	for s := range merged {
		total.Info += s.Info
		total.Error += s.Error
		total.Warn += s.Warn
	}
	fmt.Printf("info: %d, error, %d, warn: %d \n", total.Info, total.Error, total.Warn)
}
