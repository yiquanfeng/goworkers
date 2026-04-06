package main

import (
	"fmt"
	"sync"
	"os"
	"strconv"
)

func main() {
	var wg sync.WaitGroup
	count, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("err", err)
		return
	}
	for i:= 0; i<count;i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("worker %d\n", id)
		}(i)
	}
	wg.Wait()
}