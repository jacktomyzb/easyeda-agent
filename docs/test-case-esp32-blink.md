# 固定测试用例 — ESP32-S3 最小系统点灯板

> **项目要求(硬规定):每次做端到端测试,都要把这个用例用 `easyeda-agent`
> 流程脊柱完整跑一遍**——放置 → 编组 → 布线 → `sch layout-lint` → DRC → save。
> 不是只测某个单点功能。这是 agent 画原理图能力的「冒烟 + 验收」基准。

## 为什么是它

ESP32-S3 最小系统 + 一个点灯 LED 是嵌入式里最经典的「能跑起来」板子,
部件数适中(8 件、~6 个网络),足够同时压到:**库优先放置、模块编组、芯片+外围就近、
电源 flag、信号本地线、去耦、layout-lint 覆盖检测、DRC、autosave**。
小到可重复、可靠;大到是真设计,不是玩具。

## BOM(核心 8 件,全部在 `standard-parts.json`)

库 `libraryUuid = 0819f05c4eef4c71ace90d822a990e87`

| 位号 | 器件 | 类别 key | deviceUuid | 作用 |
|---|---|---|---|---|
| U1 | ESP32-S3-WROOM-1 | `mcu.esp32s3_wroom1` | `ebc5227ec05f4bcbbb5581e49a5f7cc6` | 主控模块(flash+晶振+天线集成) |
| C1 | 100nF | `cap.100nf_0402` | `87bb635d0b2f489a9f60e7cd225beb3c` | 3V3 去耦(贴 3V3 脚) |
| C2 | 10µF | `cap.10uf_0805` | `6e5726223dd84f70bc3b626fc7d1f72c` | 3V3 体电容 |
| C3 | 100nF | `cap.100nf_0402` | `87bb635d0b2f489a9f60e7cd225beb3c` | EN 复位 RC 电容 |
| R1 | 10k | `res.10k_0402` | `c3b9baa5ef2e4070a4c0f9e9cd04fe6e` | EN 上拉到 3V3 |
| R2 | 10k | `res.10k_0402` | `c3b9baa5ef2e4070a4c0f9e9cd04fe6e` | IO0(boot)上拉到 3V3 |
| R3 | 330 | `res.330_0402` | `a60160c6e65140998078961749427162` | LED 限流 |
| LED1 | 红色 LED | `led.red_0805` | `06303f8c50b646d88d0dd08d2ec9692c` | 点灯指示 |

> 供电假设:板外提供已稳压的 **3.3V**(用 3V3/GND 电源 flag 接入)。
> 完整版扩展(可选,不属核心冒烟):USB-C(`conn.usb_c_16p`)+ AMS1117-3.3
> (`ldo.ams1117_3v3`)+ 输入/输出电容,做成自带 USB 供电的板子。

## 网络表(按功能;**引脚名放置后从实际符号读**,别信预设)

| 网络 | 连接 |
|---|---|
| **3V3** | U1 的 3V3 脚、C1+、C2+、R1、R2 → `3V3` 电源 flag |
| **GND** | U1 全部 GND 脚 + EPAD、C1−、C2−、C3−、LED1 阴极(K)→ `GND` 电源 flag |
| **EN** | U1.EN —— R1 另一端(来自 3V3)、C3+(到 GND);构成上电 RC 复位 |
| **IO0** | U1.IO0(strap)—— R2 另一端(上拉 3V3);高电平=正常启动 |
| **BLINK** | U1 的某个 GPIO(如 IO2)→ R3 → LED1 阳极(A);LED1.K → GND |

电气意图:上电 3V3 稳定 → EN 经 RC 延时拉高复位 → IO0 上拉=正常 boot →
固件翻转 GPIO → LED 点灯闪烁。

## 跑测流程(照 `easyeda-agent` 的 `design-flow.md` 阶段门)

1. **S0 预分析** — `easyeda health`;确认 8 件都在 `standard-parts.json`;模块划分:
   电源组(C2/3V3·GND flag)、MCU 组(U1+C1+R1+R2+C3 去耦/复位/strap)、点灯组(R3+LED1)。
2. **S1 分页** — 本板小,1 页(P1)即可;💾 `sch save`。
3. **S2 编组** — 规划三组分区:MCU 居中,电源在左,点灯在右,组间留通道。
4. **S3 按组摆放** — 逐组 `sch place` + `sch modify` 设位号(U1/C1…);去耦贴 3V3 脚、
   C3/R1 贴 EN、R2 贴 IO0、R3+LED1 成对。💾 过 layout-lint 后 `sch save`。
5. **S4 布线** — 放置后 `sch list`(读真实引脚坐标)→ 信号本地正交线
   (EN、IO0、BLINK),电源用 `connect_pin direction=` 出 3V3/GND flag(绝不穿引脚)。💾 save。
6. **S5 校验门** — `easyeda sch layout-lint`(0 overlap)+ `easyeda sch drc`(0 fatal)。
7. **S6 调整闭环** — 有问题就 move/align/补线,重跑校验,直到两门全清,💾 `sch save`。

## 验收标准(全过才算通过)

- [ ] 8 件全部放置且位号已分配(无 `C?`/`R?`)
- [ ] `easyeda sch layout-lint` → **0 overlap**(WARN 间距可接受但要解释)
- [ ] `easyeda sch drc` → **0 fatal**;无关键悬空网络
- [ ] 5 个功能网络(3V3/GND/EN/IO0/BLINK)连通正确
- [ ] 已落盘(autosave 触发 或 显式 `sch save`,audit 里有 `schematic.save ok=true`)
- [ ] 跑测在干净工程上做;**测完清理**(`sch prim-delete` 全删)还原,除非要留存复核

## 首跑实测发现的问题(2026-06，待修)

这个用例第一次端到端跑就抓出 4 个 CLI↔连接器一致性 bug——正是它存在的意义:

1. **`sch wire --points` 嵌套格式失败。** EDA `sch_PrimitiveWire.create` 只认**扁平**
   数组 `[x1,y1,x2,y2,...]`;连接器把嵌套 `[[x,y],[x,y]]` 原样透传 → `create failed!`。
   但 CLI `--help` 示例写的是嵌套。**临时绕过**:`easyeda call schematic.wire.create
   --payload '{"points":[x1,y1,x2,y2]}'`。**修法**:连接器 `schematicWireCreate` 把嵌套
   归一化为扁平(单一真源,惠及所有调用方)。
2. **`sch connect --kind gnd` 失败。** 连接器要 `ground`(及 `analog_ground` 等),CLI
   `--help` 写的是 `gnd/agnd/pgnd`。**临时**:用 `--kind ground`。**修法**:CLI 接受
   `gnd→ground` 别名,或对齐 help 与连接器枚举。
3. **DRC 返回聚合 `[{count:3,type:warn}]`,无逐条明细。** 连接器 DRC 归一化没展开
   violations,定位困难。**修法**:展开每条 violation 的 rule/message/坐标。
4. **`sch list` 缺 `--include-pins` flag。** 连接器支持 `includePins`,CLI 没暴露;只能走
   `call`。**修法**:给 `sch list` 加 `--include-pins`(已有 `--include-bbox` 先例)。

> 修完这 4 个后**重跑本用例**,应能全程用规范 CLI 子命令(不再依赖 `call` 绕过)。

## 备注

- 测试工程用 `ceshi`(`--project ceshi`),或任一空白原理图工程。
- ESP32-S3-WROOM-1 GND 脚多(含 EPAD),`sch list --include-pins` 读全;别漏 EPAD。
- 这是**回归基准**:layout-lint / autosave / design-flow 任何改动后都重跑此用例。
