import axios from 'axios';

const api = axios.create({
    baseURL: '/api/v1',
    timeout: 10000,
});

// Request interceptor
api.interceptors.request.use(
    (config) => {
        // 可以在這裡添加 token
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// Response interceptor
api.interceptors.response.use(
    (response) => {
        return response.data;
    },
    (error) => {
        console.error('API Error:', error);
        return Promise.reject(error);
    }
);

export default api;
