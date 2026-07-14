# 使用官方 Python 基础镜像
FROM python:3.10-slim

# 设置工作目录
WORKDIR /app

# 设置环境变量（优化 Python 运行）
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# 复制 requirements.txt 并安装依赖（利用 Docker 缓存）
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# 复制所有 Python 文件
COPY *.py ./

# 复制 data 目录（静态打包进镜像）
COPY data/ ./data/

# 暴露端口（8765）
EXPOSE 8765

# 设置容器启动命令（你的完整启动命令）
CMD ["python", "deepseek_agent.py", "--serve", "--host", "0.0.0.0", "--port", "8765"]