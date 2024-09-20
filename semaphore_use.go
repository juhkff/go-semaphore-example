package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

var key = 2214

func worker(index int, startSignal <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	<-startSignal // 等待启动信号
	//同步处理
	id, err, err2 := SemGet(key) // 在接收到启动信号后调用 semGet
	if int(id) < 0 || err != 0 || err2 != nil {
		log.Fatalf("线程%d 获取信号量组失败: %v, %v\n", index, err.Error(), err2.Error())
		return
	}
	_, _, err = SemLock(int(id))
	if err != 0 {
		log.Fatalf("线程%d 获取锁失败: %v\n", index, err.Error())
	}
	defer func() {
		_, _, err = SemRelease(int(id))
		if err != 0 {
			log.Fatalf("线程%d 释放锁失败: %v\n", index, err.Error())
		}
		fmt.Printf("线程%d 进行锁释放，对应的信号量值为%d\n", index, SemShow(int(id)))
	}()
	fmt.Printf("线程%d 进行锁获取，查询的信号量值为%d\n", index, SemShow(int(id)))
	time.Sleep(1 * time.Second)
}

func main() {
	var wg sync.WaitGroup
	startSignal := make(chan struct{})

	// 启动多个 goroutine
	for {
		for i := 1; i <= 50; i++ {
			wg.Add(1)
			go worker(i, startSignal, &wg)
		}
		// 等待一段时间，然后关闭通道，发送启动信号
		// time.Sleep(3 * time.Second)
		time.Sleep(300 * time.Millisecond)
		close(startSignal)
	}
	// 等待所有 goroutine 完成
	// wg.Wait()
	// fmt.Println("done")
}
