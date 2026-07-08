# esp32-barrier v0.1 — 整板网表计划(放置/连线蓝图)

> 来源:reference-circuits.json(6 电路,双源验证)+ 固件 lckfb 引脚图 + spec.json 决策。
> 位号已全局化:U1=S3 U2=AMS1117 U3=CH340C U4=W25Q64 U5=XL1509 U6=CC1101 U7=SP3485;
> C1xx/L1xx/R1xx=S3 核心,C2xx=LDO,C3xx/R3xx/Q*=USB烧录,C5xx/L5xx/D5xx=电源,
> C6xx/L6xx/R6xx/X2=CC1101,R7xx/C7xx/D7xx=RS485,R8xx/C8xx=SD,R9xx/LED1=指示。

## 页面分配(按 spec.json)
- **P1**:MCU 核心(U1/U4/X1/去耦阵)+ 2.4G π匹配+IPEX + USB/烧录(J2/U3/Q1/Q2/SW1/SW2)+ LED
- **P2**:电源(J1/D502/U5/L501/电解 + U2 LDO)+ CC1101(U6/X2/巴伦滤波/IPEX)+ SD(J3)+ RS485(U7/J4)+ 现场接口(J5/J6)

## 电源轨
- `+12V_RAW`: J1.1 → D502(SS34).A
- `+12V`: D502.K, C501+, C503, U5.VIN
- `SW_5V`: U5.OUTPUT, D501.K, L501.1
- `+5V`: L501.2, C502+, U5.FEEDBACK(固定5V直连), **D503(SS34).K**(USB 防倒灌 OR:D503.A=VBUS), U2(AMS1117).VIN, C201(10µF)
- `VBUS`: J2.A4B9/B4A9 → D503.A(注:VBUS 经二极管并轨,+12V 在场时 buck 主供)
- `3V3`: U2.VOUT, C202(22µF), C203(100nF) → U1 全部 VDD 组(经各自去耦)、U3.VCC+V3、U6(经 L601 磁珠)、U7.VCC、J3.VDD、J5.3V3、上拉阵
- `GND`: 全部
- ⚠️ U5.ON/OFF 接 GND(低有效使能,悬空也开;**绝不能接 VIN**)

## U1 ESP32-S3 核心(按 pin 名接,QFN-56)
- XTAL_P → L101(**0Ω 代 24nH**,24nH 缺货)→ X1.XIN;XTAL_N → X1.XOUT;X1 双帽 C101/C102=**18pF**(CL12pF 晶振:series/2+2.5≈11.5pF ✓);X1.GND×2→GND
- CHIP_PU: R102 10k→3V3, C103 1µF→GND, SW2, Q1.C, R303 10k→3V3(与 R102 并存,按 DevKitC 保留), C301/C303 100nF
- GPIO0: R103 10k→3V3, SW1, Q2.C, C302 100nF
- VDD3P3(2,3): L102 2.0nH ← C110 10µF+C111 1µF(VDD33 侧);pin 侧 C112/C113 100nF
- VDDA(55,56): C114 1µF + C115 10nF;VDD3P3_RTC(20): C116 100nF;VDD3P3_CPU(46): C117 100nF
- VDD_SPI(29): C118 100nF + C119 1µF → 供 U4.VCC
- 主入口: C120 10µF
- LNA_IN: C104 1.5pF(shunt)→ L103 2.0nH(series,0402 代 0201)→ C105 1.5pF(shunt)→ J_ANT1.SIG(50Ω)
- Flash U4(直连,省 0Ω 系列): SPICS0→/CS(1), SPIQ→DO(2), SPIWP→/WP(3), SPID→DI(5), SPICLK→CLK(6), SPIHD→/HOLD(7), VCC(8)=VDD_SPI, GND(4)
- Strap: GPIO45/46 浮空(内部 WPD 默认 3.3V flash ✓);GPIO3 本板未用(磁簧在 GPIO8)

## 外设 GPIO 网(固件 lckfb 图)
| 网 | U1 pin | 对端 |
|---|---|---|
| CC_GDO0 | GPIO4 | U6.GDO0 |
| CC_CSN | GPIO5 | U6.CSN |
| CC_SCK | GPIO6 | U6.SCLK |
| CC_MOSI | GPIO7 | U6.SI |
| CC_MISO | GPIO15 | U6.SO |
| CC_GDO2 | GPIO16 | U6.GDO2 |
| SD_CS | GPIO10 | J3.DAT3 + R801 10k↑ |
| SD_MOSI | GPIO17 | J3.CMD + R803 10k↑ |
| SD_SCK | GPIO18 | J3.CLK |
| SD_MISO | GPIO21 | J3.DAT0 + R802 10k↑ |
| SD_DAT1/2 | — | J3.DAT1+R804↑ / DAT2+R805↑(仅上拉) |
| RS485_TX | GPIO38 | U7.DI |
| RS485_RX | GPIO39 | U7.RO |
| RS485_DE | GPIO40 | U7.DE+RE(R706 10k↓ 上电收态,common practice) |
| K230_TX | GPIO41 | J5.2 |
| K230_RX | GPIO42 | J5.3 |
| REED | GPIO8 | J6.1 + R110 10k↑3V3;J6.2=GND |
| LED_N | GPIO48 | R901 15k ← LED1.K;LED1.A→3V3(低电平亮,按固件) |
| U0TXD | GPIO43 | R305 0R → U3.RXD |
| U0RXD | GPIO44 | R304 0R ← U3.TXD |

## USB/自动下载(DevKitC V4 拓扑)
- J2.A6→USB_DP→U3.UD+;J2.A7→USB_DN→U3.UD-(单取向,B6/B7 不接——spec 已定偏差)
- J2.CC1/CC2 → R1/R2 5.1k↓GND;J2 壳/EP→GND
- U3.V3 接 VCC(3V3 供电模式)+ C304 100nF + C305 10µF
- U3.DTR→R301 10k→Q1.B;U3.RTS→R302 10k→Q2.B;Q1.E↔U3.RTS;Q2.E↔U3.DTR(交叉);Q1.C→CHIP_PU;Q2.C→GPIO0

## CC1101 射频(TI CC1101EM 434MHz rev2.0.0)
- 巴伦:RF_P→{C621 3.9pF, L621 27nH};L621.2→C624 220pF→GND;RF_N→{L631 27nH, C631 3.9pF→GND};C621.2+L631.1=RF_SE
- 滤波:RF_SE→L622 22nH→(C622 8.2pF↓)→L623 27nH→(C623 5.6pF↓)→C625 220pF→J_ANT2.SIG
- X2 26MHz(NX3225GA CL=10pF):帽 **C681=12pF、C682=15pF**(按 EM 实装,非 datasheet 27pF——晶振是 10pF CL)
- RBIAS→R601 56k→GND;DCOUPL→C651 100nF(**不得接 VDD**)
- 供电:3V3→L601 磁珠→VDD 节点(C601 1µF)→DVDD+C641 100n / DGUARD+C661 220p / AVDD9+C691 10n / AVDD11+C692 220p / AVDD14+C693 10n / AVDD15+C694 220p;GND16/19/EP

## RS485(SP3485 + Modbus 规)
- A: U7.A, R701 680R(**默认 DNF**,从站不偏置), D701(PSM712).1, J4.1
- B: U7.B, R702 680R(DNF), R703 120R 1206(经 JP701 SIP2 跳线), D701.2, J4.2
- J4.3=GND=D701.3;C701 100nF 去耦
- ⚠️ 丝印注明 A/B 按芯片惯例(与 Modbus 规范命名相反)

## 电源模块细节
- LDO U2:AMS1117-3.3,C201 10µF(in)/C202 22µF(out)/C203 100nF
- Buck 布局(P8 注意):C501/C503 贴 VIN;续流 D501 紧贴;反馈线远离 L501

## 备货/偏差备注(notes.md 素材)
- 24nH → 0Ω 代(导则允许);L103/C104/C105 π 匹配用 0402 代 0201(可焊性优先,IPEX 输出容错)
- 470µF/35V C9900003002 下单前核电压;220µF 选了 16V(5V 轨 3 倍裕量 ✓)
- PSM712 代 SM712(同规 SOT-23 RS485 TVS)
- D503 VBUS 防倒灌 OR 为 common practice(非参考设计原文)
