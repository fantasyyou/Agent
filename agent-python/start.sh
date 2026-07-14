#!/bin/bash
# 停止并删除旧容器
docker stop agent-app 2>/dev/null
docker rm agent-app 2>/dev/null

# 构建镜像
docker build -t agent-python:latest .

# 运行容器
docker run -d --name agent-app -p 8080:8080 agent-python:latest

# 显示日志
docker logs -f agent-app