package main

import (
	"fmt"
	"go-test/semaphore"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

func ChangeSemaphore() {
	// 配置文件路径在可执行文件同目录下
	wd, _ := os.Getwd()
	configFilePath := filepath.Join(wd, "config.yaml")
	config, err := semaphore.ReadConfig(configFilePath)
	if err != nil {
		fmt.Printf("打开配置文件失败: %v\n", err)
		return
	}
	semaphore.LockKey = config.LockKey
	semaphore.ConcurrentNum = config.ConcurrentNum
	semaphore.LockFilePath = filepath.Join(config.LockFilePath, "lockFile")
	lockFile, err := semaphore.GetLockFile()
	if err != nil {
		log.Fatalf("获取锁文件失败: %v\n", err)
	}
	// 获取写锁（独占锁）
	fmt.Println("获取写锁中，需等待现存任务停止锁占用...")
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX)
	if err != nil {
		fmt.Println("获取写锁失败: ", err)
		return
	}
	defer func() {
		err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		if err != nil {
			fmt.Println("释放写锁失败: ", err)
			return
		}
	}()
	r1, _, err := semaphore.SetSemaphore()
	if int(r1) < 0 {
		log.Fatalf("更新信号量值失败: %v\n", err)
		return
	}
	log.Printf("修改key: %d 信号量，目标值为%d, 修改后实际值为 %d\n", config.LockKey, config.ConcurrentNum, semaphore.SemShow(int(r1)))
}

func main() {
	ChangeSemaphore()
}
