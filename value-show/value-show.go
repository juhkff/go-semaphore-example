package main

import (
	"fmt"
	"go-test/semaphore"
)

func main() {
	id, _, _ := semaphore.SemGet() // 在接收到启动信号后调用 semGet
	fmt.Printf("查询的信号量值为%d\n", semaphore.SemShow(int(id)))
	// wd, _ := os.Getwd()
	// fmt.Printf("目录:%s\n", wd)
}
