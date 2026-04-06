import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';
import { useNotificationStore } from '../../store/notificationStore';
import { logout } from '../../api/auth';

export function Header() {
  const { user, roles, clearAuth } = useAuthStore();
  const { unreadCount } = useNotificationStore();
  const navigate = useNavigate();

  const handleLogout = async () => {
    try {
      await logout();
    } finally {
      clearAuth();
      navigate('/login');
    }
  };

  return (
    <header className="bg-white border-b px-4 md:px-6 py-3 flex items-center justify-between">
      <div className="flex items-center gap-2 md:gap-4 min-w-0">
        <span className="text-xs md:text-sm text-gray-500 truncate hidden md:inline">
          {roles.map((r) => r.charAt(0).toUpperCase() + r.slice(1)).join(', ')}
        </span>
      </div>
      <div className="flex items-center gap-3 md:gap-4">
        <button
          onClick={() => navigate('/notifications')}
          className="relative text-sm text-gray-600 hover:text-gray-900"
        >
          Notifications
          {unreadCount > 0 && (
            <span className="absolute -top-1 -right-3 bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
              {unreadCount > 9 ? '9+' : unreadCount}
            </span>
          )}
        </button>
        <span className="text-sm font-medium hidden md:inline">{user?.display_name}</span>
        <button
          onClick={handleLogout}
          className="text-sm text-gray-500 hover:text-red-600"
        >
          Logout
        </button>
      </div>
    </header>
  );
}
