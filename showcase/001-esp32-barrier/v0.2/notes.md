# v0.2 模组版 — 进行中笔记

## 2026-07-09 深夜首日(S0-S4 大部)

- **S0 拍板**(用户):WROOM-1U-N8(C2980297,IPEX 外引承接)+ 维持 4 层;
  其余决策承接 v0.1(贴片/SD 底面/USB 单取向/USB+12V/CC1101 方案)。spec.json 落档。
- **工程**:esp32-barrier-v02(uuid 7f864ed2076a461c83b11abb17ed3420),
  schematic 4fa895b5aeaafaab 三页 p1=e8cca22e56b45038 / p2=056ad9e011460821 /
  p3=2c3b22e2ffc2ae41,PCB 3f98c29aa09c2a5e。网表引擎探针 ✓(TESTNET)。
- **83 件放置+位号+autolayout 全绿**(v0.1 减 20 件)。U1 模组 41 pin 字典已读
  (GND=1/40/41,3V3=2,EN=3,IO4-7=4-7,IO15/16=8/9,IO17/18=10/11,IO8=12,
  IO10=18,IO21=23,IO48=25,IO0=27,IO38-42=31-35,RXD0/TXD0=36/37)。
- **布线战况**:P1 86/92(0 跨网)、P2 91/91 ✓ 收官、P3 69/72(1 装饰性交叉)。
- **v0.1 坑确定性复现实录**:同一 autolayout spec → P2 同坐标 (160,565) R702:2↔D701:1
  引脚重合(已用 v0.1 配方拆弹:R702→(330,565) 三针重画;教训:**搬家前先拆旧桩**,
  这次 orphan GND 旗又桥了一轮才想起);LED1:1↔R901:1 近距 5mil(P1,监视中)。
- **待办(下一会话)**:
  1. P1 剩 11 针:USB/模组巷道拥塞(J2:A7,B4A9 / U3:1,3,5,14,16 / U1:1,2,4,5)
     ——树感知结点直连需先把**备用 pin 也入障碍模型**(这次 wire-over-pin 教训);
     或清巷道(挪 C30x 行)后 fix_pins。
  2. P3 剩 3 针(U6:21/C692:1/X2:3):"already connected"假象=端点压 pin 类,
     用 endpoint_scan 定位删旗后重画。
  3. 全量网表核对(健康引擎!direct sch read 按名并网 vs spec)→ S5 三门
     (layout-lint + native DRC + check)→ 端点扫描 → 确认点②。
  4. PCB:分档确认制布局(孔→边缘→主芯片→卫星)→ P7 人机协作档
     (用户点原生自动布线)首验。
- CLI 票素材新增:autolayout 无 pin 间隙约束二次复现(同 spec 同坐标重合);
  lib search 返回值展示截断导致 uuid 抄错(应在 CLI 输出完整 uuid——已修 standard-parts)。
