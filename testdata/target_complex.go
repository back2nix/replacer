package main

import "fmt"

// Old version of SimpleFunc
func SimpleFunc() {
	fmt.Println("Old SimpleFunc from target_complex.go")
}

type Receiver struct {
	Value int
}

// Old version of MethodWithArgs
func (r *Receiver) MethodWithArgs(a int, b string) (bool, error) {
	fmt.Println("Old MethodWithArgs from target_complex.go", r.Value, a, b)
	return false, fmt.Errorf("old error")
}

func KeepThisFunc() {
	fmt.Println("KeepThisFunc from target_complex.go - I should remain.")
}

// This function is not in source_complex.go, should remain.
func TargetSpecificFunc() {
	fmt.Println("TargetSpecificFunc in target_complex.go")
}

// This method exists in source_complex.go, so it should be replaced.
func (s *Service) Serve() {
    fmt.Println("Old Service.Serve from target_complex.go")
}

type Service struct{}

func FuncWithNoReceiver() string { // This one should be replaced
	return "Old FuncWithNoReceiver from target_complex.go"
}

/*
func (r *Receiver) MethodWithArgs(a int, b string) (bool, error) {
	// A commented out version, should not be touched
	fmt.Println("Commented out MethodWithArgs")
	return true, nil
}
*/

// Some code after all functions to test where new functions might be added
var EndMarkerTargetComplexGo = true
