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

COPY customer_service.py logging_config.py ./
COPY client/ ./client/
COPY controller/ ./controller/
COPY model/ ./model/
COPY service/ ./service/

EXPOSE 8765

CMD ["python", "customer_service.py", "--host", "0.0.0.0", "--port", "8765"]
