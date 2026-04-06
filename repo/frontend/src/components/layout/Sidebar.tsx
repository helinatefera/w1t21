import { NavLink } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';
import { useState } from 'react';

const navItems = [
  { to: '/', label: 'Dashboard', roles: [] },
  { to: '/catalog', label: 'Catalog', roles: [] },
  { to: '/orders', label: 'Orders', roles: [] },
  { to: '/notifications', label: 'Notifications', roles: [] },
  { to: '/sell/new', label: 'List Item', roles: ['seller'] },
  { to: '/analytics', label: 'Analytics', roles: ['administrator', 'compliance_analyst'] },
  // A/B test run and rollback operations are permitted for Administrators and Compliance Analysts.
  { to: '/analytics/ab-tests', label: 'A/B Tests', roles: ['administrator', 'compliance_analyst'] },
  { to: '/admin/users', label: 'Users', roles: ['administrator'] },
  { to: '/admin/moderation', label: 'Moderation', roles: ['administrator', 'compliance_analyst'] },
  { to: '/admin/anomalies', label: 'Anomaly Alerts', roles: ['compliance_analyst', 'administrator'] },
];

export function Sidebar() {
  const { hasRole } = useAuthStore();
  const [collapsed, setCollapsed] = useState(false);

  const visibleItems = navItems.filter(
    (item) => item.roles.length === 0 || item.roles.some((r) => hasRole(r))
  );

  return (
    <aside
      className={`bg-white border-r min-h-screen p-4 transition-all duration-200 flex-shrink-0 ${
        collapsed ? 'w-16' : 'w-48 md:w-56 lg:w-64'
      }`}
    >
      <div className="flex items-center justify-between mb-6 md:mb-8">
        {!collapsed && (
          <div>
            <h1 className="text-lg lg:text-xl font-bold text-primary-700">LedgerMint</h1>
            <p className="text-xs text-gray-500 hidden lg:block">Digital Collectibles Exchange</p>
          </div>
        )}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="p-1 text-gray-400 hover:text-gray-600 md:hidden lg:hidden"
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? '\u25B6' : '\u25C0'}
        </button>
      </div>
      <nav className="space-y-1">
        {visibleItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            title={collapsed ? item.label : undefined}
            className={({ isActive }) =>
              `block px-2 lg:px-3 py-2 rounded text-sm transition-colors ${
                isActive
                  ? 'bg-primary-50 text-primary-700 font-medium'
                  : 'text-gray-600 hover:bg-gray-50'
              } ${collapsed ? 'text-center' : ''}`
            }
          >
            {collapsed ? item.label.charAt(0) : item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
