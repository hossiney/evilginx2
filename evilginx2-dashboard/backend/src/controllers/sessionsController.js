const Session = require('../models/Session');

/**
 * @desc    Get all sessions
 * @route   GET /api/sessions
 * @access  Private
 */
const getSessions = async (req, res) => {
  try {
    const sessions = await Session.getSessions();
    
    res.json({
      success: true,
      count: sessions.length,
      data: sessions
    });
  } catch (error) {
    console.error('Get sessions error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Get single session
 * @route   GET /api/sessions/:id
 * @access  Private
 */
const getSession = async (req, res) => {
  try {
    const session = await Session.getSessionById(req.params.id);
    
    if (!session) {
      return res.status(404).json({ 
        success: false, 
        message: 'Session not found' 
      });
    }
    
    res.json({
      success: true,
      data: session
    });
  } catch (error) {
    console.error(`Get session ${req.params.id} error:`, error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Filter sessions
 * @route   POST /api/sessions/filter
 * @access  Private
 */
const filterSessions = async (req, res) => {
  try {
    const { domain, startDate, endDate } = req.body;
    
    const filteredSessions = await Session.filterSessions({
      domain,
      startDate,
      endDate
    });
    
    res.json({
      success: true,
      count: filteredSessions.length,
      data: filteredSessions
    });
  } catch (error) {
    console.error('Filter sessions error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * @desc    Get session statistics
 * @route   GET /api/sessions/stats
 * @access  Private
 */
const getSessionStats = async (req, res) => {
  try {
    const sessions = await Session.getSessions();
    
    // Calculate stats
    const stats = {
      totalSessions: sessions.length,
      
      // Count sessions by phishlet/domain
      byDomain: sessions.reduce((acc, session) => {
        const domain = session.phishlet;
        acc[domain] = (acc[domain] || 0) + 1;
        return acc;
      }, {}),
      
      // Count sessions with credentials captured
      withCredentials: sessions.filter(s => s.username && s.password).length,
      
      // Count sessions by day (last 7 days)
      byDay: getSessionsByDay(sessions)
    };
    
    res.json({
      success: true,
      data: stats
    });
  } catch (error) {
    console.error('Get session stats error:', error);
    res.status(500).json({ 
      success: false, 
      message: 'Server error' 
    });
  }
};

/**
 * Helper function to group sessions by day (last 7 days)
 */
const getSessionsByDay = (sessions) => {
  const now = new Date();
  const result = {};
  
  // Initialize last 7 days with 0 sessions
  for (let i = 6; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i);
    const dateStr = date.toISOString().split('T')[0];
    result[dateStr] = 0;
  }
  
  // Count sessions by day
  sessions.forEach(session => {
    const date = new Date(session.create_time * 1000);
    const dateStr = date.toISOString().split('T')[0];
    
    // Only include last 7 days
    if (result[dateStr] !== undefined) {
      result[dateStr]++;
    }
  });
  
  return result;
};

module.exports = {
  getSessions,
  getSession,
  filterSessions,
  getSessionStats
}; 