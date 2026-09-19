export interface User {
  _id: string;
  username: string;
  role: string;
}

export interface Project {
  _id: string;
  name: string;
  description: string;
  baseUrl: string;
  userId: string;
  createdAt: string;
}

export interface ConditionRule {
  field: string;
  operator: string;
  value: string;
  responseBody: string;
  statusCode: number;
}

export interface MockAPI {
  _id: string;
  projectId: string;
  path: string;
  method: string;
  statusCode: number;
  responseBody: string;
  responseHeaders: Record<string, string>;
  delay: number;
  conditions: ConditionRule[];
  currentVersion: number;
  createdAt: string;
}

export interface APIVersion {
  _id: string;
  apiId: string;
  version: number;
  path: string;
  method: string;
  statusCode: number;
  responseBody: string;
  responseHeaders: Record<string, string>;
  delay: number;
  conditions: ConditionRule[];
  editorId: string;
  editorName: string;
  createdAt: string;
}

export interface RequestLog {
  _id: string;
  projectId: string;
  apiId?: string;
  apiVersion?: number;
  method: string;
  path: string;
  headers: Record<string, string>;
  body: unknown;
  query: Record<string, string>;
  responseStatus: number;
  responseBody: unknown;
  createdAt: string;
}

export interface Pagination {
  page: number;
  limit: number;
  total: number;
}

export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}
