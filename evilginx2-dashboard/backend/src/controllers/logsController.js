const Log = require('../models/Log');

/**
 * @desc    Get all logs
 * @route   GET /api/logs
 * @access  Private
 */
const getLogs = async (req, res) => {
  try {
    const logs = await Log.getSessionLogs();
    
    res.json({
      success: true,
      count: logs.length,
      data: logs
    });
  } catch (error) {
    console.error('Get logs error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Filter logs
 * @route   POST /api/logs/filter
 * @access  Private
 */
const filterLogs = async (req, res) => {
  try {
    const { sessionId, startDate, endDate } = req.body;
    
    const filteredLogs = await Log.filterLogs({
      sessionId,
      startDate,
      endDate
    });
    
    res.json({
      success: true,
      count: filteredLogs.length,
      data: filteredLogs
    });
  } catch (error) {
    console.error('Filter logs error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Get session logs
 * @route   GET /api/logs/session/:id
 * @access  Private
 */
const getSessionLogs = async (req, res) => {
  try {
    const sessionId = req.params.id;
    
    const logs = await Log.filterLogs({ sessionId });
    
    res.json({
      success: true,
      count: logs.length,
      data: logs
    });
  } catch (error) {
    console.error(`Get logs for session ${req.params.id} error:`, error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Get all credentials
 * @route   GET /api/logs/credentials
 * @access  Private
 */
const getCredentials = async (req, res) => {
  try {
    const credentials = await Log.getCredentials();
    
    res.json({
      success: true,
      count: credentials.length,
      data: credentials
    });
  } catch (error) {
    console.error('Get credentials error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

module.exports = {
  getLogs,
  filterLogs,
  getSessionLogs,
  getCredentials
}; 