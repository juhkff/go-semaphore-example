package main

import (
	"fmt"
	"go-test/semaphore"
	"log"
)

var configFile = "/home/juhkff/projects/go-test/console/config.yaml"

type Config struct {
	LockKey       int `yaml:"lockKey"`
	ConcurrentNum int `yaml:"concurrentNum"`
}

func main() {
	config, err := semaphore.ReadConfig(configFile)
	if err != nil {
		fmt.Printf("打开配置文件失败: %v\n", err)
		return
	}
	semaphore.LockKey = config.LockKey
	semaphore.ConcurrentNum = config.ConcurrentNum
	r1, _, err := semaphore.SetSemaphore()
	if int(r1) < 0 {
		log.Fatalf("更新信号量值失败: %v\n", err)
		return
	}

	log.Printf("key: %d 信号量的目标值为%d, 实际值为 %d\n", config.LockKey, config.ConcurrentNum, semaphore.SemShow(int(r1)))
}
