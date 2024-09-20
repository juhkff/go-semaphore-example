package main

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

/*
#include <sys/sem.h>
#include <sys/types.h> // Add this line to include the necessary header file
#include <sys/ipc.h> // Add this line to include the necessary header file
typedef struct sembuf sembuf;
// 手动定义semun联合体
union semun {
    int val;                // Value for SETVAL
    struct semid_ds *buf;   // Buffer for IPC_STAT, IPC_SET
    unsigned short *array;  // Array for GETALL, SETALL
    struct seminfo *__buf;  // Buffer for IPC_INFO (Linux特有)
};

// 由于semctl是一个variadic函数，需要一个包装函数来正确传递semun参数
int semctl_setval(int semid, int semnum, int cmd, int val) {
    union semun arg;
    arg.val = val;
    return semctl(semid, semnum, cmd, arg);
}
*/
import "C"

var key = 6353
var concurrentNum = 30

// var semId = semget(key)
var semId int

var lockKey = 7799

var setLock = 1

// var lockId = put_on_out_function()
var lockId uintptr

// 放在外部函数中
func put_on_out_function() {
	r1, _, err := syscall.Syscall(syscall.SYS_SEMGET, uintptr(lockKey), uintptr(1), uintptr(C.IPC_CREAT|00666))
	if int(r1) < 0 {
		log.Printf("lock信号量创建出错: %v\n", err)
		// todo: 重试机制
		return
	}
	lockId = r1
	r2 := uintptr(C.semctl_setval(C.int(r1), 0, C.SETVAL, C.int(setLock)))
	if int(r2) < 0 {
		log.Printf("lock信号量设值失败\n")
		//todo: 重试机制
		return
	}
	r1, _, _ = syscall.Syscall(syscall.SYS_SEMCTL, uintptr(r1), 0, uintptr(C.GETVAL))
	fmt.Printf("信号量值为%d\n", r1)
}

func qualifyLock(index int) (r1 uintptr, r2 uintptr, err syscall.Errno) {
	fmt.Printf("线程%d 开始qualifyLock\n", index)
	r1, r2, err = syscall.Syscall(syscall.SYS_SEMGET, uintptr(key), uintptr(1), uintptr(C.IPC_CREAT|00666))
	if int(r1) < 0 {
		fmt.Printf("线程%d qualifyLock 失败！%v\n", index, err)
	}
	// 使用包装函数调用semctl
	ret := C.semctl_setval(C.int(r1), 0, C.SETVAL, C.int(concurrentNum))
	if ret < 0 {
		log.Printf("线程%d 设值失败\n", index)
	} else {
		log.Printf("线程%d 设值成功, 现信号量值为%d\n", index, semShow(index, int(r1)))
	}
	return
}

func semget(index int) int {
	//线程安全
	r1, _, _ := syscall.Syscall(syscall.SYS_SEMGET, uintptr(key), uintptr(1), uintptr(00666))
	if int(r1) < 0 {
		for !lockGet() {
		}
		log.Printf("线程%d 获取互斥锁成功，开始 qualifyLock\n", index)
		r1, _, _ = syscall.Syscall(syscall.SYS_SEMGET, uintptr(key), uintptr(1), uintptr(00666))
		if int(r1) < 0 {
			r1, _, _ = qualifyLock(index)
		} else {
			log.Printf("线程%d 无需再创建信号量\n", index)
		}
		// 释放锁
		for !lockRelease() {
		}
		log.Printf("线程%d 释放互斥锁成功，结束 qualifyLock\n", index)
	}
	return int(r1)
}

func lockGet() bool {
	stSemBuf := C.sembuf{
		sem_num: 0,
		sem_op:  -1,
		sem_flg: C.IPC_NOWAIT | C.SEM_UNDO,
	}

	r1, _, _ := syscall.Syscall(syscall.SYS_SEMOP, uintptr(lockId), uintptr(unsafe.Pointer(&stSemBuf)), 1)
	return int(r1) == 0
}

func lockRelease() bool {
	stSemBuf := C.sembuf{
		sem_num: 0,
		sem_op:  1,
		sem_flg: C.IPC_NOWAIT | C.SEM_UNDO,
	}
	r1, _, _ := syscall.Syscall(syscall.SYS_SEMOP, uintptr(lockId), uintptr(unsafe.Pointer(&stSemBuf)), 1)
	return int(r1) == 0
}

func semLock(index int, semid int) int {

	stSemBuf := C.sembuf{
		sem_num: 0,
		sem_op:  -1,
		sem_flg: C.SEM_UNDO,
	}

	r1, r2, err := syscall.Syscall(syscall.SYS_SEMOP, uintptr(semid), uintptr(unsafe.Pointer(&stSemBuf)), 1)
	if int(r1) < 0 {
		log.Printf("线程%d 请求信号量出错: %v,%v,%v\n", index, r1, r2, err)
	}
	return int(r1)
}

func semRelease(index int, semid int) int {
	stSemBuf := C.sembuf{
		sem_num: 0,
		sem_op:  1,
		sem_flg: C.SEM_UNDO,
	}

	r1, r2, err := syscall.Syscall(syscall.SYS_SEMOP, uintptr(semid), uintptr(unsafe.Pointer(&stSemBuf)), 1)
	if int(r1) < 0 {
		log.Printf("线程%d 请求信号量出错: %v,%v,%v\n", index, r1, r2, err)
	}
	return int(r1)
}

func semShow(index int, semid int) int {
	r1, r2, err := syscall.Syscall(syscall.SYS_SEMCTL, uintptr(semid), 0, uintptr(C.GETVAL))
	if int(r1) < 0 {
		log.Printf("线程%d 查值出错: %v,%v,%v on id %d\n", index, r1, r2, err, semid)
	}
	return int(r1)
}

func worker(index int, startSignal <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	<-startSignal       // 等待启动信号
	id := semget(index) // 在接收到启动信号后调用 semget
	fmt.Printf("线程%d 的对应的信号量值为%d, lock信号量结果: %d\n", index, semShow(index, id), semLock(index, id))
	time.Sleep(1 * time.Second)
	fmt.Printf("线程%d 的对应的信号量值为%d, release信号量结果: %d\n", index, semShow(index, id), semRelease(index, id))
}

func main() {
	put_on_out_function()
	semId = semget(key)
	var wg sync.WaitGroup
	startSignal := make(chan struct{})

	// 启动多个 goroutine
	for i := 1; i <= 50; i++ {
		wg.Add(1)
		go worker(i, startSignal, &wg)
	}

	// 等待一段时间，然后关闭通道，发送启动信号
	time.Sleep(3 * time.Second)
	close(startSignal)

	// 等待所有 goroutine 完成
	wg.Wait()
	fmt.Println("done")
}
