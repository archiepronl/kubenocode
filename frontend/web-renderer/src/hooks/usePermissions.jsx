import React, { createContext, useContext, useState, useEffect } from 'react';

const PermissionsContext = createContext({
  permissions: [],
  loading: true,
  checkPermission: () => false,
});

export const PermissionsProvider = ({ children, token }) => {
  const [permissions, setPermissions] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchPermissions = async () => {
      if (!token) {
        setPermissions([]);
        setLoading(false);
        return;
      }
      
      try {
        const res = await fetch('/api/auth/permissions', {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });
        if (res.ok) {
          const data = await res.json();
          setPermissions(data.permissions || []);
        } else {
          setPermissions([]);
        }
      } catch (err) {
        console.error('Failed to fetch permissions', err);
        setPermissions([]);
      } finally {
        setLoading(false);
      }
    };

    fetchPermissions();
  }, [token]);

  const checkPermission = (requiredPermission) => {
    return permissions.includes(requiredPermission) || permissions.includes('admin');
  };

  return (
    <PermissionsContext.Provider value={{ permissions, loading, checkPermission }}>
      {children}
    </PermissionsContext.Provider>
  );
};

export const usePermissions = () => useContext(PermissionsContext);

export const PermissionGate = ({ requiredPermission, children, fallback = null }) => {
  const { checkPermission, loading } = usePermissions();
  
  if (loading) return null;
  
  if (checkPermission(requiredPermission)) {
    return <>{children}</>;
  }
  
  return fallback ? <>{fallback}</> : null;
};
