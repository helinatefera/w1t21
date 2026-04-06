import api from './client';
import type { AuthResponse } from '../types';

export const login = (username: string, password: string) =>
  api.post<AuthResponse>('/auth/login', { username, password });

export const refresh = () => api.post('/auth/refresh');

export const me = () => api.get<AuthResponse>('/auth/me');

export const logout = () => api.post('/auth/logout');
