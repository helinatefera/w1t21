import api from './client';
import type { Message, PaginatedResponse } from '../types';

export const listMessages = (orderId: string, page = 1, pageSize = 50) =>
  api.get<PaginatedResponse<Message>>(`/orders/${orderId}/messages`, {
    params: { page, page_size: pageSize },
  });

export const sendMessage = (orderId: string, body: string, attachment?: File) => {
  const formData = new FormData();
  formData.append('body', body);
  if (attachment) {
    formData.append('attachment', attachment);
  }
  return api.post<Message>(`/orders/${orderId}/messages`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};
