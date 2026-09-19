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
          <a-table-column title="当前版本" data-index="currentVersion" width="100">
            <template #cell="{ record }">
              <a-tag v-if="record.currentVersion > 0" color="arcoblue">v{{ record.currentVersion }}</a-tag>
              <a-tooltip v-else content="该接口创建于版本功能上线前，编辑后自动生成版本">
                <span class="legacy-version">—</span>
              </a-tooltip>
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
          <a-table-column title="操作" width="220">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" size="small" @click="handleEdit(record)">
                  编辑
                </a-button>
                <a-button type="text" size="small" @click="openVersions(record)">
                  历史版本
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
        <a-alert v-if="editingAPI" type="info" class="edit-version-tip">
          保存后将基于当前版本 v{{ editingBaseVersion }} 生成新版本 v{{ editingBaseVersion + 1 }}，历史版本不可变。
        </a-alert>
      </a-form>
    </a-modal>

    <a-drawer
      v-model:visible="versionDrawerVisible"
      :width="760"
      :title="`历史版本${versionApi ? ` · ${versionApi.method} ${versionApi.path}` : ''}`"
      unmount-on-close
    >
      <div v-if="versionApi" class="version-drawer">
        <a-alert type="info" class="version-tip">
          当前生效版本：<a-tag color="arcoblue" size="small">v{{ versionApi.currentVersion }}</a-tag>
          每次编辑都会生成新的不可变版本，可将任意历史版本重新发布（回滚）。
        </a-alert>
        <a-table :data="projectStore.versions" :pagination="false" size="small">
          <template #columns>
            <a-table-column title="版本" width="110">
              <template #cell="{ record }">
                <a-space size="mini">
                  <a-tag color="arcoblue">v{{ record.version }}</a-tag>
                  <a-tag v-if="record.version === versionApi.currentVersion" color="green" size="small">当前</a-tag>
                </a-space>
              </template>
            </a-table-column>
            <a-table-column title="操作人" data-index="createdByName" width="100" />
            <a-table-column title="时间" width="170">
              <template #cell="{ record }">
                {{ formatDate(record.createdAt) }}
              </template>
            </a-table-column>
            <a-table-column title="响应摘要">
              <template #cell="{ record }">
                <code class="version-summary">{{ summarize(record) }}</code>
              </template>
            </a-table-column>
            <a-table-column title="操作" width="170">
              <template #cell="{ record }">
                <a-space>
                  <a-button type="text" size="small" @click="previewVersion(record)">
                    预览
                  </a-button>
                  <a-button
                    type="text"
                    size="small"
                    status="success"
                    :disabled="record.version === versionApi.currentVersion"
                    @click="handlePublish(record)"
                  >
                    发布此版本
                  </a-button>
                </a-space>
              </template>
            </a-table-column>
          </template>
          <template #empty>
            <a-empty description="暂无历史版本，编辑接口后自动生成" />
          </template>
        </a-table>
      </div>
    </a-drawer>

    <a-modal
      v-model:visible="previewVisible"
      :title="previewing ? `版本预览 · v${previewing.version}` : '版本预览'"
      :width="800"
      :footer="false"
      unmount-on-close
    >
      <div v-if="previewing" class="version-preview">
        <a-descriptions :column="2" size="small" bordered>
          <a-descriptions-item label="方法">
            <a-tag :color="getMethodColor(previewing.method)">{{ previewing.method }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="路径">
            <code>{{ previewing.path }}</code>
          </a-descriptions-item>
          <a-descriptions-item label="状态码">{{ previewing.statusCode }}</a-descriptions-item>
          <a-descriptions-item label="延迟">{{ previewing.delay || 0 }}ms</a-descriptions-item>
          <a-descriptions-item label="操作人">{{ previewing.createdByName }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatDate(previewing.createdAt) }}</a-descriptions-item>
        </a-descriptions>
        <template v-if="hasHeaders(previewing)">
          <h4>响应头</h4>
          <pre class="preview-json">{{ formatJson(previewing.responseHeaders) }}</pre>
        </template>
        <template v-if="previewing.conditions && previewing.conditions.length > 0">
          <h4>条件响应</h4>
          <pre class="preview-json">{{ formatJson(previewing.conditions) }}</pre>
        </template>
        <h4>响应体</h4>
        <MonacoEditor :model-value="previewing.responseBody" language="json" read-only />
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import { IconPlus, IconCopy, IconDelete } from '@arco-design/web-vue/es/icon';
import { useProjectStore } from '../store';
import type { MockAPI, APIVersion } from '../types';
import MonacoEditor from '../components/MonacoEditor.vue';

const route = useRoute();
const projectStore = useProjectStore();

const showCreateModal = ref(false);
const editingAPI = ref<MockAPI | null>(null);
const editingBaseVersion = ref(0);
const apiForm = ref({
  method: 'GET',
  path: '',
  statusCode: 200,
  responseBody: '{}',
  delay: 0,
  conditions: [] as any[]
});

const versionDrawerVisible = ref(false);
const versionApi = ref<MockAPI | null>(null);
const previewVisible = ref(false);
const previewing = ref<APIVersion | null>(null);

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

function hasHeaders(version: APIVersion) {
  return version.responseHeaders && Object.keys(version.responseHeaders).length > 0;
}

function summarize(version: APIVersion) {
  const body = (version.responseBody || '').replace(/\s+/g, ' ');
  const summary = `${version.statusCode} ${body}`;
  return summary.length > 60 ? `${summary.slice(0, 60)}…` : summary;
}

function handleEdit(api: MockAPI) {
  editingAPI.value = api;
  editingBaseVersion.value = api.currentVersion;
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
      const result = await projectStore.updateAPI(projectId.value, editingAPI.value._id, {
        ...apiForm.value,
        baseVersion: editingBaseVersion.value
      });
      if (result.success) {
        Message.success(`已保存，当前版本 v${result.data?.currentVersion ?? ''}`);
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
    if (error.response?.data?.code === 40900) {
      // 版本冲突：关闭编辑框并刷新到最新版本，避免基于过期版本重复提交
      showCreateModal.value = false;
      resetForm();
    }
    await projectStore.fetchAPIs(projectId.value);
  }
}

function handleDelete(api: MockAPI) {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除 API「${api.method} ${api.path}」吗？其全部历史版本将一并删除。`,
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

async function openVersions(api: MockAPI) {
  versionApi.value = api;
  versionDrawerVisible.value = true;
  try {
    await projectStore.fetchVersions(projectId.value, api._id);
  } catch (error: any) {
    Message.error(error.response?.data?.error || '加载版本失败');
  }
}

function previewVersion(version: APIVersion) {
  previewing.value = version;
  previewVisible.value = true;
}

function handlePublish(version: APIVersion) {
  if (!versionApi.value) return;
  const api = versionApi.value;
  Modal.confirm({
    title: '发布版本',
    content: `确定要将「${api.method} ${api.path}」回滚到 v${version.version} 吗？当前 v${api.currentVersion} 的内容将被覆盖（历史版本仍会保留）。`,
    onOk: async () => {
      try {
        const result = await projectStore.publishVersion(
          projectId.value,
          api._id,
          version.version,
          api.currentVersion
        );
        if (result.success && result.data) {
          versionApi.value = result.data;
          Message.success(`已发布 v${version.version}`);
          await projectStore.fetchVersions(projectId.value, api._id);
        }
      } catch (error: any) {
        Message.error(error.response?.data?.error || '发布失败');
        // 并发发布冲突：刷新列表与版本，同步到最新状态
        await projectStore.fetchAPIs(projectId.value);
        const fresh = projectStore.apis.find((a) => a._id === api._id);
        if (fresh) {
          versionApi.value = fresh;
        }
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
  editingBaseVersion.value = 0;
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

.legacy-version {
  color: #86909c;
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

.edit-version-tip {
  margin-top: 12px;
}

.version-drawer {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-tip {
  flex-shrink: 0;
}

.version-summary {
  font-size: 12px;
  color: #4e5969;
}

.version-preview h4 {
  margin: 16px 0 8px;
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
