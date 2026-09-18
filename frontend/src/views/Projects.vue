<template>
  <div>
    <div class="page-header">
      <h2>项目列表</h2>
      <a-button type="primary" @click="showCreateModal = true">
        <template #icon><icon-plus /></template>
        新建项目
      </a-button>
    </div>

    <a-card :loading="projectStore.loading">
      <a-table :data="projectStore.projects" :pagination="false">
        <template #columns>
          <a-table-column title="项目名称" data-index="name" width="200" />
          <a-table-column title="描述" data-index="description">
            <template #cell="{ record }">
              {{ record.description || '暂无描述' }}
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="createdAt" width="200">
            <template #cell="{ record }">
              {{ formatDate(record.createdAt) }}
            </template>
          </a-table-column>
          <a-table-column title="操作" width="150">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" size="small" @click="goToProject(record)">
                  进入
                </a-button>
                <a-button type="text" size="small" status="danger" @click="handleDelete(record)">
                  删除
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
        <template #empty>
          <a-empty description="暂无项目，点击右上角新建项目" />
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="showCreateModal" title="新建项目" @ok="handleCreate">
      <a-form :model="createForm" layout="vertical">
        <a-form-item field="name" label="项目名称">
          <a-input v-model="createForm.name" placeholder="请输入项目名称" />
        </a-form-item>
        <a-form-item field="description" label="项目描述">
          <a-textarea v-model="createForm.description" placeholder="请输入项目描述" :auto-size="{ minRows: 3 }" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import { IconPlus } from '@arco-design/web-vue/es/icon';
import { useProjectStore } from '../store';
import type { Project } from '../types';

const router = useRouter();
const projectStore = useProjectStore();

const showCreateModal = ref(false);
const createForm = ref({
  name: '',
  description: ''
});

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleString('zh-CN');
}

function goToProject(project: Project) {
  projectStore.setCurrentProject(project);
  router.push(`/projects/${project._id}/apis`);
}

async function handleCreate() {
  if (!createForm.value.name) {
    Message.warning('请输入项目名称');
    return;
  }

  try {
    const result = await projectStore.createProject(createForm.value);
    if (result.success) {
      Message.success('创建成功');
      showCreateModal.value = false;
      createForm.value = { name: '', description: '' };
    }
  } catch (error: any) {
    Message.error(error.response?.data?.error || '创建失败');
  }
}

async function handleDelete(project: Project) {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除项目「${project.name}」吗？此操作不可恢复。`,
    onOk: async () => {
      try {
        const result = await projectStore.deleteProject(project._id);
        if (result.success) {
          Message.success('删除成功');
        }
      } catch (error: any) {
        Message.error(error.response?.data?.error || '删除失败');
      }
    }
  });
}

onMounted(() => {
  projectStore.fetchProjects();
});
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
}
</style>
