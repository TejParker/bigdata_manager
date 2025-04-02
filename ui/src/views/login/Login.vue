<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <h2 class="text-2xl font-bold mb-8 text-center">大数据集群管理平台</h2>
      </div>
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        layout="vertical"
        @submit="handleSubmit"
      >
        <a-form-item field="username" label="用户名">
          <a-input
            v-model="form.username"
            placeholder="请输入用户名"
            allow-clear
          >
            <template #prefix>
              <icon-user />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item field="password" label="密码">
          <a-input-password
            v-model="form.password"
            placeholder="请输入密码"
            allow-clear
          >
            <template #prefix>
              <icon-lock />
            </template>
          </a-input-password>
        </a-form-item>
        <div class="mb-4">
          <a-checkbox v-model="rememberMe">记住我</a-checkbox>
        </div>
        <a-form-item>
          <a-button
            type="primary"
            html-type="submit"
            long
            :loading="loading"
          >
            登录
          </a-button>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Message } from '@arco-design/web-vue';
import { useUserStore } from '@/stores/user';

const router = useRouter();
const route = useRoute();
const userStore = useUserStore();
const formRef = ref(null);
const loading = ref(false);
const rememberMe = ref(false);

const form = reactive({
  username: '',
  password: '',
});

const rules = {
  username: [
    { required: true, message: '请输入用户名' },
  ],
  password: [
    { required: true, message: '请输入密码' },
  ],
};

const handleSubmit = async () => {
  formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true;
      
      try {
        const result = await userStore.login(form.username, form.password);
        
        if (result.success) {
          Message.success('登录成功');
          const redirectPath = route.query.redirect || '/';
          router.push(redirectPath);
        } else {
          Message.error(result.message);
        }
      } catch (error) {
        console.error('登录失败', error);
        Message.error('登录失败，请稍后再试');
      } finally {
        loading.value = false;
      }
    }
  });
};
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(to right, #1e3c72, #2a5298);
}

.login-card {
  width: 360px;
  padding: 30px;
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.1);
}

.login-header {
  margin-bottom: 20px;
}
</style> 