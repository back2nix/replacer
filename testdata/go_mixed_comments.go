package main

import "fmt"

// func PlainCommentedFunc() {}

/*
func BlockCommentedFunc1() {}
*/

/* Partial block comment */ func BlockCommentedFunc2() { // Эта функция НЕ будет считаться закомментированной, так как блок /* ... */ перед ней закрыт.
	fmt.Println("This function (BlockCommentedFunc2) body")
}

func RealFunction1() { // Эта функция должна быть извлечена
	fmt.Println("Real1")
}

/* Блок комментария здесь.
   Он влияет только на то, что внутри него.
   Поскольку он ЗАКРЫТ этим -> */

func PotentiallyAffectedByUnclosedComment() { // Этот блок НЕ закомментирован, так как предыдущий /* ... */ был закрыт.
	fmt.Println("Affected?")
}

func AnotherUnaffectedFunctionAfterPotentialClosure() { // Этот блок также НЕ закомментирован.
    fmt.Println("Unaffected if block was closed")
}

/*
another block
*/

func FinalFunction() { // Эта функция должна быть извлечена
    fmt.Println("Final")
}
