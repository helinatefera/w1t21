import api from './client';
import type { Collectible, CollectibleTxHistory, PaginatedResponse } from '../types';

export const listCollectibles = (page = 1, pageSize = 20, status = 'published') =>
  api.get<PaginatedResponse<Collectible>>('/collectibles', { params: { page, page_size: pageSize, status } });

export const listMyCollectibles = (page = 1, pageSize = 20) =>
  api.get<PaginatedResponse<Collectible>>('/collectibles/mine', { params: { page, page_size: pageSize } });

export const getCollectible = (id: string) =>
  api.get<{ collectible: Collectible; transaction_history: CollectibleTxHistory[] }>(`/collectibles/${id}`);

export const createCollectible = (data: {
  title: string;
  description: string;
  price_cents: number;
  currency?: string;
  contract_address?: string;
  chain_id?: number;
  token_id?: string;
  metadata_uri?: string;
  image_url?: string;
}) => api.post<Collectible>('/collectibles', data);

export const updateCollectible = (id: string, data: Partial<Collectible>) =>
  api.patch<Collectible>(`/collectibles/${id}`, data);

export const hideCollectible = (id: string, reason: string) =>
  api.patch(`/collectibles/${id}/hide`, { reason });

export const publishCollectible = (id: string) =>
  api.patch(`/collectibles/${id}/publish`);
