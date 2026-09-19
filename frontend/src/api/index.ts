import axios from 'axios';
import type { User, Project, MockAPI, APIVersion, RequestLog, Pagination, ApiResponse } from '../types';

// The Go backend wraps every response in { code, message, data }.
// This interceptor normalizes it to the frontend's { success, data, message } shape.
interface NormalizedError extends Error {
  response?: {
    data?: {
      error?: string;
      message?: string;
      code?: number;
    };
  };
}

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => {
    const body = response.data;
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) {
        response.data = { success: true, data: body.data, message: body.message || 'ok' };
      } else {
        const err: NormalizedError = new Error(body.message || '请求失败');
        err.response = { data: { error: body.message || '请求失败', code: body.code } };
        return Promise.reject(err);
      }
    }
    return response;
  },
  (error: NormalizedError & { response?: { status?: number; data?: { error?: string; message?: string } } }) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    // expose a uniform `error` field for the UI
    if (error.response?.data && error.response.data.error === undefined && error.response.data.message) {
      error.response.data.error = error.response.data.message;
    }
    return Promise.reject(error);
  }
);

export const authApi = {
  login: (username: string, password: string) =>
    api.post<ApiResponse<{ token: string; user: User }>>('/auth/login', { username, password }),
  register: (username: string, password: string) =>
    api.post<ApiResponse<{ token: string; user: User }>>('/auth/register', { username, password }),
  getCurrentUser: () => api.get<ApiResponse<User>>('/auth/me')
};

export const projectApi = {
  getProjects: () => api.get<ApiResponse<Project[]>>('/projects'),
  getProject: (id: string) => api.get<ApiResponse<Project>>(`/projects/${id}`),
  createProject: (data: { name: string; description: string }) =>
    api.post<ApiResponse<Project>>('/projects', data),
  updateProject: (id: string, data: { name: string; description: string }) =>
    api.put<ApiResponse<Project>>(`/projects/${id}`, data),
  deleteProject: (id: string) => api.delete<ApiResponse<void>>(`/projects/${id}`)
};

export const mockApiApi = {
  getAPIs: (projectId: string) => api.get<ApiResponse<MockAPI[]>>(`/projects/${projectId}/apis`),
  getAPI: (projectId: string, id: string) => api.get<ApiResponse<MockAPI>>(`/projects/${projectId}/apis/${id}`),
  createAPI: (projectId: string, data: Partial<MockAPI>) =>
    api.post<ApiResponse<MockAPI>>(`/projects/${projectId}/apis`, data),
  updateAPI: (projectId: string, id: string, data: Partial<MockAPI> & { baseVersion?: number }) =>
    api.put<ApiResponse<MockAPI>>(`/projects/${projectId}/apis/${id}`, data),
  deleteAPI: (projectId: string, id: string) => api.delete<ApiResponse<void>>(`/projects/${projectId}/apis/${id}`),
  getVersions: (projectId: string, id: string) =>
    api.get<ApiResponse<APIVersion[]>>(`/projects/${projectId}/apis/${id}/versions`),
  publishVersion: (projectId: string, id: string, version: number, expectedVersion: number) =>
    api.post<ApiResponse<MockAPI>>(`/projects/${projectId}/apis/${id}/versions/${version}/publish`, { expectedVersion })
};

export const requestLogApi = {
  getLogs: (projectId: string, page = 1, pageSize = 50) =>
    api.get<ApiResponse<{ logs: RequestLog[]; pagination: Pagination }>>(`/projects/${projectId}/logs`, {
      params: { page, page_size: pageSize }
    }),
  clearLogs: (projectId: string) => api.delete<ApiResponse<void>>(`/projects/${projectId}/logs`)
};
