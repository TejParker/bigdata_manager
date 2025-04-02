<template>
  <div class="layout">
    <a-layout class="h-full">
      <a-layout-header class="header">
        <div class="logo">
          <h1 class="text-white text-xl">大数据集群管理平台</h1>
        </div>
        <div class="flex-1"></div>
        <div class="user-area">
          <a-dropdown trigger="click">
            <a-avatar class="mr-2">{{ userInfo?.user?.username?.charAt(0) || 'U' }}</a-avatar>
            <span class="text-white">{{ userInfo?.user?.username || '用户' }}</span>
            <template #content>
              <a-doption @click="openSettings">
                <icon-settings class="mr-2" />设置
              </a-doption>
              <a-doption @click="logout">
                <icon-poweroff class="mr-2" />退出
              </a-doption>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>
      <a-layout>
        <a-layout-sider
          collapsible
          :width="200"
          breakpoint="lg"
          :collapsed="collapsed"
          @collapse="onCollapse"
        >
          <a-menu
            :default-selected-keys="[selectedKey]"
            :default-open-keys="['cluster', 'monitor']"
            :style="{ width: '100%' }"
            @menu-item-click="handleMenuClick"
          >
            <a-menu-item key="dashboard">
              <template #icon><icon-dashboard /></template>
              仪表盘
            </a-menu-item>
            <a-sub-menu key="cluster">
              <template #icon><icon-apps /></template>
              <template #title>集群管理</template>
              <a-menu-item key="clusters">集群列表</a-menu-item>
              <a-menu-item key="hosts">主机管理</a-menu-item>
              <a-menu-item key="services">服务管理</a-menu-item>
            </a-sub-menu>
            <a-sub-menu key="monitor">
              <template #icon><icon-computer /></template>
              <template #title>监控与告警</template>
              <a-menu-item key="monitor">监控中心</a-menu-item>
              <a-menu-item key="alerts">告警管理</a-menu-item>
            </a-sub-menu>
            <a-menu-item key="logs">
              <template #icon><icon-file /></template>
              日志管理
            </a-menu-item>
            <a-menu-item key="settings">
              <template #icon><icon-settings /></template>
              系统设置
            </a-menu-item>
          </a-menu>
        </a-layout-sider>
        <a-layout class="layout-content">
          <a-layout-content class="content">
            <router-view v-slot="{ Component }">
              <transition name="fade">
                <keep-alive :include="['Dashboard', 'Clusters', 'Hosts', 'Services']">
                  <component :is="Component" />
                </keep-alive>
              </transition>
            </router-view>
          </a-layout-content>
          <a-layout-footer class="footer">
            大数据集群管理平台 &copy; {{ new Date().getFullYear() }}
          </a-layout-footer>
        </a-layout>
      </a-layout>
    </a-layout>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useUserStore } from '@/stores/user';

const router = useRouter();
const userStore = useUserStore();
const collapsed = ref(false);
const selectedKey = computed(() => {
  const route = router.currentRoute.value;
  const path = route.path.split('/')[1];
  return path || 'dashboard';
});

const userInfo = computed(() => userStore.userInfo);

onMounted(() => {
  if (!userStore.userInfo) {
    userStore.fetchUserInfo();
  }
});

const onCollapse = (value) => {
  collapsed.value = value;
};

const handleMenuClick = (key) => {
  router.push({ name: key.charAt(0).toUpperCase() + key.slice(1) });
};

const openSettings = () => {
  router.push({ name: 'Settings' });
};

const logout = () => {
  userStore.logout();
  router.push({ name: 'Login' });
};
</script>

<style scoped>
.layout {
  height: 100%;
}

.header {
  display: flex;
  align-items: center;
  background-color: #001529;
  padding: 0 20px;
}

.logo {
  height: 32px;
  margin: 16px 0;
  display: flex;
  align-items: center;
}

.user-area {
  display: flex;
  align-items: center;
  color: white;
  cursor: pointer;
}

.layout-content {
  padding: 0 16px;
}

.content {
  margin: 16px 0;
  padding: 16px;
  background: #fff;
  min-height: calc(100vh - 160px);
}

.footer {
  text-align: center;
  color: rgba(0, 0, 0, 0.45);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style> 