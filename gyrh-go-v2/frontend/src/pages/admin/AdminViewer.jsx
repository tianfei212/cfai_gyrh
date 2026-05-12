import React from 'react';
import { AppShell } from '../../app/AppShell';
import { adminScreens } from '../../constants';

export function AdminViewer() {
  return <AppShell mode="admin" navigationItems={adminScreens} />;
}
