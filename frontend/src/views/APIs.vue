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
          <a-table-column title="方法" data-index="method" :width="100">
            <template #cell="{ record }">
              <a-tag :color="getMethodColor(record.method)">{{ record.method }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="路径" data-index="path">
            <template #cell="{ record }">
              <code>{{ record.path }}</code>
            </template>
          </a-table-column>
          <a-table-column title="状态码" data-index="statusCode" :width="100">
            <template #cell="{ record }">
              <a-tag>{{ record.statusCode }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="当前版本" data-index="currentVersion" :width="100">
            <template #cell="{ record }">
              <a-tag color="arcoblue">v{{ record.currentVersion }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="延迟" data-index="delay" :width="80">
            <template #cell="{ record }">
              {{ record.delay || 0 }}ms
            </template>
          </a-table-column>
          <a-table-column title="Mock URL" :width="250">
            <template #cell="{ record }">
              <div class="url-container">
                <code class="mock-url">{{ getMockUrl(record) }}</code>
                <a-button type="text" size="mini" @click="copyUrl(getMockUrl(record))">
                  <icon-copy />
                </a-button>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="210">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" size="small" @click="handleEdit(record)">
                  编辑
                </a-button>
                <a-button type="text" size="small" @click="handleHistory(record)">
                  历史
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
      <a-alert v-if="editingAPI" class="edit-tip" type="info">
        正在编辑版本 v{{ editingAPI.currentVersion }}，保存后将生成新版本 v{{ editingAPI.currentVersion + 1 }} 并立即生效
      </a-alert>
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
                <a-row :gutter="8" align="center">
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

    <a-drawer
      v-model:visible="showHistory"
      :title="`版本历史 · ${historyAPI?.method || ''} ${historyAPI?.path || ''}`"
      :width="760"
    >
      <a-table :data="projectStore.versions" :pagination="false" :loading="versionsLoading">
        <template #columns>
          <a-table-column title="版本" data-index="version" :width="110">
            <template #cell="{ record }">
              <a-space>
                <a-tag :color="record.version === historyAPI?.currentVersion ? 'green' : 'gray'">
                  v{{ record.version }}
                </a-tag>
                <a-tag v-if="record.version === historyAPI?.currentVersion" color="green" size="small">
                  当前
                </a-tag>
              </a-space>
            </template>
          </a-table-column>
          <a-table-column title="方法" data-index="method" :width="90">
            <template #cell="{ record }">
              <a-tag :color="getMethodColor(record.method)" size="small">{{ record.method }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="路径" data-index="path">
            <template #cell="{ record }">
              <code>{{ record.path }}</code>
            </template>
          </a-table-column>
          <a-table-column title="操作人" data-index="editorName" :width="110">
            <template #cell="{ record }">
              {{ record.editorName || '-' }}
            </template>
          </a-table-column>
          <a-table-column title="时间" data-index="createdAt" :width="170">
            <template #cell="{ record }">
              {{ formatDate(record.createdAt) }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="130">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" size="small" @click="handlePreview(record)">
                  预览
                </a-button>
                <a-button
                  v-if="record.version !== historyAPI?.currentVersion"
                  type="text"
                  size="small"
                  status="success"
                  @click="handlePublish(record)"
                >
                  发布
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
        <template #empty>
          <a-empty description="暂无版本记录" />
        </template>
      </a-table>
    </a-drawer>

    <a-modal
      v-model:visible="showPreview"
      :title="`版本预览 · v${previewVersion?.version || ''}`"
      :width="760"
      :footer="false"
    >
      <template v-if="previewVersion">
        <a-descriptions :column="2" bordered size="small" class="preview-meta">
          <a-descriptions-item label="方法">{{ previewVersion.method }}</a-descriptions-item>
          <a-descriptions-item label="路径">{{ previewVersion.path }}</a-descriptions-item>
          <a-descriptions-item label="状态码">{{ previewVersion.statusCode }}</a-descriptions-item>
          <a-descriptions-item label="延迟">{{ previewVersion.delay || 0 }}ms</a-descriptions-item>
          <a-descriptions-item label="操作人">{{ previewVersion.editorName || '-' }}</a-descriptions-item>
          <a-descriptions-item label="时间">{{ formatDate(previewVersion.createdAt) }}</a-descriptions-item>
        </a-descriptions>
        <h4 class="preview-section">响应体</h4>
        <MonacoEditor :model-value="previewVersion.responseBody" language="json" read-only />
        <template v-if="previewVersion.conditions && previewVersion.conditions.length > 0">
          <h4 class="preview-section">条件响应</h4>
          <pre class="preview-json">{{ formatJson(previewVersion.conditions) }}</pre>
        </template>
        <template v-if="hasResponseHeaders">
          <h4 class="preview-section">响应头</h4>
          <pre class="preview-json">{{ formatJson(previewVersion.responseHeaders) }}</pre>
        </template>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import { IconPlus, IconCopy, IconDelete } from '@arco-design/web-vue/es/icon';
import { useProjectStore } from '../store';
import { mockApiApi } from '../api';
import type { MockAPI, APIVersion } from '../types';
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

const showHistory = ref(false);
const historyAPI = ref<MockAPI | null>(null);
const versionsLoading = ref(false);
const showPreview = ref(false);
const previewVersion = ref<APIVersion | null>(null);

const projectId = computed(() => route.params.projectId as string);

const hasResponseHeaders = computed(
  () => previewVersion.value && Object.keys(previewVersion.value.responseHeaders || {}).length > 0
);

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

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleString('zh-CN');
}

function formatJson(data: any) {
  if (!data) return '{}';
  try {
    return JSON.stringify(typeof data === 'string' ? JSON.parse(data) : data, null, 2);
  } catch {
    return JSON.stringify(data, null, 2);
  }
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
      // baseVersion 是乐观锁令牌：若期间他人已保存，后端返回 409 且当前版本不变
      const result = await projectStore.updateAPI(projectId.value, editingAPI.value._id, {
        ...apiForm.value,
        baseVersion: editingAPI.value.currentVersion
      });
      if (result.success) {
        Message.success(`已保存为版本 v${result.data!.currentVersion}`);
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
    if (error.response?.status === 409) {
      Message.error(error.response?.data?.error || '接口已被他人修改，请刷新后重试');
      projectStore.fetchAPIs(projectId.value);
      showCreateModal.value = false;
      resetForm();
      return;
    }
    Message.error(error.response?.data?.error || '保存失败');
  }
}

function handleDelete(api: MockAPI) {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除 API「${api.method} ${api.path}」吗？其全部版本历史将一并删除。`,
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

function handleHistory(api: MockAPI) {
  historyAPI.value = api;
  showHistory.value = true;
  loadVersions();
}

async function loadVersions() {
  if (!historyAPI.value) return;
  versionsLoading.value = true;
  try {
    await projectStore.fetchVersions(projectId.value, historyAPI.value._id);
  } catch (error: any) {
    Message.error(error.response?.data?.error || '加载版本历史失败');
  } finally {
    versionsLoading.value = false;
  }
}

async function handlePreview(record: APIVersion) {
  if (!historyAPI.value) return;
  try {
    const response = await mockApiApi.getVersion(projectId.value, historyAPI.value._id, record.version);
    if (response.data.success) {
      previewVersion.value = response.data.data!;
      showPreview.value = true;
    }
  } catch (error: any) {
    Message.error(error.response?.data?.error || '加载版本详情失败');
  }
}

function handlePublish(record: APIVersion) {
  if (!historyAPI.value) return;
  const api = historyAPI.value;
  Modal.confirm({
    title: '发布版本',
    content: `确定要将 v${record.version} 发布为当前版本吗？当前版本 v${api.currentVersion} 的配置将被替换（历史版本仍保留）。`,
    onOk: async () => {
      try {
        const result = await projectStore.publishVersion(
          projectId.value,
          api._id,
          record.version,
          api.currentVersion
        );
        if (result.success) {
          Message.success(`已发布 v${record.version}`);
          historyAPI.value = result.data!;
          await loadVersions();
        }
      } catch (error: any) {
        if (error.response?.status === 409) {
          Message.error(error.response?.data?.error || '接口已被他人修改，请刷新后重试');
          await projectStore.fetchAPIs(projectId.value);
          const fresh = projectStore.apis.find((a) => a._id === api._id);
          if (fresh) {
            historyAPI.value = fresh;
          }
          await loadVersions();
          return;
        }
        Message.error(error.response?.data?.error || '发布失败');
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

.edit-tip {
  margin-bottom: 16px;
}

.preview-meta {
  margin-bottom: 16px;
}

.preview-section {
  margin: 16px 0 8px 0;
  font-size: 13px;
  color: #4e5969;
}

.preview-json {
  margin: 0;
  padding: 12px;
  background: #f7f8fa;
  border-radius: 4px;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
}
</style>
