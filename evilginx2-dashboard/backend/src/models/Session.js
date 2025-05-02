const fs = require('fs');
const path = require('path');
const { config } = require('../config/evilginx');
const { promisify } = require('util');

// Use BuntDB to read Evilginx2 database
// Note: In a production environment, you might want to use a more robust database connector
// or directly interact with the Evilginx2 API if available

class Session {
  constructor() {
    this.dbPath = config.dbPath;
  }

  /**
   * Safely read the Evilginx2 database file
   * This is a simple implementation - in a real-world scenario,
   * you would use proper database connection/libraries
   */
  async readDatabase() {
    try {
      const readFile = promisify(fs.readFile);
      const data = await readFile(this.dbPath, 'utf8');
      
      // Mock parsing - actual implementation would need proper parsing
      // of the BuntDB format used by Evilginx2
      // This is placeholder implementation
      return JSON.parse(data);
    } catch (error) {
      console.error('Error reading Evilginx2 database:', error);
      
      // Return mock data for development purposes
      return this.getMockSessions();
    }
  }

  /**
   * Get all sessions
   */
  async getSessions() {
    try {
      // In a real implementation, this would properly read the BuntDB database
      // For now, we'll use mock data
      return this.getMockSessions();
    } catch (error) {
      console.error('Error fetching sessions:', error);
      throw new Error('Failed to fetch sessions');
    }
  }

  /**
   * Get session by ID
   */
  async getSessionById(id) {
    try {
      const sessions = await this.getSessions();
      return sessions.find(session => session.id === parseInt(id));
    } catch (error) {
      console.error(`Error fetching session ${id}:`, error);
      throw new Error(`Failed to fetch session ${id}`);
    }
  }

  /**
   * Filter sessions by criteria
   */
  async filterSessions(criteria = {}) {
    try {
      const sessions = await this.getSessions();
      
      return sessions.filter(session => {
        // Filter by phishlet/domain if specified
        if (criteria.domain && session.phishlet !== criteria.domain) {
          return false;
        }
        
        // Filter by date range if specified
        if (criteria.startDate && session.createTime < new Date(criteria.startDate).getTime()/1000) {
          return false;
        }
        
        if (criteria.endDate && session.createTime > new Date(criteria.endDate).getTime()/1000) {
          return false;
        }
        
        return true;
      });
    } catch (error) {
      console.error('Error filtering sessions:', error);
      throw new Error('Failed to filter sessions');
    }
  }

  /**
   * Generate mock session data for development
   */
  getMockSessions() {
    const now = Math.floor(Date.now() / 1000);
    return [
      {
        id: 1,
        phishlet: 'gmail',
        landing_url: 'https://gmail.com/signin',
        username: 'victim@gmail.com',
        password: 'password123',
        custom: {},
        tokens: {
          'session': 'ABC123',
          'auth': 'XYZ456'
        },
        session_id: 'sess_1',
        useragent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36',
        remote_addr: '192.168.1.101',
        create_time: now - 3600,
        update_time: now - 1800
      },
      {
        id: 2,
        phishlet: 'facebook',
        landing_url: 'https://facebook.com/login',
        username: 'victim@example.com',
        password: 'fb_password',
        custom: {},
        tokens: {
          'session': 'FB123',
          'auth': 'FB456'
        },
        session_id: 'sess_2',
        useragent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1',
        remote_addr: '192.168.1.102',
        create_time: now - 7200,
        update_time: now - 3600
      },
      {
        id: 3,
        phishlet: 'linkedin',
        landing_url: 'https://linkedin.com/login',
        username: 'professional@example.com',
        password: 'linkedin_pass',
        custom: {},
        tokens: {
          'session': 'LI123',
          'auth': 'LI456'
        },
        session_id: 'sess_3',
        useragent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36',
        remote_addr: '192.168.1.103',
        create_time: now - 86400,
        update_time: now - 43200
      }
    ];
  }
}

module.exports = new Session(); 