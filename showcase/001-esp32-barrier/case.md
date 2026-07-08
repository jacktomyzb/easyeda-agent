# Case 001 — esp32-barrier 道闸改造板

- **作者 / 提交人**:zhoushoujian(项目发起人;首个真实用户案例)
- **需求来源**:真实固件工程 `xiaozhi-server-go/firmware/esp32-barrier`(已在两块开发板上跑通,现做专用板)
- **交互模式**:里程碑确认档(ADR-0002 走查 #1,记录同步 `docs/milestone-walkthrough.md`)
- **状态**:S0 方案确认中
- **版本历史**:
  - `v0.1-draft`(2026-07-08):需求提取 + S0 方案书,等待决策确认

## 原始需求(用户口吻)

> 给我提取 esp32-barrier 这个固件工程的需求设计一个 PCB。这个就是 cc1101 + SD 卡 + esp32 的,
> 这次不用模组,改为芯片那种设计。带上 CH340 + 自动下载电路,加 2 个基本 BOOT/RESET 按键 + LED。

## 从固件提取的硬需求(佐证:BOARD_PINOUTS.md / README.md / Kconfig)

| 模块 | 依据 | 引脚(采用固件"立创 lckfb"原始配置) |
|---|---|---|
| ESP32-S3 芯片级 | `CONFIG_IDF_TARGET="esp32s3"` | QFN-56 裸片 + 外置 SPI flash + 40MHz 晶振 + 2.4G 天线(BLE 配网 + ESP-NOW 都要用) |
| CC1101 315/433MHz | README「RF replay」 | SPI2:GDO0=4 CSN=5 SCK=6 MOSI=7 MISO=15 GDO2=16;独立 433M 天线 |
| SD 卡(microSD) | Kconfig BARRIER_SD_* | SPI3:CS=10 SCK=18 MOSI=17 MISO=21 |
| RS485 Modbus | README「DI 状态采集」 | TX=38 RX=39 DE=40 → 3.3V 收发器 + 接线端子 |
| K230 上位机 UART | host_link | TX=41 RX=42 → 排针引出 |
| 磁簧输入 | Reed B1 | GPIO8 → 端子引出 |
| 烧录/调试 | 用户点名 | UART0(43/44)→ CH340C → USB-C;Q1/Q2 自动下载 + BOOT/RESET 按键 |
| 状态 LED | led_status.h 注释 | IO48,**低电平点亮**(3V3→LED→15k→IO48,按固件原文) |
