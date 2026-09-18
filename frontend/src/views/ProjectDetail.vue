<template>
  <div>
    <div class="page-header">
      <a-breadcrumb>
        <a-breadcrumb-item href="#/projects">项目列表</a-breadcrumb-item>
        <a-breadcrumb-item>{{ currentProject?.name }}</a-breadcrumb-item>
      </a-breadcrumb>
    </div>

    <a-tabs v-model:active-key="activeTab" type="card">
      <a-tab-pane key="apis" title="API 配置">
        <router-view />
      </a-tab-pane>
      <a-tab-pane key="logs" title="请求日志">
        <router-view />
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useProjectStore } from '../store';

const route = useRoute();
const router = useRouter();
const projectStore = useProjectStore();

const activeTab = ref('apis');

const currentProject = computed(() => projectStore.currentProject);

onMounted(() => {
  if (!projectStore.currentProject) {
    const projectId = route.params.projectId as string;
    const project = projectStore.projects.find((p) => p._id === projectId);
    if (project) {
      projectStore.setCurrentProject(project);
    }
  }

  if (route.path.includes('/logs')) {
    activeTab.value = 'logs';
  } else {
    activeTab.value = 'apis';
  }
});

watch(activeTab, (newTab) => {
  const projectId = route.params.projectId as string;
  if (newTab === 'logs') {
    router.push(`/projects/${projectId}/logs`);
  } else {
    router.push(`/projects/${projectId}/apis`);
  }
});
</script>

<style scoped>
.page-header {
  margin-bottom: 20px;
}
</style>
