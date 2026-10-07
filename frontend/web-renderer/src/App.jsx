import React, { useState } from 'react';
import { PermissionsProvider } from "./hooks/usePermissions.jsx";
import { Dashboard } from './components/Dashboard.jsx';
import { WizardEditor } from './components/WizardEditor.jsx';


const Button = ({ children, primary, onClick, style, type="button", className="" }) => (
  <button type={type} onClick={onClick} className={className} style={{
    background: primary ? '#4F46E5' : '#FFFFFF',
    color: primary ? '#FFFFFF' : '#334155',
    border: primary ? 'none' : '1px solid #CBD5E1',
    padding: '10px 16px',
    borderRadius: '6px',
    fontWeight: '500',
    cursor: 'pointer',
    fontSize: '0.875rem',
    transition: 'all 0.2s',
    width: '100%',
    ...style
  }}>
    {children}
  </button>
);

const Login = ({ onLogin }) => (
  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', background: '#F8F9FA', color: '#0F172A' }}>
    <div style={{ background: '#FFFFFF', padding: '40px', borderRadius: '12px', boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)', width: '400px', border: '1px solid #E2E8F0' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', marginBottom: '32px' }}>
        <div style={{ width: '48px', height: '48px', background: '#4F46E5', borderRadius: '8px', marginBottom: '16px' }}></div>
        <h1 style={{ fontSize: '1.5rem', fontWeight: '700', margin: 0 }}>Sign in</h1>
        <div style={{ color: '#64748B', fontSize: '0.875rem', marginTop: '8px' }}>NextKube</div>
      </div>
      
      <form onSubmit={(e) => { e.preventDefault(); onLogin(); }} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
        <div>
          <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '8px' }}>Email</label>
          <input type="email" placeholder="name@example.com" required style={{ width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }} />
        </div>
        <div>
          <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '8px' }}>Password</label>
          <input type="password" placeholder="••••••••" required style={{ width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }} />
        </div>
        
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem', color: '#475569', cursor: 'pointer' }}>
            <input type="checkbox" style={{ cursor: 'pointer' }} />
            Remember me
          </label>
          <a href="#" style={{ color: '#4F46E5', textDecoration: 'none', fontSize: '0.875rem', fontWeight: '500' }}>Forgot password?</a>
        </div>
        
        <Button primary type="submit" style={{ marginTop: '8px' }}>Sign in</Button>
      </form>
      
      <div style={{ textAlign: 'center', marginTop: '32px', fontSize: '0.75rem', color: '#94A3B8' }}>
        Mock sign-in · Demo mode
      </div>
    </div>
  </div>
);


export function App() {
  const [currentView, setCurrentView] = useState('login'); // 'login' | 'dashboard' | 'editor'
  const [editingApp, setEditingApp] = useState(null);
  const [authToken, setAuthToken] = useState(null);

  const handleLogin = (e) => {
    // In a real app, this would be an API call returning a JWT.
    // For now, we mock a token.
    setAuthToken('mock-jwt-token');
    setCurrentView('dashboard');
  };

  const handleOpenEditor = (appName) => {
    setEditingApp(appName);
    setCurrentView('editor');
  };

  const handleBackToDashboard = () => {
    setCurrentView('dashboard');
    setEditingApp(null);
  };

  const handleLogout = () => {
    setAuthToken(null);
    setCurrentView('login');
  };

  return (
    <PermissionsProvider token={authToken}>
      <div style={{ width: '100vw', height: '100vh', background: '#F8F9FA', overflow: 'hidden', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif' }}>
        {currentView === 'login' ? (
          <Login onLogin={handleLogin} />
        ) : currentView === 'dashboard' ? (
          <Dashboard onOpenEditor={handleOpenEditor} onLogout={handleLogout} />
        ) : (
          <WizardEditor appName={editingApp} onBack={handleBackToDashboard} />
        )}
      </div>
    </PermissionsProvider>
  );
}
