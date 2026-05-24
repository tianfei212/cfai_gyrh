import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './theme/tokens.css';
import './theme/glass.css';
import './theme/admin.css';
import './theme/kiosk.css';
import './styles.css';
import './theme/liquid-glass.css';

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
