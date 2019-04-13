package utils

import "time"

// SetInterval ...
// Source: https://www.loxodrome.io/post/set-timeout-interval-go/
func SetInterval(someFunc func(), milliseconds int, async bool) chan bool {

	// How often to fire the passed in function
	// in milliseconds
	interval := time.Duration(milliseconds) * time.Millisecond

	// Setup the ticket and the channel to signal
	// the ending of the interval
	ticker := time.NewTicker(interval)
	clear := make(chan bool)

	// Put the selection in a go routine
	// so that the for loop is none blocking
	go func() {
		for {

			select {
			case <-ticker.C:
				if async {
					// This won't block
					go someFunc()
				} else {
					// This will block
					someFunc()
				}
			case <-clear:
				ticker.Stop()
				return
			}

		}
	}()

	// We return the channel so we can pass in
	// a value to it to clear the interval
	return clear

}

// SetTimeout ...
// Source: https://www.loxodrome.io/post/set-timeout-interval-go/
func SetTimeout(someFunc func(), milliseconds int) {

	timeout := time.Duration(milliseconds) * time.Millisecond

	// This spawns a goroutine and therefore does not block
	time.AfterFunc(timeout, someFunc)

}
