const express = require('express');
const router = express.Router();
const { login, getCurrentUser, changePassword } = require('../controllers/authController');
const { auth } = require('../middleware/auth');

// @route   POST /api/auth/login
// @desc    Authenticate user & get token
// @access  Public
router.post('/login', login);

// @route   GET /api/auth/me
// @desc    Get current user
// @access  Private
router.get('/me', auth, getCurrentUser);

// @route   POST /api/auth/change-password
// @desc    Change password
// @access  Private
router.post('/change-password', auth, changePassword);

module.exports = router; 