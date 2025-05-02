document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('login-form');
    const usernameInput = document.getElementById('username');
    const passwordInput = document.getElementById('password');
    const loginButton = document.getElementById('login-button');
    const errorMessage = document.getElementById('error-message');
    
    // التركيز التلقائي على حقل اسم المستخدم
    usernameInput.focus();
    
    // معالجة تسجيل الدخول
    loginForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // التحقق من إدخال اسم المستخدم وكلمة المرور
        const username = usernameInput.value.trim();
        const password = passwordInput.value.trim();
        
        if (!username || !password) {
            showError('يرجى إدخال اسم المستخدم وكلمة المرور');
            return;
        }
        
        // تعطيل زر تسجيل الدخول وإظهار حالة التحميل
        loginButton.disabled = true;
        loginButton.innerHTML = '<i class="fas fa-spinner fa-spin"></i> جاري تسجيل الدخول...';
        
        try {
            // إرسال طلب تسجيل الدخول إلى API
            const response = await fetch('/api/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, password })
            });
            
            const data = await response.json();
            console.log('استجابة تسجيل الدخول:', data);
            
            if (data.success) {
                // تخزين الـ token في التخزين المحلي
                localStorage.setItem('authToken', data.auth_token);
                
                // توجيه المستخدم إلى صفحة لوحة التحكم
                window.location.href = '/static/dashboard.html';
            } else {
                // إظهار رسالة الخطأ
                showError(data.message || 'فشل تسجيل الدخول. يرجى التحقق من بيانات الاعتماد');
                
                // إعادة تفعيل زر تسجيل الدخول
                resetLoginButton();
            }
        } catch (error) {
            console.error('Login error:', error);
            showError('حدث خطأ أثناء الاتصال بالخادم. يرجى المحاولة مرة أخرى');
            resetLoginButton();
        }
    });
    
    // إظهار رسالة الخطأ
    function showError(message) {
        errorMessage.textContent = message;
        errorMessage.style.display = 'block';
        
        // هز حقول الإدخال لتنبيه المستخدم
        usernameInput.classList.add('shake');
        passwordInput.classList.add('shake');
        
        // إزالة تأثير الهز بعد انتهاء الرسوم المتحركة
        setTimeout(() => {
            usernameInput.classList.remove('shake');
            passwordInput.classList.remove('shake');
        }, 500);
    }
    
    // إعادة تعيين زر تسجيل الدخول
    function resetLoginButton() {
        loginButton.disabled = false;
        loginButton.innerHTML = 'تسجيل الدخول';
    }
    
    // التحقق مما إذا كان المستخدم مسجل الدخول بالفعل
    const authToken = localStorage.getItem('authToken');
    if (authToken) {
        // إذا كان المستخدم مسجل الدخول بالفعل، انقله إلى لوحة التحكم
        window.location.href = '/static/dashboard.html';
    }
}); 