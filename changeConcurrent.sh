#!/bin/bash

# 定义要锁定的文件
lockfile="../go-test/changeFile"

# 尝试获取写锁
(
  flock -x 200

  # 在这里放置你的代码
  # 这部分代码会在获取到锁之后执行
  echo "已获取锁，执行操作"
  sleep 5 # 假设的操作

) 200>$lockfile

echo "操作完成，锁已释放"