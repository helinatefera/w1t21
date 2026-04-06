import { Outlet } from 'react-router-dom';
import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { getUnreadCount } from '../../api/notifications';
import { getAssignments } from '../../api/analytics';
import { useNotificationStore } from '../../store/notificationStore';
import { useABStore } from '../../store/abStore';

export function AppShell() {
  const setUnreadCount = useNotificationStore((s) => s.setUnreadCount);
  const setAssignments = useABStore((s) => s.setAssignments);

  const { data } = useQuery({
    queryKey: ['unreadCount'],
    queryFn: () => getUnreadCount().then((r) => r.data),
    refetchInterval: 30000,
  });

  const { data: abData } = useQuery({
    queryKey: ['abAssignments'],
    queryFn: () => getAssignments().then((r) => r.data),
    staleTime: 60000,
    refetchInterval: 60000,
  });

  useEffect(() => {
    if (data?.unread_count !== undefined) {
      setUnreadCount(data.unread_count);
    }
  }, [data, setUnreadCount]);

  useEffect(() => {
    if (abData) {
      setAssignments(abData);
    }
  }, [abData, setAssignments]);

  return (
    <div className="flex min-h-screen bg-gray-50">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0">
        <Header />
        <main className="flex-1 p-4 md:p-6 lg:p-8 overflow-x-hidden">
          <div className="max-w-7xl mx-auto">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
