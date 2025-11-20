# 排班系統前端專案

## 專案架構
```
frontend/
├── src/
│   ├── main.tsx           # 應用程式入口
│   ├── App.tsx            # 主元件 (包含 Layout 和路由)
│   ├── pages/             # 頁面元件
│   │   ├── ScheduleCalendar.tsx  # 排班行事曆
│   │   └── UserList.tsx          # 使用者列表
│   ├── services/          # API 服務層
│   │   └── api.ts
│   ├── store/             # 狀態管理
│   │   └── userStore.ts
│   ├── types/             # TypeScript 型別定義
│   │   └── index.ts
│   └── utils/             # 工具函式
│       └── api.ts         # Axios 實例
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## 安裝與執行

### 前置需求
請先安裝 Node.js (建議版本 18.x 或以上)
下載連結：https://nodejs.org/

### 安裝步驟

1. **安裝依賴**
```bash
cd frontend
npm install
```

2. **啟動開發伺服器**
```bash
npm run dev
```

前端伺服器將在 `http://localhost:3000` 啟動

3. **建置生產版本**
```bash
npm run build
```

## 功能說明

### 1. 排班行事曆
- 顯示月曆視圖
- 每日顯示早中晚班的已排人數/需求人數
- 點擊日期可以預約班別
- 即時統計今日需求、已排班與缺額

### 2. 使用者管理
- 顯示所有使用者列表
- 選擇當前操作使用者（用於預約班別）

## 技術棧
- React 18
- TypeScript
- Vite
- Ant Design (UI 框架)
- Axios (HTTP 客戶端)
- Zustand (狀態管理)
- React Router (路由)
- Day.js (日期處理)

## API 端點
前端透過 Vite Proxy 將 `/api` 請求代理到後端 `http://localhost:8080`

主要 API：
- `GET /api/v1/shifts/schedule?month=YYYY-MM` - 取得月份班表
- `POST /api/v1/shifts/book` - 預約班別
- `POST /api/v1/shifts/requirements` - 設定班表需求
- `GET /api/v1/users/` - 取得使用者列表
