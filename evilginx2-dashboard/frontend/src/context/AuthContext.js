import React, { createContext, useState, useEffect } from 'react';
import axios from 'axios';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('token'));
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Set auth token in axios headers
  if (token) {
    axios.defaults.headers.common['x-auth-token'] = token;
  } else {
    delete axios.defaults.headers.common['x-auth-token'];
  }

  // Load user from token if available
  useEffect(() => {
    const loadUser = async () => {
      if (!token) {
        setLoading(false);
        return;
      }

      try {
        const res = await axios.get('http://5.199.168.182:5000/api/auth/me');
        
        if (res.data.success) {
          setIsAuthenticated(true);
          setUser(res.data.user);
        } else {
          localStorage.removeItem('token');
          setToken(null);
          setIsAuthenticated(false);
          setUser(null);
          setError(res.data.message);
        }
      } catch (err) {
        localStorage.removeItem('token');
        setToken(null);
        setIsAuthenticated(false);
        setUser(null);
        setError(err.response?.data?.message || 'Authentication error');
      }
      
      setLoading(false);
    };

    loadUser();
  }, [token]);

  // Login user
  const login = async (username, password) => {
    try {
      setError(null);
      const res = await axios.post('http://5.199.168.182:5000/api/auth/login', { username, password });
      
      if (res.data.success) {
        localStorage.setItem('token', res.data.token);
        setToken(res.data.token);
        setIsAuthenticated(true);
        setUser(res.data.user);
        return true;
      } else {
        setError(res.data.message);
        return false;
      }
    } catch (err) {
      setError(err.response?.data?.message || 'Login failed');
      return false;
    }
  };

  // Logout user
  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setIsAuthenticated(false);
    setUser(null);
  };

  // Change password
  const changePassword = async (oldPassword, newPassword) => {
    try {
      setError(null);
      const res = await axios.post('http://5.199.168.182:5000/api/auth/change-password', {
        oldPassword,
        newPassword
      });
      
      if (res.data.success) {
        return { success: true, message: res.data.message };
      } else {
        setError(res.data.message);
        return { success: false, message: res.data.message };
      }
    } catch (err) {
      const message = err.response?.data?.message || 'Failed to change password';
      setError(message);
      return { success: false, message };
    }
  };

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        user,
        loading,
        error,
        login,
        logout,
        changePassword
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export default AuthContext; 