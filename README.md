# 🗓️ Golang 排班系統 (Schedule System)

這是一套完整的前後端分離排班系統，使用 **Golang + Gin + GORM + MySQL** 開發後端 API，搭配 **React + TypeScript + Ant Design** 打造現代化前端介面，支援員工管理、班表需求設定、排班預約與行事曆視圖，適合企業內部或團隊使用。

![排班系統介面](./docs/screenshot.png)

## ✨ 主要功能

### 📅 排班行事曆
- 月曆視圖顯示所有班表狀態
- 每日顯示早中晚班的「已排人數/需求人數」
- 點擊日期快速預約班別
- 即時統計今日需求、已排班與缺額

### 👥 使用者管理
- 顯示所有員工資訊
- 選擇當前操作使用者
- 支援 CRUD 操作

### ⚙️ 班表需求管理
- 設定每日各班別所需人數
- 彈性調整不同日期的需求

## 🚀 技術棧

### 後端
- **框架**: [Go](https://go.dev/) + [Gin](https://github.com/gin-gonic/gin)
- **ORM**: [GORM](https://gorm.io/)
- **資料庫**: MySQL
- **API**: RESTful API + CORS 支援

### 前端
- **框架**: [React 18](https://react.dev/) + [TypeScript](https://www.typescriptlang.org/)
- **建置工具**: [Vite](https://vitejs.dev/)
- **UI 框架**: [Ant Design](https://ant.design/)
- **狀態管理**: [Zustand](https://github.com/pmndrs/zustand)
- **HTTP 客戶端**: [Axios](https://axios-http.com/)
- **路由**: [React Router](https://reactrouter.com/)

## 📦 主要依賴

### 後端
```
github.com/gin-gonic/gin     # Web Framework
gorm.io/gorm                 # ORM 工具
gorm.io/driver/mysql         # MySQL 驅動
```

### 前端
```
react, react-dom             # React 核心
antd                         # UI 組件庫
axios                        # HTTP 客戶端
zustand                      # 狀態管理
react-router-dom             # 路由管理
dayjs                        # 日期處理
```

## 📁 專案結構

```
schedule-system/
├── main.go                          # 後端入口
├── go.mod, go.sum                   # Go 依賴管理
├── models/
│   ├── user.go                      # 使用者模型
│   └── shift.go                     # 班表模型 (ShiftRequirement, UserSchedule)
├── controllers/
│   ├── user_controller.go           # 使用者 API 控制器
│   └── shift_controller.go          # 班表 API 控制器
├── routes/
│   └── router.go                    # API 路由設定 (含 CORS)
├── db/
│   ├── connection.go                # 資料庫連線
│   └── migrate.go                   # 資料表遷移
├── frontend/                        # React 前端專案
│   ├── src/
│   │   ├── main.tsx                 # 前端入口
│   │   ├── App.tsx                  # 主元件
│   │   ├── pages/
│   │   │   ├── ScheduleCalendar.tsx # 排班行事曆頁面
│   │   │   └── UserList.tsx         # 使用者列表頁面
│   │   ├── services/
│   │   │   └── api.ts               # API 服務層
│   │   ├── store/
│   │   │   └── userStore.ts         # 狀態管理
│   │   ├── types/
│   │   │   └── index.ts             # TypeScript 型別定義
│   │   └── utils/
│   │       └── api.ts               # Axios 實例
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── start-backend.bat                # 後端啟動腳本
├── start-frontend.bat               # 前端啟動腳本
└── 啟動指南.md                      # 詳細啟動說明

```

## 🚀 快速開始

### 前置需求

1. **Go** (建議 1.21 或以上)
   - 下載：https://go.dev/dl/

2. **Node.js** (建議 18.x 或以上)
   - 下載：https://nodejs.org/

3. **MySQL** (建議 8.0 或以上)
   - 確保 MySQL 服務運行中
   - 建立資料庫：`CREATE DATABASE schedule_system;`

### 資料庫設定

編輯 `db/connection.go` 中的連線字串：
```go
dsn := "root:root1234@tcp(127.0.0.1:3306)/schedule_system?charset=utf8mb4&parseTime=True&loc=Local"
```

### 快速啟動（使用腳本）

#### 方法一：使用啟動腳本（推薦）

**終端機 1 - 啟動後端：**
```powershell
.\start-backend.bat
```

**終端機 2 - 啟動前端：**
```powershell
.\start-frontend.bat
```

#### 方法二：手動啟動

**後端（終端機 1）：**
```bash
go run main.go
```
後端將運行於 `http://localhost:8080`

**前端（終端機 2）：**
```bash
cd frontend
npm install    # 首次需要執行
npm run dev
```
前端將運行於 `http://localhost:3000`

### 訪問系統

在瀏覽器開啟 `http://localhost:3000`

## 📊 資料庫 Schema

### users (使用者表)
- `id` - 主鍵
- `name` - 姓名
- `email` - 電子郵件（唯一）
- `role` - 角色
- `status` - 狀態 (1=啟用, 0=停用)

### shift_requirements (班表需求表)
- `id` - 主鍵
- `date` - 日期
- `shift_type` - 班別 (morning/afternoon/evening)
- `required_count` - 需求人數
- 唯一索引：`(date, shift_type)`

### user_schedules (使用者排班表)
- `id` - 主鍵
- `user_id` - 使用者 ID
- `date` - 日期
- `shift_type` - 班別
- 唯一索引：`(user_id, date, shift_type)`

## 🔌 API 端點

### 使用者管理
- `GET /api/v1/users/` - 取得所有使用者
- `GET /api/v1/users/:id` - 取得單一使用者
- `POST /api/v1/users/` - 建立使用者
- `PUT /api/v1/users/:id` - 更新使用者
- `DELETE /api/v1/users/:id` - 刪除使用者

### 班表管理
- `POST /api/v1/shifts/requirements` - 設定班表需求
- `GET /api/v1/shifts/schedule?month=YYYY-MM` - 取得月份班表
- `POST /api/v1/shifts/book` - 預約班別

## 🧪 測試範例

### 建立使用者
```bash
curl -X POST http://localhost:8080/api/v1/users/ \
  -H "Content-Type: application/json" \
  -d '{
    "name": "測試員工",
    "email": "test@example.com",
    "role": "employee",
    "status": 1
  }'
```

### 設定班表需求
```bash
curl -X POST http://localhost:8080/api/v1/shifts/requirements \
  -H "Content-Type: application/json" \
  -d '{
    "date": "2025-11-21",
    "shift_type": "morning",
    "required_count": 3
  }'
```

### 預約班別
```bash
curl -X POST http://localhost:8080/api/v1/shifts/book \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "date": "2025-11-21",
    "shift_type": "morning"
  }'
```

## 📖 詳細文件

- [啟動指南](./啟動指南.md) - 完整的啟動步驟與測試說明
- [前端 README](./frontend/README.md) - 前端專案詳細說明

## 🛠️ 開發工具

專案提供了幾個實用工具腳本：

- `cleanup_db.go` - 清理資料表（解決建表錯誤時使用）
- `check_table.go` - 檢查資料表結構

## ⚠️ 常見問題

### Q: 後端啟動失敗，顯示資料庫連線錯誤？
**A**: 請確認 MySQL 服務正在運行，並檢查 `db/connection.go` 中的連線資訊是否正確。

### Q: 前端顯示 npm 指令找不到？
**A**: 請確認 Node.js 已正確安裝，並重新開啟終端機視窗。

### Q: 行事曆沒有顯示任何資料？
**A**: 請使用 API 或資料庫先設定至少一天的班表需求。

### Q: 資料表建立失敗？
**A**: 執行 `go run cleanup_db.go` 清理舊表後重試。

## 🎯 後續改進方向

- [ ] 實作 JWT 使用者認證
- [ ] 新增管理員與一般員工權限區分
- [ ] 支援排班衝突檢測
- [ ] 新增班表匯出功能（Excel/PDF）
- [ ] 實作通知系統
- [ ] 優化行動裝置介面
- [ ] 支援批次設定班表需求

## 📝 授權

MIT License

## 👨‍💻 作者

Schedule System Team

---

如有任何問題或建議，歡迎提交 Issue 或 Pull Request！