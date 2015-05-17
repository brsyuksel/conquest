package conquest

import (
	"errors"
	"github.com/robertkrimen/otto"
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
		vm:       vm,
		conquest: NewConquest(),
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
