const express = require('express');
const router = express.Router();
const { getLogs, filterLogs, getSessionLogs, getCredentials } = require('../controllers/logsController');
const { auth } = require('../middleware/auth');

// Apply auth middleware to all routes
router.use(auth);

// @route   GET /api/logs
// @desc    Get all logs
// @access  Private
router.get('/', getLogs);

// @route   POST /api/logs/filter
// @desc    Filter logs
// @access  Private
router.post('/filter', filterLogs);

// @route   GET /api/logs/credentials
// @desc    Get all credentials
// @access  Private
router.get('/credentials', getCredentials);

// @route   GET /api/logs/session/:id
// @desc    Get logs for a specific session
// @access  Private
router.get('/session/:id', getSessionLogs);

module.exports = router; 