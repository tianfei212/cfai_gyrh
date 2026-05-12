import React, { startTransition, useMemo, useState } from 'react';
import { GlowBackdrop, ControlRail } from '../components/Layout';
import { adminScreens } from '../constants';
import { DashboardScreen } from '../screens/DashboardScreen';
import { HistoryScreen } from '../screens/HistoryScreen';
import { BackgroundManagerScreen } from '../screens/BackgroundManagerScreen';
import { SkillManagerScreen } from '../screens/SkillManagerScreen';
import { StyleManagerScreen } from '../screens/StyleManagerScreen';
import { PreviewScreen } from '../screens/PreviewScreen';
import { CaptureScreen } from '../screens/CaptureScreen';
import { RenderingScreen } from '../screens/RenderingScreen';
import { LoginScreen } from '../screens/LoginScreen';
import { LogoutScreen } from '../screens/LogoutScreen';
import { getNextModel } from '../utils/modelProvider';
import { normalizePreviewSelection } from '../utils/previewSelection';

export function AppShell({ mode = 'admin', navigationItems = adminScreens }) {
  const [screen, setScreen] = useState('dashboard');
  const [model, setModel] = useState('G'); // W: Wan, G: Gemini, GPT: 302 GPT Image
  const [selectedBg, setSelectedBg] = useState(null);
  const [capturedImage, setCapturedImage] = useState(null);
  const [previewMode, setPreviewMode] = useState('compare');

  const activeScreen = useMemo(
    () => adminScreens.find((item) => item.key === screen) ?? adminScreens[0],
    [screen],
  );

  function changeScreen(nextScreen) {
    console.log(`[App] Navigate to screen: ${nextScreen}`);
    startTransition(() => {
      setScreen(nextScreen);
    });
  }

  const toggleModel = () => {
    setModel((prev) => {
      const next = getNextModel(prev);
      console.log(`[App] Toggle model: ${next}`);
      return next;
    });
  };
  const goHome = () => changeScreen('dashboard');
  const goHistory = () => changeScreen('history');
  const goBackgrounds = () => changeScreen('backgrounds');
  const goSkills = () => changeScreen('skills');
  const goStyles = () => changeScreen('styles');
  const goCapture = (bg) => {
    setSelectedBg(bg);
    changeScreen('capture');
  };
  const goPreview = (selection) => {
    const nextPreview = normalizePreviewSelection(selection);
    setCapturedImage(nextPreview.image);
    setPreviewMode(nextPreview.mode);
    changeScreen('preview');
  };
  const logout = () => changeScreen(mode === 'kiosk' ? 'dashboard' : 'logout');

  const goLogin = () => changeScreen('login');

  const navHandlers = {
    onHome: goHome,
    onHistory: goHistory,
    onBackgrounds: goBackgrounds,
    onSkills: goSkills,
    onStyles: goStyles,
    onCapture: goCapture,
    onPreview: goPreview,
    onLogout: logout,
    onLogoutAction: goLogin,
    onToggleModel: toggleModel,
    model,
    selectedBg,
    capturedImage,
    previewMode,
    mode,
  };

  return (
    <div className={`app-shell app-mode-${mode} screen-${screen}`}>
      <GlowBackdrop />
      {!activeScreen.hideInNav && (
        <ControlRail
          screen={screen}
          onSelect={changeScreen}
          items={navigationItems}
        />
      )}
      <div className="canvas-frame">
        {screen === 'dashboard' ? <DashboardScreen {...navHandlers} /> : null}
        {screen === 'history' ? <HistoryScreen {...navHandlers} /> : null}
        {screen === 'backgrounds' && mode === 'admin' ? <BackgroundManagerScreen {...navHandlers} /> : null}
        {screen === 'skills' && mode === 'admin' ? <SkillManagerScreen {...navHandlers} /> : null}
        {screen === 'styles' && mode === 'admin' ? <StyleManagerScreen {...navHandlers} /> : null}
        {screen === 'preview' ? <PreviewScreen {...navHandlers} /> : null}
        {screen === 'capture' ? <CaptureScreen {...navHandlers} /> : null}
        {screen === 'rendering' ? <RenderingScreen {...navHandlers} /> : null}
        {screen === 'login' && mode === 'admin' ? <LoginScreen {...navHandlers} /> : null}
        {screen === 'logout' && mode === 'admin' ? <LogoutScreen {...navHandlers} /> : null}
      </div>
      <div className="status-pill">
        <span className="status-dot" />
        <span>{activeScreen.label}</span>
      </div>
    </div>
  );
}
