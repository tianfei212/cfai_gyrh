import React from 'react';
import { CenteredStage } from '../components/Layout';

export function LoginScreen({ onHome }) {
  return (
    <CenteredStage>
      <div className="auth-card">
        <div className="avatar-mark" />
        <h1>欢迎登录</h1>
        <p>使用账号继续你的创作流程</p>
        <label className="field-block">
          <span>账号</span>
          <input placeholder="请输入手机号 / 邮箱" />
        </label>
        <label className="field-block">
          <span>密码</span>
          <input placeholder="请输入密码" type="password" />
        </label>
        <div className="helper-row">
          <span>记住我</span>
          <span>忘记密码?</span>
        </div>
        <button className="auth-submit" type="button" onClick={onHome}>登录</button>
        <p className="auth-footnote">没有账号？立即注册</p>
      </div>
    </CenteredStage>
  );
}
