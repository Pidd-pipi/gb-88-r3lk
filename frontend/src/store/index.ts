import { defineStore } from 'pinia';
import type { User, Project, MockAPI, APIVersion, RequestLog } from '../types';
import { authApi, projectApi, mockApiApi, requestLogApi } from '../api';

interface AuthState {
  token: string | null;
  user: User | null;
  isAuthenticated: boolean;
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: localStorage.getItem('token'),
    user: localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!) : null,
    isAuthenticated: !!localStorage.getItem('token')
  }),

  actions: {
    async login(username: string, password: string) {
      const response = await authApi.login(username, password);
      if (response.data.success) {
        this.token = response.data.data!.token;
        this.user = response.data.data!.user;
        this.isAuthenticated = true;
        localStorage.setItem('token', this.token!);
        localStorage.setItem('user', JSON.stringify(this.user));
      }
      return response.data;
    },

    async register(username: string, password: string) {
      const response = await authApi.register(username, password);
      if (response.data.success) {
        this.token = response.data.data!.token;
        this.user = response.data.data!.user;
        this.isAuthenticated = true;
        localStorage.setItem('token', this.token!);
        localStorage.setItem('user', JSON.stringify(this.user));
      }
      return response.data;
    },

    logout() {
      this.token = null;
      this.user = null;
      this.isAuthenticated = false;
      localStorage.removeItem('token');
      localStorage.removeItem('user');
    }
  }
});

interface ProjectState {
  projects: Project[];
  currentProject: Project | null;
  apis: MockAPI[];
  versions: APIVersion[];
  logs: RequestLog[];
  loading: boolean;
}

export const useProjectStore = defineStore('project', {
  state: (): ProjectState => ({
    projects: [],
    currentProject: null,
    apis: [],
    versions: [],
    logs: [],
    loading: false
  }),

  actions: {
    async fetchProjects() {
      this.loading = true;
      try {
        const response = await projectApi.getProjects();
        if (response.data.success) {
          this.projects = response.data.data!;
        }
      } finally {
        this.loading = false;
      }
    },

    async createProject(data: { name: string; description: string }) {
      const response = await projectApi.createProject(data);
      if (response.data.success) {
        this.projects.unshift(response.data.data!);
      }
      return response.data;
    },

    async deleteProject(id: string) {
      const response = await projectApi.deleteProject(id);
      if (response.data.success) {
        this.projects = this.projects.filter((p) => p._id !== id);
      }
      return response.data;
    },

    setCurrentProject(project: Project) {
      this.currentProject = project;
    },

    async fetchAPIs(projectId: string) {
      this.loading = true;
      try {
        const response = await mockApiApi.getAPIs(projectId);
        if (response.data.success) {
          this.apis = response.data.data!;
        }
      } finally {
        this.loading = false;
      }
    },

    async createAPI(projectId: string, data: Partial<MockAPI>) {
      const response = await mockApiApi.createAPI(projectId, data);
      if (response.data.success) {
        this.apis.unshift(response.data.data!);
      }
      return response.data;
    },

    async updateAPI(projectId: string, id: string, data: Partial<MockAPI> & { baseVersion?: number }) {
      const response = await mockApiApi.updateAPI(projectId, id, data);
      if (response.data.success) {
        const index = this.apis.findIndex((a) => a._id === id);
        if (index !== -1) {
          this.apis[index] = response.data.data!;
        }
      }
      return response.data;
    },

    async deleteAPI(projectId: string, id: string) {
      const response = await mockApiApi.deleteAPI(projectId, id);
      if (response.data.success) {
        this.apis = this.apis.filter((a) => a._id !== id);
      }
      return response.data;
    },

    async fetchVersions(projectId: string, apiId: string) {
      const response = await mockApiApi.getVersions(projectId, apiId);
      if (response.data.success) {
        this.versions = response.data.data!;
      }
      return response.data;
    },

    async publishVersion(projectId: string, apiId: string, version: number, expectedVersion: number) {
      const response = await mockApiApi.publishVersion(projectId, apiId, version, expectedVersion);
      if (response.data.success) {
        const index = this.apis.findIndex((a) => a._id === apiId);
        if (index !== -1) {
          this.apis[index] = response.data.data!;
        }
      }
      return response.data;
    },

    async fetchLogs(projectId: string) {
      this.loading = true;
      try {
        const response = await requestLogApi.getLogs(projectId);
        if (response.data.success) {
          this.logs = response.data.data!.logs;
        }
      } finally {
        this.loading = false;
      }
    },

    async clearLogs(projectId: string) {
      const response = await requestLogApi.clearLogs(projectId);
      if (response.data.success) {
        this.logs = [];
      }
      return response.data;
    }
  }
});
