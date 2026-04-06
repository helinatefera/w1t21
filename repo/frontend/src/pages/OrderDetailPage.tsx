import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getOrder, confirmOrder, processOrder, completeOrder, cancelOrder } from '../api/orders';
import { useAuthStore } from '../store/authStore';
import { useABStore } from '../store/abStore';
import { formatCents, formatDate } from '../utils/formatters';
import { useState } from 'react';

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user, hasRole } = useAuthStore();
  const checkoutVariant = useABStore((s) => s.getVariant('checkout_flow'));
  const isExpress = checkoutVariant === 'express';
  const queryClient = useQueryClient();
  const [cancelReason, setCancelReason] = useState('');
  const [showCancel, setShowCancel] = useState(false);

  const { data: order, isLoading } = useQuery({
    queryKey: ['order', id],
    queryFn: () => getOrder(id!).then((r) => r.data),
    enabled: !!id,
  });

  const transitionMutation = useMutation({
    mutationFn: (action: string) => {
      switch (action) {
        case 'confirm': return confirmOrder(id!);
        case 'process': return processOrder(id!);
        case 'complete': return completeOrder(id!);
        case 'cancel': return cancelOrder(id!, cancelReason);
        default: throw new Error('Unknown action');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      setShowCancel(false);
    },
  });

  if (isLoading) return <p className="text-gray-500">Loading...</p>;
  if (!order) return <p className="text-gray-500">Order not found.</p>;

  const isSeller = order.seller_id === user?.id;
  const isBuyer = order.buyer_id === user?.id;

  return (
    <div className="max-w-3xl mx-auto">
      <Link to="/orders" className="text-sm text-primary-600 hover:underline mb-4 block">Back to Orders</Link>

      <div className="bg-white rounded-lg shadow-sm border p-6">
        <div className="flex justify-between items-start mb-6">
          <div>
            <h1 className="text-xl font-bold">Order #{order.id.slice(0, 8)}</h1>
            <p className="text-sm text-gray-500">Created {formatDate(order.created_at)}</p>
          </div>
          <span className={`px-3 py-1 rounded-full text-sm font-medium ${
            order.status === 'completed' ? 'bg-green-100 text-green-800' :
            order.status === 'cancelled' ? 'bg-red-100 text-red-800' :
            'bg-blue-100 text-blue-800'
          }`}>
            {order.status}
          </span>
        </div>

        <div className="grid grid-cols-2 gap-4 mb-6">
          <div>
            <p className="text-sm text-gray-500">Price</p>
            <p className="text-2xl font-bold text-primary-700">{formatCents(order.price_snapshot_cents)}</p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Collectible</p>
            <Link to={`/catalog/${order.collectible_id}`} className="text-primary-600 hover:underline text-sm">
              View item
            </Link>
          </div>
        </div>

        {order.fulfillment_tracking && (
          <div className="mb-6 p-4 bg-gray-50 rounded">
            <h3 className="text-sm font-medium mb-2">Fulfillment</h3>
            <pre className="text-xs text-gray-600">{JSON.stringify(order.fulfillment_tracking, null, 2)}</pre>
          </div>
        )}

        {order.cancellation_reason && (
          <div className="mb-6 p-4 bg-red-50 rounded">
            <p className="text-sm text-red-700">Cancellation reason: {order.cancellation_reason}</p>
          </div>
        )}

        {/* Actions — checkout_flow experiment: "express" merges seller steps */}
        <div className="flex gap-2 flex-wrap" data-ab-variant={checkoutVariant ?? 'default'}>
          {isExpress && isSeller && ['pending', 'confirmed', 'processing'].includes(order.status) && (
            <button onClick={() => transitionMutation.mutate(
              order.status === 'pending' ? 'confirm' :
              order.status === 'confirmed' ? 'process' : 'complete'
            )} className="px-4 py-2 bg-green-600 text-white rounded text-sm hover:bg-green-700">
              Express Fulfill
            </button>
          )}
          {!isExpress && isSeller && order.status === 'pending' && (
            <button onClick={() => transitionMutation.mutate('confirm')} className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">
              Confirm Order
            </button>
          )}
          {!isExpress && isSeller && order.status === 'confirmed' && (
            <button onClick={() => transitionMutation.mutate('process')} className="px-4 py-2 bg-purple-600 text-white rounded text-sm hover:bg-purple-700">
              Start Processing
            </button>
          )}
          {!isExpress && isSeller && order.status === 'processing' && (
            <button onClick={() => transitionMutation.mutate('complete')} className="px-4 py-2 bg-green-600 text-white rounded text-sm hover:bg-green-700">
              Mark Completed
            </button>
          )}
          {(isBuyer || isSeller) && ['pending', 'confirmed'].includes(order.status) && (
            <button onClick={() => setShowCancel(true)} className="px-4 py-2 bg-red-100 text-red-700 rounded text-sm hover:bg-red-200">
              Cancel Order
            </button>
          )}
          <Link to={`/orders/${id}/messages`} className="px-4 py-2 bg-gray-100 text-gray-700 rounded text-sm hover:bg-gray-200">
            Messages
          </Link>
        </div>

        {showCancel && (
          <div className="mt-4 p-4 bg-red-50 rounded">
            <textarea
              value={cancelReason}
              onChange={(e) => setCancelReason(e.target.value)}
              placeholder="Reason for cancellation..."
              className="w-full p-2 border rounded mb-2"
              rows={2}
            />
            <div className="flex gap-2">
              <button onClick={() => transitionMutation.mutate('cancel')} disabled={!cancelReason} className="px-3 py-1 bg-red-600 text-white rounded text-sm disabled:opacity-50">
                Confirm Cancel
              </button>
              <button onClick={() => setShowCancel(false)} className="px-3 py-1 bg-gray-200 rounded text-sm">
                Never mind
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
