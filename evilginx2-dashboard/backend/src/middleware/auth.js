const jwt = require('jsonwebtoken');
const { config } = require('../config/evilginx');
const User = require('../models/User');

/**
 * Authentication middleware to protect routes
 */
const auth = (req, res, next) => {
  // Get token from header
  const token = req.header('x-auth-token');

  // Check if no token
  if (!token) {
    return res.status(401).json({ 
      success: false, 
      message: 'No authentication token, access denied' 
    });
  }

  try {
    // Verify token
    const decoded = jwt.verify(token, config.jwtSecret);
    
    // Add user from payload
    const user = User.findById(decoded.id);
    
    if (!user) {
      return res.status(401).json({ 
        success: false, 
        message: 'User no longer exists' 
      });
    }
    
    req.user = user;
    next();
  } catch (err) {
    res.status(401).json({ 
      success: false, 
      message: 'Token is not valid' 
    });
  }
};

/**
 * Admin authorization middleware
 */
const adminOnly = (req, res, next) => {
  if (!req.user || req.user.role !== 'admin') {
    return res.status(403).json({ 
      success: false, 
      message: 'Admin access required' 
    });
  }
  next();
};

module.exports = {
  auth,
  adminOnly
}; 