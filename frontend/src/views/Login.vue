<template>
  <div class="login-container">
    <a-card class="login-card" title="MockHub · 在线 API Mock 服务">
      <a-form :model="form" @submit="handleLogin">
        <a-form-item field="username" label="用户名">
          <a-input v-model="form.username" placeholder="请输入用户名" />
        </a-form-item>
        <a-form-item field="password" label="密码">
          <a-input-password v-model="form.password" placeholder="请输入密码" />
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" html-type="submit" :loading="loading">
              登录
            </a-button>
            <a-button @click="handleRegister" :loading="loading">
              注册
            </a-button>
          </a-space>
        </a-form-item>
      </a-form>
      <div class="tips">
        <p>测试账号：dev / dev123</p>
        <p>管理员账号：lead / lead123</p>
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { Message } from '@arco-design/web-vue';
import { useAuthStore } from '../store';

const router = useRouter();
const authStore = useAuthStore();

const loading = ref(false);
const form = ref({
  username: '',
  password: ''
});

async function handleLogin() {
  if (!form.value.username || !form.value.password) {
    Message.warning('请输入用户名和密码');
    return;
  }

  loading.value = true;
  try {
    const result = await authStore.login(form.value.username, form.value.password);
    if (result.success) {
      Message.success('登录成功');
      router.push('/projects');
    } else {
      Message.error(result.error || '登录失败');
    }
  } catch (error: any) {
    Message.error(error.response?.data?.error || '登录失败');
  } finally {
    loading.value = false;
  }
}

async function handleRegister() {
  if (!form.value.username || !form.value.password) {
    Message.warning('请输入用户名和密码');
    return;
  }

  loading.value = true;
  try {
    const result = await authStore.register(form.value.username, form.value.password);
    if (result.success) {
      Message.success('注册成功');
      router.push('/projects');
    } else {
      Message.error(result.error || '注册失败');
    }
  } catch (error: any) {
    Message.error(error.response?.data?.error || '注册失败');
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 400px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.tips {
  margin-top: 20px;
  padding: 15px;
  background: #f7f8fa;
  border-radius: 4px;
  font-size: 12px;
  color: #6b778c;
}

.tips p {
  margin: 5px 0;
}
</style>
