import React from 'react';
import { AdminViewer } from '../pages/admin/AdminViewer';
import { KioskViewer } from '../pages/kiosk/KioskViewer';

export function AppRoutes() {
  const path = window.location.pathname;
  if (path.startsWith('/admin_viewer')) {
    return <AdminViewer />;
  }
  return <KioskViewer />;
}
