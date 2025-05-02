const fs = require('fs');
const path = require('path');
const { config } = require('../config/evilginx');
const { promisify } = require('util');

class Log {
  constructor() {
    this.logsPath = config.logsPath;
  }

  /**
   * Get all log files from the Evilginx2 logs directory
   */
  async getLogFiles() {
    try {
      const readdir = promisify(fs.readdir);
      const files = await readdir(this.logsPath);
      return files.filter(file => file.endsWith('.log') || file.endsWith('.creds'));
    } catch (error) {
      console.error('Error reading log directory:', error);
      return [];
    }
  }

  /**
   * Read a specific log file
   */
  async readLogFile(filename) {
    try {
      const readFile = promisify(fs.readFile);
      const filePath = path.join(this.logsPath, filename);
      const data = await readFile(filePath, 'utf8');
      return data;
    } catch (error) {
      console.error(`Error reading log file ${filename}:`, error);
      throw new Error(`Failed to read log file ${filename}`);
    }
  }

  /**
   * Parse credential logs from .creds files
   */
  async getCredentials() {
    try {
      const logFiles = await this.getLogFiles();
      const credFiles = logFiles.filter(file => file.endsWith('.creds'));
      
      let allCredentials = [];
      
      for (const file of credFiles) {
        const content = await this.readLogFile(file);
        const credentials = this.parseCredentialsFile(content, file);
        allCredentials = [...allCredentials, ...credentials];
      }
      
      return allCredentials;
    } catch (error) {
      console.error('Error getting credentials:', error);
      
      // Return mock data for development
      return this.getMockCredentials();
    }
  }

  /**
   * Parse a credentials file content
   */
  parseCredentialsFile(content, filename) {
    try {
      // Extract phishlet name from filename (e.g., "gmail.creds" -> "gmail")
      const phishlet = path.basename(filename, '.creds');
      
      // Parse the file content
      // This is a simple implementation - the actual format may vary
      const lines = content.split('\n').filter(line => line.trim());
      
      return lines.map(line => {
        const parts = line.split(':');
        if (parts.length >= 2) {
          const timestamp = parts[0].trim();
          const credentials = parts.slice(1).join(':').trim();
          
          return {
            timestamp,
            phishlet,
            credentials,
            raw: line
          };
        }
        return null;
      }).filter(Boolean);
    } catch (error) {
      console.error(`Error parsing credentials file ${filename}:`, error);
      return [];
    }
  }

  /**
   * Get session logs from .log files
   */
  async getSessionLogs() {
    try {
      const logFiles = await this.getLogFiles();
      const sessionFiles = logFiles.filter(file => file.endsWith('.log'));
      
      let allLogs = [];
      
      for (const file of sessionFiles) {
        const content = await this.readLogFile(file);
        const logs = this.parseLogFile(content, file);
        allLogs = [...allLogs, ...logs];
      }
      
      return allLogs;
    } catch (error) {
      console.error('Error getting session logs:', error);
      
      // Return mock data for development
      return this.getMockLogs();
    }
  }

  /**
   * Parse a log file content
   */
  parseLogFile(content, filename) {
    try {
      // Extract session ID from filename (e.g., "session_abc123.log" -> "abc123")
      const sessionId = path.basename(filename, '.log').replace('session_', '');
      
      // Parse the file content
      // This is a simple implementation - the actual format may vary
      const lines = content.split('\n').filter(line => line.trim());
      
      return lines.map(line => {
        // Simple parsing assuming format: [timestamp] [level] message
        const match = line.match(/^\[(.*?)\] \[(.*?)\] (.*)$/);
        if (match) {
          const [, timestamp, level, message] = match;
          return {
            timestamp,
            level,
            message,
            sessionId,
            raw: line
          };
        }
        return {
          sessionId,
          raw: line
        };
      });
    } catch (error) {
      console.error(`Error parsing log file ${filename}:`, error);
      return [];
    }
  }

  /**
   * Filter logs by criteria
   */
  async filterLogs(criteria = {}) {
    try {
      const logs = await this.getSessionLogs();
      
      return logs.filter(log => {
        // Filter by session ID if specified
        if (criteria.sessionId && log.sessionId !== criteria.sessionId) {
          return false;
        }
        
        // Filter by date range if specified
        if (criteria.startDate) {
          const logDate = new Date(log.timestamp);
          const startDate = new Date(criteria.startDate);
          if (logDate < startDate) return false;
        }
        
        if (criteria.endDate) {
          const logDate = new Date(log.timestamp);
          const endDate = new Date(criteria.endDate);
          if (logDate > endDate) return false;
        }
        
        return true;
      });
    } catch (error) {
      console.error('Error filtering logs:', error);
      throw new Error('Failed to filter logs');
    }
  }

  /**
   * Generate mock log data for development
   */
  getMockLogs() {
    return [
      {
        timestamp: '2023-04-01 10:15:23',
        level: 'inf',
        message: 'New visitor: session_1',
        sessionId: 'sess_1',
        raw: '[2023-04-01 10:15:23] [inf] New visitor: session_1'
      },
      {
        timestamp: '2023-04-01 10:15:45',
        level: 'inf',
        message: 'Captured login attempt for: victim@gmail.com',
        sessionId: 'sess_1',
        raw: '[2023-04-01 10:15:45] [inf] Captured login attempt for: victim@gmail.com'
      },
      {
        timestamp: '2023-04-01 10:16:12',
        level: 'success',
        message: 'Authentication successful',
        sessionId: 'sess_1',
        raw: '[2023-04-01 10:16:12] [success] Authentication successful'
      },
      {
        timestamp: '2023-04-01 11:30:01',
        level: 'inf',
        message: 'New visitor: session_2',
        sessionId: 'sess_2',
        raw: '[2023-04-01 11:30:01] [inf] New visitor: session_2'
      },
      {
        timestamp: '2023-04-01 11:31:20',
        level: 'inf',
        message: 'Captured login attempt for: victim@example.com',
        sessionId: 'sess_2',
        raw: '[2023-04-01 11:31:20] [inf] Captured login attempt for: victim@example.com'
      }
    ];
  }

  /**
   * Generate mock credential data for development
   */
  getMockCredentials() {
    return [
      {
        timestamp: '2023-04-01 10:15:45',
        phishlet: 'gmail',
        credentials: 'victim@gmail.com:password123',
        raw: '2023-04-01 10:15:45: victim@gmail.com:password123'
      },
      {
        timestamp: '2023-04-01 11:31:20',
        phishlet: 'facebook',
        credentials: 'victim@example.com:fb_password',
        raw: '2023-04-01 11:31:20: victim@example.com:fb_password'
      },
      {
        timestamp: '2023-04-02 09:45:12',
        phishlet: 'linkedin',
        credentials: 'professional@example.com:linkedin_pass',
        raw: '2023-04-02 09:45:12: professional@example.com:linkedin_pass'
      }
    ];
  }
}

module.exports = new Log(); 