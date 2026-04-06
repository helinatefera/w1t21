import api from './client';
import type {
  ABTest,
  ABTestAssignment,
  ABTestResult,
  AnomalyEvent,
  ContentPerformance,
  FunnelResponse,
  IPRule,
  PaginatedResponse,
  RetentionCohort,
  User,
} from '../types';

// Analytics
export const getFunnel = (days = 7) =>
  api.get<FunnelResponse>('/analytics/funnel', { params: { days } });

export const getRetention = (days = 30) =>
  api.get<RetentionCohort[]>('/analytics/retention', { params: { days } });

export const getContentPerformance = (limit = 20) =>
  api.get<ContentPerformance[]>('/analytics/content-performance', { params: { limit } });

// A/B Tests
export const listABTests = () => api.get<ABTest[]>('/ab-tests');

export const getABTest = (id: string) =>
  api.get<{ test: ABTest; results: ABTestResult[] }>(`/ab-tests/${id}`);

export const createABTest = (data: {
  name: string;
  description: string;
  traffic_pct: number;
  start_date: string;
  end_date: string;
  control_variant: string;
  test_variant: string;
  rollback_threshold_pct: number;
}) => api.post<ABTest>('/ab-tests', data);

export const rollbackABTest = (id: string) => api.post(`/ab-tests/${id}/rollback`);

export const getAssignments = () => api.get<ABTestAssignment[]>('/ab-tests/assignments');

export const getExperimentRegistry = () =>
  api.get<Record<string, { description: string; variants: string[] }>>('/ab-tests/registry');

// Admin
export const listIPRules = () => api.get<IPRule[]>('/admin/ip-rules');

export const createIPRule = (cidr: string, action: string) =>
  api.post<IPRule>('/admin/ip-rules', { cidr, action });

export const deleteIPRule = (id: string) => api.delete(`/admin/ip-rules/${id}`);

export const listAnomalies = (page = 1, pageSize = 20, acknowledged?: boolean) =>
  api.get<PaginatedResponse<AnomalyEvent>>('/admin/anomalies', {
    params: { page, page_size: pageSize, acknowledged: acknowledged?.toString() },
  });

export const acknowledgeAnomaly = (id: string) => api.patch(`/admin/anomalies/${id}/acknowledge`);

export const getMetrics = () => api.get('/admin/metrics');

// Users
export const listUsers = (page = 1, pageSize = 20) =>
  api.get<PaginatedResponse<User>>('/users', { params: { page, page_size: pageSize } });

export const createUser = (data: { username: string; password: string; display_name: string; email?: string }) =>
  api.post<User>('/users', data);

export const addRole = (userId: string, roleName: string) =>
  api.post(`/users/${userId}/roles`, { role_name: roleName });

export const removeRole = (userId: string, roleId: string) =>
  api.delete(`/users/${userId}/roles/${roleId}`);

export const unlockUser = (userId: string) => api.post(`/users/${userId}/unlock`);

// Dashboard
export const getDashboard = () => api.get<{
  open_orders: number;
  owned_collectibles: number;
  unread_notifications: number;
  seller_open_orders: number;
  listed_items: number;
}>('/dashboard');
