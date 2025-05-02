const bcrypt = require('bcryptjs');
const { config } = require('../config/evilginx');

/**
 * Simple user model for dashboard authentication
 * In a production environment, you would want to use a database
 */
class User {
  constructor() {
    // In-memory user store (in a real app, this would be a database)
    this.users = [{
      id: 1,
      username: config.defaultAdmin.username,
      password: this.hashPassword(config.defaultAdmin.password),
      role: 'admin'
    }];
  }

  /**
   * Hash a password
   */
  hashPassword(password) {
    const salt = bcrypt.genSaltSync(10);
    return bcrypt.hashSync(password, salt);
  }

  /**
   * Verify a password
   */
  verifyPassword(password, hashedPassword) {
    return bcrypt.compareSync(password, hashedPassword);
  }

  /**
   * Find a user by username
   */
  findByUsername(username) {
    return this.users.find(user => user.username === username);
  }

  /**
   * Find a user by ID
   */
  findById(id) {
    return this.users.find(user => user.id === parseInt(id));
  }

  /**
   * Authenticate a user
   */
  authenticate(username, password) {
    const user = this.findByUsername(username);
    
    if (!user) {
      return null;
    }
    
    if (!this.verifyPassword(password, user.password)) {
      return null;
    }
    
    // Return user without password
    const { password: _, ...userWithoutPassword } = user;
    return userWithoutPassword;
  }

  /**
   * Create a new user (admin only)
   */
  createUser(userData) {
    const newUser = {
      id: this.users.length + 1,
      username: userData.username,
      password: this.hashPassword(userData.password),
      role: userData.role || 'user'
    };
    
    this.users.push(newUser);
    
    // Return user without password
    const { password: _, ...userWithoutPassword } = newUser;
    return userWithoutPassword;
  }

  /**
   * Change user password
   */
  changePassword(userId, oldPassword, newPassword) {
    const userIndex = this.users.findIndex(user => user.id === parseInt(userId));
    
    if (userIndex === -1) {
      return false;
    }
    
    const user = this.users[userIndex];
    
    if (!this.verifyPassword(oldPassword, user.password)) {
      return false;
    }
    
    this.users[userIndex].password = this.hashPassword(newPassword);
    return true;
  }
}

module.exports = new User(); 