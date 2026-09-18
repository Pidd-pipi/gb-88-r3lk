<template>
  <div>
    <div class="page-header">
      <h3>API 配置</h3>
      <a-button type="primary" @click="showCreateModal = true">
        <template #icon><icon-plus /></template>
        新建 API
      </a-button>
    </div>

    <a-card :loading="projectStore.loading">
      <a-table :data="projectStore.apis" :pagination="false">
        <template #columns>
          <a-table-column title="方法" data-index="method" width="100">
            <template #cell="{ record }">
              <a-tag :color="getMethodColor(record.method)">{{ record.method }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="路径" data-index="path">
            <template #cell="{ record }">
              <code>{{ record.path }}</code>
            </template>
          </a-table-column>
          <a-table-column title="状态码" data-index="statusCode" width="100">
            <template #cell="{ record }">
              <a-tag>{{ record.statusCode }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="延迟" data-index="delay" width="80">
            <template #cell="{ record }">
              {{ record.delay || 0 }}ms
            </template>
          </a-table-column>
          <a-table-column title="Mock URL" width="250">
            <template #cell="{ record }">
              <div class="url-container">
                <code class="mock-url">{{ getMockUrl(record) }}</code>
                <a-button type="text" size="mini" @click="copyUrl(getMockUrl(record))">
                  <icon-copy />
                </a-button>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="操作" width="150">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" size="small" @click="handleEdit(record)">
                  编辑
                </a-button>
                <a-button type="text" size="small" status="danger" @click="handleDelete(record)">
                  删除
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
        <template #empty>
          <a-empty description="暂无 API，点击右上角新建 API" />
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:visible="showCreateModal"
      :title="editingAPI ? '编辑 API' : '新建 API'"
      @ok="handleSave"
      :width="800"
    >
      <a-form :model="apiForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item field="method" label="请求方法">
              <a-select v-model="apiForm.method" style="width: 100%">
                <a-option value="GET">GET</a-option>
                <a-option value="POST">POST</a-option>
                <a-option value="PUT">PUT</a-option>
                <a-option value="DELETE">DELETE</a-option>
                <a-option value="PATCH">PATCH</a-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item field="statusCode" label="响应状态码">
              <a-input-number v-model="apiForm.statusCode" :min="100" :max="599" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item field="path" label="API 路径">
          <a-input v-model="apiForm.path" placeholder="/api/users/:id" />
        </a-form-item>
        <a-form-item field="delay" label="响应延迟 (毫秒)">
          <a-input-number v-model="apiForm.delay" :min="0" style="width: 200px" />
        </a-form-item>
        <a-form-item field="responseBody" label="响应体 (JSON)">
          <MonacoEditor v-model="apiForm.responseBody" language="json" />
        </a-form-item>
        <a-collapse>
          <a-collapse-panel header="条件响应配置">
            <div class="conditions-section">
              <a-button type="outline" size="small" @click="addCondition">
                <template #icon><icon-plus /></template>
                添加条件
              </a-button>
              <div v-for="(condition, index) in apiForm.conditions" :key="index" class="condition-item">
                <a-row :gutter="8" align="middle">
                  <a-col :span="5">
                    <a-input v-model="condition.field" placeholder="字段名" />
                  </a-col>
                  <a-col :span="4">
                    <a-select v-model="condition.operator" style="width: 100%">
                      <a-option value="equals">等于</a-option>
                      <a-option value="contains">包含</a-option>
                      <a-option value="startsWith">开头</a-option>
                      <a-option value="endsWith">结尾</a-option>
                    </a-select>
                  </a-col>
                  <a-col :span="5">
                    <a-input v-model="condition.value" placeholder="匹配值" />
                  </a-col>
                  <a-col :span="3">
                    <a-input-number v-model="condition.statusCode" :min="100" :max="599" style="width: 100%" />
                  </a-col>
                  <a-col :span="6">
                    <a-input v-model="condition.responseBody" placeholder="响应体" />
                  </a-col>
                  <a-col :span="1">
                    <a-button type="text" status="danger" @click="removeCondition(index)">
                      <icon-delete />
                    </a-button>
                  </a-col>
                </a-row>
              </div>
            </div>
          </a-collapse-panel>
        </a-collapse>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import { IconPlus, IconCopy, IconDelete } from '@arco-design/web-vue/es/icon';
import { useProjectStore } from '../store';
import type { MockAPI } from '../types';
import MonacoEditor from '../components/MonacoEditor.vue';

const route = useRoute();
const projectStore = useProjectStore();

const showCreateModal = ref(false);
const editingAPI = ref<MockAPI | null>(null);
const apiForm = ref({
  method: 'GET',
  path: '',
  statusCode: 200,
  responseBody: '{}',
  delay: 0,
  conditions: [] as any[]
});

const projectId = computed(() => route.params.projectId as string);

function getMethodColor(method: string) {
  const colors: Record<string, string> = {
    GET: 'green',
    POST: 'blue',
    PUT: 'orange',
    DELETE: 'red',
    PATCH: 'purple'
  };
  return colors[method] || 'gray';
}

function getMockUrl(api: MockAPI) {
  return `/mock/${projectId.value}${api.path}`;
}

function copyUrl(url: string) {
  navigator.clipboard.writeText(window.location.origin + url);
  Message.success('已复制');
}

function handleEdit(api: MockAPI) {
  editingAPI.value = api;
  apiForm.value = {
    method: api.method,
    path: api.path,
    statusCode: api.statusCode,
    responseBody: api.responseBody,
    delay: api.delay,
    conditions: [...api.conditions]
  };
  showCreateModal.value = true;
}

async function handleSave() {
  if (!apiForm.value.path) {
    Message.warning('请输入 API 路径');
    return;
  }

  try {
    if (editingAPI.value) {
      const result = await projectStore.updateAPI(projectId.value, editingAPI.value._id, apiForm.value);
      if (result.success) {
        Message.success('更新成功');
      }
    } else {
      const result = await projectStore.createAPI(projectId.value, apiForm.value);
      if (result.success) {
        Message.success('创建成功');
      }
    }
    showCreateModal.value = false;
    resetForm();
  } catch (error: any) {
    Message.error(error.response?.data?.error || '保存失败');
  }
}

function handleDelete(api: MockAPI) {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除 API「${api.method} ${api.path}」吗？`,
    onOk: async () => {
      try {
        const result = await projectStore.deleteAPI(projectId.value, api._id);
        if (result.success) {
          Message.success('删除成功');
        }
      } catch (error: any) {
        Message.error(error.response?.data?.error || '删除失败');
      }
    }
  });
}

function addCondition() {
  apiForm.value.conditions.push({
    field: '',
    operator: 'equals',
    value: '',
    responseBody: '{}',
    statusCode: 200
  });
}

function removeCondition(index: number) {
  apiForm.value.conditions.splice(index, 1);
}

function resetForm() {
  editingAPI.value = null;
  apiForm.value = {
    method: 'GET',
    path: '',
    statusCode: 200,
    responseBody: '{}',
    delay: 0,
    conditions: []
  };
}

onMounted(() => {
  projectStore.fetchAPIs(projectId.value);
});
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h3 {
  margin: 0;
}

.url-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mock-url {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.conditions-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.condition-item {
  padding: 12px;
  background: #f7f8fa;
  border-radius: 4px;
}
</style>
