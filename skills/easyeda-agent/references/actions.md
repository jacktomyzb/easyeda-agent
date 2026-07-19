# EasyEDA Action Reference

Run `easyeda actions` for the authoritative machine-readable list.

## Playbook 回放(`easyeda apply`)— 批量步骤的首选载体

> **多步批量操作(>~5 步)不要再写 shell/python 胶水脚本**——写成 playbook JSON,
> `easyeda apply` 按步执行,自带变量捕获、门禁、journal 断点续跑。
> 完整格式与错误处理语义见 `docs/design-apply-playbook.md`(单一真源)。

```bash
easyeda apply steps.json                    # 顺序执行(meta.project 定目标工程)
easyeda apply steps.json --dry-run          # 预检 + 打印计划,不执行
easyeda apply steps.json --project demo2    # CLI flag > 文件(同一份打到另一工程)
easyeda apply steps.json --var LIB=<uuid>   # 变量复写(参数化)
easyeda apply steps.json --resume           # 按 journal 跳过已完成步骤(恢复 captured 变量)
easyeda apply steps.json --from 12 --to 30  # 区间执行
easyeda apply steps.json --yes              # 放行确认门控(delete/clear/rip-up/import 类)
```

要点(实现与设计一致,已单测+真机验证):
- 每步 `action:`(typed action)或 `run:`(Cobra 子命令,如 `pcb auto-place`)二选一;
  `notify:` 弹 toast。`payload/flags/args` 内 `${VAR}` 替换。
- `capture: {"U1": "$.primitiveId"}` 把结果存变量给后步用(id 跨会话会变,这是复现的命门)。
- `assert: {"$.score": ">=95", "$.overlaps": "==0"}` = 门禁,不过即停(`onFail: stop` 默认)。
- **错误纪律**:失败即终止;只读步骤自动重试 2 次;**变更类步骤超时不自动重试**
  (mutation 可能已生效——先读回校验再 `--resume`);变更步骤可带 `verify:` 读回块自证。
- journal 头带 playbook sha,文件改动会拒绝 `--resume`(改用 `--from`)。

**录制导出**:`easyeda audit export --playbook --day 2026-07-03 --since 15:17 --until 15:19
-o replay.json` 把真实会话(审计日志)提取成 playbook——只留变更步骤、自动挤压 autosave
风暴、**自动接线 capture/${var}**(后步引用前步 result.primitiveId 时);引用「窗外出生」
裸 id 的步骤会标 `raw-id` 警告(只能对同一板态回放,先 review)。⚠️ 提取物可能含
rip-up/clear 等破坏性步骤——整册回放前先 `--dry-run` 看计划,或用 `--from/--to` 只放安全区间
(已实证:esp32 移件段 18 步区间回放,幂等,lint 保持 100)。

## Navigation

- `system.health` — daemon + connector 可用性，已连接窗口列表
- `project.current` — 当前工程 uuid / name / teamUuid
- `document.current` — 当前激活文档 uuid / tabId / documentType
- `document.open` — 按 UUID 打开任意文档（原理图或 PCB）
- `schematic.pages.list` — 工程内全部原理图及页面
- `schematic.page.open` — 切换到指定原理图页（兼容旧用法）

## Sheet / 图页管理 + 明细表（title block）

均映射 `eda.dmt_Schematic.*`。**注意：EasyEDA Pro 无设置纸张尺寸(A4/A3)的公开 API**；可编辑的「图纸」属性就是明细表(title block)。CLI：`easyeda sch …`。

- `schematic.titleblock.get` — 读当前（或指定 `pageUuid`）图页的明细表：`showTitleBlock` + 各字段 `titleBlockData`。**改前先 get 拿到字段 key** → `easyeda sch titleblock-get`
- `schematic.titleblock.modify` — 调整明细表：显隐 + 字段值（只传要改的项，未知 key 被忽略）→ `easyeda sch titleblock --show` / `--data '{"Title":{"value":"电源模块"}}'`
- `schematic.page.create` — 新建图页（`schematicUuid`）→ `easyeda sch page-new --schematic <uuid>`
- `schematic.page.rename` — 重命名图页 → `easyeda sch page-rename --page <uuid> --name ...`
- `schematic.page.delete` — 删除图页（**需确认**，无 undo）→ `easyeda sch page-delete --page <uuid>`
- `schematic.rename` — 重命名整张原理图文档（非单页；可能联动复用模块符号 + PCB）→ `easyeda sch rename --schematic <uuid> --name ...`

## View（画布视图快捷键，原理图 + PCB 通用）

作用于当前聚焦的画布，等价于编辑器工具栏/快捷键。CLI：`easyeda view …`。

- `view.fit` — 适应全部（`K` 快捷键）；缩放至显示全部图元 → `easyeda view fit`
- `view.fit_selection` — 适应选中；先 `schematic.select` 再缩放至选中图元 → `easyeda view fit-selection`
- `view.zoom` — 缩放到坐标/比例（x/y/scale，scale 为百分比，省略则保持当前值）→ `easyeda view zoom --scale 200`
- `view.region` — 缩放到矩形区域（left/right/top/bottom，单位：原理图 0.01inch、PCB mil）→ `easyeda view region --left 0 --right 1000 --top 1000 --bottom 0`

## Inspect Schematic

- `schematic.components.list` — 当前页（或全页）所有元件，可含 pins
- `schematic.select` — 按 primitiveId 选中图元
- `schematic.snapshot` — 截取当前渲染区域为 PNG artifact。**默认先「适应全部」再截**（整张图入画，无需另调 `view.fit`）；`easyeda sch snapshot --no-fit` 保留当前视口。**局部截图**：先 `easyeda view region --left --right --top --bottom`（或 `view zoom --x --y --scale`）框住目标区域，再 `easyeda sch snapshot --no-fit` 截该视口

## Mutate Schematic

- `schematic.component.place` — 从库放置元件（libraryUuid + uuid + x/y）
- `schematic.component.modify` — 修改位置、位号、BOM 属性等
- `schematic.component.delete` — 删除元件（需确认）
- `schematic.wire.create` — 创建导线折线
- `schematic.netflag.create` — 创建电源/地/网络端口/短路 flag
- `schematic.power.connect_pin` — 复合操作：从 pin 拉导线 + 在末端放 flag（防止 flag-on-pin DRC fatal）
- `schematic.pin.disconnect` — `connect_pin` 的逆操作：把某 pin 的 stub 导线**连同**末端 netflag/netport 一并删除，避免只删 flag 留下孤儿 stub（EasyEDA 会给残留 wire 分配 `$3N…` 自动网名，`sch check` 现已能识别报 WARN）。按 `--pin U1:5`、`pinX`/`pinY` 坐标(`sch autoconnect --replace` 换网时用)或 `--flag-id`/`--wire-id` 定位。CLI：`easyeda sch disconnect --pin U1:5`
- `schematic.pin.set_no_connect` — 给引脚打/清「非连接标识」(NC, X 标记)，告诉 DRC 该脚是故意悬空。按 `--designator` + `--pin`（可多个）定位；`--clear` 清除。CLI：`easyeda sch no-connect --designator U1 --pin 23,24`
- `schematic.rebind.footprint` — 换封装（五步绑定法）。`modify` 改不了已放置件的封装引用，故走 `lib_Device.modify → delete → create → 恢复位号/坐标/属性`；导入器件 `libraryUuid` 为空时先在工程库反查补齐。按封装名精确匹配（同名多命中或未命中会报错，可用 `--footprint-uuid` 直连）。**重建会换新 primitiveId，导线可能需重连——务必跑 `sch drc`/`sch check` 复核连通性。** CLI：`easyeda sch rebind-footprint --id <primitiveId> --footprint <name>`
- `schematic.rebind.symbol` — 换符号，机制同上（五步绑定法）。CLI：`easyeda sch rebind-symbol --id <primitiveId> --symbol <name>`
- `schematic.save` — 保存原理图（需确认）

## Library

- `schematic.library.search` — 自由文本搜索立创/EasyEDA 器件库，返回 libraryUuid + uuid。当 `query` 为纯 LCSC C 号（`^C\d+$`）时自动切换为精确模式，仅保留 `lcsc`/`supplierId` 严格相等的条目；无精确命中则报错。传 `allowFuzzy`（CLI `--allow-fuzzy`）可保留原模糊排序结果

## Verify & Export

- `schematic.drc.check` — 调官方 `eda.sch_Drc.check` 作为 SDK DRC 门。当前 EasyEDA build 可能只返回 boolean/聚合结果,即使 `includeVerboseError=true` 也不保证有逐条 UI warning；CLI: `easyeda sch drc [--json]`。**不要单靠它宣称“官方 UI DRC 干净”**。
- `schematic.check` — 我们的逐条重建检查:从 primitives + 官方 `sch_ManufactureData.getNetlistFile()` 交叉校验,报告 net-marker/wire-name mismatch、multi-net wire、floating-pin、wire-crossing、wire-over-pin。CLI: `easyeda sch check [--json] [--strict]`。
- `schematic.export.netlist` — 导出网表为 artifact。底层必须走官方推荐的 `eda.sch_ManufactureData.getNetlistFile(fileName, netlistType)` 并读取返回的 `File`;不要使用已废弃的 `eda.sch_Netlist.getNetlist()`。官方文档标注 `getNetlist()` obsolete 且建议替代为 `getNetlistFile()`,并且 upstream issue [easyeda/pro-api-sdk#30](https://github.com/easyeda/pro-api-sdk/issues/30) 已复现它在含悬空引脚的原理图上可能无限卡死。CLI: `easyeda sch netlist`
- `schematic.export.bom` — 导出 BOM（csv 或 xlsx）为 artifact。CLI `easyeda bom export --type csv` **默认在导出后就地补全 LCSC C 号**（按 Manufacturer Part 关联 `standard-parts.json`，把 `Supplier Part` 从 `<MPN>.1` 改写为可下单的 C 号）；`--enrich=false` 关闭，xlsx 不补全（二进制）。补全是 best-effort（缺 python3/脚本只告警、导出仍成功）。脚本自动解析顺序：`--script` > `$EASYEDA_SKILLS_DIR/easyeda-agent/scripts/bom-enrich.py` > 二进制/工作目录向上找 `skills/` > PATH；安装版二进制在 `/usr/local/bin` 时设 `EASYEDA_SKILLS_DIR` 最稳。

## PCB（Phase 2，只读）

- `pcb.documents.list` — 工程内所有 PCB 文档（uuid + name）
- `pcb.components.list` — PCB 上的封装/器件（可含 pads）
- `pcb.layers.list` — PCB 层列表 + 当前层 + 铜层数（会先激活 PCB tab 保证 `currentLayer` 可读回；无当前层时附带 `visibleLayers` 作为显示状态证据）→ `easyeda pcb layers`
- `pcb.layers.set_current` — 切换当前编辑层（`--layer` 接受 id|层名|top|bottom|inner1）→ `easyeda pcb layer-set --layer bottom`
- `pcb.layers.visibility` — 显示/隐藏/聚焦层做视觉 QA：`--preset top-only|bottom-only|copper-only|silk-only`，或 `--show/--hide`（可加 `--exclusive` 只留所选）→ `easyeda pcb layer-visibility --preset bottom-only`
- `pcb.view.side` — 切到顶面/底面视图（选该面铜层为当前层 + 聚焦该面铜+丝印），随后 `pcb snapshot` 即反映该面。注意：EasyEDA 无原生画布翻面 API，这是「层聚焦」近似而非物理翻板 → `easyeda pcb view-side --side bottom`
- `pcb.nets.list` — PCB 全部网络

## PCB DRC 子规则 + 网络类（@beta）

EasyEDA 的 `eda.pcb_Drc.*` 暴露一组 **DRC 子规则**接口,把规则的 scope 从全局缩小到 **单个网络 / 网络对 / 区域**;另有一组 **网络类 (net class)** CRUD 给网络分组。所有方法标 `@beta` —— 返回的 `IPCB_NetRuleItem` / `IPCB_NetByNetRuleItem` / `IPCB_RegionRuleItem` 字段名在不同 build 间有 shape variance,**写之前先读回看清实际字段名**,再以同字段名构造 payload。Connector 侧的 handler 用 `netOfRule()`/`netPairOfRule()`/`regionIdOfRule()` 顺序尝试多套字段名,`deepMergeInto` 做 recursive deep-merge 不会因未知字段失败。

**写动作都触发 platform trap**:成功写入会把「系统预设」变成「板级自定义配置」副本(同 `overwriteCurrentRuleConfiguration`),预期且必须 —— 写完跑 `pcb drc` 验证规则生效。

- `pcb.drc.net_rules` — 读 per-net DRC 覆盖(网络规则:某网络的 trackWidth/clearance/via 大小覆盖)。verbatim `eda.pcb_Drc.getNetRules()` → `easyeda pcb net-rules`
- `pcb.drc.net_rules.set` — 写 per-net 覆盖。三种输入形态:`mode=replace` + `netRules` 全覆盖;`mode=merge` + `upserts` 按 `net` 匹配深合;或 `patches` 数组 `[{net, patch:{trackWidth, clearance, viaDrill, viaDiameter, ...}}]`(结构化 flag 用)。`removeNets` 在 merge 模式下按网名删条目。**需确认**。→ `easyeda pcb net-rules-set --mode=merge --rules '[...]'` 或结构化 `easyeda pcb net-rule --net USB_DP --track-width 12`
- `pcb.drc.net_by_net_rules` — 读 per-net-pair clearance 覆盖(网络间规则)。SDK 返回 object map,handler 用 `Object.values()` 规范成数组。→ `easyeda pcb net-by-net-rules`
- `pcb.drc.net_by_net_rules.set` — 写网络间规则。条目按 `{netA, netB}` pair 匹配(顺序无关,canonical-ordered tuple)。三种输入形态同上;`removePairs` 在 merge 模式下删条目。**需确认**。→ `easyeda pcb net-by-net-rules-set` 或 `easyeda pcb net-by-net-rule --net-a SW --net-b VREF --clearance 16`
- `pcb.drc.region_rules` — 读 per-region DRC 覆盖(区域规则)。**不是** `pcb.region.create` 那种 board-level keep-out primitive —— 后者是画布上的多边形,这里是规则的几何 scope。→ `easyeda pcb region-rules`
- `pcb.drc.region_rules.set` — 写区域规则。按 `regionId` 或 `name` 匹配。三种输入形态同上;`removeIds` 删条目。**需确认**。→ `easyeda pcb region-rules-set` 或 `easyeda pcb region-rule --region-id r1 --clearance 20`
- `pcb.netclass.list` — 列所有 **网络类**(网络类 = 一组网络分组,可共享一条 width/clearance 规则)及其成员 + 颜色。→ `easyeda pcb net-class` / `net-class-list`
- `pcb.netclass.create` — 新建网络类 + 初始成员 + 颜色(`#rrggbb` 或空)。同名类已存在时返回 false(no-op)。→ `easyeda pcb net-class-create --name USB --nets USB_DP,USB_DM`
- `pcb.netclass.delete` — 按名删网络类。**成员网络的 per-net 规则不会被删**(用 `pcb.drc.net_rules.set --remove-nets` 单独清)。**需确认**。→ `easyeda pcb net-class-delete --name USB`
- `pcb.netclass.rename` — 改名(`modifyNetClassName`,原名不存在或新名冲突会失败)。→ `easyeda pcb net-class-rename --name USB --new-name USB_PAIR`
- `pcb.netclass.add_net` — 加一个/一组网络到类(`addNetToNetClass`,幂等)。→ `easyeda pcb net-class-add-net --name USB --nets USB_DP,USB_DM`
- `pcb.netclass.remove_net` — 从类中移走一个/一组网络(`removeNetFromNetClass`)。最后一名被移走**不会**自动删类 —— 配合 `pcb.netclass.delete`。→ `easyeda pcb net-class-remove-net --name USB --nets USB_DP`

> **网络类 vs 规则**:网络类只是「分组」 —— 给一组网络起名,本身不带规则。要实际约束规则,先把网络加进类,再用 `pcb.drc.net_rules.set` / `pcb net-rule` 给类里的每个网络赋同样的 width/clearance 覆盖。典型流程:`net-class-create --name Power --nets +3V3,+5V` → `net-rule --net +3V3 --track-width 16` → `net-rule --net +5V --track-width 20`。
>
> **InvalidatesStage: post_route_checked**:网络/网络间/区域规则写入可能让现有走线变成 sub-clearance,workflow state 里的 `post_route_checked` 标记会被失效,后续 `pcb drc` 重新仲裁。

## Board（板子/组合 — 原理图↔PCB 绑定）

一个 **Board = 1 张原理图 + 1 块 PCB**，原理图与 PCB 就是通过它「组合」在一起（`import_changes` 也沿此链接同步）。Board 以**名称**标识。CLI：`easyeda board …`。

- `board.list` / `board.current` — 列出全部组合（名称 + 原理图 + PCB）/ 当前组合
- `board.create` — 把原理图和/或 PCB 绑成新组合（`--schematic` / `--pcb`）；游离 PCB 在 `import_changes` 前的修复手段
- `board.rename` — 重命名组合（`--name` → `--new`）
- `board.copy` — 复制组合（连同原理图 + PCB）
- `board.delete` — 删除组合（**需确认**，无 undo）

## Confirmation Required

- `schematic.component.delete`
- `schematic.page.delete`（删除图页，无 undo）
- `board.delete`（删除组合/板子，无 undo）
- `schematic.save`（未明确要求保存时）
- 生成的多步 mutation 计划
- `debug.exec_js`（任何情况）
