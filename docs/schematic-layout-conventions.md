# 原理图布局约定 (Schematic Layout Conventions)

When an AI agent (via `easyeda-agent`) generates or modifies a schematic, it must follow these conventions. They are derived from EE best practices plus empirical study of real LCEDA / EasyEDA Pro reference designs (see §7).

## 0. 坐标系与单位

- EasyEDA Pro 原理图网格单位 = `0.01 inch`（1 grid step = 10 raw units）。
- 所有坐标必须**对齐网格**（10 的倍数）。`x % 10 == 0 && y % 10 == 0`。
- A3 typical sheet ≈ `2400 × 1600` units (= 24" × 16")。A4 ≈ `1700 × 1100`。
- 元件中心 `(x, y)` = 元件参考点；元件 pin 在中心周围。

## 1. 分区 (Zone Map)

A3 / 类 A3 图纸划成 **3×3 九宫格**。模块按功能落到指定区：

```
+-----------------------+-----------------------+-----------------------+
| (TL) 输入电源 / 接口   | (TC) 时钟 / 复位       | (TR) 状态 LED / 调试    |
| Type-C, DC IN, 充电   | 晶振, RC, MOSFET      | LED, 串口 header       |
|                       |                       |                       |
+-----------------------+-----------------------+-----------------------+
| (ML) DC-DC / LDO      | (MC) 主 MCU + 去耦    | (MR) 射频 / 传感器       |
| 降压, 升压, LDO, 滤波  | ESP32 / STM32 等       | Wi-Fi, BLE, IMU, GNSS  |
|                       |                       |                       |
+-----------------------+-----------------------+-----------------------+
| (BL) 电池 / 接地       | (BC) 外设 IC          | (BR) I/O 连接器 / 大模块  |
| Bat connector, GND   | EEPROM, RTC, USB Hub  | 4G/LTE 模块, FPC, FFC  |
+-----------------------+-----------------------+-----------------------+
```

**Rules of placement**：

- **电源向左**（TL/ML/BL 列）—— 电流从左向右流，符合阅读习惯
- **MCU 居中**（MC）—— 它是信号的"枢纽"，所有外设朝它收敛
- **射频/传感器/I/O 向右**（TR/MR/BR 列）—— 时序/数据从 MCU 发散
- **大模块（pin > 50 或 bbox 一边 > 200）放在角落**（BL/BR/TR）—— 给中部留布线空间
- **同一功能簇相邻**：晶振紧贴 MCU；去耦电容紧贴 IC 电源 pin（≤ 30 units）；上拉电阻紧贴拉的那个 pin

## 2. 模块间距

> Goal：既不松散（图纸利用率低）也不拥挤（线路相互干扰）。

设模块 A 中心 `(xa, ya)`、bbox 宽 `wa`、高 `ha`；B 同。

**最小中心距**：
```
min_dx = (wa + wb) / 2 + buffer
buffer:
  - 小元件 (bbox 边 ≤ 50): 80
  - 中等 IC (bbox 边 60–150): 120
  - 大模块 (bbox 边 > 150): 200
```

**推荐中心距**（典型布局）：
| 邻居类型 | 中心距 (units) | 备注 |
|---|---|---|
| R / C / L 离散件之间 | 80–120 | 0603 元件 bbox ≈ 40 |
| 小 IC 之间 (8–16 pin) | 200–280 | 留出 designator 标签空间 |
| 中 IC 之间 (16–48 pin) | 280–400 | |
| MCU 邻射频/sensor | 400–600 | 大模块需要旁路空间 |
| 主 MCU 邻晶振 | 60–120 | **紧贴**，提高时钟完整性 |
| IC 邻去耦电容 | 20–40 | **极近**，1 个 grid step 内 |

**行间距**（top row → middle row → bottom row）：
- 短模块（h ≤ 80）间距 200–300
- 高模块（h > 200）间距 300–500
- 跨行长 wire 走外围（不穿过模块中央）

## 3. Wire 长度与走线约定

### 3.1 短桩 (pin lead-out)

每个 pin **必须有非零长度 wire** 引出（EasyEDA DRC 不认重叠点为连接，见 [skills/easyeda-schematic/SKILL.md](../skills/easyeda-schematic/SKILL.md#easyeda-electrical-rules)）。

| 用途 | 推荐长度 (units) | 方向 |
|---|---|---|
| pin → netflag (Power / GND / NetPort) | 20–40 | 顺着 pin 朝外 |
| pin → 邻 pin（同行 IC） | ≥ 60 | 直线，y 共线 |
| pin → 共享网络 net label | 20–60 | 朝标签方向 |
| pin → 去耦电容 | 20 | 极短，越紧越好 |

### 3.2 直角约定 (right-angle routing)

- **所有 wire 走水平或竖直**，不出现斜线（45°、任意角度均不允许）
- 拐弯用**一段水平 + 一段竖直**两段 wire（或单 wire 多 endpoint：`[x1,y1, x2,y1, x2,y2]`）
- 不允许"T 形" 三线交点未显式标 junction
- 长 wire 拐两次以上 → 考虑用 net label 代替（同名 label 表示同一网络，避免视觉缠绕）

### 3.3 电源/接地特殊约定

| | 方向 | 推荐长度 | netflag kind |
|---|---|---|---|
| `+3V3` / `+5V` / `VBAT` | netflag 朝**上** (rotation 0 或 90) | pin → netflag 20–40 | `power` |
| `GND` / `AGND` | netflag 朝**下** (rotation 180 或 270) | pin → netflag 20–40 | `ground` |
| `IN/OUT` 端口 | 朝外侧 (rotation 0/180) | pin → netflag 20–60 | `net_port_in/out/bi` |

电源/地的 netflag **绝不**与 pin 同坐标——必须用 wire 引出一段。

### 3.4 线宽

EasyEDA 默认 lineWidth = 1。约定：
- **信号线**：1（默认）
- **电源线**（VCC/GND 主干道）：2
- **总线** (`BUS_xxx`)：3

通过 `schematic.wire.create` 的 `lineWidth` 参数指定。

## 4. 命名约定

| 类型 | 风格 | 例 |
|---|---|---|
| 电源 net | `+大写带正负号` | `+3V3`, `+5V`, `+12V`, `-5V` |
| 模拟地 / 数字地 | 大写 | `GND`, `AGND`, `DGND`, `PGND` |
| 数字信号 | `MODULE_FUNC` 大写下划线 | `UART_TX`, `SPI_MOSI`, `I2C_SDA`, `LED_R` |
| 总线 | `BASE[N..0]` | `DATA[7..0]`, `ADDR[15..0]` |
| 复位 / 中断 | `nRESET`, `nINT`，前缀小 n 表示低有效 | `nRESET`, `nINT_IMU` |

## 5. Designator 前缀

| 前缀 | 元件类 |
|---|---|
| `R` | 电阻 |
| `C` | 电容 |
| `L` | 电感 |
| `D` | 二极管 (含 LED) |
| `Q` | 晶体管 / MOSFET |
| `U` | IC（一般、芯片、模块） |
| `J` | 连接器 |
| `X` | 晶振 |
| `SW` / `K` | 开关 |
| `TP` | 测试点 |
| `H` / `MH` | 安装孔 |

LED 也可用 `LED1` 这种语义化命名（兼容 `D1`），EasyEDA 不强制 `D` 前缀。

## 6. 去耦电容 (decoupling) 规则

每个数字 IC、模拟 IC、模块的 **VCC pin** 都应有：
- **0.1 μF (100 nF) 陶瓷电容** ≤ 30 units 处旁路到 GND（高频）
- 若模块电流大（> 50 mA），并联 **10 μF 钽/陶电容**（低频/储能）
- 多个 VCC pin 的大芯片（如 ESP32-S3 有 3 个 VDDA/VDD3P3）：**每个 pin 都要一个 0.1 μF**

由 Skill 自动布线时，去耦电容应在元件 `place` 后立刻 place 在其 VCC pin 旁。

## 7. 真实参考：motobox2026 (motorcycle tracker)

15 个 part，2400 × 1600 grid 范围（采集自 connector 实测）：

| Designator | 元件 | 类别 | 坐标 (x, y) | bbox (W × H) | pin |
|---|---|---|---|---|---|
| **U1** | ESP32-S3-WROOM-1U-N8R8 | 主 MCU (MC 区) | (1385, 210) | 190 × 220 | 41 |
| U2 | TPS54360 | 降压 DC-DC (TL/ML) | (150, 115) | 100 × 50 | 9 |
| U3 | BQ24074 | 电池充电管理 (ML) | (420, 175) | 120 × 150 | 17 |
| U4 | JW5033S | DC-DC (TL) | (675, 110) | 70 × 20 | 6 |
| U5 | SY8089 | DC-DC (TL) | (905, 110) | 70 × 20 | 5 |
| U6 | LC29H | GNSS 模块 (TR/MR) | (1740, 155) | 200 × 110 | 24 |
| U7 | LSM6DSV | IMU (TR) | (2085, 130) | 170 × 60 | 14 |
| U8 | SD NAND | 存储 (BL/BC) | (120, 500) | 40 × 40 | 9 |
| U9 | Air780EG | 4G LTE 模块 (BR 角) | (1590, 750) | 180 × 540 | 109 |
| U10 | CH334F | USB Hub (BL/BC) | (370, 540) | 140 × 130 | 25 |
| U11 | CH340K | USB-UART (BC) | (645, 500) | 90 × 60 | 11 |
| J1 | Wafer 2P | 电池接口 (BL/MC) | (1120, 125) | 30 × 50 | 4 |
| J2 | Type-C 12P | USB (BC) | (885, 535) | 70 × 110 | 16 |
| X1 | 32.768 kHz xtal | 晶振 (邻 MCU) | (1110, 490) | 60 × 20 | 4 |
| LED1 | 0603 White | 状态指示 (TR/MR) | (1320, 485) | 40 × ? | 2 |

**观察结论**：
- **电源链 (TPS54360 → BQ24074 → JW5033 → SY8089)** 从左到右排在上排 y=110-175。符合"电源向左+电流向右流"。
- **主 MCU U1 在中右** (1385, 210)，靠近右排射频/sensor（U6 GNSS, U7 IMU）。
- **大模块 U9 Air780EG (180×540, 109 pin)** 占 BR 角，独立成块。
- **USB-C 接口 J2 + USB Hub U10 + USB-UART U11** 形成 USB 子系统集中在 BC/BL 中下区。
- **晶振 X1 (1110, 490) 距 U1 中心 ≈ 384 units**——稍远，可优化到 200 内。
- 行间距 = top 行 (y~115-175) → 下行 (y~485-540) ≈ 320-360 units。

## 8. 自动化布局的执行步骤

当 Skill / Agent 自动放元件时：

1. **分类**：每个待放元件按 `symbolName` / `Manufacturer Part` 模糊匹配到分类 → 落到 §1 九宫格的某区
2. **排序**：同区内按"上游 → 下游"信号流向排（电源链：输入 → 转换 → 输出；信号链：sensor → MCU → 外设）
3. **下笔**：从区中心格点开始，按 §2 间距规则放邻居。优先填 x 方向，超过区宽就换 y
4. **布线**：每个 pin 用 §3 短桩规则引出。电源 pin → netflag (power, 朝上)，地 pin → netflag (ground, 朝下)
5. **去耦**：每个 IC 的 VCC pin 30 units 内 place 0.1μF
6. **验证**：跑 `schematic.drc.check`，违规返回参考区/间距规则定位修复

## 9. 边界与开放问题

- 这是 **schematic** 约定，不是 PCB 约定。PCB layout 另有独立约定（trace 宽度、layer 用途、impedance）。
- 对超大模块（pin > 100），九宫格容纳能力有限，可能要分多页（用 `schematic.pages.list` + `schematic.page.open`）。
- 多页之间通过 `net_port` (`createNetPort('IN/OUT/BI')`) 在页间建立电气连接，net 名称相同视为同网。
- `getCurrentRenderedAreaImage` 返回的是当前 viewport 的截图，不是全图——大图需要 `dmt_EditorControl.openDocument` + `sch_Document.navigateToRegion` 控制视野后再 snapshot。
