package main

import (
	"go-test/semaphore"
	"log"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <lockKey> <new_value>\n", os.Args[0])
		return
	}

	lockKey, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid lockKey: %v\n", err)
		return
	}

	newValue, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid new_value: %v\n", err)
		return
	}

	semaphore.ConcurrentNum = newValue
	r1, _, err := semaphore.SetSemaphore(lockKey)
	if int(r1) < 0 {
		log.Fatalf("更新信号量值失败: %v\n", err)
		return
	}

	log.Printf("信号量 %d 的目标值为%d, 实际值为 %d\n", lockKey, newValue, semaphore.SemShow(int(r1)))
}
