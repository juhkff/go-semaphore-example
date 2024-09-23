package semaphore

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"gopkg.in/yaml.v3"
)

/*
#include <sys/sem.h>
typedef struct sembuf sembuf;
*/
import "C"

// LockKey 默认信号量key，会被配置文件的值覆盖
var LockKey = 2216

// ConcurrentNum 默认并发量，会被配置文件的值覆盖
var ConcurrentNum = 30

// LockFilePath 默认锁文件路径，会被配置文件的值覆盖
var LockFilePath = "/usr/local/orchestrator/hook/lockFile"

var configFilePath string

type Config struct {
	LockKey       int    `yaml:"lockKey"`
	ConcurrentNum int    `yaml:"concurrentNum"`
	LockFilePath  string `yaml:"lockFilePath"`
}

func init() {
	// 配置文件路径在可执行文件同目录下
	wd, _ := os.Getwd()
	configFilePath = filepath.Join(wd, "config.yaml")
	configFile, err := ReadConfig(configFilePath)
	if err != nil {
		log.Fatalf("初始化: 读取配置失败: %v\n", err)
	} else {
		LockKey = configFile.LockKey
		ConcurrentNum = configFile.ConcurrentNum
		LockFilePath = configFile.LockFilePath
		LockFilePath = filepath.Join(LockFilePath, "lockFile")
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

	// 准备SETVAL命令的参数
	cmd := uintptr(C.SETVAL)
	val := uintptr(ConcurrentNum)

	// 调用SYS_SEMCTL设置信号量值
	_, _, err = syscall.Syscall6(syscall.SYS_SEMCTL, r1, 0, cmd, val, 0, 0)
	if err != 0 {
		return
	}
	return
}

func GetLockFile() (file *os.File, err error) {
	if !fileExists(LockFilePath) {
		file, err = os.OpenFile(LockFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			if os.IsExist(err) {
				//lock_file已被其它进程创建
				file, err = os.OpenFile(LockFilePath, os.O_RDWR|os.O_CREATE, 0666)
			}
		}
	} else {
		file, err = os.OpenFile(LockFilePath, os.O_RDWR|os.O_CREATE, 0666)
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
		file, err2 = GetLockFile()
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
		log.Printf("请求信号量出错: %v\n", err)
		return
	}
	//申请共享锁
	file, err2 := GetLockFile()
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
		log.Printf("释放信号量出错: %v\n", err)
		return
	}
	//申请共享锁
	file, err2 := GetLockFile()
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
	defer func(configFile *os.File) {
		err = configFile.Close()
		if err != nil {
			return
		}
	}(configFile)
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
