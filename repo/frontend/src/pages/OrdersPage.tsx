import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { listOrders } from '../api/orders';
import { useAuthStore } from '../store/authStore';
import { formatCents, formatDate } from '../utils/formatters';
import type { OrderStatus } from '../types';

const statusColors: Record<OrderStatus, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  confirmed: 'bg-blue-100 text-blue-800',
  processing: 'bg-purple-100 text-purple-800',
  completed: 'bg-green-100 text-green-800',
  cancelled: 'bg-red-100 text-red-800',
};

export function OrdersPage() {
  const [page, setPage] = useState(1);
  const { hasRole } = useAuthStore();
  const [viewAs, setViewAs] = useState<'buyer' | 'seller'>(hasRole('seller') ? 'seller' : 'buyer');

  const { data, isLoading } = useQuery({
    queryKey: ['orders', page, viewAs],
    queryFn: () => listOrders(page, 20, viewAs).then((r) => r.data),
  });

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3 mb-4 md:mb-6">
        <h1 className="text-xl md:text-2xl font-bold">Orders</h1>
        {hasRole('seller') && hasRole('buyer') && (
          <div className="flex gap-2">
            <button
              onClick={() => setViewAs('buyer')}
              className={`px-3 py-1.5 text-sm rounded transition-colors ${viewAs === 'buyer' ? 'bg-primary-600 text-white' : 'bg-gray-100'}`}
            >
              As Buyer
            </button>
            <button
              onClick={() => setViewAs('seller')}
              className={`px-3 py-1.5 text-sm rounded transition-colors ${viewAs === 'seller' ? 'bg-primary-600 text-white' : 'bg-gray-100'}`}
            >
              As Seller
            </button>
          </div>
        )}
      </div>

      {isLoading ? (
        <p className="text-gray-500">Loading...</p>
      ) : (
        <div className="space-y-3">
          {data?.data?.map((order) => (
            <Link
              key={order.id}
              to={`/orders/${order.id}`}
              className="block bg-white p-3 md:p-4 rounded-lg shadow-sm border hover:shadow-md transition-shadow"
            >
              <div className="flex justify-between items-center gap-4">
                <div className="min-w-0">
                  <p className="font-medium text-gray-900 truncate">Order #{order.id.slice(0, 8)}</p>
                  <p className="text-sm text-gray-500">{formatDate(order.created_at)}</p>
                </div>
                <div className="text-right flex-shrink-0">
                  <span className={`px-2 py-1 text-xs rounded-full font-medium ${statusColors[order.status]}`}>
                    {order.status}
                  </span>
                  <p className="text-base md:text-lg font-bold text-primary-700 mt-1">
                    {formatCents(order.price_snapshot_cents)}
                  </p>
                </div>
              </div>
            </Link>
          ))}
          {(!data?.data || data.data.length === 0) && (
            <p className="text-gray-500">No orders found.</p>
          )}
        </div>
      )}

      {data && data.total_pages > 1 && (
        <div className="flex justify-center gap-2 mt-6">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 md:px-4 py-2 bg-white border rounded text-sm disabled:opacity-50">Previous</button>
          <span className="px-3 md:px-4 py-2 text-sm text-gray-600">Page {page} of {data.total_pages}</span>
          <button onClick={() => setPage((p) => p + 1)} disabled={page >= data.total_pages} className="px-3 md:px-4 py-2 bg-white border rounded text-sm disabled:opacity-50">Next</button>
        </div>
      )}
    </div>
  );
}
