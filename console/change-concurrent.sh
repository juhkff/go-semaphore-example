#!/bin/bash

# 检查是否传入了两个参数
# if [ "$#" -ne 2 ]; then
# echo "Usage: $0 <lockKey> <new_value>"
# exit 1
# fi

# lockKey=$1
# newValue=$2

# 定义要锁定的文件
lockfile="/home/juhkff/projects/go-test/file/changeFile"

go build -o ./change-semaphore ./change-semaphore.go

# 获取文件的写锁
flock -x $lockfile -c "./change-semaphore"
