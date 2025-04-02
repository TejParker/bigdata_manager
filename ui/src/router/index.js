import { createRouter, createWebHistory } from 'vue-router';
import { useUserStore } from '@/stores/user';

// 布局组件
const Layout = () => import('@/layouts/DefaultLayout.vue');

// 路由配置
const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/Login.vue'),
    meta: {
      title: '登录',
      requiresAuth: false
    }
  },
  {
    path: '/',
    component: Layout,
    meta: {
      requiresAuth: true
    },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/Dashboard.vue'),
        meta: { title: '仪表盘' }
      },
      {
        path: 'clusters',
        name: 'Clusters',
        component: () => import('@/views/clusters/ClusterList.vue'),
        meta: { title: '集群管理' }
      },
      {
        path: 'clusters/:id',
        name: 'ClusterDetail',
        component: () => import('@/views/clusters/ClusterDetail.vue'),
        meta: { title: '集群详情' }
      },
      {
        path: 'hosts',
        name: 'Hosts',
        component: () => import('@/views/hosts/HostList.vue'),
        meta: { title: '主机管理' }
      },
      {
        path: 'hosts/:id',
        name: 'HostDetail',
        component: () => import('@/views/hosts/HostDetail.vue'),
        meta: { title: '主机详情' }
      },
      {
        path: 'services',
        name: 'Services',
        component: () => import('@/views/services/ServiceList.vue'),
        meta: { title: '服务管理' }
      },
      {
        path: 'services/:id',
        name: 'ServiceDetail',
        component: () => import('@/views/services/ServiceDetail.vue'),
        meta: { title: '服务详情' }
      },
      {
        path: 'monitor',
        name: 'Monitor',
        component: () => import('@/views/monitor/MonitorDashboard.vue'),
        meta: { title: '监控中心' }
      },
      {
        path: 'alerts',
        name: 'Alerts',
        component: () => import('@/views/monitor/AlertList.vue'),
        meta: { title: '告警管理' }
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('@/views/logs/LogViewer.vue'),
        meta: { title: '日志管理' }
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/settings/Settings.vue'),
        meta: { title: '系统设置' }
      }
    ]
  },
  {
    // 匹配所有未找到的路由
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '页面未找到', requiresAuth: false }
  }
];

const router = createRouter({
  history: createWebHistory('/'),
  routes
});

// 路由守卫
router.beforeEach((to, from, next) => {
  // 设置页面标题
  document.title = to.meta.title ? `${to.meta.title} - 大数据集群管理平台` : '大数据集群管理平台';
  
  // 权限校验
  const userStore = useUserStore();
  const requiresAuth = to.matched.some(record => record.meta.requiresAuth);
  
  if (requiresAuth && !userStore.isLoggedIn) {
    next({ name: 'Login', query: { redirect: to.fullPath } });
  } else {
    next();
  }
});

export default router; 