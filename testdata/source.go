package main

import "fmt"

func Hello() {
    fmt.Println("Hello from source!")
}

func (s *Service) Serve() {
    fmt.Println("Service is serving from source!")
}

type Service struct{}
