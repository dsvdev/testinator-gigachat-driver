package main

import (
	"fmt"
	"github.com/dsvdev/testinator-gigachat-driver/pkg/llm"
)

func main() {
	d := llm.NewGigachatDriver()
	fmt.Println(d.SendRequest("Привет! Я тестирую твой API и если ты пришлешь ответ это значит что все сработало!"))
}
