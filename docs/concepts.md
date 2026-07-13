# 核心概念与对象(拉通认知)

> 本文是 easyeda-agent 布局/布线域的**共享词汇表**——把散在代码注释、memory、
> 对话里的概念统一到项目层,让后续会话、贡献者、Skill 用**同一套心智模型**。
> 新概念先落这里,再在代码/Skill/memory 里引用。相关:[cli-design.md](./cli-design.md) ·
> [e2e-automation-acceptance.md](./e2e-automation-acceptance.md) ·
> `skills/easyeda-agent/references/design-flow.md`(流程脊柱)。

---

## 一、电气层

### 网(net)
一条把若干**引脚(pad)**连在一起的电气网络。原理图的连接、PCB 的连通性都以网为单位。
一个器件的每个 pad 带一个 `net` 名(如 `USB_DP`、`GND`、`3V3`);同名即同网、电气相连。

### local net vs global net —— `isGlobalNet`
- **global(全局网)**:`GND` / `VCC` / `3V3` / `5V` / `VBUS`… 电源与地,**几乎每颗芯片都连**。
- **local(局部/信号网)**:`USB_DP` / `MCU_RX` / `EN` / `IO0`… 把**特定两三颗器件**绑在一起的信号网。
- **为什么区分**:判断「谁是谁的电气搭档」只能看 **local 网**——global 网谁都连,拿它找搭档
  会选到「几何上近但电气无关」的芯片。工具里 `isGlobalNet()` 就是这条线;net-aware 决策
  一律 **local 优先,无 local 才退回全部网**。

### 网感知(net-aware) vs 几何(geometric)—— 本项目的核心决策二分
| | 依据 | 例子(连接器分组选边) |
|---|---|---|
| **几何** | 只看 X/Y 坐标 | 「这些连接器质心离哪条板边最近 → 就贴那条边」——不问它们连到谁 |
| **网感知** | 看电气连接(net) | 「这些连接器主要连到哪颗芯片(local 网)→ 贴那颗芯片所在的边」 |

**判据一句话:让「谁挨着谁」由电路连接决定,而不是由当前谁离谁近决定。**
几何决策会把 USB 拉到离它搭档 CH340 很远的边 → 差分对横穿全板 → 交叉暴涨;网感知把 USB
放到 CH340 同侧 → 短、少交叉。实测(ceshi 同种子 A/B):几何 61 交叉 → 网感知 28 交叉。

---

## 二、布局对象与分档(placement tiers)

`pcb place-constrained` 按**固定的四档序**布局,高档先锁死、低档只能在其空隙里填:

| 档 | 对象 | 处理 |
|---|---|---|
| **T1 孔** | M3 安装孔 / 挖槽(layer-12 fill) | 障碍,永不移动;其他件避让 +60mil 垫圈净空 |
| **T2 边缘件** | 连接器 / 模组 / 天线(`board_edge=true`) | 贴板边 + 锁定;**user-facing 分组、any 就近**(见下)|
| **T3 主芯片** | ≥8 脚芯片 / 晶振 / `anchor=true` | 保持种子位、冻结(**工具不做主芯片 floorplan——种子由 agent 给**)|
| **T4 卫星** | 去耦/上下拉/LED/按键 | 螺旋合法化,**net-aware 吸到搭档芯片**;不冲突的保持原位(不扰手调)|

### edge role —— 边缘件的两种边语义(块声明)
块 `placement.<ref>.edge` 声明边缘件的**边角色**,工具据此分流(`edgeRoleOf`):
- **`user-facing`**:USB / SD / 螺钉端子 / 排针——用户插拔的外部 I/O。**≥2 件时分组到同一条
  共享边、沿边居中紧凑排布**;共享边由 **net-aware** 选(搭档芯片所在边),外部 I/O 聚一处
  又不拉长网。diag 标 `:grouped`。
- **`any`**:RF 天线 / 无线模组——必须在**某条**边(缩短天线走线),但哪条都行 → 保持各自最近边。

匹配:优先读块 hint(独特 designator 前缀 JP/SW/LED/ANT),通用前缀 J*/U* 走 device-name
regex 兜底(regex 是块数据的镜像)。**仍不解析自由文本 `orientation`**——那要 `blocks show` 摊给用户。

### partner chip(搭档芯片)
一个连接器/卫星**电气驱动的那颗固定主芯片**(经共享 local 网)。net-aware 布局的目标点:
卫星吸到它、连接器组贴它所在的边。工具里用 `mainNetPads`(net→主芯片焊盘)+ `nearestPad` 找。

---

## 三、块数据模型(circuit-block library)

`internal/blocks/data/*.json` 是布局/选型/拓扑知识的**声明式单一源**(见 [[improvements-sink-to-blocks]]:
改进沉淀成块数据,不做成工具启发式误伤别人板)。一个块携带多维 map,各阶段按需读:

| 字段 | 内容 | 谁消费 |
|---|---|---|
| `parts.<role>` | role → `standard-parts.json` 的 LCSC/UUID | 选型(零思考,照抄)|
| `internal_nets` / `ports` | 块内网表 + 边界端口(功能名) | 原理图实例化(validated 块信任照抄,只验跨块重绑)|
| `placement.<ref>` | `board_edge` / `edge` / `side` / `orientation` / `severity` / `reason` | T2 边缘件(edge 已消费;side/orientation 待补)|
| `openings` | `[{match, local}]` 连接器开口本地方向 | T2 朝向(旋到开口朝板外)|
| `pcb_layout` | `*-adjacency`(去耦/晶振贴脚)/ `rf-keepout` / `ep-*` | T4 贴脚(待消费)/ P4 禁布 / P8 热焊盘 |
| `signals` | 差分对 / 阻抗 / 等长 | P7.0 关键网先行 |
| `silk` | 逐脚标注 | P9 丝印 |

**source-of-truth 分层**:选型 = standard-parts;拓扑 = internal_nets;放置 = placement/pcb_layout。
工具**消费**块数据,不复制、不猜。

---

## 四、可信判据(reliable oracles)——判对错只信这些

| 判什么 | ✅ 唯一可信 | ❌ 不可信 |
|---|---|---|
| 网络连通 | `pcb drc` 的 **Connection Error 数**(0=通) | `pcb track-list` 计数(#103:已布线板读 0)|
| 器件覆盖/间距 | `pcb/sch layout-lint` | 截图(stale/blank)|
| 手焊可达 | `layout-lint --gate` #99(每件 ≥1 侧 ≥60mil)| — |
| 阶段门 | `workflow` 机械强制(指纹绑定,mutation 自动失效)| 人肉记状态 |
| 布线间距 | `pcb drc` 的 **Clearance Error 数** | — |
| 读 mutation 后的状态 | 先 `doc reload` 再读(见 [[pcb-stale-reads-need-doc-reload]])| mutation 后第一次读 |

---

## 五、已消化 vs 待补(工具消费块数据的进度)

- ✅ **T2 edge 语义**:user-facing 分组 + net-aware 选边(commit 1dbc065 / e94275d)。
- ⬜ **T2 `side`** → 器件贴装层(top/bottom);**`orientation` 自由文本**仍要人读。
- ⬜ **T4 `pcb_layout` adjacency** → 卫星贴脚距离硬约束。
- ⬜ **T3 主芯片信号流 floorplan**:块**不编码**主芯片相对位置 → 板子大小主要由 agent 的种子决定,
  这是「偏散板」的更大根因(见 [e2e-automation-acceptance.md](./e2e-automation-acceptance.md) §5)。
- ⬜ **末端 headless 干净布线**:route-short 稀疏板启发式,非平凡板需 tier② 原生自动布线。
