import React from 'react';
import { CenteredStage } from '../components/Layout';

export function LogoutScreen({ onHome, onLogoutAction }) {
  return (
    <CenteredStage muted>
      <div className="logout-modal">
        <div className="avatar-mark" />
        <h1>确认退出登录？</h1>
        <p>退出后需要重新输入账号与密码。</p>
        <div className="dual-actions">
          <button className="auth-secondary" type="button" onClick={onHome}>取消</button>
          <button className="auth-submit small" type="button" onClick={onLogoutAction}>退出</button>
        </div>
      </div>
    </CenteredStage>
  );
}
