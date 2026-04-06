import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getPreferences, updatePreferences } from '../api/notifications';
import { useState, useEffect } from 'react';

const TEMPLATE_SLUGS = [
  { slug: 'order_confirmed', label: 'Order Confirmed', status: true },
  { slug: 'order_processing', label: 'Order Processing', status: true },
  { slug: 'order_completed', label: 'Order Completed', status: true },
  { slug: 'order_cancelled', label: 'Order Cancelled', status: true },
  { slug: 'refund_approved', label: 'Refund Approved', status: true },
  { slug: 'arbitration_opened', label: 'Arbitration Opened', status: false },
  { slug: 'review_posted', label: 'Review Posted', status: false },
];

// Presets map subscription modes to per-slug enabled states.
// "status" marks whether a slug is a status-change event.
const MODE_PRESETS: Record<string, Record<string, boolean>> = {
  all_events: Object.fromEntries(TEMPLATE_SLUGS.map(({ slug }) => [slug, true])),
  status_only: Object.fromEntries(TEMPLATE_SLUGS.map(({ slug, status }) => [slug, status])),
};

const SUBSCRIPTION_MODES = [
  { value: 'all_events', label: 'All Events', description: 'Receive notifications for every event type.' },
  { value: 'status_only', label: 'Status Changes Only', description: 'Only receive order status changes and refund outcomes.' },
] as const;

export function NotificationPrefsPage() {
  const queryClient = useQueryClient();
  const [prefs, setPrefs] = useState<Record<string, boolean>>({});
  const [subscriptionMode, setSubscriptionMode] = useState<string>('all_events');

  const { data } = useQuery({
    queryKey: ['notificationPrefs'],
    queryFn: () => getPreferences().then((r) => r.data),
  });

  useEffect(() => {
    if (data?.preferences) {
      setPrefs(data.preferences);
    }
    if (data?.subscription_mode) {
      setSubscriptionMode(data.subscription_mode);
    }
  }, [data]);

  const mutation = useMutation({
    mutationFn: (vars: { prefs: Record<string, boolean>; mode: string }) =>
      updatePreferences(vars.prefs, vars.mode),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notificationPrefs'] }),
  });

  const toggle = (slug: string) => {
    const updated = { ...prefs, [slug]: !prefs[slug] };
    setPrefs(updated);
    mutation.mutate({ prefs: updated, mode: subscriptionMode });
  };

  const changeMode = (mode: string) => {
    setSubscriptionMode(mode);
    // Apply preset: auto-toggle individual slugs to match the selected mode.
    const preset = MODE_PRESETS[mode] ?? MODE_PRESETS.all_events;
    const updated = { ...prefs, ...preset };
    setPrefs(updated);
    mutation.mutate({ prefs: updated, mode });
  };

  // In status_only mode, auxiliary slugs are server-filtered regardless of
  // individual toggles. Dim them to make this clear.
  const isDisabledByMode = (slug: string) =>
    subscriptionMode === 'status_only' && !TEMPLATE_SLUGS.find((t) => t.slug === slug)?.status;

  return (
    <div className="max-w-lg mx-auto">
      <h1 className="text-2xl font-bold mb-6">Notification Preferences</h1>

      <div className="bg-white rounded-lg shadow-sm border p-4 mb-6">
        <h2 className="text-sm font-semibold text-gray-700 mb-3">Subscription Mode</h2>
        <div className="space-y-2">
          {SUBSCRIPTION_MODES.map(({ value, label, description }) => (
            <label key={value} className="flex items-start gap-3 cursor-pointer p-2 rounded hover:bg-gray-50">
              <input
                type="radio"
                name="subscription_mode"
                value={value}
                checked={subscriptionMode === value}
                onChange={() => changeMode(value)}
                className="mt-0.5 h-4 w-4 text-primary-600 border-gray-300"
              />
              <div>
                <span className="text-sm font-medium text-gray-900">{label}</span>
                <p className="text-xs text-gray-500">{description}</p>
              </div>
            </label>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg shadow-sm border divide-y">
        {TEMPLATE_SLUGS.map(({ slug, label }) => {
          const dimmed = isDisabledByMode(slug);
          return (
            <div key={slug} className={`flex justify-between items-center p-4 ${dimmed ? 'opacity-50' : ''}`}>
              <div>
                <span className="text-sm text-gray-900">{label}</span>
                {dimmed && <p className="text-xs text-gray-400">Excluded by Status Changes Only mode</p>}
              </div>
              <button
                onClick={() => toggle(slug)}
                disabled={dimmed}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  prefs[slug] !== false && !dimmed ? 'bg-primary-600' : 'bg-gray-300'
                } ${dimmed ? 'cursor-not-allowed' : ''}`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    prefs[slug] !== false && !dimmed ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
