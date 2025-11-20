import React from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider } from 'antd'
import zhTW from 'antd/locale/zh_TW'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <ConfigProvider locale={zhTW}>
            <App />
        </ConfigProvider>
    </React.StrictMode>,
)
