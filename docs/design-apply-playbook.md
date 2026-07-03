# 设计:`easyeda apply` — 声明式步骤回放(playbook)

> **动机**(esp32MiniRequire 探针轮次 #1 实证):完整画一块板,agent 写了 10 个一次性
> bash 脚本,内容全是同构胶水——循环调 `easyeda` 子命令、记日志、防超时、断点续跑。
> 这层编排该内置:**一份 JSON 步骤文件 + `easyeda apply` 按步执行**,用户/agent 不再写
> 任何 shell/python 组合脚本;同一份文件即是「复现脚本 + 回归用例 + 教学示例」。

## 命令接口

```bash
easyeda apply steps.json                    # 顺序执行,自动写 journal,失败即停
easyeda apply steps.json --dry-run          # 只校验格式/变量/动作名,打印计划
easyeda apply steps.json --resume           # 按 journal 跳过已完成步骤,断点续跑
easyeda apply steps.json --from 12 --to 30  # 区间执行(调试单段)
easyeda apply steps.json --yes              # 放行确认门控步骤(delete/clear/import)
easyeda audit export --playbook > replay.json   # ★ 从真实会话的审计日志生成 playbook
```

## 文件格式(v1)

```jsonc
{
  "version": 1,
  "meta": { "name": "esp32-mini-sch", "project": "ceshi", "doc": "P1" },
  "vars":  { "LIB": "0819f05c4eef4c71ace90d822a990e87" },
  "steps": [
    // ① 典型步骤:typed action + payload(daemon 已有全量校验)
    { "id": "place-u1", "action": "schematic.component.place",
      "payload": { "libraryUuid": "${LIB}", "uuid": "ebc5227e…", "x": 760, "y": 430 },
      "capture": { "U1": "$.primitiveId" } },              // ← 结果取值存变量

    // ② 引用前步捕获的变量(解决「id 每次会变,静态脚本无法跨会话复现」)
    { "id": "desig-u1", "action": "schematic.component.modify",
      "payload": { "primitiveId": "${U1}", "patch": { "designator": "U1" } } },

    // ③ CLI 复合命令层(auto-place/route-short/power-planes/pour-fit/silk-align
    //    这些最有价值的工具不是单一 action,必须能编排)
    { "id": "autoplace", "run": "pcb auto-place", "flags": { "assembly-gap": 40 } },

    // ④ 门禁步骤:失败即停(对应 design-flow 的硬门)
    { "id": "gate-lint", "run": "pcb layout-lint",
      "assert": { "overlaps": 0, "score": ">=95" }, "onFail": "stop" },

    // ⑤ 检查点存盘 / 切文档 / 提示
    { "id": "save-1", "action": "pcb.save", "checkpoint": true },
    { "id": "to-sch", "run": "doc switch", "args": ["P1"] },
    { "id": "note",   "notify": "P3 完成,进入布线" }
  ]
}
```

**步骤五要素**:`action|run`(做什么)· `payload|flags|args`(参数,支持 `${var}`)·
`capture`(JSONPath 取结果)· `assert/onFail`(门禁)· 执行策略(`retry`、
`timeoutSec`、`continueOnError`,默认 0/20/false)。

## 关键设计决策

1. **刻意不做编程语言**——无条件分支、无循环。60 行数据就是 60 步。生成侧(agent/
   audit 导出)负责展开循环;回放侧保持傻瓜化、可 diff、可断点。这是与"再写一门脚本
   语言"的本质区别。
2. **双层寻址**:`action:`(typed action,daemon 校验 payload)+ `run:`(Cobra 子命令
   层)。缺一不可——复合工具都在 CLI 层。
3. **变量捕获是复现的命门**:primitiveId/坐标每次会话都变(load-bearing gotcha:
   pull fresh pids before mutating),`capture` + `${}` 替换让同一份文件跨会话可复现。
4. **journal 即状态**(`<file>.journal.jsonl`,每步一行 id/status/耗时/结果摘要):
   `--resume` 跳过已完成;超时/崩溃后原地续跑——本轮 place 阶段 2 分钟超时被迫改后台
   脚本的问题从根上消失。
5. **审计日志 → playbook 导出**是杀手级闭环:探索性会话跑完,
   `easyeda audit export --playbook` 直接得到干净步骤文件 → 提交为回归用例。
   esp32 案例可固化为 `examples/esp32-mini/{schematic,pcb}.playbook.json`。
6. **确认门控延续**:destructive 步骤(delete/clear/import_changes)默认逐步询问,
   `--yes` 整册放行(与现有 CLI 门控语义一致)。
7. **平台坑封装成"宏步骤"**:如 `run: pcb via-hop`(#31 的 fill 键合 workaround)、
   PLANE 翻转配方——playbook 引用宏,不要求用户知道坑。

## 实现落点

- `internal/app/cmd_apply.go`:解析 + 变量替换 + journal + 逐步分发(action → daemon
  `/action`;run → 进程内调用对应 Cobra 命令,复用既有实现,零重复)。
- `internal/app/cmd_audit.go`:`audit export --playbook`(审计条目 → 步骤,过滤只读
  action,合并 save)。
- Skill 同步:`references/actions.md` 增补 apply 章节;design-flow 各阶段附
  「可导出为 playbook」提示。
- 回归:`make lint-test` 加 playbook 格式 fixture;examples/ 下放 esp32 案例双文件。

## 验收(探针轮次 #2 的一部分)

用两份 playbook(sch + pcb)从零重放 esp32MiniRequire 全程,人工零脚本,
门禁步骤全过 → 即宣告 B 列「批量编排」缺口关闭。
