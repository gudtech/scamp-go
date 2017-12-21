package scamp

import "fmt"
import "strconv"
import "syscall"

type highResTimestamp float64

func (ts highResTimestamp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%f", ts)), nil
}

func getTimeOfDay() (ts highResTimestamp, err error) {
	var tval syscall.Timeval
	syscall.Gettimeofday(&tval)

	f, err := strconv.ParseFloat(fmt.Sprintf("%d.%d", tval.Sec, tval.Usec), 64)
	if err != nil {
		fmt.Printf("error creating timestamp: `%s`", err)
		return
	}

	ts = highResTimestamp(f)
	return
}
