# EasyEDA Action Reference

Run `easyeda actions` for the authoritative machine-readable list.

## Navigation

- `system.health` — daemon + connector 可用性，已连接窗口列表
- `project.current` — 当前工程 uuid / name / teamUuid
- `document.current` — 当前激活文档 uuid / tabId / documentType
- `document.open` — 按 UUID 打开任意文档（原理图或 PCB）
- `schematic.pages.list` — 工程内全部原理图及页面
- `schematic.page.open` — 切换到指定原理图页（兼容旧用法）

## Inspect Schematic

- `schematic.components.list` — 当前页（或全页）所有元件，可含 pins
- `schematic.select` — 按 primitiveId 选中图元
- `schematic.snapshot` — 截取当前渲染区域为 PNG artifact

## Mutate Schematic

- `schematic.component.place` — 从库放置元件（libraryUuid + uuid + x/y）
- `schematic.component.modify` — 修改位置、位号、BOM 属性等
- `schematic.component.delete` — 删除元件（需确认）
- `schematic.wire.create` — 创建导线折线
- `schematic.netflag.create` — 创建电源/地/网络端口/短路 flag
- `schematic.power.connect_pin` — 复合操作：从 pin 拉导线 + 在末端放 flag（防止 flag-on-pin DRC fatal）
- `schematic.save` — 保存原理图（需确认）

## Library

- `schematic.library.search` — 自由文本搜索立创/EasyEDA 器件库，返回 libraryUuid + uuid

## Verify & Export

- `schematic.drc.check` — 运行 DRC，返回 passed + violations
- `schematic.export.netlist` — 导出网表为 artifact
- `schematic.export.bom` — 导出 BOM（csv 或 xlsx）为 artifact

## PCB（Phase 2，只读）

- `pcb.documents.list` — 工程内所有 PCB 文档（uuid + name）
- `pcb.components.list` — PCB 上的封装/器件（可含 pads）
- `pcb.layers.list` — PCB 层列表 + 当前层 + 铜层数
- `pcb.nets.list` — PCB 全部网络

## Confirmation Required

- `schematic.component.delete`
- `schematic.save`（未明确要求保存时）
- 生成的多步 mutation 计划
- `debug.exec_js`（任何情况）
