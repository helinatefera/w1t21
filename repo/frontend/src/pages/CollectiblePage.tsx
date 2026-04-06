import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getCollectible } from '../api/collectibles';
import { createOrder } from '../api/orders';
import { useAuthStore } from '../store/authStore';
import { useABStore } from '../store/abStore';
import { formatCents, formatDate } from '../utils/formatters';
import { useState } from 'react';

export function CollectiblePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { hasRole, user } = useAuthStore();
  const queryClient = useQueryClient();
  const [orderError, setOrderError] = useState('');
  const catalogVariant = useABStore((s) => s.getVariant('catalog_layout'));
  const isListVariant = catalogVariant === 'list';

  const { data, isLoading } = useQuery({
    queryKey: ['collectible', id],
    queryFn: () => getCollectible(id!).then((r) => r.data),
    enabled: !!id,
  });

  const orderMutation = useMutation({
    mutationFn: () => {
      const idempotencyKey = crypto.randomUUID();
      return createOrder(id!, idempotencyKey);
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      navigate(`/orders/${res.data.id}`);
    },
    onError: (err: any) => {
      setOrderError(err.response?.data?.error?.message || 'Failed to place order');
    },
  });

  if (isLoading) return <p className="text-gray-500">Loading...</p>;
  if (!data) return <p className="text-gray-500">Collectible not found.</p>;

  const { collectible, transaction_history } = data;
  const isOwner = collectible.seller_id === user?.id;

  return (
    <div className="max-w-5xl mx-auto">
      <button onClick={() => navigate(-1)} className="text-sm text-primary-600 hover:underline mb-4 block">
        Back
      </button>

      <div className="bg-white rounded-lg shadow-sm border overflow-hidden" data-ab-variant={catalogVariant ?? 'default'}>
        <div className={isListVariant ? '' : 'md:flex'}>
          <div className={isListVariant ? '' : 'md:w-1/2 lg:w-2/5'}>
            {collectible.image_url ? (
              <img
                src={collectible.image_url}
                alt={collectible.title}
                className={isListVariant
                  ? 'w-full h-64 sm:h-80 object-cover'
                  : 'w-full h-64 md:h-80 lg:h-96 object-cover'}
              />
            ) : (
              <div className={`bg-gray-100 flex items-center justify-center text-gray-400 ${
                isListVariant ? 'w-full h-64 sm:h-80' : 'w-full h-64 md:h-80 lg:h-96'
              }`}>
                No image
              </div>
            )}
          </div>
          <div className={isListVariant
            ? 'p-4 md:p-6 lg:p-8'
            : 'p-4 md:p-6 lg:p-8 md:w-1/2 lg:w-3/5'}>
            <h1 className="text-xl md:text-2xl lg:text-3xl font-bold text-gray-900 mb-2">{collectible.title}</h1>
            <p className="text-2xl md:text-3xl font-bold text-primary-700 mb-4">
              {formatCents(collectible.price_cents, collectible.currency)}
            </p>
            <p className="text-gray-600 mb-6">{collectible.description}</p>

            {(collectible.contract_address || collectible.chain_id || collectible.token_id) && (
              <div className="space-y-1 text-sm text-gray-500 mb-4">
                {collectible.contract_address && (
                  <p>Contract: <span className="font-mono">{collectible.contract_address}</span></p>
                )}
                {collectible.chain_id && <p>Chain ID: {collectible.chain_id}</p>}
                {collectible.token_id && <p>Token ID: <span className="font-mono">{collectible.token_id}</span></p>}
              </div>
            )}

            <div className="text-xs text-gray-400 mb-4">
              <p>{collectible.view_count} views</p>
              <p>Listed {formatDate(collectible.created_at)}</p>
            </div>

            {hasRole('buyer') && !isOwner && (
              <div>
                <button
                  onClick={() => orderMutation.mutate()}
                  disabled={orderMutation.isPending}
                  className="w-full bg-primary-600 text-white py-3 rounded font-medium hover:bg-primary-700 disabled:opacity-50"
                >
                  {orderMutation.isPending ? 'Placing Order...' : 'Place Order'}
                </button>
                {orderError && (
                  <p className="text-sm text-red-600 mt-2">{orderError}</p>
                )}
              </div>
            )}

            {isOwner && (
              <button
                onClick={() => navigate(`/sell/${id}/edit`)}
                className="w-full bg-gray-100 text-gray-700 py-3 rounded font-medium hover:bg-gray-200"
              >
                Edit Listing
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Immutable Transaction History — always visible to sellers, visible to others when data exists */}
      {(isOwner || (transaction_history && transaction_history.length > 0)) && (
        <div className="mt-6 bg-white rounded-lg shadow-sm border p-4 md:p-6">
          <div className="flex items-center gap-2 mb-4">
            <h2 className="text-lg font-semibold">Transaction History</h2>
            <span className="text-xs bg-gray-100 text-gray-500 px-2 py-0.5 rounded-full">Immutable</span>
          </div>
          {transaction_history && transaction_history.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="text-left text-xs text-gray-500 uppercase border-b">
                    <th className="pb-2 pr-4">Tx Hash</th>
                    <th className="pb-2 pr-4">From</th>
                    <th className="pb-2 pr-4">To</th>
                    <th className="pb-2 pr-4 hidden md:table-cell">Block</th>
                    <th className="pb-2">Date</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {transaction_history.map((tx) => (
                    <tr key={tx.id} className="text-gray-700">
                      <td className="py-2 pr-4 font-mono text-xs">{tx.tx_hash.slice(0, 10)}...</td>
                      <td className="py-2 pr-4 font-mono text-xs">{tx.from_address.slice(0, 10)}...</td>
                      <td className="py-2 pr-4 font-mono text-xs">{tx.to_address.slice(0, 10)}...</td>
                      <td className="py-2 pr-4 text-xs hidden md:table-cell">{tx.block_number}</td>
                      <td className="py-2 text-xs text-gray-400">{formatDate(tx.timestamp)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-gray-400">No past sales recorded for this collectible yet.</p>
          )}
        </div>
      )}
    </div>
  );
}
