# 🗓️ Golang 排班系統 (Schedule System)

這是一套使用 **Golang + Gin + GORM + MySQL** 所開發的排班系統，支援員工管理、班表設定、自動排班邏輯與 API 存取，適合企業內部或團隊使用。

## 🚀 技術棧

- Backend: [Go](https://go.dev/) + [Gin](https://github.com/gin-gonic/gin)
- ORM: [GORM](https://gorm.io/)
- DB: MySQL
- Auth: JWT（可選）
- Version control: Git + GitHub
  
## 📦 主要依賴
- `github.com/gin-gonic/gin` - Web Framework
- `gorm.io/gorm` - ORM 工具
- `gorm.io/driver/mysql` - MySQL 驅動

## 📁 專案結構

```bash
.
├── main.go         # 專案入口
├── go.mod          # Go 模組檔案，紀錄套件依賴
├── go.sum          # 套件完整版本鎖定
├── README.md       # 專案說明
├── config/
│   └── db.go                  # MySQL 資料庫連線
├── models/
│   └── user.go                # User 模型
├── controllers/
│   └── user_controller.go     # User API 控制器
├── routes/
│   └── router.go              #  所有 API 路由集中管理