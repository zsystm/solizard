package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// create signal channel for handling program termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	// catch signals and handle program termination
	go func() {
		sig := <-sigs
		fmt.Printf("terminating program... (reason: %v)\n", sig)
		done <- true
	}()
	// run the main program
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("terminating program... (reason: %v)\n", r)
				done <- true
			}
		}()
		err := Run()
		if err != nil {
			fmt.Printf("terminating program... (reason: %v)\n", err)
			done <- true
		}
	}()

	<-done
	fmt.Println("terminated")
}
