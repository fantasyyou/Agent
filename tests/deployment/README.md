# 部署回归测试

`test_start_sh_volume_lifecycle.py` 验证根目录 `start.sh` 的容器和数据卷管理：

- 重复部署复用四个固定数据卷和五个依赖容器。
- 停止的依赖恢复启动并接入网络。
- 删除依赖容器后重新挂载原卷，保留数据。
- 发现错误挂载或额外匿名卷时阻止部署。

在仓库根目录运行：

```bash
python -m unittest discover -s tests/deployment -v
```

需要 Python、Bash 和 curl；Windows 可使用 Git Bash，或通过 `BASH_BIN` 指定 Bash 路径。`test_` 前缀用于自动发现测试文件。

测试使用模拟 Docker 和健康检查，不需要真实 Docker，也不会操作虚拟机。临时项目副本位于 `.tmp-startup-*`，正常结束后自动删除。此测试不替代真实镜像启动、服务连接或持久化验收。
