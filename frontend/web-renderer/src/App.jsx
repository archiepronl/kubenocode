import React, { useState } from 'react';
import { Dashboard } from './components/Dashboard.jsx';
import { WorkflowCanvas } from './components/Canvas.jsx';

export function App() {
  const [currentView, setCurrentView] = useState('dashboard'); // 'dashboard' | 'editor'
  const [editingApp, setEditingApp] = useState(null);

  const handleOpenEditor = (appName) => {
    setEditingApp(appName);
    setCurrentView('editor');
  };

  const handleBackToDashboard = () => {
    setCurrentView('dashboard');
    setEditingApp(null);
  };

  return (
    <div style={{ width: '100vw', height: '100vh', background: '#0a0a0a', overflow: 'auto' }}>
      {currentView === 'dashboard' ? (
        <Dashboard onOpenEditor={handleOpenEditor} />
      ) : (
        <WorkflowCanvas appName={editingApp} onBack={handleBackToDashboard} />
      )}
    </div>
  );
}
