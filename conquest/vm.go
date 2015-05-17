package conquest

import (
	"github.com/robertkrimen/otto"
	"errors"
)

func RunScript(file string) (conquest *Conquest, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case error:
				err = r.(error)
			case string:
				err = errors.New(r.(string))
			}
		}
	}()
	vm := otto.New()

	conjs := JSConquest{
		vm: vm,
		conquest: &Conquest{
			Proto:         "HTTP/1.1",
			Initials:      map[string]map[string]string{},
			TotalUsers:    10,
			TotalRequests: 100,
		},
	}
	if err = vm.Set("conquest", conjs); err != nil {
		return
	}

	script, err := vm.Compile(file, nil)
	if err != nil {
		return
	}

	_, err = vm.Run(script)
	if err != nil {
		return
	}
	conquest = conjs.conquest
	return
}
