package main

import "fmt"

func Hello() {
    fmt.Println("Hello from target!")
}

func Bye() {
    fmt.Println("Goodbye from target!")
}

type Service struct{}

func (s *Service) OldServe() {
    fmt.Println("Old serving logic")
}
