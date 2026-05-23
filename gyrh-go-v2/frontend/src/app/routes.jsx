import React, { useEffect, useState } from 'react';
import { AdminViewer } from '../pages/admin/AdminViewer';
import { KioskViewer } from '../pages/kiosk/KioskViewer';
import { LoginScreen } from '../screens/LoginScreen';
import {
  buildLoginRedirectPath,
  canAccessPath,
  getPostLoginPath,
  getStoredFrontendSession,
  validateFrontendSession,
} from '../services/frontendAuth';

export function AppRoutes() {
  const path = window.location.pathname;
  const isLoginPath = path.startsWith('/login');
  const nextPath = new URLSearchParams(window.location.search).get('next') || '/';
  const [session, setSession] = useState(() => getStoredFrontendSession());
  const [isChecking, setIsChecking] = useState(Boolean(session));
  const needsLoginRedirect = !isChecking && !session && !isLoginPath;
  const needsForbiddenRedirect = !isChecking && Boolean(session) && !isLoginPath && !canAccessPath(path, session);

  useEffect(() => {
    if (needsLoginRedirect) {
      window.location.replace(buildLoginRedirectPath(path, window.location.search));
    } else if (needsForbiddenRedirect) {
      window.location.replace('/login');
    }
  }, [needsLoginRedirect, needsForbiddenRedirect, path]);

  useEffect(() => {
    let cancelled = false;
    async function checkSession() {
      const validSession = await validateFrontendSession(session);
      if (!cancelled) {
        setSession(validSession);
        setIsChecking(false);
      }
    }
    if (session?.token) {
      checkSession();
    } else {
      setIsChecking(false);
    }
    return () => {
      cancelled = true;
    };
  }, []);

  if (isChecking) {
    return <LoginScreen />;
  }

  if (needsLoginRedirect) {
    return <LoginScreen />;
  }

  if (isLoginPath) {
    return <LoginScreen onLoginSuccess={(nextSession) => {
      setSession(nextSession);
      window.location.replace(getPostLoginPath(nextSession, nextPath));
    }} />;
  }

  if (needsForbiddenRedirect) {
    return <LoginScreen />;
  }

  if (path.startsWith('/admin_viewer')) {
    return <AdminViewer />;
  }
  return <KioskViewer />;
}
