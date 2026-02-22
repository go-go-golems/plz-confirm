package main

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

func main() {
	vm := goja.New()
	done := make(chan error, 1)

	go func() {
		_, err := vm.RunString(`
var n = 0;
while (true) {
  n++;
}
`)
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	vm.Interrupt("deadline exceeded")

	select {
	case err := <-done:
		if err == nil {
			fmt.Println("unexpected: script ended without interruption")
			return
		}
		fmt.Printf("interrupted err type=%T\n", err)
		fmt.Printf("interrupted err=%v\n", err)
	case <-time.After(2 * time.Second):
		fmt.Println("timeout waiting for interrupted run to return")
	}
}
