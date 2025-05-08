document.addEventListener('DOMContentLoaded', function() {
    const loginForm = document.getElementById('login-form');
    const userTokenInput = document.getElementById('userToken');
    const loginButton = document.getElementById('login-button');
    const errorMessage = document.getElementById('error-message');
    
    // Auto focus on token field
    userTokenInput.focus();
    
    // Handle login form submission
    loginForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // Validate token input
        const userToken = userTokenInput.value.trim();
        
        if (!userToken) {
            showError('Please enter your access token');
            return;
        }
        
        // Disable login button and show loading state
        loginButton.disabled = true;
        loginButton.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Verifying...';
        
        try {
            // Send token verification request to API
            const response = await fetch('/auth/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ userToken })
            });
            
            if (!response.ok) {
                throw new Error('Token verification failed');
            }
            
            const data = await response.json();
            console.log('Verification response:', data);
            
            if (data.success) {
                // Store tokens in local storage
                localStorage.setItem('userToken', userToken);
                if (data.data && data.data.auth_token) {
                    localStorage.setItem('authToken', data.data.auth_token);
                }
                
                console.log('Authentication successful, redirecting to dashboard...');
                // Redirect user to dashboard
                window.location.href = '/static/dashboard.html';
            } else {
                // Show error message
                showError(data.message || 'Verification failed. Please check your token');
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
        
        // Shake input field to alert user
        userTokenInput.classList.add('shake');
        
        // Remove shake effect after animation completes
        setTimeout(() => {
            userTokenInput.classList.remove('shake');
        }, 500);
    }
    
    // Reset login button
    function resetLoginButton() {
        loginButton.disabled = false;
        loginButton.innerHTML = '<i class="fas fa-sign-in-alt"></i> <span>Access Dashboard</span>';
    }
    
    // Check if the user is already logged in
    const authToken = localStorage.getItem('authToken');
    if (authToken) {
        // If user is already logged in, redirect to dashboard
        window.location.href = '/static/dashboard.html';
    }
}); 