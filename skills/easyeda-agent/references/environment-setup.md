# 环境自举 — agent 自己把「可用的 EasyEDA 环境」拉起来

`NO_CONNECTOR` / `windows: []` 不是终点。有 chrome-devtools MCP 时,agent 可以
**自己**完成:开 web 编辑器 → 打开目标工程 → 确认连接器附着 → (需要时)热重载新
连接器。全流程 2026-07-07 在 ceshi 工程真机跑通。没有浏览器控制工具时才退回
「请用户手工打开 EasyEDA」。

## 0. 判定当前环境

```bash
easyeda daemon health
```

- `status: found` + `windows: []` → daemon 活着、没有编辑器。走下面的自举。
- 连 daemon 都没有 → 先 `make dev`(开发)或 `easyeda daemon start &`。
- `windows[]` 有条目 → 环境就绪,看 `context` 是否是目标工程/文档,不是就
  `easyeda doc switch <name> --project <name>`。

## 1. 打开 web 编辑器 + 目标工程(chrome-devtools MCP)

桌面客户端没开时,web 编辑器 `https://pro.lceda.cn/editor` 是完全等价的宿主
(同一 Chromium webview,连接器装在浏览器 profile 的 IndexedDB 里,登录态也在
profile 里持久化)。

```
1. new_page → https://pro.lceda.cn/editor#id=<projectUuid>
   ⚠️ #id= 直达【只在全新页面加载时生效】——已加载的编辑器里改 hash / 再
   navigate 同页 都不会触发打开工程。
2. 不知道 projectUuid?先开裸编辑器,take_snapshot 首页,工程树里每个工程是
   link "名字" url="…#id=<uuid>" —— uuid 直接读出来。
   或者对树节点用 click(dblClick: true) 真实双击(合成 MouseEvent dispatch
   无效,框架不吃)。
3. 等连接器附着(编辑器 boot + 连接器握手要 15~30s):
   until easyeda daemon health | grep -q connectorVersion; do sleep 3; done
4. 附着后 context.documentType 是 "home"/"blank" —— 还要
   easyeda doc switch PCB1 --project <name> 切到目标文档。
```

前提(一次性,人工):该 profile 里已导入过连接器 `.eext` 且开了
**允许外部交互**;登录过嘉立创账号。之后每次自举都无人工步骤。

## 2. 热重载连接器(改了 extension/ 之后)

不卸载、不重导入、不弹文件对话框——直接覆写 IndexedDB 里的执行文件。
详细原理见仓库 `docs/dev-environment.md` §5;要点:

```
1. make eext                        # 产出 extension/dist/index.js(19 万字节级)
2. 起本地 WS 文件服务器(编辑器是 HTTPS,fetch http://127.0.0.1 被
   mixed-content 拦,ws://127.0.0.1 放行——连接器本身就靠它):
   一个 ~30 行 node 脚本,收 {action:"getFile"} 回 {content:<base64>}。
3. evaluate_script 在编辑器页里执行:
   - DB = User_<teamUuid>_v6(teamUuid 从 easyeda project info 读)
   - store extensionsObjectStorage,key = <extensionUuid>|dist/index.js,
     把 record.source 换成 new File([bytes],'index.js')
   - store extensionsIndex,key = <extensionUuid>,把 config.version 改成新
     版本号(isAllowExternalInteractions 别动,权限就是这个布尔)
4. navigate reload 页面(#id= 还在,工程随 boot 重开)
5. until … grep connectorVersion → 应显示新版本(版本号编译在 bundle 里,
   变了就是新代码在跑的铁证)
```

extensionUuid 在 `extension/extension.json`。IndexedDB 结构非官方稳定 API
(今天 `_v6`),schema 升版要重核对 store 名。

## 3. 已踩过的坑

- **chrome-devtools MCP 多实例抢 profile**:多个会话/IDE(Claude Code、
  VSCode、opencode…)各起一个 chrome-devtools-mcp,全都用同一个
  `~/.cache/chrome-devtools-mcp/chrome-profile`,同一时刻只有一个 Chrome 能
  持有 → 其余实例所有调用报 "The browser is already running"。**修法**:
  `pkill -f "user-data-dir=.../chrome-devtools-mcp/chrome-profile"` 杀掉占
  profile 的孤儿 Chrome,紧接着发一个工具调用让**本会话**实例重启拿回句柄。
  profile 持久:登录态、EasyEDA 扩展、IndexedDB 全保留,重启零损失。
  多人/多会话同时驱动同一 profile 没有仲裁机制——**约定串行使用**,并发必冲突。
- **编辑后同网大面积「断连」**:对布线/填充做手术式增删后,DRC 可能突然报一串
  同网(常见 GND)Connection Error——这是**铺铜介导的连通性失效**,不是真断。
  `easyeda pcb pour-rebuild` 重灌后复测即恢复(ceshi 实测 11→1)。
  via-hop / via-delete / track-delete / fill delete 之后,若 DRC 报同网断连,
  先 pour-rebuild 再判断。
- **后台窗口 DRC 永不完成**:见 `pcb.md` DRC 条目——切前台单发,daemon 已防
  重入(`ACTION_BUSY`)。
- **headless 环境(CI / ClawFlow operator)不能做运行时验收**:没有编辑器就
  没有 DRC/check 的运行时产物;正确行为是失败并说明,绝不伪造通过。

## 4. 一次完整自举的实测时间线(2026-07-07,ceshi)

health(no windows)→ new_page #id 直达 → 25s 附着(0.8.4)→ make eext →
WS 服务器 + IndexedDB 覆写(199105 字节,0.8.4→0.8.9)→ reload → 30s 附着
0.8.9 → doc switch PCB1 → via-hop / via-delete / drc --json 全部真机验证 →
pour-rebuild 还原 DRC 基线。全程无人工。
