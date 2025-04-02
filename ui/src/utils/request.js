import axios from 'axios';
import { Message } from '@arco-design/web-vue';
import router from '@/router';

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    // 对于业务错误，直接返回完整响应，在业务代码中处理
    return response;
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response;
      
      switch (status) {
        case 401:
          // 未认证，重定向到登录页
          if (router.currentRoute.value.name !== 'Login') {
            Message.error('登录已过期，请重新登录');
            localStorage.removeItem('token');
            localStorage.removeItem('userInfo');
            router.push({
              name: 'Login',
              query: { redirect: router.currentRoute.value.fullPath },
            });
          }
          break;
        case 403:
          // 权限不足
          Message.error(data.message || '权限不足');
          break;
        case 404:
          // 资源未找到
          Message.error(data.message || '请求的资源不存在');
          break;
        case 500:
          // 服务器错误
          Message.error(data.message || '服务器内部错误，请稍后再试');
          break;
        default:
          // 其他错误
          Message.error(data.message || `请求失败(${status})`);
      }
    } else if (error.request) {
      // 网络错误，请求发出但未收到响应
      Message.error('网络错误，无法连接到服务器');
    } else {
      // 请求配置错误
      Message.error('请求错误: ' + error.message);
    }
    
    return Promise.reject(error);
  }
);

export default request; 