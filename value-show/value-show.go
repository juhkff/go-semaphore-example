package main

import (
	"fmt"
	"go-test/semaphore"
)

var key = 2216

func main() {
	id, _, _ := semaphore.SemGet(key) // 在接收到启动信号后调用 semGet
	fmt.Printf("查询的信号量值为%d\n", semaphore.SemShow(int(id)))
}
