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
            loginButton.textContent = 'جاري التحقق...';
            loginButton.disabled = true;
            errorMessage.textContent = '';
            errorMessage.style.display = 'none';
            
            // التحقق من القيمة
            const token = tokenInput.value.trim();
            
            if (!token) {
                showError('يرجى إدخال توكن صالح');
                resetButton();
                return;
            }
            
            try {
                console.log('إرسال طلب تحقق للتوكن:', token.substring(0, 5) + '...');
                
                // إرسال طلب API
                const response = await fetch('/auth/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ userToken: token })
                });
                
                console.log('استجابة التحقق من التوكن:', response.status);
                
                // معالجة الاستجابة
                if (!response.ok) {
                    throw new Error(`فشل التحقق من التوكن (${response.status})`);
                }
                
                const data = await response.json();
                console.log('تم تلقي بيانات استجابة:', data.success ? 'نجاح' : 'فشل');
                
                if (data.success && data.data && data.data.auth_token) {
                    // حفظ التوكن
                    const authToken = data.data.auth_token;
                    console.log('تم استلام توكن صالح:', authToken.substring(0, 5) + '...');
                    
                    // حفظ قيم التوكن
                    localStorage.setItem('userToken', token);
                    localStorage.setItem('authToken', authToken);
                    
                    // إنشاء كوكي بنفس القيمة (احتياطي)
                    setCookie('Authorization', authToken, 1); // صالح لمدة يوم واحد
                    
                    console.log('تم تخزين التوكن وإعادة التوجيه إلى لوحة التحكم');
                    
                    // إعادة التوجيه إلى اللوحة
                    window.location.href = '/static/dashboard.html';
                } else {
                    throw new Error('استجابة غير صالحة من الخادم');
                }
            } catch (error) {
                console.error('خطأ في تسجيل الدخول:', error);
                showError('فشل التحقق: ' + error.message);
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
        loginButton.textContent = 'تسجيل الدخول';
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
                console.log('توكن سابق صالح، إعادة التوجيه إلى لوحة التحكم');
                window.location.href = '/static/dashboard.html';
            } else {
                console.log('توكن سابق غير صالح، البقاء على صفحة تسجيل الدخول');
                // مسح التوكنات غير الصالحة
                localStorage.removeItem('authToken');
                deleteCookie('Authorization');
            }
        } catch (error) {
            console.error('خطأ أثناء التحقق من جلسة سابقة:', error);
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
    
    console.log('تعيين الكوكي:', name, 'بالقيمة:', value.substring(0, 5) + '...', 'على النطاق:', domain || 'افتراضي');
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