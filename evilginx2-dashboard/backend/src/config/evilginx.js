const path = require('path');
const fs = require('fs');

// Configuration for Evilginx2 data access
// Update these paths to match your Evilginx2 installation
const config = {
  // Default path to Evilginx2 database file (adjust as needed)
  dbPath: process.env.EVILGINX_DB_PATH || path.join(process.env.HOME || process.env.USERPROFILE, '.evilginx/data.db'),
  
  // Path to Evilginx2 logs directory (adjust as needed)
  logsPath: process.env.EVILGINX_LOGS_PATH || path.join(process.env.HOME || process.env.USERPROFILE, '.evilginx/logs'),
  
  // Dashboard authentication config
  jwtSecret: process.env.JWT_SECRET || 'evilginx2_dashboard_secret_key',
  jwtExpire: process.env.JWT_EXPIRE || '24h',
  
  // Default dashboard admin credentials
  defaultAdmin: {
    username: process.env.ADMIN_USERNAME || 'admin',
    password: process.env.ADMIN_PASSWORD || 'admin123',
  }
};

// Verify that the DB file exists
const verifyConfig = () => {
  try {
    if (!fs.existsSync(config.dbPath)) {
      console.warn(`Warning: Evilginx2 database file not found at ${config.dbPath}`);
      console.warn('Update the EVILGINX_DB_PATH environment variable or check your Evilginx2 installation');
    }
    
    if (!fs.existsSync(config.logsPath)) {
      console.warn(`Warning: Evilginx2 logs directory not found at ${config.logsPath}`);
      console.warn('Update the EVILGINX_LOGS_PATH environment variable or check your Evilginx2 installation');
    }
  } catch (err) {
    console.error('Error verifying Evilginx2 configuration:', err);
  }
};

module.exports = {
  config,
  verifyConfig
}; 