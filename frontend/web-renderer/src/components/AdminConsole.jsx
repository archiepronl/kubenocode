import React, { useEffect, useState } from 'react';

const Card = ({ children, style }) => (
  <div style={{ background: '#FFFFFF', borderRadius: '8px', border: '1px solid #E2E8F0', padding: '20px', boxShadow: '0 1px 2px rgba(0,0,0,0.05)', ...style }}>
    {children}
  </div>
);

const Button = ({ children, primary, onClick, style, type="button", disabled=false }) => (
  <button type={type} onClick={onClick} disabled={disabled} style={{
    background: primary ? (disabled ? '#9CA3AF' : '#4F46E5') : '#FFFFFF',
    color: primary ? '#FFFFFF' : '#334155',
    border: primary ? 'none' : '1px solid #CBD5E1',
    padding: '8px 16px',
    borderRadius: '6px',
    fontWeight: '500',
    cursor: disabled ? 'not-allowed' : 'pointer',
    fontSize: '0.875rem',
    transition: 'all 0.2s',
    ...style
  }}>
    {children}
  </button>
);

const PERMISSIONS_LIST = [
  { id: 'app:create', label: 'Create App' },
  { id: 'app:read', label: 'Read App' },
  { id: 'app:update', label: 'Update App' },
  { id: 'app:delete', label: 'Delete App' },
  { id: 'admin', label: 'Super Admin' },
];

export function AdminConsole() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [savingUser, setSavingUser] = useState(null);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/auth/users');
      if (res.ok) {
        setUsers(await res.json() || []);
      }
    } catch (err) {
      console.error("Failed to fetch users", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleTogglePermission = (username, permissionId) => {
    setUsers(users.map(u => {
      if (u.username === username) {
        const hasPerm = u.permissions.includes(permissionId);
        return {
          ...u,
          permissions: hasPerm 
            ? u.permissions.filter(p => p !== permissionId)
            : [...u.permissions, permissionId]
        };
      }
      return u;
    }));
  };

  const handleSave = async (user) => {
    setSavingUser(user.username);
    try {
      await fetch(`/api/auth/users/${encodeURIComponent(user.username)}/permissions`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ permissions: user.permissions })
      });
      // In a real app we'd show a success toast here
    } catch (err) {
      console.error("Failed to save permissions", err);
    } finally {
      setSavingUser(null);
    }
  };

  return (
    <div style={{ flex: 1, padding: '32px 48px', overflowY: 'auto' }}>
      <div style={{ marginBottom: '32px' }}>
        <h1 style={{ fontSize: '1.875rem', fontWeight: '700', margin: '0 0 8px 0' }}>Security & RBAC Admin</h1>
        <p style={{ color: '#64748B', margin: 0 }}>Manage user roles and atomic permissions across the enterprise.</p>
      </div>

      <Card style={{ padding: 0, overflow: 'hidden' }}>
        <div style={{ padding: '20px 24px', borderBottom: '1px solid #E2E8F0', background: '#F8FAFC' }}>
          <h3 style={{ margin: 0, fontSize: '1rem', fontWeight: '600' }}>Permission Matrix</h3>
        </div>
        
        {loading ? (
          <div style={{ padding: '24px', color: '#64748B' }}>Loading users...</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid #E2E8F0', color: '#64748B', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                <th style={{ padding: '16px 24px', fontWeight: '600' }}>User</th>
                {PERMISSIONS_LIST.map(p => (
                  <th key={p.id} style={{ padding: '16px 24px', fontWeight: '600', textAlign: 'center' }}>
                    {p.label}
                  </th>
                ))}
                <th style={{ padding: '16px 24px', fontWeight: '600', textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map(user => (
                <tr key={user.username} style={{ borderBottom: '1px solid #F1F5F9' }}>
                  <td style={{ padding: '16px 24px', fontWeight: '500' }}>{user.username}</td>
                  
                  {PERMISSIONS_LIST.map(p => (
                    <td key={p.id} style={{ padding: '16px 24px', textAlign: 'center' }}>
                      <input 
                        type="checkbox" 
                        checked={user.permissions.includes(p.id)}
                        onChange={() => handleTogglePermission(user.username, p.id)}
                        style={{ cursor: 'pointer', width: '16px', height: '16px' }}
                      />
                    </td>
                  ))}
                  
                  <td style={{ padding: '16px 24px', textAlign: 'right' }}>
                    <Button 
                      primary 
                      onClick={() => handleSave(user)}
                      disabled={savingUser === user.username}
                    >
                      {savingUser === user.username ? 'Saving...' : 'Apply via K8s'}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
}
