console.log('تم تحميل صفحة تسجيل الدخول');

document.addEventListener('DOMContentLoaded', function() {
    // تحقق إذا كان المستخدم قد سجل الدخول بالفعل
    const storedAuthToken = localStorage.getItem('authToken');
    if (storedAuthToken) {
        console.log('تم العثور على توكن مصادقة مخزن، التحقق من صلاحيته...');
        checkExistingLogin(storedAuthToken);
    }

    const form = document.getElementById('login-form');
    const tokenInput = document.getElementById('token');
    const errorMessage = document.getElementById('error-message');
    const loginButton = document.getElementById('login-button');

    // التركيز على حقل الإدخال
    if (tokenInput) {
        tokenInput.focus();
    }

    // معالج تقديم النموذج
    if (form) {
        form.addEventListener('submit', async function(e) {
            e.preventDefault();
            
            // تجهيز نمط التحميل للزر
            loginButton.classList.add('loading');
            loginButton.textContent = 'loading...';
            loginButton.disabled = true;
            errorMessage.textContent = '';
            errorMessage.style.display = 'none';
            
            // التحقق من القيمة
            const token = tokenInput.value.trim();
            
            if (!token) {
                showError('Please enter a valid token');
                resetButton();
                return;
            }
            
            try {
                console.log('Sending token verification request:', token.substring(0, 5) + '...');
                
                // إرسال طلب API
                const response = await fetch('/auth/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ userToken: token })
                });
                
                console.log('Token verification response:', response.status);
                
                // معالجة الاستجابة
                if (!response.ok) {
                    throw new Error(`Token verification failed (${response.status})`);
                }
                
                const data = await response.json();
                console.log('Received response data:', data.success ? 'success' : 'failure');
                
                if (data.success && data.data && data.data.auth_token) {
                    // حفظ التوكن
                    const authToken = data.data.auth_token;
                        console.log('Received valid token:', authToken.substring(0, 5) + '...');
                    
                    // حفظ قيم التوكن بطرق متعددة لضمان الوصول
                    localStorage.setItem('userToken', token);
                    localStorage.setItem('authToken', authToken);
                    
                    // تعيين كوكي صريح (مباشر) مع إعدادات متساهلة
                    document.cookie = `Authorization=${authToken}; path=/; max-age=86400; SameSite=Lax`;
                    
                    // إضافة تأخير قصير قبل التوجيه للتأكد من حفظ البيانات
                    console.log('Token stored, redirecting to dashboard after short delay...');
                    
                    setTimeout(() => {
                        // محاولة تعيين كوكي مرة أخرى
                        document.cookie = `Authorization=${authToken}; path=/; max-age=86400; SameSite=Lax`;
                        // التوجيه للمسار المباشر
                        window.location.href = '/dashboard';
                    }, 500);
                    
                    return;
                } else {
                    throw new Error('Invalid server response');
                }
            } catch (error) {
                console.error('Login error:', error);
                showError('Login failed: ' + error.message);
                resetButton();
            }
        });
    }
    
    // دالة لإظهار رسالة الخطأ
    function showError(message) {
        errorMessage.textContent = message;
        errorMessage.style.display = 'block';
    }
    
    // دالة لإعادة تعيين حالة زر تسجيل الدخول
    function resetButton() {
        loginButton.classList.remove('loading');
        loginButton.textContent = 'Login';
        loginButton.disabled = false;
    }
    
    // التحقق من جلسة موجودة بالفعل
    async function checkExistingLogin(token) {
        try {
            // إرسال طلب بسيط مع التوكن للتحقق من صلاحيته
            const response = await fetch('/api/dashboard', {
                method: 'GET',
                headers: {
                    'Authorization': token
                }
            });
            
            if (response.ok) {
                console.log('Previous valid token, redirecting to dashboard');
                window.location.href = '/static/dashboard.html';
            } else {
                console.log('Previous invalid token, staying on login page');
                // مسح التوكنات غير الصالحة
                localStorage.removeItem('authToken');
                deleteCookie('Authorization');
            }
        } catch (error) {
            console.error('Error verifying previous session:', error);
        }
    }
});

// دالة مساعدة لتعيين كوكي
function setCookie(name, value, days) {
    let expires = '';
    if (days) {
        const date = new Date();
        date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
        expires = '; expires=' + date.toUTCString();
    }
    
    // تعيين نطاق الكوكي على نطاق أوسع للمشاركة بين النطاقات الفرعية
    const hostParts = window.location.hostname.split('.');
    let domain = '';
    
    if (hostParts.length >= 2) {
        // استخدم النطاق الرئيسي بدلاً من المضيف الكامل
        domain = '; domain=.' + hostParts[hostParts.length - 2] + '.' + hostParts[hostParts.length - 1];
    }
    
    console.log('Setting cookie:', name, 'with value:', value.substring(0, 5) + '...', 'on domain:', domain || 'default');
    document.cookie = name + '=' + value + expires + domain + '; path=/; SameSite=Lax';
}

// دالة مساعدة لحذف كوكي
function deleteCookie(name) {
    document.cookie = name + '=; Max-Age=-99999999; path=/';
}

// دالة مساعدة للحصول على قيمة كوكي
function getCookie(name) {
    const nameEQ = name + '=';
    const ca = document.cookie.split(';');
    for (let i = 0; i < ca.length; i++) {
        let c = ca[i];
        while (c.charAt(0) === ' ') c = c.substring(1, c.length);
        if (c.indexOf(nameEQ) === 0) return c.substring(nameEQ.length, c.length);
    }
    return null;
} 