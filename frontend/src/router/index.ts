import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../store';

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    name: 'Layout',
    component: () => import('../views/Layout.vue'),
    meta: { requiresAuth: true },
    redirect: '/projects',
    children: [
      {
        path: 'projects',
        name: 'Projects',
        component: () => import('../views/Projects.vue')
      },
      {
        path: 'projects/:projectId',
        name: 'ProjectDetail',
        component: () => import('../views/ProjectDetail.vue'),
        redirect: (to) => `/projects/${to.params.projectId as string}/apis`,
        children: [
          {
            path: 'apis',
            name: 'APIs',
            component: () => import('../views/APIs.vue')
          },
          {
            path: 'logs',
            name: 'Logs',
            component: () => import('../views/Logs.vue')
          }
        ]
      }
    ]
  }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore();
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login');
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/projects');
  } else {
    next();
  }
});

export default router;
