package main

import "fmt"

// A simple function
func SimpleFunc() {
	fmt.Println("New SimpleFunc from source_complex.go")
}

type Receiver struct {
	Value int
}

// MethodWithArgs has arguments and return values
func (r *Receiver) MethodWithArgs(a int, b string) (bool, error) {
	fmt.Println("New MethodWithArgs from source_complex.go", r.Value, a, b)
	return true, nil
}

func AnotherFunc() { // With a comment on the same line
	fmt.Println("New AnotherFunc from source_complex.go")
}

// This function is only in source, should be added to target.
func SimpleFuncNeighbor() {
	fmt.Println("SimpleFuncNeighbor from source_complex.go, to be added.")
}

func (s *Service) Serve() { // Overlap with original source.go
    fmt.Println("Service is serving from source_complex.go!")
}

type Service struct{} // Ensure type definition doesn't break parsing

func FuncWithNoReceiver() string {
	return "FuncWithNoReceiver from source_complex.go"
}

// func CommentedOutFunc() {
//	fmt.Println("This should not be extracted or replaced")
// }

/*
func MultiLineCommentedOutFunc() {
	fmt.Println("This also should not be extracted or replaced")
}
*/
