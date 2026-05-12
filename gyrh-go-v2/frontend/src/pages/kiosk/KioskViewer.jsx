import React from 'react';
import { AppShell } from '../../app/AppShell';
import { kioskScreens } from '../../constants';

export function KioskViewer() {
  return <AppShell mode="kiosk" navigationItems={kioskScreens} />;
}
