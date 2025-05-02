const jwt = require('jsonwebtoken');
const { config } = require('../config/evilginx');
const User = require('../models/User');

/**
 * @desc    Authenticate user & get token
 * @route   POST /api/auth/login
 * @access  Public
 */
const login = async (req, res) => {
  try {
    const { username, password } = req.body;
    
    // Validate input
    if (!username || !password) {
      return res.status(400).json({ 
        success: false, 
        message: 'Please provide username and password' 
      });
    }
    
    // Authenticate user
    const user = User.authenticate(username, password);
    
    if (!user) {
      return res.status(401).json({ 
        success: false, 
        message: 'Invalid credentials' 
      });
    }
    
    // Create JWT
    const token = jwt.sign(
      { id: user.id },
      config.jwtSecret,
      { expiresIn: config.jwtExpire }
    );
    
    res.json({
      success: true,
      token,
      user
    });
  } catch (error) {
    console.error('Login error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Get current user
 * @route   GET /api/auth/me
 * @access  Private
 */
const getCurrentUser = async (req, res) => {
  try {
    res.json({
      success: true,
      user: req.user
    });
  } catch (error) {
    console.error('Get current user error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Change password
 * @route   POST /api/auth/change-password
 * @access  Private
 */
const changePassword = async (req, res) => {
  try {
    const { oldPassword, newPassword } = req.body;
    
    // Validate input
    if (!oldPassword || !newPassword) {
      return res.status(400).json({ 
        success: false, 
        message: 'Please provide old and new password' 
      });
    }
    
    // Change password
    const success = User.changePassword(req.user.id, oldPassword, newPassword);
    
    if (!success) {
      return res.status(400).json({ 
        success: false, 
        message: 'Invalid old password' 
      });
    }
    
    res.json({
      success: true,
      message: 'Password changed successfully'
    });
  } catch (error) {
    console.error('Change password error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

module.exports = {
  login,
  getCurrentUser,
  changePassword
}; 