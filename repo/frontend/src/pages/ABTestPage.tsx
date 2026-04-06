import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listABTests, rollbackABTest, createABTest, getExperimentRegistry } from '../api/analytics';
import { formatDateTimeHuman } from '../utils/formatters';
import { useState, useEffect } from 'react';

export function ABTestPage() {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);

  // Fetch the experiment registry from the backend — single source of truth.
  const { data: registeredExperiments } = useQuery({
    queryKey: ['experimentRegistry'],
    queryFn: () => getExperimentRegistry().then((r) => r.data),
    staleTime: 300000,
  });

  const experimentNames = registeredExperiments ? Object.keys(registeredExperiments) : [];

  const [form, setForm] = useState({
    name: '', description: '', traffic_pct: 50,
    start_date: '', end_date: '',
    control_variant: '',
    test_variant: '',
    rollback_threshold_pct: 15,
  });

  // Initialise form defaults once registry loads.
  useEffect(() => {
    if (registeredExperiments && experimentNames.length > 0 && !form.name) {
      const firstName = experimentNames[0];
      const exp = registeredExperiments[firstName];
      setForm((f) => ({
        ...f,
        name: firstName,
        control_variant: exp?.variants[0] || '',
        test_variant: exp?.variants[1] || '',
      }));
    }
  }, [registeredExperiments]);

  const { data: tests } = useQuery({
    queryKey: ['abtests'],
    queryFn: () => listABTests().then((r) => r.data),
  });

  const rollbackMutation = useMutation({
    mutationFn: rollbackABTest,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['abtests'] });
      queryClient.invalidateQueries({ queryKey: ['abAssignments'] });
    },
  });

  const createMutation = useMutation({
    mutationFn: () => createABTest(form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['abtests'] });
      setShowCreate(false);
    },
  });

  const statusColors: Record<string, string> = {
    draft: 'bg-gray-100 text-gray-800',
    running: 'bg-green-100 text-green-800',
    rolled_back: 'bg-red-100 text-red-800',
    completed: 'bg-blue-100 text-blue-800',
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">A/B Tests</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="px-4 py-2 bg-primary-600 text-white rounded text-sm hover:bg-primary-700">
          {showCreate ? 'Cancel' : 'New Test'}
        </button>
      </div>

      {showCreate && registeredExperiments && (
        <div className="bg-white p-6 rounded-lg shadow-sm border mb-6">
          <h2 className="text-lg font-semibold mb-4">Create A/B Test</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Experiment</label>
              <select value={form.name} onChange={(e) => {
                const name = e.target.value;
                const exp = registeredExperiments[name];
                setForm({ ...form, name, control_variant: exp?.variants[0] || '', test_variant: exp?.variants[1] || '' });
              }} className="w-full px-3 py-2 border rounded">
                {experimentNames.map((n) => (
                  <option key={n} value={n}>{n} — {registeredExperiments[n].description}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Traffic %</label>
              <input type="number" value={form.traffic_pct} onChange={(e) => setForm({ ...form, traffic_pct: Number(e.target.value) })}
                className="w-full px-3 py-2 border rounded" min={1} max={100} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Start Date</label>
              <input type="datetime-local" value={form.start_date} onChange={(e) => setForm({ ...form, start_date: e.target.value })}
                className="w-full px-3 py-2 border rounded" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">End Date</label>
              <input type="datetime-local" value={form.end_date} onChange={(e) => setForm({ ...form, end_date: e.target.value })}
                className="w-full px-3 py-2 border rounded" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Control Variant</label>
              <select value={form.control_variant} onChange={(e) => setForm({ ...form, control_variant: e.target.value })}
                className="w-full px-3 py-2 border rounded">
                {(registeredExperiments[form.name]?.variants || []).map((v) => (
                  <option key={v} value={v}>{v}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Test Variant</label>
              <select value={form.test_variant} onChange={(e) => setForm({ ...form, test_variant: e.target.value })}
                className="w-full px-3 py-2 border rounded">
                {(registeredExperiments[form.name]?.variants || []).map((v) => (
                  <option key={v} value={v}>{v}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Rollback Threshold %</label>
              <input type="number" value={form.rollback_threshold_pct} onChange={(e) => setForm({ ...form, rollback_threshold_pct: Number(e.target.value) })}
                className="w-full px-3 py-2 border rounded" min={1} max={100} />
            </div>
          </div>
          <div className="mt-4">
            <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="Description" className="w-full px-3 py-2 border rounded" rows={2} />
          </div>
          <button onClick={() => createMutation.mutate()} className="mt-4 px-4 py-2 bg-primary-600 text-white rounded text-sm">
            Create Test
          </button>
        </div>
      )}

      <div className="space-y-3">
        {tests?.map((test) => (
          <div key={test.id} className="bg-white p-4 md:p-5 rounded-lg shadow-sm border">
            <div className="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-3">
              <div>
                <h3 className="font-medium">{test.name}</h3>
                <p className="text-sm text-gray-500 mt-1">{test.description}</p>
                <div className="text-xs text-gray-400 mt-2 space-y-0.5">
                  <p>Start: {formatDateTimeHuman(test.start_date)}</p>
                  <p>End: {formatDateTimeHuman(test.end_date)}</p>
                  <p>Traffic: {test.traffic_pct}% | Rollback threshold: {test.rollback_threshold_pct}%</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className={`px-2 py-1 text-xs rounded-full ${statusColors[test.status] || 'bg-gray-100'}`}>
                  {test.status}
                </span>
                {test.status === 'running' && (
                  <button
                    onClick={() => rollbackMutation.mutate(test.id)}
                    className="px-3 py-1 text-xs bg-red-100 text-red-700 rounded hover:bg-red-200"
                  >
                    Rollback
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
