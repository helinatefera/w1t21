import api from './client';
import type { Order, PaginatedResponse } from '../types';

export const createOrder = (collectibleId: string, idempotencyKey: string) =>
  api.post<Order>('/orders', { collectible_id: collectibleId }, {
    headers: { 'Idempotency-Key': idempotencyKey },
  });

export const listOrders = (page = 1, pageSize = 20, role = 'buyer') =>
  api.get<PaginatedResponse<Order>>('/orders', { params: { page, page_size: pageSize, role } });

export const getOrder = (id: string) => api.get<Order>(`/orders/${id}`);

export const confirmOrder = (id: string) => api.post<Order>(`/orders/${id}/confirm`);

export const processOrder = (id: string) => api.post<Order>(`/orders/${id}/process`);

export const completeOrder = (id: string) => api.post<Order>(`/orders/${id}/complete`);

export const cancelOrder = (id: string, reason: string) =>
  api.post<Order>(`/orders/${id}/cancel`, { reason });

export const updateFulfillment = (id: string, data: { carrier?: string; tracking_number?: string }) =>
  api.patch(`/orders/${id}/fulfillment`, data);
