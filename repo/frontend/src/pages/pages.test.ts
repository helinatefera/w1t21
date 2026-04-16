import { describe, it, expect } from 'vitest';
import { LoginPage } from './LoginPage';
import { DashboardPage } from './DashboardPage';
import { CatalogPage } from './CatalogPage';
import { OrdersPage } from './OrdersPage';
import { OrderDetailPage } from './OrderDetailPage';
import { NotificationsPage } from './NotificationsPage';
import { NotificationPrefsPage } from './NotificationPrefsPage';
import { AnalyticsPage } from './AnalyticsPage';
import { ABTestPage } from './ABTestPage';
import { AdminUsersPage } from './AdminUsersPage';
import { AdminModerationPage } from './AdminModerationPage';
import { AnomalyAlertsPage } from './AnomalyAlertsPage';
import { CollectiblePage } from './CollectiblePage';
import { CollectibleFormPage } from './CollectibleFormPage';
import { MessagesPage } from './MessagesPage';

describe('Page exports', () => {
  const pages = [
    { name: 'LoginPage', component: LoginPage },
    { name: 'DashboardPage', component: DashboardPage },
    { name: 'CatalogPage', component: CatalogPage },
    { name: 'OrdersPage', component: OrdersPage },
    { name: 'OrderDetailPage', component: OrderDetailPage },
    { name: 'NotificationsPage', component: NotificationsPage },
    { name: 'NotificationPrefsPage', component: NotificationPrefsPage },
    { name: 'AnalyticsPage', component: AnalyticsPage },
    { name: 'ABTestPage', component: ABTestPage },
    { name: 'AdminUsersPage', component: AdminUsersPage },
    { name: 'AdminModerationPage', component: AdminModerationPage },
    { name: 'AnomalyAlertsPage', component: AnomalyAlertsPage },
    { name: 'CollectiblePage', component: CollectiblePage },
    { name: 'CollectibleFormPage', component: CollectibleFormPage },
    { name: 'MessagesPage', component: MessagesPage },
  ];

  for (const { name, component } of pages) {
    it(`${name} exports a function component`, () => {
      expect(typeof component).toBe('function');
    });
  }
});
