package scamp

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

var depfilePath = flag.String("gdep", "./gdep.json", "path to the gdep.json file")

// soaDependencies represents SOA dependencies for the service
type soaDependencies struct {
	Offers   []serviceAction `json:"offers"`
	Requires []soaAction     `json:"requires"`
}

type soaAction struct {
	Action string   `json:"action"`
	Deps   []string `json:"deps"`
}

type serviceAction struct {
	Name string `json:"name"`
}

// ReadDeps reads service dependencies from gdep.yml file
func readDeps(path string) (deps *soaDependencies, err error) {
	f, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("could not open %s: %s ", path, err)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file data was nil")
	}

	err = json.Unmarshal([]byte(data), &deps)
	if err != nil {
		return
	}

	return
}

func checkRequirement(action soaAction) error {
	for _, dep := range action.Deps {
		serviceProxies, err := DefaultCache.SearchByMungedAction(dep)
		if err != nil {
			return fmt.Errorf("dependency error: %s", err)
		}
		if serviceProxies == nil || len(serviceProxies) == 0 {
			return fmt.Errorf("%s not found in discovery cache: %s", dep, err)
		}
	}

	return nil
}
