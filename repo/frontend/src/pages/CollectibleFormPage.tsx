import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createCollectible, getCollectible, updateCollectible } from '../api/collectibles';

interface CollectibleFormState {
  title: string;
  description: string;
  price_cents: number;
  currency: string;
  contract_address: string;
  chain_id: number;
  token_id: string;
  metadata_uri: string;
  image_url: string;
}

const initialForm: CollectibleFormState = {
  title: '',
  description: '',
  price_cents: 0,
  currency: 'USD',
  contract_address: '',
  chain_id: 0,
  token_id: '',
  metadata_uri: '',
  image_url: '',
};

export function CollectibleFormPage() {
  const { id } = useParams<{ id: string }>();
  const isEdit = !!id;
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [form, setForm] = useState<CollectibleFormState>(initialForm);
  const [error, setError] = useState('');

  const { data: editData } = useQuery({
    queryKey: ['collectible', id],
    queryFn: () => getCollectible(id!).then((r) => r.data),
    enabled: isEdit,
  });

  useEffect(() => {
    if (editData) {
      const c = editData.collectible;
      setForm({
        title: c.title,
        description: c.description,
        price_cents: c.price_cents,
        currency: c.currency,
        contract_address: c.contract_address || '',
        chain_id: c.chain_id || 0,
        token_id: c.token_id || '',
        metadata_uri: c.metadata_uri || '',
        image_url: c.image_url || '',
      });
    }
  }, [editData]);

  const createMutation = useMutation({
    mutationFn: () => createCollectible({
      ...form,
      price_cents: Number(form.price_cents),
      chain_id: form.chain_id || undefined,
      token_id: form.token_id || undefined,
    }),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['collectibles'] });
      navigate(`/catalog/${res.data.id}`);
    },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed'),
  });

  const updateMutation = useMutation({
    mutationFn: () => updateCollectible(id!, {
      title: form.title,
      description: form.description,
      price_cents: Number(form.price_cents),
      image_url: form.image_url,
      metadata_uri: form.metadata_uri,
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['collectible', id] });
      navigate(`/catalog/${id}`);
    },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed'),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isEdit) updateMutation.mutate();
    else createMutation.mutate();
  };

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">{isEdit ? 'Edit Listing' : 'Create Listing'}</h1>

      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-sm border p-6 space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Title *</label>
          <input type="text" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })}
            className="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-primary-500" required maxLength={300} />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
          <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
            className="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-primary-500" rows={4} maxLength={5000} />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Price (cents) *</label>
            <input type="number" value={form.price_cents} onChange={(e) => setForm({ ...form, price_cents: Number(e.target.value) })}
              className="w-full px-3 py-2 border rounded focus:ring-2 focus:ring-primary-500" required min={1} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Currency</label>
            <input type="text" value={form.currency} onChange={(e) => setForm({ ...form, currency: e.target.value })}
              className="w-full px-3 py-2 border rounded" maxLength={3} />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Image URL</label>
          <input type="url" value={form.image_url} onChange={(e) => setForm({ ...form, image_url: e.target.value })}
            className="w-full px-3 py-2 border rounded" />
        </div>

        {!isEdit && (
          <>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Contract Address</label>
              <input type="text" value={form.contract_address} onChange={(e) => setForm({ ...form, contract_address: e.target.value })}
                className="w-full px-3 py-2 border rounded font-mono text-sm" maxLength={42} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Metadata URI</label>
              <input type="url" value={form.metadata_uri} onChange={(e) => setForm({ ...form, metadata_uri: e.target.value })}
                className="w-full px-3 py-2 border rounded" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Chain ID</label>
                <input type="number" value={form.chain_id || ''} onChange={(e) => setForm({ ...form, chain_id: Number(e.target.value) })}
                  className="w-full px-3 py-2 border rounded" min={1} placeholder="e.g. 1" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Token ID</label>
                <input type="text" value={form.token_id} onChange={(e) => setForm({ ...form, token_id: e.target.value })}
                  className="w-full px-3 py-2 border rounded font-mono text-sm" placeholder="Auto-generated if empty" />
              </div>
            </div>
            <p className="text-xs text-gray-400">Chain ID and Token ID are required when Contract Address is provided, optional otherwise.</p>
          </>
        )}

        {error && <div className="text-sm text-red-600 bg-red-50 p-2 rounded">{error}</div>}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={createMutation.isPending || updateMutation.isPending || !form.title || !form.price_cents}
            className="px-6 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {createMutation.isPending || updateMutation.isPending
              ? (isEdit ? 'Updating...' : 'Creating...')
              : (isEdit ? 'Update' : 'Create Listing')}
          </button>
          <button type="button" onClick={() => navigate(-1)} className="px-6 py-2 bg-gray-100 rounded hover:bg-gray-200">Cancel</button>
        </div>
      </form>
    </div>
  );
}
