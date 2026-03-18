# 🏥 醫療排班系統 (Medical Schedule System)

這是一個專為醫療人員設計的「循環排班系統」，支援 4 週 (28天) 一週期的循環模板，並能自動產出月度班表。系統具備複雜的排班約束檢查（如「做 6 休 1」、「大小夜連續至少 2 天」、「night88 不可連兩日」等），確保排班符合法規與人性化需求。

---

## ✨ 核心特色
- **循環排班模板**：以 28 天為一週期，自動映射至各月份。
- **智慧約束檢查 (v6)**：含 R1~R8 硬性約束，自動過濾不合規的排班。
- **夜班升級機制**：支持 `day88` 搭配 `night88` 的連動排班邏輯。
- **假期自動分配**：根據剩餘配額與人力需求，智慧分配每月休假。
- **現代化介面**：使用 Ant Design 打造，具備視覺化行事曆與即時警告提示。

---

## 🛠️ 技術棧

| 層級 | 技術 |
|------|------|
| **後端核心** | [Go](https://go.dev/) (Gin Web Framework) |
| **資料庫 ORM** | [GORM](https://gorm.io/) + MySQL |
| **前端框架** | [React 18](https://react.dev/) + TypeScript |
| **建置工具** | [Vite](https://vitejs.dev/) |
| **UI 組件** | [Ant Design 5](https://ant.design/) |
| **狀態管理** | [Zustand](https://github.com/pmndrs/zustand) |

---

## 🚀 啟動指南

要重新啟動系統，請分別在兩個終端機視窗中執行後端與前端：

### 1. 後端 (Go Server)
後端服務負責 API 提供與排班演算法計算。

### 2. 啟動後端服務

#### 一般啟動 (手動重啟)
```powershell
# 確保已安裝 Go 並設定好資料庫 DSN
go run main.go
```

#### 開發啟動 (自動重啟/Hot Reload) ⭐ **推薦**
本專案已配置 `air` 工具，修改程式碼後後端會自動重啟：
```powershell
# 第一次使用需先確認 GOPATH/bin 在環境變數中，或直接執行：
C:\Users\sky12\go\bin\air
```
- **API 地址**: `http://localhost:8080`
- **資料庫配置**: 位於 `db/connection.go`

### 3. 啟動前端服務
前端負責圖形化管理介面。

```bash
cd frontend
# 若是首次啟動或有新增套件
npm install 

# 啟動開發伺服器
npm run dev
```
- **訪問地址**: `http://localhost:3000` (或終端機輸出的 Vite 地址)

---

## 📁 專案架構

```text
schedule-system/
├── main.go             # 後端入口
├── controllers/        # 控制器 (核心邏輯)
│   ├── monthly_controller.go  # 月度排班演算法
│   ├── scheduling_logic.go    # 約束判斷 (canAssign)
│   └── employee_controller.go  # 員工管理
├── models/             # 資料模型 (Employee, CycleTemplate...)
├── db/                 # 資料庫連線與遷移
├── .agents/            # Agent 技能與知識庫 (包含排班規則 SKILL.md)
└── frontend/           # 前端 React 專案庫
    └── src/
        ├── pages/      # 頁面 (MonthlySchedule, EmployeeList...)
        ├── types/      # TypeScript 定義
        └── store/      # Zustand 狀態管理
```

---

## 💡 開發備註
- **排班起算日**：2026/03/15 (C1 循環開始)。
- **時區**：預設使用 `Asia/Taipei` (UTC+8)。
- **排班規則**：詳細規則請參閱 `.agents/skills/scheduling-rules/SKILL.md`。

---

## 📝 授權
MIT License. Created by Medical Schedule Team.