package main

import (
	"log"
	"os"
	"syscall"
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

var lockFile = "../go-test/lockFile"

// 并发量
var concurrentNum = 30

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// todo :返回值为r1和val的情况
func setSemaphore(key int) (r1 uintptr, r2 uintptr, err syscall.Errno) {
	r1, r2, err = syscall.Syscall(syscall.SYS_SEMGET, uintptr(key), uintptr(1), uintptr(C.IPC_CREAT|00666))
	if int(r1) < 0 {
		log.Printf("信号量初始化失败: %v\n", err)
		return
	}
	val := C.semctl_setval(C.int(r1), 0, C.SETVAL, C.int(concurrentNum))
	if val < 0 {
		//todo :打印errno
		log.Printf("信号量设值失败\n")
	} else {
		//todo :测试后删除
		defer log.Printf("信号量设值成功, 值为%d\n", SemShow(int(r1)))
	}
	return
}

func getLockFile() (file *os.File, err error) {
	if !fileExists(lockFile) {
		file, err = os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			if os.IsExist(err) {
				//lock_file已被其它进程创建
				file, err = os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE, 0666)
			} else {
				log.Printf("创建锁文件失败: %v\n", err)
			}
		}
	} else {
		file, err = os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("获取锁文件失败: %v\n", err)
		}
	}
	return
}

// SemGet 获取信号量组ID
// return : r1-semId, err-error, err2-syscall.Errno
func SemGet(key int) (r1 uintptr, err syscall.Errno, err2 error) {
	r1, _, err = syscall.Syscall(syscall.SYS_SEMGET, uintptr(key), uintptr(1), uintptr(00666))
	//第一次运行需要初始化信号量
	if int(r1) < 0 {
		//创建或获取文件用于锁
		//todo :异常处理
		file, _ := getLockFile()
		//获取锁
		err2 = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if err2 != nil {
			log.Printf("文件锁Lock失败: %v\n", err2)
			return
		}
		//确保释放锁
		defer func() {
			err2 = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
			if err2 != nil {
				log.Printf("文件锁Unlock失败: %v\n", err2)
				return
			}
		}()
		r1, _, err = syscall.Syscall(syscall.SYS_SEMGET, uintptr(key), uintptr(1), uintptr(00666))
		//初始化信号量
		if int(r1) < 0 {
			r1, _, err = setSemaphore(key)
		}
	}
	return
}

func SemLock(semId int) (r1 uintptr, r2 uintptr, err syscall.Errno) {
	stSemBuf := C.sembuf{
		sem_num: 0,
		sem_op:  -1,
		sem_flg: C.IPC_NOWAIT | C.SEM_UNDO,
	}

	r1, r2, err = syscall.Syscall(syscall.SYS_SEMOP, uintptr(semId), uintptr(unsafe.Pointer(&stSemBuf)), 1)
	if int(r1) < 0 {
		log.Printf("请求锁出错: %v\n", err.Error())
	}
	return
}

func SemRelease(semId int) (r1 uintptr, r2 uintptr, err syscall.Errno) {
	stSemBuf := C.sembuf{
		sem_num: 0,
		sem_op:  1,
		sem_flg: C.SEM_UNDO,
	}

	r1, r2, err = syscall.Syscall(syscall.SYS_SEMOP, uintptr(semId), uintptr(unsafe.Pointer(&stSemBuf)), 1)
	if int(r1) < 0 {
		log.Printf("释放锁出错: %v\n", err)
	}
	return
}

func SemShow(semId int) int {
	r1, r2, err := syscall.Syscall(syscall.SYS_SEMCTL, uintptr(semId), 0, uintptr(C.GETVAL))
	if int(r1) < 0 {
		log.Printf("信号量读取出错: %v,%v,%v on id %d\n", r1, r2, err, semId)
	}
	return int(r1)
}
