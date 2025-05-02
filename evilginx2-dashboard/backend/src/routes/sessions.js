const express = require('express');
const router = express.Router();
const { getSessions, getSession, filterSessions, getSessionStats } = require('../controllers/sessionsController');
const { auth } = require('../middleware/auth');

// Apply auth middleware to all routes
router.use(auth);

// @route   GET /api/sessions
// @desc    Get all sessions
// @access  Private
router.get('/', getSessions);

// @route   GET /api/sessions/stats
// @desc    Get sessions statistics
// @access  Private
router.get('/stats', getSessionStats);

// @route   POST /api/sessions/filter
// @desc    Filter sessions
// @access  Private
router.post('/filter', filterSessions);

// @route   GET /api/sessions/:id
// @desc    Get single session
// @access  Private
router.get('/:id', getSession);

module.exports = router; 