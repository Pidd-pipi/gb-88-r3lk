<template>
  <div>
    <div class="page-header">
      <h3>请求日志</h3>
      <a-space>
        <a-button @click="refreshLogs">
          <template #icon><icon-refresh /></template>
          刷新
        </a-button>
        <a-button status="danger" @click="handleClear">
          <template #icon><icon-delete /></template>
          清空日志
        </a-button>
      </a-space>
    </div>

    <a-card :loading="projectStore.loading">
      <a-timeline reverse>
        <a-timeline-item
          v-for="log in projectStore.logs"
          :key="log._id"
          :color="getLogColor(log.responseStatus)"
        >
          <div class="log-item">
            <div class="log-header">
              <a-tag :color="getMethodColor(log.method)" size="small">
                {{ log.method }}
              </a-tag>
              <code class="log-path">{{ log.path }}</code>
              <a-tag :color="getStatusCodeColor(log.responseStatus)" size="small">
                {{ log.responseStatus }}
              </a-tag>
              <a-tooltip :content="log.apiVersion > 0 ? '实际生效的接口版本' : '该请求由未版本化的旧接口处理'">
                <a-tag v-if="log.apiVersion > 0" color="purple" size="small">v{{ log.apiVersion }}</a-tag>
                <a-tag v-else size="small">—</a-tag>
              </a-tooltip>
              <span class="log-time">{{ formatDate(log.createdAt) }}</span>
            </div>
            <a-collapse bordered>
              <a-collapse-panel header="请求详情">
                <a-row :gutter="16">
                  <a-col :span="12">
                    <div class="log-section">
                      <h4>请求头</h4>
                      <pre class="log-json">{{ formatJson(log.headers) }}</pre>
                    </div>
                    <div class="log-section">
                      <h4>查询参数</h4>
                      <pre class="log-json">{{ formatJson(log.query) }}</pre>
                    </div>
                  </a-col>
                  <a-col :span="12">
                    <div class="log-section">
                      <h4>请求体</h4>
                      <pre class="log-json">{{ formatJson(log.body) }}</pre>
                    </div>
                    <div class="log-section">
                      <h4>响应体</h4>
                      <pre class="log-json">{{ formatJson(log.responseBody) }}</pre>
                    </div>
                  </a-col>
                </a-row>
              </a-collapse-panel>
            </a-collapse>
          </div>
        </a-timeline-item>
      </a-timeline>
      <a-empty v-if="!projectStore.loading && projectStore.logs.length === 0" description="暂无请求日志" />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import { IconRefresh, IconDelete } from '@arco-design/web-vue/es/icon';
import { useProjectStore } from '../store';

const route = useRoute();
const projectStore = useProjectStore();

const projectId = computed(() => route.params.projectId as string);

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

function getStatusCodeColor(status: number) {
  if (status >= 200 && status < 300) return 'green';
  if (status >= 300 && status < 400) return 'blue';
  if (status >= 400 && status < 500) return 'orange';
  return 'red';
}

function getLogColor(status: number) {
  if (status >= 200 && status < 300) return 'green';
  if (status >= 400) return 'red';
  return 'blue';
}

function refreshLogs() {
  projectStore.fetchLogs(projectId.value);
}

function handleClear() {
  Modal.confirm({
    title: '确认清空',
    content: '确定要清空所有请求日志吗？此操作不可恢复。',
    onOk: async () => {
      try {
        const result = await projectStore.clearLogs(projectId.value);
        if (result.success) {
          Message.success('已清空');
        }
      } catch (error: any) {
        Message.error(error.response?.data?.error || '操作失败');
      }
    }
  });
}

onMounted(() => {
  projectStore.fetchLogs(projectId.value);
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

.log-item {
  width: 100%;
}

.log-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.log-path {
  flex: 1;
  font-size: 13px;
  color: #1d2129;
}

.log-time {
  font-size: 12px;
  color: #86909c;
}

.log-section {
  margin-bottom: 16px;
}

.log-section h4 {
  margin: 0 0 8px 0;
  font-size: 13px;
  color: #4e5969;
}

.log-json {
  margin: 0;
  padding: 12px;
  background: #f7f8fa;
  border-radius: 4px;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
}
</style>
