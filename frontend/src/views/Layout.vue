<template>
  <a-layout style="min-height: 100vh">
    <a-layout-header class="header">
      <div class="logo">
        <icon-apps class="logo-icon" />
        <span>MockHub · 在线 API Mock 服务</span>
      </div>
      <div class="header-right">
        <span class="username">{{ authStore.user?.username }}</span>
        <a-button type="text" @click="handleLogout">退出</a-button>
      </div>
    </a-layout-header>
    <a-layout>
      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import { useAuthStore } from '../store';
import { IconApps } from '@arco-design/web-vue/es/icon';

const router = useRouter();
const authStore = useAuthStore();

function handleLogout() {
  authStore.logout();
  router.push('/login');
}
</script>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #0f172a;
  color: white;
  padding: 0 24px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 600;
}

.logo-icon {
  font-size: 24px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.username {
  color: #94a3b8;
}

.content {
  padding: 24px;
  background: #f7f8fa;
}
</style>
