"""验证 start.sh 的依赖容器与数据卷生命周期。

使用模拟 Docker 检查重复启动、容器恢复、原卷复用和错误挂载保护。
此测试不会连接真实 Docker，不替代虚拟机上的实际部署验收。

仓库根目录运行：python -m unittest discover -s tests/deployment -v
需要 Python、Bash 和 curl；Windows 可使用 Git Bash。
"""
import json
import contextlib
import io
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
import threading

ROOT = Path(__file__).resolve().parents[2]
# 优先使用显式指定的 Bash；Windows 找不到时尝试 Git Bash 默认安装目录。
BASH = os.getenv("BASH_BIN") or shutil.which("bash")
if not BASH and os.name == "nt":
    candidate = Path("C:/Program Files/Git/bin/bash.exe")
    if candidate.exists():
        BASH = str(candidate)

# 模拟 Docker 的最小状态机，只实现 start.sh 使用的命令。
# calls 保存操作历史；volumes 中的数据标记用于判断原卷是否被重置。
FAKE_DOCKER = r'''#!/usr/bin/env python
import json, os, sys
from pathlib import Path
p = Path(os.environ['STARTUP_TEST_STATE'])
s = json.loads(p.read_text())
a = sys.argv[1:]
s['calls'].append(a)
def save(): p.write_text(json.dumps(s))
def value(flag): return a[a.index(flag)+1]
def fail(): save(); sys.exit(1)
if a[:2] == ['volume', 'inspect']:
    if a[2] not in s['volumes']: fail()
elif a[:2] == ['volume', 'create']:
    # 同名卷已存在时保持原数据，模拟命名卷的幂等行为。
    s['volumes'].setdefault(a[2], 'existing-data-marker')
elif a[:2] == ['container', 'inspect']:
    if a[2] not in s['containers']: fail()
elif a[0] == 'inspect':
    # 按查询模板返回运行状态、指定目录挂载或卷数量。
    c = s['containers'].get(a[-1])
    if c is None: fail()
    template = value('-f')
    if '.State.Running' in template: print(str(c['running']).lower())
    elif '.Destination' in template:
        print('volume:' + c['volume'] if c['volume'] else '')
    else: print(' '.join(['volume'] * c.get('volume_count', bool(c['volume']))))
elif a[:2] == ['network', 'inspect']:
    if not s['network']: fail()
    if '-f' in a: print(' '.join(k for k,v in s['containers'].items() if v['network']))
elif a[:2] == ['network', 'create']: s['network'] = True
elif a[:2] == ['network', 'connect']: s['containers'][a[-1]]['network'] = True
elif a[0] == 'start': s['containers'][a[1]]['running'] = True
elif a[0] == 'run' and '--name' in a:
    # 同名容器不能重复创建；只统计数据卷，排除只读配置文件挂载。
    name = value('--name')
    if name in s['containers']: fail()
    volumes = [a[i+1].split(':')[0] for i,x in enumerate(a) if x == '-v' and not a[i+1].endswith(':ro')]
    s['containers'][name] = {'running':True, 'network':True, 'volume':volumes[0] if volumes else '', 'volume_count':len(volumes)}
elif a[0] == 'rm':
    for name in a[1:]: s['containers'].pop(name, None)
elif a[0] == 'exec' and 'redis-cli' in a: print('PONG')
save()
'''

@unittest.skipUnless(BASH and shutil.which("curl"), "需要 Bash 和 curl")
class StartShVolumeLifecycleTest(unittest.TestCase):
    """以完整启动脚本为入口，验证依赖复用和数据卷保护规则。"""

    def setUp(self):
        """每项测试使用独立项目副本和模拟状态，结束后自动清理。"""
        # 不读取真实 .env，使用虚构密钥，避免触碰实际服务或泄露配置。
        self.tmp = tempfile.TemporaryDirectory(prefix=".tmp-startup-", dir=ROOT)
        self.addCleanup(self.tmp.cleanup)
        self.folder = Path(self.tmp.name)
        (self.folder / "agent-python").mkdir()
        (self.folder / "agent-python/.env").write_bytes(b"DEEPSEEK_API_KEY=test-key\n")
        shutil.copyfile(ROOT / "start.sh", self.folder / "start.sh")
        self.bin = self.folder / "bin"
        self.bin.mkdir()
        state_path = self.folder / "state.json"
        # 本机 HTTP 服务处理模拟命令，避免每条命令都启动 Python 解释器。
        class Handler(BaseHTTPRequestHandler):
            def do_POST(handler):
                # NUL 分隔参数，保留空格、引号和内联配置中的换行。
                arguments = handler.rfile.read(int(handler.headers['Content-Length'])).decode().split('\0')[:-1]
                code = FAKE_DOCKER.replace("p = Path(os.environ['STARTUP_TEST_STATE'])", "p = Path("+repr(str(state_path))+")").replace('a = sys.argv[1:]', 'a = '+repr(arguments))
                output = io.StringIO()
                status = 200
                with contextlib.redirect_stdout(output):
                    # 只执行本文件内的模拟代码，参数用 repr 编码。
                    try: exec(code, {})
                    except SystemExit: status = 400
                handler.send_response(status)
                handler.end_headers()
                handler.wfile.write(output.getvalue().encode())
            def log_message(handler, *args): pass
        server = HTTPServer(('127.0.0.1', 0), Handler)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        # 临时 docker 命令置于 PATH 最前；真实 curl 仅访问本机模拟服务。
        real_curl = Path(shutil.which('curl')).as_posix()
        docker_stub = "#!/usr/bin/env bash\nprintf '%s\\0' \"$@\" | '"+real_curl+"' --noproxy '*' -fsS --data-binary @- http://127.0.0.1:"+str(server.server_port)+"/\n"
        for name, body in {"docker":docker_stub}.items():
            path = self.bin / name
            path.write_bytes(body.encode("utf-8"))
            path.chmod(0o755)
        self.state = self.folder / "state.json"
        self.write({"calls":[], "volumes":{}, "containers":{}, "network":False})

    def read(self):
        """读取模拟的卷、容器和操作历史。"""
        return json.loads(self.state.read_text())

    def write(self, state):
        """预置状态，例如容器停止或被删除，但数据卷仍存在。"""
        self.state.write_text(json.dumps(state))

    def run_script(self, success=True):
        """执行项目副本中的 start.sh，检查预期退出状态。"""
        # 禁止 Git Bash 把容器路径转换为 Windows 路径。
        env = dict(os.environ, STARTUP_TEST_STATE=str(self.state), PATH=str(self.bin)+os.pathsep+os.environ['PATH'], MSYS_NO_PATHCONV='1')
        with (self.folder / 'stdout.log').open('w') as out, (self.folder / 'stderr.log').open('w') as err:
            try:
                # 健康检查和主机地址使用固定结果，本测试只验证部署编排。
                # -x 保存执行步骤，失败时显示末尾日志以便定位。
                harness = 'curl() { return 0; }; hostname() { echo 192.0.2.1; }; export -f curl hostname; exec bash -x start.sh'
                result = subprocess.run([BASH, "-c", harness], cwd=self.folder, env=env, stdout=out, stderr=err, timeout=40)
            except subprocess.TimeoutExpired:
                self.fail('Simulation timed out: '+(self.folder / 'stderr.log').read_text()[-2500:])
        result.stdout = (self.folder / 'stdout.log').read_text()
        result.stderr = (self.folder / 'stderr.log').read_text()
        self.assertEqual(0 if success else 1, result.returncode, result.stdout+result.stderr)
        return result

    def test_repeated_start_creates_exactly_four_unique_volumes(self):
        """连续启动两次：四个卷和五个依赖容器都只创建一次。"""
        self.run_script()
        self.run_script()
        state = self.read()
        self.assertEqual({'es_data','mysql_data','redis_data','agent_jaeger_data'}, set(state['volumes']))
        self.assertEqual(4, len([a for a in state['calls'] if a[:2] == ['volume','create']]))
        dependencies = {'es','mysql','agent-redis','agent-jaeger','agent-collector'}
        runs = [a for a in state['calls'] if a[0]=='run' and '--name' in a and a[a.index('--name')+1] in dependencies]
        self.assertEqual(5, len(runs))
        jaeger = next(a for a in runs if a[a.index('--name')+1]=='agent-jaeger')
        # /tmp 使用内存目录，防止 Jaeger 镜像额外生成匿名卷。
        self.assertIn('--tmpfs', jaeger)
        self.assertFalse(state['containers']['agent-collector']['volume'])
        self.assertFalse(any(a[:2]==['volume','rm'] for a in state['calls']))

    def test_stopped_dependencies_are_started_without_recreation(self):
        """依赖停止且断网后，再次启动只恢复容器和网络连接。"""
        self.run_script()
        state = self.read()
        for name in ['es','mysql','agent-redis','agent-jaeger','agent-collector']:
            state['containers'][name]['running'] = False
            state['containers'][name]['network'] = False
        state['calls'] = []
        self.write(state)
        self.run_script()
        calls = self.read()['calls']
        self.assertEqual(5, len([a for a in calls if a[0]=='start']))
        self.assertEqual(5, len([a for a in calls if a[:2]==['network','connect']]))
        self.assertFalse(any(a[:2]==['volume','create'] for a in calls))

    def test_missing_container_reuses_existing_volume_and_data(self):
        """MySQL 容器被删除后，重建仍挂载原卷并保留数据标记。"""
        self.run_script()
        state = self.read()
        del state['containers']['mysql']
        state['calls'] = []
        self.write(state)
        self.run_script()
        state = self.read()
        self.assertEqual('mysql_data', state['containers']['mysql']['volume'])
        self.assertEqual('existing-data-marker', state['volumes']['mysql_data'])
        self.assertFalse(any(a[:2]==['volume','create'] for a in state['calls']))

    def test_wrong_existing_mount_aborts_before_creating_storage(self):
        """已有 ES 挂载其他卷时，应在创建任何新存储之前退出。"""
        state = self.read()
        state['containers']['es'] = {'running':True,'network':False,'volume':'other_data'}
        self.write(state)
        result = self.run_script(success=False)
        self.assertIn('Refusing to switch data storage', result.stderr)
        self.assertFalse(any(a[:2]==['volume','create'] for a in self.read()['calls']))

    def test_extra_anonymous_volume_is_not_silently_accepted(self):
        """Jaeger 存在额外匿名卷时应提示核对，不能默默接受。"""
        state = self.read()
        state['containers']['agent-jaeger'] = {'running':True,'network':False,'volume':'agent_jaeger_data','volume_count':2}
        self.write(state)
        result = self.run_script(success=False)
        self.assertIn('additional volumes', result.stderr)
        self.assertFalse(any(a[:2]==['volume','create'] for a in self.read()['calls']))

if __name__ == '__main__':
    unittest.main()
