import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '../store/authStore';
import { useNotificationStore } from '../store/notificationStore';
import { getDashboard } from '../api/analytics';
import { listMyCollectibles, listCollectibles } from '../api/collectibles';
import { listOrders } from '../api/orders';
import { formatCents, formatDate } from '../utils/formatters';
import { Link } from 'react-router-dom';
import type { OrderStatus } from '../types';

const statusColors: Record<OrderStatus, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  confirmed: 'bg-blue-100 text-blue-800',
  processing: 'bg-purple-100 text-purple-800',
  completed: 'bg-green-100 text-green-800',
  cancelled: 'bg-red-100 text-red-800',
};

export function DashboardPage() {
  const { user, roles, hasRole } = useAuthStore();
  const { unreadCount } = useNotificationStore();

  const { data: dashboard } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => getDashboard().then((r) => r.data),
  });

  const { data: recentItems } = useQuery({
    queryKey: ['collectibles', 1],
    queryFn: () => listCollectibles(1, 6, 'published').then((r) => r.data),
  });

  const { data: myItems } = useQuery({
    queryKey: ['myCollectibles'],
    queryFn: () => listMyCollectibles(1, 6).then((r) => r.data),
    enabled: hasRole('seller'),
  });

  const { data: openOrders } = useQuery({
    queryKey: ['orders', 1, 'buyer'],
    queryFn: () => listOrders(1, 5, 'buyer').then((r) => r.data),
  });

  return (
    <div>
      <h1 className="text-xl md:text-2xl lg:text-3xl font-bold mb-4 md:mb-6">
        Welcome back, {user?.display_name}
      </h1>

      {/* Stats cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-6 mb-6 md:mb-8">
        <div className="bg-white p-4 md:p-6 rounded-lg shadow-sm border">
          <h3 className="text-sm font-medium text-gray-500">Open Orders</h3>
          <p className="text-2xl md:text-3xl font-bold text-primary-700 mt-1">{dashboard?.open_orders ?? 0}</p>
          <Link to="/orders" className="text-sm text-primary-600 hover:underline mt-2 block">
            View orders
          </Link>
        </div>
        <div className="bg-white p-4 md:p-6 rounded-lg shadow-sm border">
          <h3 className="text-sm font-medium text-gray-500">Unread Notifications</h3>
          <p className="text-2xl md:text-3xl font-bold text-primary-700 mt-1">{unreadCount}</p>
          <Link to="/notifications" className="text-sm text-primary-600 hover:underline mt-2 block">
            View notifications
          </Link>
        </div>
        <div className="bg-white p-4 md:p-6 rounded-lg shadow-sm border sm:col-span-2 lg:col-span-1">
          <h3 className="text-sm font-medium text-gray-500">Owned Items</h3>
          <p className="text-2xl md:text-3xl font-bold text-primary-700 mt-1">{dashboard?.owned_collectibles ?? 0}</p>
          <Link to="/orders" className="text-sm text-primary-600 hover:underline mt-2 block">
            View purchases
          </Link>
        </div>
        {hasRole('seller') && (
          <>
            <div className="bg-white p-4 md:p-6 rounded-lg shadow-sm border">
              <h3 className="text-sm font-medium text-gray-500">Seller Open Orders</h3>
              <p className="text-2xl md:text-3xl font-bold text-primary-700 mt-1">{dashboard?.seller_open_orders ?? 0}</p>
              <Link to="/orders" className="text-sm text-primary-600 hover:underline mt-2 block">
                Manage sales
              </Link>
            </div>
            <div className="bg-white p-4 md:p-6 rounded-lg shadow-sm border">
              <h3 className="text-sm font-medium text-gray-500">Listed Items</h3>
              <p className="text-2xl md:text-3xl font-bold text-primary-700 mt-1">{dashboard?.listed_items ?? 0}</p>
              <Link to="/sell/new" className="text-sm text-primary-600 hover:underline mt-2 block">
                List new item
              </Link>
            </div>
          </>
        )}
      </div>

      {/* Open Orders section */}
      {openOrders?.data && openOrders.data.length > 0 && (
        <div className="mb-6 md:mb-8">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-base md:text-lg font-semibold">Open Orders</h2>
            <Link to="/orders" className="text-sm text-primary-600 hover:underline">View all</Link>
          </div>
          <div className="space-y-2">
            {openOrders.data.filter((o) => o.status !== 'completed' && o.status !== 'cancelled').slice(0, 5).map((order) => (
              <Link
                key={order.id}
                to={`/orders/${order.id}`}
                className="flex justify-between items-center bg-white p-3 rounded-lg shadow-sm border hover:shadow-md transition-shadow"
              >
                <div className="min-w-0">
                  <p className="font-medium text-gray-900 text-sm">Order #{order.id.slice(0, 8)}</p>
                  <p className="text-xs text-gray-500">{formatDate(order.created_at)}</p>
                </div>
                <div className="text-right flex-shrink-0 flex items-center gap-3">
                  <span className={`px-2 py-0.5 text-xs rounded-full font-medium ${statusColors[order.status]}`}>
                    {order.status}
                  </span>
                  <span className="text-sm font-bold text-primary-700">
                    {formatCents(order.price_snapshot_cents)}
                  </span>
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* My Listings (seller) */}
      {hasRole('seller') && (
        <div className="mb-6 md:mb-8">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-base md:text-lg font-semibold">My Listings</h2>
            <div className="flex items-center gap-3">
              <Link to="/sell/new" className="text-sm px-3 py-1 bg-primary-600 text-white rounded hover:bg-primary-700 transition-colors">
                + Add Collectible
              </Link>
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {myItems?.data?.map((item) => (
              <Link
                key={item.id}
                to={`/catalog/${item.id}`}
                className="bg-white p-4 rounded-lg shadow-sm border hover:shadow-md transition-shadow"
              >
                {item.image_url && (
                  <img src={item.image_url} alt={item.title} className="w-full h-36 md:h-40 object-cover rounded mb-3" />
                )}
                <h3 className="font-medium text-gray-900">{item.title}</h3>
                <div className="flex justify-between items-center mt-1">
                  <p className="text-lg font-bold text-primary-700">
                    {formatCents(item.price_cents, item.currency)}
                  </p>
                  <span className={`text-xs px-2 py-0.5 rounded-full ${item.status === 'published' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'}`}>
                    {item.status}
                  </span>
                </div>
              </Link>
            ))}
            {(!myItems?.data || myItems.data.length === 0) && (
              <div className="col-span-full text-center py-8 bg-white rounded-lg border">
                <p className="text-gray-500 mb-3">You haven't listed any collectibles yet.</p>
                <Link to="/sell/new" className="text-sm px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 transition-colors">
                  List Your First Item
                </Link>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Recent collectibles */}
      <div>
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-base md:text-lg font-semibold">Recent Collectibles</h2>
          <Link to="/catalog" className="text-sm text-primary-600 hover:underline">View all</Link>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {recentItems?.data?.map((item) => (
            <Link
              key={item.id}
              to={`/catalog/${item.id}`}
              className="bg-white p-4 rounded-lg shadow-sm border hover:shadow-md transition-shadow"
            >
              {item.image_url && (
                <img src={item.image_url} alt={item.title} className="w-full h-36 md:h-40 object-cover rounded mb-3" />
              )}
              <h3 className="font-medium text-gray-900">{item.title}</h3>
              <p className="text-lg font-bold text-primary-700 mt-1">
                {formatCents(item.price_cents, item.currency)}
              </p>
            </Link>
          ))}
          {(!recentItems?.data || recentItems.data.length === 0) && (
            <p className="text-gray-500 col-span-full">No collectibles available yet.</p>
          )}
        </div>
      </div>
    </div>
  );
}
