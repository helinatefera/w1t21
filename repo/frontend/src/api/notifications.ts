import api from './client';
import type { Notification, NotificationPreferences, PaginatedResponse } from '../types';

export const listNotifications = (page = 1, pageSize = 20, unread = false) =>
  api.get<PaginatedResponse<Notification>>('/notifications', {
    params: { page, page_size: pageSize, unread: unread ? 'true' : undefined },
  });

export const getUnreadCount = () =>
  api.get<{ unread_count: number }>('/notifications', { params: { count: 'true', unread: 'true' } });

export const markRead = (id: string) => api.patch(`/notifications/${id}/read`);

export const markAllRead = () => api.post('/notifications/read-all');

export const retryNotification = (id: string) => api.post(`/notifications/${id}/retry`);

export const getPreferences = () => api.get<NotificationPreferences>('/notifications/preferences');

export const updatePreferences = (preferences: Record<string, boolean>, subscriptionMode?: string) =>
  api.put('/notifications/preferences', { preferences, subscription_mode: subscriptionMode });
