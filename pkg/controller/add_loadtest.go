package controller

import (
	"github.com/verfio/fortio-operator/pkg/controller/loadtest"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, loadtest.Add)
}
