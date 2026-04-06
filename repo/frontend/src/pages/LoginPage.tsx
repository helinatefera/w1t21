import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { login } from '../api/auth';
import { useAuthStore } from '../store/authStore';

const loginSchema = z.object({
  username: z
    .string()
    .min(3, 'Username must be at least 3 characters')
    .max(100, 'Username must be at most 100 characters'),
  password: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .max(128, 'Password must be at most 128 characters'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export function LoginPage() {
  const [serverError, setServerError] = useState('');
  const [loading, setLoading] = useState(false);
  const { setAuth } = useAuthStore();
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    mode: 'onChange',
    resolver: async (values) => {
      const result = loginSchema.safeParse(values);
      if (result.success) {
        return { values: result.data, errors: {} };
      }
      const fieldErrors: Record<string, { type: string; message: string }> = {};
      for (const issue of result.error.issues) {
        const field = issue.path[0] as string;
        if (!fieldErrors[field]) {
          fieldErrors[field] = { type: 'validation', message: issue.message };
        }
      }
      return { values: {}, errors: fieldErrors };
    },
  });

  const onSubmit = async (data: LoginFormData) => {
    setServerError('');
    setLoading(true);

    try {
      const { data: resp } = await login(data.username, data.password);
      setAuth(resp.user, resp.roles);
      navigate('/');
    } catch (err: any) {
      const msg = err.response?.data?.error?.message || 'Login failed';
      setServerError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="bg-white p-6 sm:p-8 rounded-lg shadow-md w-full max-w-sm md:max-w-md">
        <h1 className="text-2xl md:text-3xl font-bold text-center text-primary-700 mb-2">LedgerMint</h1>
        <p className="text-sm text-center text-gray-500 mb-6">Digital Collectibles Exchange</p>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
            <input
              type="text"
              {...register('username')}
              className={`w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary-500 ${
                errors.username ? 'border-red-400 focus:ring-red-400' : 'border-gray-300'
              }`}
              autoFocus
            />
            {errors.username && (
              <p className="text-xs text-red-600 mt-1">{errors.username.message}</p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input
              type="password"
              {...register('password')}
              className={`w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary-500 ${
                errors.password ? 'border-red-400 focus:ring-red-400' : 'border-gray-300'
              }`}
            />
            {errors.password && (
              <p className="text-xs text-red-600 mt-1">{errors.password.message}</p>
            )}
          </div>

          {serverError && (
            <div className="text-sm text-red-600 bg-red-50 p-2 rounded">{serverError}</div>
          )}

          <button
            type="submit"
            disabled={loading || !!errors.username || !!errors.password}
            className="w-full bg-primary-600 text-white py-2.5 md:py-3 rounded font-medium hover:bg-primary-700 disabled:opacity-50 transition-colors"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}
