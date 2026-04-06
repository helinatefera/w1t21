import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listUsers, createUser, addRole, unlockUser } from '../api/analytics';
import { formatDate } from '../utils/formatters';

export function AdminUsersPage() {
  const [page, setPage] = useState(1);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ username: '', password: '', display_name: '', email: '' });
  const [roleForm, setRoleForm] = useState({ userId: '', roleName: '' });
  const queryClient = useQueryClient();

  const { data } = useQuery({
    queryKey: ['adminUsers', page],
    queryFn: () => listUsers(page).then((r) => r.data),
  });

  const createMutation = useMutation({
    mutationFn: (data: { username: string; password: string; display_name: string; email: string }) => createUser(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['adminUsers'] });
      setShowCreate(false);
      setForm({ username: '', password: '', display_name: '', email: '' });
    },
  });

  const addRoleMutation = useMutation({
    mutationFn: ({ userId, roleName }: { userId: string; roleName: string }) => addRole(userId, roleName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['adminUsers'] });
      setRoleForm({ userId: '', roleName: '' });
    },
  });

  const unlockMutation = useMutation({
    mutationFn: unlockUser,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['adminUsers'] }),
  });

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">User Management</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="px-4 py-2 bg-primary-600 text-white rounded text-sm">
          {showCreate ? 'Cancel' : 'Create User'}
        </button>
      </div>

      {showCreate && (
        <div className="bg-white p-6 rounded-lg shadow-sm border mb-6">
          <div className="grid grid-cols-2 gap-4">
            <input type="text" placeholder="Username" value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              className="px-3 py-2 border rounded" />
            <input type="password" placeholder="Password (min 8 chars)" value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              className="px-3 py-2 border rounded" />
            <input type="text" placeholder="Display Name" value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              className="px-3 py-2 border rounded" />
            <input type="email" placeholder="Email (optional)" value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="px-3 py-2 border rounded" />
          </div>
          <button
            onClick={() => createMutation.mutate(form)}
            disabled={createMutation.isPending || !form.username || !form.password || !form.display_name}
            className="mt-4 px-4 py-2 bg-primary-600 text-white rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-primary-700 transition-colors"
          >
            {createMutation.isPending ? 'Creating...' : 'Create'}
          </button>
          {createMutation.isError && (
            <p className="mt-2 text-sm text-red-600">
              Failed to create user. {(createMutation.error as any)?.response?.data?.error || 'Please try again.'}
            </p>
          )}
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Display Name</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {data?.data?.map((user) => (
              <tr key={user.id}>
                <td className="px-4 py-3 text-sm font-medium">{user.username}</td>
                <td className="px-4 py-3 text-sm">{user.display_name}</td>
                <td className="px-4 py-3 text-sm">
                  {user.is_locked ? (
                    <span className="text-red-600 font-medium">Locked</span>
                  ) : (
                    <span className="text-green-600">Active</span>
                  )}
                </td>
                <td className="px-4 py-3 text-sm text-gray-500">{formatDate(user.created_at)}</td>
                <td className="px-4 py-3 text-sm">
                  <div className="flex gap-2">
                    {user.is_locked && (
                      <button onClick={() => unlockMutation.mutate(user.id)} className="text-xs text-primary-600 hover:underline">
                        Unlock
                      </button>
                    )}
                    <select
                      value=""
                      onChange={(e) => {
                        if (e.target.value) {
                          addRoleMutation.mutate({ userId: user.id, roleName: e.target.value });
                        }
                      }}
                      className="text-xs border rounded px-1"
                    >
                      <option value="">Add role...</option>
                      <option value="buyer">Buyer</option>
                      <option value="seller">Seller</option>
                      <option value="administrator">Admin</option>
                      <option value="compliance_analyst">Compliance</option>
                    </select>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
