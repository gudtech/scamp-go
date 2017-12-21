package scamp

import (
	"encoding/json"
	"fmt"
	"os"
)

func panicjson(thing interface{}) {
	thingBytes, _ := json.Marshal(thing)
	fmt.Printf("%s\n", thingBytes)
	os.Exit(1)
}
