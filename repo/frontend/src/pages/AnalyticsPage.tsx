import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getFunnel, getRetention, getContentPerformance } from '../api/analytics';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line } from 'recharts';

export function AnalyticsPage() {
  const [funnelDays, setFunnelDays] = useState(7);
  const [retentionDays, setRetentionDays] = useState(30);

  const { data: funnel } = useQuery({
    queryKey: ['funnel', funnelDays],
    queryFn: () => getFunnel(funnelDays).then((r) => r.data),
  });

  const { data: retention } = useQuery({
    queryKey: ['retention', retentionDays],
    queryFn: () => getRetention(retentionDays).then((r) => r.data),
  });

  const { data: performance } = useQuery({
    queryKey: ['contentPerformance'],
    queryFn: () => getContentPerformance(10).then((r) => r.data),
  });

  const funnelChartData = funnel ? [
    { name: 'Views', count: funnel.views },
    { name: 'Orders', count: funnel.orders },
  ] : [];

  return (
    <div>
      <h1 className="text-xl md:text-2xl font-bold mb-4 md:mb-6">Analytics</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Funnel */}
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-lg font-semibold">View-to-Order Funnel</h2>
            <select value={funnelDays} onChange={(e) => setFunnelDays(Number(e.target.value))}
              className="text-sm border rounded px-2 py-1">
              <option value={7}>7 days</option>
              <option value={30}>30 days</option>
            </select>
          </div>
          {funnel && (
            <div className="mb-4">
              <p className="text-sm text-gray-500">
                Conversion rate: <span className="font-bold text-primary-700">{(funnel.rate * 100).toFixed(2)}%</span>
              </p>
            </div>
          )}
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={funnelChartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Bar dataKey="count" fill="#3b82f6" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Retention */}
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-lg font-semibold">Retention</h2>
            <select value={retentionDays} onChange={(e) => setRetentionDays(Number(e.target.value))}
              className="text-sm border rounded px-2 py-1">
              <option value={7}>7 days</option>
              <option value={30}>30 days</option>
            </select>
          </div>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={retention || []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="cohort_date" tick={{ fontSize: 10 }} />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="retention_rate" stroke="#3b82f6" />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Content Performance */}
        <div className="bg-white p-6 rounded-lg shadow-sm border lg:col-span-2">
          <h2 className="text-lg font-semibold mb-4">Content Performance</h2>
          <table className="min-w-full divide-y divide-gray-200">
            <thead>
              <tr>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Title</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Views</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Orders</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Conv. Rate</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {performance?.map((item) => (
                <tr key={item.collectible_id}>
                  <td className="px-4 py-2 text-sm">{item.title}</td>
                  <td className="px-4 py-2 text-sm">{item.views}</td>
                  <td className="px-4 py-2 text-sm">{item.orders}</td>
                  <td className="px-4 py-2 text-sm">{(item.conversion_rate * 100).toFixed(2)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
