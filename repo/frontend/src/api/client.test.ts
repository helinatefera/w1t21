import { describe, it, expect } from 'vitest';
import api from './client';

describe('API client', () => {
  it('exports an axios instance', () => {
    expect(api).toBeDefined();
    expect(typeof api.get).toBe('function');
    expect(typeof api.post).toBe('function');
    expect(typeof api.patch).toBe('function');
    expect(typeof api.delete).toBe('function');
  });

  it('has baseURL set to /api', () => {
    expect(api.defaults.baseURL).toBe('/api');
  });

  it('sends credentials with requests', () => {
    expect(api.defaults.withCredentials).toBe(true);
  });

  it('has request interceptors configured', () => {
    // Axios interceptor manager has handlers array
    const reqInterceptors = (api.interceptors.request as any).handlers;
    expect(reqInterceptors.length).toBeGreaterThan(0);
  });

  it('has response interceptors configured', () => {
    const resInterceptors = (api.interceptors.response as any).handlers;
    expect(resInterceptors.length).toBeGreaterThan(0);
  });
});
