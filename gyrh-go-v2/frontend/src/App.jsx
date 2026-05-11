import React, { startTransition, useMemo, useState } from 'react';
import { GlowBackdrop, ControlRail } from './components/Layout';
import { screens } from './constants';
import { DashboardScreen } from './screens/DashboardScreen';
import { HistoryScreen } from './screens/HistoryScreen';
import { BackgroundManagerScreen } from './screens/BackgroundManagerScreen';
import { SkillManagerScreen } from './screens/SkillManagerScreen';
import { StyleManagerScreen } from './screens/StyleManagerScreen';
import { PreviewScreen } from './screens/PreviewScreen';
import { CaptureScreen } from './screens/CaptureScreen';
import { RenderingScreen } from './screens/RenderingScreen';
import { LoginScreen } from './screens/LoginScreen';
import { LogoutScreen } from './screens/LogoutScreen';
import { normalizePreviewSelection } from './utils/previewSelection';

function App() {
  const [screen, setScreen] = useState('dashboard');
  const [model, setModel] = useState('W'); // 'W' for Wan, 'G' for Gemini
  const [selectedBg, setSelectedBg] = useState(null);
  const [capturedImage, setCapturedImage] = useState(null);
  const [previewMode, setPreviewMode] = useState('compare');

  const activeScreen = useMemo(
    () => screens.find((item) => item.key === screen) ?? screens[0],
    [screen],
  );

  function changeScreen(nextScreen) {
    console.log(`[App] Navigate to screen: ${nextScreen}`);
    startTransition(() => {
      setScreen(nextScreen);
    });
  }

  const toggleModel = () => {
    setModel(prev => {
      const next = prev === 'W' ? 'G' : 'W';
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
  const logout = () => changeScreen('logout');

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
    model: model,
    selectedBg: selectedBg,
    capturedImage: capturedImage,
    previewMode: previewMode,
  };

  return (
    <div className={`app-shell screen-${screen}`}>
      <GlowBackdrop />
      {!activeScreen.hideInNav && (
        <ControlRail
          screen={screen}
          onSelect={changeScreen}
        />
      )}
      <div className="canvas-frame">
        {screen === 'dashboard' ? <DashboardScreen {...navHandlers} /> : null}
        {screen === 'history' ? <HistoryScreen {...navHandlers} /> : null}
        {screen === 'backgrounds' ? <BackgroundManagerScreen {...navHandlers} /> : null}
        {screen === 'skills' ? <SkillManagerScreen {...navHandlers} /> : null}
        {screen === 'styles' ? <StyleManagerScreen {...navHandlers} /> : null}
        {screen === 'preview' ? <PreviewScreen {...navHandlers} /> : null}
        {screen === 'capture' ? <CaptureScreen {...navHandlers} /> : null}
        {screen === 'rendering' ? <RenderingScreen {...navHandlers} /> : null}
        {screen === 'login' ? <LoginScreen {...navHandlers} /> : null}
        {screen === 'logout' ? <LogoutScreen {...navHandlers} /> : null}
      </div>
      <div className="status-pill">
        <span className="status-dot" />
        <span>{activeScreen.label}</span>
      </div>
    </div>
  );
}

export default App;
