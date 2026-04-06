import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listAnomalies, acknowledgeAnomaly } from '../api/analytics';
import { formatDate } from '../utils/formatters';

export function AnomalyAlertsPage() {
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();

  const { data } = useQuery({
    queryKey: ['anomalies', page],
    queryFn: () => listAnomalies(page, 20, false).then((r) => r.data),
  });

  const ackMutation = useMutation({
    mutationFn: acknowledgeAnomaly,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['anomalies'] }),
  });

  const typeLabels: Record<string, string> = {
    excessive_cancellations: 'Excessive Cancellations',
    repeated_checkout_failures: 'Repeated Checkout Failures',
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Anomaly Alerts</h1>

      <div className="space-y-3">
        {data?.data?.map((event) => (
          <div key={event.id} className={`bg-white p-4 rounded-lg shadow-sm border ${!event.acknowledged ? 'border-l-4 border-l-red-500' : ''}`}>
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-medium text-red-700">
                  {typeLabels[event.anomaly_type] || event.anomaly_type}
                </h3>
                <p className="text-sm text-gray-500 mt-1">User: {event.user_id.slice(0, 8)}...</p>
                <p className="text-xs text-gray-400 mt-1">{formatDate(event.created_at)}</p>
                <div className="text-xs text-gray-600 mt-2 bg-gray-50 p-2 rounded">
                  {Object.entries(event.details).map(([k, v]) => (
                    <p key={k}>{k}: {String(v)}</p>
                  ))}
                </div>
              </div>
              {!event.acknowledged && (
                <button
                  onClick={() => ackMutation.mutate(event.id)}
                  className="px-3 py-1 text-xs bg-yellow-100 text-yellow-800 rounded hover:bg-yellow-200"
                >
                  Acknowledge
                </button>
              )}
            </div>
          </div>
        ))}
        {(!data?.data || data.data.length === 0) && (
          <p className="text-gray-500">No anomaly alerts.</p>
        )}
      </div>
    </div>
  );
}
