import { defineStore } from 'pinia';
import axios from 'axios';

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userInfo: JSON.parse(localStorage.getItem('userInfo') || 'null'),
  }),
  
  getters: {
    isLoggedIn: (state) => !!state.token,
    hasPrivilege: (state) => (privilege) => {
      if (!state.userInfo || !state.userInfo.privileges) return false;
      return state.userInfo.privileges.includes(privilege);
    }
  },
  
  actions: {
    async login(username, password) {
      try {
        const response = await axios.post('/api/v1/login', {
          username,
          password
        });
        
        if (response.data.success) {
          const { token, user_id } = response.data.data;
          this.token = token;
          localStorage.setItem('token', token);
          
          // 获取用户信息
          await this.fetchUserInfo();
          
          return { success: true };
        } else {
          return { success: false, message: response.data.message };
        }
      } catch (error) {
        if (error.response) {
          return { success: false, message: error.response.data.message || '登录失败' };
        }
        return { success: false, message: '网络错误，请稍后再试' };
      }
    },
    
    async fetchUserInfo() {
      try {
        const response = await axios.get('/api/v1/user/info', {
          headers: { Authorization: `Bearer ${this.token}` }
        });
        
        if (response.data.success) {
          this.userInfo = response.data.data;
          localStorage.setItem('userInfo', JSON.stringify(response.data.data));
        }
      } catch (error) {
        console.error('获取用户信息失败', error);
      }
    },
    
    logout() {
      this.token = '';
      this.userInfo = null;
      localStorage.removeItem('token');
      localStorage.removeItem('userInfo');
    }
  }
}); 