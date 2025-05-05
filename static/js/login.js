document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('login-form');
    const usernameInput = document.getElementById('username');
    const passwordInput = document.getElementById('password');
    const loginButton = document.getElementById('login-button');
    const errorMessage = document.getElementById('error-message');
    
    // Auto focus on username field
    usernameInput.focus();
    
    // Handle login form submission
    loginForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // Validate username and password input
        const username = usernameInput.value.trim();
        const password = passwordInput.value.trim();
        
        if (!username || !password) {
            showError('Please enter username and password');
            return;
        }
        
        // Disable login button and show loading state
        loginButton.disabled = true;
        loginButton.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Logging in...';
        
        try {
            // Send login request to API
            const response = await fetch('/api/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, password })
            });
            
            const data = await response.json();
            console.log('Login response:', data);
            
            if (data.success) {
                // Store token in local storage
                localStorage.setItem('authToken', data.auth_token);
                
                // Redirect user to dashboard
                window.location.href = '/static/dashboard.html';
            } else {
                // Show error message
                showError(data.message || 'Login failed. Please check your credentials');
                
                // Re-enable login button
                resetLoginButton();
            }
        } catch (error) {
            console.error('Login error:', error);
            showError('An error occurred while connecting to the server. Please try again');
            resetLoginButton();
        }
    });
    
    // Show error message
    function showError(message) {
        errorMessage.textContent = message;
        errorMessage.style.display = 'block';
        
        // Shake input fields to alert user
        usernameInput.classList.add('shake');
        passwordInput.classList.add('shake');
        
        // Remove shake effect after animation completes
        setTimeout(() => {
            usernameInput.classList.remove('shake');
            passwordInput.classList.remove('shake');
        }, 500);
    }
    
    // Reset login button
    function resetLoginButton() {
        loginButton.disabled = false;
        loginButton.innerHTML = 'Login';
    }
    
    // Check if the user is already logged in
    const authToken = localStorage.getItem('authToken');
    if (authToken) {
        // If user is already logged in, redirect to dashboard
        window.location.href = '/static/dashboard.html';
    }
}); 