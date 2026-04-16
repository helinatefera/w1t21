import { describe, it, expect, beforeEach } from 'vitest';
import { useNotificationStore } from './notificationStore';

describe('notificationStore', () => {
  beforeEach(() => {
    useNotificationStore.setState({ unreadCount: 0 });
  });

  it('starts with zero unread count', () => {
    expect(useNotificationStore.getState().unreadCount).toBe(0);
  });

  it('setUnreadCount updates the count', () => {
    useNotificationStore.getState().setUnreadCount(5);
    expect(useNotificationStore.getState().unreadCount).toBe(5);
  });

  it('setUnreadCount can set to zero', () => {
    useNotificationStore.getState().setUnreadCount(10);
    useNotificationStore.getState().setUnreadCount(0);
    expect(useNotificationStore.getState().unreadCount).toBe(0);
  });
});
