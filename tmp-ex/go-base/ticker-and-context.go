package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ticker1 := time.Tick(500 * time.Millisecond)
	ticker2 := time.Tick(1000 * time.Millisecond)
	// timeout := time.After(10 * time.Second)
	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	for {
		select {
		case <-timeout.Done():
			fmt.Println(timeout.Err())
			return
		case <-ticker1:
			fmt.Println("ticker1 trigger")
		case <-ticker2:
			fmt.Println("ticker2 trigger")
		}
	}

}
