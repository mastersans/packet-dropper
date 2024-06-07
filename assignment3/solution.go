package main

import "fmt"

func main() {
	// Creating a bucket that can hold up to 10 tasks
	taskBucket := make(chan func(), 10)

	// Starting up four worker bees
	for i := 0; i < 4; i++ {
		go func() {
			// Each worker bee keeps an eye on the bucket
			for task := range taskBucket {
				// If there's a task in the bucket, the worker bee grabs it and get it done
				task()
			}
		}()
	}

	// Adding a task to the bucket
	taskBucket <- func() {
		fmt.Println("HERE1")
	}

	// Saying hello
	fmt.Println("Hello")

	// Waiting for all worker bees to finish their tasks before ending the program
	// Without this line, the program might end before all tasks are complete
	close(taskBucket)
}
