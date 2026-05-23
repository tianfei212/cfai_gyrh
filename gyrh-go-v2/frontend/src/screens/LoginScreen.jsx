import React, { useState } from 'react';
import { CenteredStage } from '../components/Layout';
import { loginFrontend } from '../services/frontendAuth';

export function LoginScreen({ onHome, onLoginSuccess }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setError('');
    setIsSubmitting(true);
    try {
      const session = await loginFrontend(username.trim(), password);
      if (onLoginSuccess) {
        onLoginSuccess(session);
      } else {
        onHome();
      }
    } catch (err) {
      setError(err.message || '登录失败');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <CenteredStage>
      <form className="auth-card" autoComplete="off" onSubmit={handleSubmit}>
        <div className="avatar-mark" />
        <h1>欢迎登录</h1>
        <p>请输入账号和密码继续访问系统</p>
        <label className="field-block">
          <span>账号</span>
          <input
            autoComplete="off"
            onChange={(event) => setUsername(event.target.value)}
            placeholder="请输入账号"
            name="gyrh-login-user"
            spellCheck="false"
            value={username}
          />
        </label>
        <label className="field-block">
          <span>密码</span>
          <input
            autoComplete="new-password"
            onChange={(event) => setPassword(event.target.value)}
            placeholder="请输入密码"
            name="gyrh-login-pass"
            type="password"
            value={password}
          />
        </label>
        {error ? <p className="auth-error">{error}</p> : null}
        <button className="auth-submit" type="submit" disabled={isSubmitting}>
          {isSubmitting ? '登录中...' : '登录'}
        </button>
        <p className="auth-footnote">账号信息由管理员统一管理</p>
      </form>
    </CenteredStage>
  );
}
