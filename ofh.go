package main

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lnanomsg
// #include <nanomsg/track.h>
import "C"

import (
	"fmt"
	"io/ioutil"
	"time"
)

// return the number of open files a process has
func OpenFileHandles(jobservPid int) []string {

	// wait until process shows up in /proc
	waited := 0
	for {
		pt := *ProcessTable()
		_, jsAlive := pt[jobservPid]
		if jsAlive {
			break
		}
		time.Sleep(50 * time.Millisecond)
		waited++
		if waited > 10 {
			panic(fmt.Sprintf("jobserv with expected pid %d did not show up in /proc after 10 waits", jobservPid))
		}
	}
	fmt.Printf("\njobserv with expected pid %d was *found* in /proc after %d waits of 50msec\n", jobservPid, waited)

	// read all the open files
	fileInfoSlice, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", jobservPid))
	if err != nil {
		panic(err)
	}

	res := make([]string, len(fileInfoSlice))
	for i := range fileInfoSlice {
		res[i] = fileInfoSlice[i].Name()
	}

	return res
}

func call_efdtr_dump(i int) {
   fmt.Printf("PRE call to efdtr_dump(), i = %d\n", i)
   C.efdtr_dump(0, nil)
   fmt.Printf("POST call to efdtr_dump(), i = %d\n", i)
}