import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listCollectibles, hideCollectible, publishCollectible } from '../api/collectibles';
import { formatCents, formatDate } from '../utils/formatters';

export function AdminModerationPage() {
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState('');
  const [hideReason, setHideReason] = useState('');
  const [hidingId, setHidingId] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data } = useQuery({
    queryKey: ['moderation', page, status],
    queryFn: () => listCollectibles(page, 20, status).then((r) => r.data),
  });

  const hideMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => hideCollectible(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['moderation'] });
      setHidingId(null);
      setHideReason('');
    },
  });

  const publishMutation = useMutation({
    mutationFn: publishCollectible,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['moderation'] }),
  });

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Moderation Queue</h1>
        <div className="flex gap-2">
          {['', 'published', 'hidden'].map((s) => (
            <button key={s} onClick={() => { setStatus(s); setPage(1); }}
              className={`px-3 py-1 text-sm rounded ${status === s ? 'bg-primary-600 text-white' : 'bg-gray-100'}`}>
              {s || 'All'}
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        {data?.data?.map((item) => (
          <div key={item.id} className="bg-white p-4 rounded-lg shadow-sm border">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-medium">{item.title}</h3>
                <p className="text-sm text-gray-500">{formatCents(item.price_cents)} | {item.view_count} views</p>
                <p className="text-xs text-gray-400 mt-1">Listed {formatDate(item.created_at)}</p>
                {item.hidden_reason && (
                  <p className="text-xs text-red-500 mt-1">Hidden: {item.hidden_reason}</p>
                )}
              </div>
              <div className="flex gap-2 items-center">
                <span className={`px-2 py-1 text-xs rounded-full ${
                  item.status === 'published' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                }`}>{item.status}</span>
                {item.status === 'published' && (
                  <button onClick={() => setHidingId(item.id)} className="text-xs text-red-600 hover:underline">
                    Hide
                  </button>
                )}
                {item.status === 'hidden' && (
                  <button onClick={() => publishMutation.mutate(item.id)} className="text-xs text-green-600 hover:underline">
                    Publish
                  </button>
                )}
              </div>
            </div>

            {hidingId === item.id && (
              <div className="mt-3 flex gap-2">
                <input type="text" placeholder="Reason for hiding..." value={hideReason}
                  onChange={(e) => setHideReason(e.target.value)}
                  className="flex-1 px-3 py-1 border rounded text-sm" />
                <button onClick={() => hideMutation.mutate({ id: item.id, reason: hideReason })}
                  disabled={!hideReason} className="px-3 py-1 bg-red-600 text-white rounded text-sm disabled:opacity-50">
                  Confirm
                </button>
                <button onClick={() => { setHidingId(null); setHideReason(''); }}
                  className="px-3 py-1 bg-gray-100 rounded text-sm">Cancel</button>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
