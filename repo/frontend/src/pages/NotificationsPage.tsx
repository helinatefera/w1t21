import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listNotifications, markRead, markAllRead, retryNotification } from '../api/notifications';
import { formatRelativeTime } from '../utils/formatters';

const statusConfig: Record<string, { label: string; className: string }> = {
  pending:            { label: 'Pending',    className: 'bg-blue-100 text-blue-700' },
  delivered:          { label: 'Delivered',  className: 'bg-green-100 text-green-700' },
  failed:             { label: 'Failed',     className: 'bg-yellow-100 text-yellow-800' },
  permanently_failed: { label: 'Permanently Failed', className: 'bg-red-100 text-red-700' },
};

export function NotificationsPage() {
  const [page, setPage] = useState(1);
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [retryError, setRetryError] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['notifications', page, unreadOnly],
    queryFn: () => listNotifications(page, 20, unreadOnly).then((r) => r.data),
  });

  const markReadMutation = useMutation({
    mutationFn: markRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['unreadCount'] });
    },
  });

  const markAllMutation = useMutation({
    mutationFn: markAllRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['unreadCount'] });
    },
  });

  const retryMutation = useMutation({
    mutationFn: retryNotification,
    onSuccess: () => {
      setRetryError(null);
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
    onError: (err: any) => {
      const message = err.response?.data?.error?.message || 'Retry failed. Please try again later.';
      setRetryError(message);
    },
  });

  const canRetry = (status: string) => status === 'failed' || status === 'permanently_failed';

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Notifications</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setUnreadOnly(!unreadOnly)}
            className={`px-3 py-1 text-sm rounded ${unreadOnly ? 'bg-primary-600 text-white' : 'bg-gray-100'}`}
          >
            {unreadOnly ? 'Showing Unread' : 'Show All'}
          </button>
          <button
            onClick={() => markAllMutation.mutate()}
            className="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200"
          >
            Mark All Read
          </button>
        </div>
      </div>

      {retryError && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">
          <span className="font-medium">Retry error:</span> {retryError}
          <button onClick={() => setRetryError(null)} className="ml-2 text-red-500 hover:underline text-xs">Dismiss</button>
        </div>
      )}

      {isLoading ? (
        <p className="text-gray-500">Loading...</p>
      ) : (
        <div className="space-y-2">
          {data?.data?.map((notif) => {
            const config = statusConfig[notif.status];
            const showStatus = notif.status !== 'delivered';
            return (
              <div
                key={notif.id}
                className={`bg-white p-4 rounded-lg shadow-sm border ${!notif.is_read ? 'border-l-4 border-l-primary-500' : ''}`}
              >
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className={`text-sm font-medium ${!notif.is_read ? 'text-gray-900' : 'text-gray-600'}`}>
                      {notif.rendered_title}
                    </h3>
                    <p className="text-sm text-gray-500 mt-1">{notif.rendered_body}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <p className="text-xs text-gray-400">{formatRelativeTime(notif.created_at)}</p>
                      {showStatus && config && (
                        <span className={`text-xs px-1.5 py-0.5 rounded-full ${config.className}`}>
                          {config.label}
                        </span>
                      )}
                      {notif.status === 'failed' && notif.retry_count > 0 && (
                        <span className="text-xs text-gray-400">
                          (retry {notif.retry_count}/{notif.max_retries})
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-2 items-center flex-shrink-0">
                    {canRetry(notif.status) && (
                      <button
                        onClick={() => retryMutation.mutate(notif.id)}
                        disabled={retryMutation.isPending}
                        className="text-xs px-2 py-1 bg-yellow-100 text-yellow-800 rounded hover:bg-yellow-200 disabled:opacity-50"
                      >
                        {retryMutation.isPending ? 'Retrying...' : 'Retry'}
                      </button>
                    )}
                    {!notif.is_read && (
                      <button
                        onClick={() => markReadMutation.mutate(notif.id)}
                        className="text-xs text-primary-600 hover:underline"
                      >
                        Mark read
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
          {(!data?.data || data.data.length === 0) && (
            <p className="text-gray-500">No notifications.</p>
          )}

          {data && data.total_pages > 1 && (
            <div className="flex justify-center gap-2 pt-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="text-sm text-gray-500 py-1">
                Page {page} of {data.total_pages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(data.total_pages, p + 1))}
                disabled={page >= data.total_pages}
                className="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
