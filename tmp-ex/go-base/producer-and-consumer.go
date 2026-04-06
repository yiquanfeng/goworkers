package main

import (
	"fmt"
	"sync"
)

func worker(ch chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for v := range ch {
		v := v * v
		fmt.Println(v)
	}
}

func producer(ch chan int) {
	for i := 0; i < 10; i++ {
		ch <- i
	}
	close(ch)
}

func main() {
	var wg sync.WaitGroup
	ch := make(chan int, 3)
	go producer(ch)
	wg.Add(3)
	go worker(ch, &wg)
	go worker(ch, &wg)
	go worker(ch, &wg)
	wg.Wait()
}
