package semaphore

import (
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"

	"gopkg.in/yaml.v3"
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

var lockFile = "/home/juhkff/projects/go-test/file/lockFile"
var changeFile = "/home/juhkff/projects/go-test/file/changeFile"
var configFile = "/home/juhkff/projects/go-test/config/config.yaml"

// 默认信号量key，会被配置文件的值覆盖
var LockKey = 2216

// 默认并发量，会被配置文件的值覆盖
var ConcurrentNum = 30

type Config struct {
	LockKey       int `yaml:"lockKey"`
	ConcurrentNum int `yaml:"concurrentNum"`
}

func init() {
	config, err := ReadConfig(configFile)
	if err != nil {
		log.Fatalf("初始化: 读取配置失败: %v\n", err)
	} else {
		ConcurrentNum = config.ConcurrentNum
		log.Printf("读取设定并发量: %d\n", ConcurrentNum)
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func SetSemaphore() (r1 uintptr, r2 uintptr, err syscall.Errno) {
	r1, r2, err = syscall.Syscall(syscall.SYS_SEMGET, uintptr(LockKey), uintptr(1), uintptr(C.IPC_CREAT|00666))
	if int(r1) < 0 {
		return
	}
	val := uintptr(C.semctl_setval(C.int(r1), 0, C.SETVAL, C.int(ConcurrentNum)))
	if int(val) < 0 {
		//todo :打印errno
		log.Printf("信号量设值失败\n")
		r1 = val
		return
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
			}
		}
	} else {
		file, err = os.OpenFile(lockFile, os.O_RDWR|os.O_CREATE, 0666)
	}
	return
}

func getChangeFile() (file *os.File, err error) {
	if !fileExists(changeFile) {
		file, err = os.OpenFile(changeFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			if os.IsExist(err) {
				//changeFile已被其它进程创建
				file, err = os.OpenFile(changeFile, os.O_RDWR|os.O_CREATE, 0666)
			}
		}
	} else {
		file, err = os.OpenFile(changeFile, os.O_RDWR|os.O_CREATE, 0666)
	}
	return
}

// SemGet 获取信号量组ID
// return : r1-semId, err-syscall.Errno, err2-error
func SemGet() (r1 uintptr, err syscall.Errno, err2 error) {
	r1, _, err = syscall.Syscall(syscall.SYS_SEMGET, uintptr(LockKey), uintptr(1), uintptr(00666))
	var file *os.File
	//第一次运行需要初始化信号量
	if int(r1) < 0 {
		//创建或获取文件用于锁
		file, err2 = getLockFile()
		if err2 != nil {
			log.Printf("获取锁文件失败: %v\n", err2)
			return
		}
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
		//二次验证
		r1, _, err = syscall.Syscall(syscall.SYS_SEMGET, uintptr(LockKey), uintptr(1), uintptr(00666))
		if int(r1) < 0 {
			//初始化信号量
			r1, _, err = SetSemaphore()
			if int(r1) < 0 {
				log.Printf("信号量初始化失败: %v\n", err)
				return
			}
		}
	}
	return
}

func SemLock(semId int) (r1 uintptr, r2 uintptr, err syscall.Errno) {
	stSemBuf := C.sembuf{
		sem_num: 0,
		sem_op:  -1,
		sem_flg: C.SEM_UNDO,
	}

	r1, r2, err = syscall.Syscall(syscall.SYS_SEMOP, uintptr(semId), uintptr(unsafe.Pointer(&stSemBuf)), 1)
	if int(r1) < 0 {
		log.Printf("请求锁出错: %v\n", err)
		return
	}
	//申请共享锁
	file, err2 := os.OpenFile(changeFile, os.O_RDWR|os.O_CREATE, 0666)
	if err2 != nil {
		log.Printf("获取共享锁文件失败: %v\n", err2)
		return
	}
	//获取锁
	err2 = syscall.Flock(int(file.Fd()), syscall.LOCK_SH)
	if err2 != nil {
		log.Printf("共享锁Lock失败: %v\n", err2)
		return
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
		return
	}
	//申请共享锁
	file, err2 := getChangeFile()
	if err2 != nil {
		log.Printf("获取共享锁文件失败: %v\n", err2)
		return
	}
	//获取锁
	err2 = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if err2 != nil {
		log.Printf("共享锁释放失败: %v\n", err2)
		return
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

func ReadConfig(filePath string) (config Config, err error) {
	configFile, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("无法打开配置文件: %v\n", err)
		return
	}
	defer configFile.Close()
	configData, err := io.ReadAll(configFile)
	if err != nil {
		fmt.Printf("无法读取配置文件: %v\n", err)
		return
	}
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return config, fmt.Errorf("无法解析配置文件: %v", err)
	}
	//加载配置文件
	return config, nil
}
