// Authentication JavaScript

function showTab(tabName, event = null) {
    // Hide all forms
    const forms = document.querySelectorAll('.auth-form');
    forms.forEach(form => form.classList.remove('active'));
    
    // Remove active class from all tabs
    const tabs = document.querySelectorAll('.tab-btn');
    tabs.forEach(tab => tab.classList.remove('active'));
    
    // Show selected form and activate tab
    document.getElementById(tabName).classList.add('active');
    
    // If event is provided, activate the clicked tab
    if (event && event.target) {
        event.target.classList.add('active');
    } else {
        // Find and activate the tab button for the selected form
        const tabButtons = document.querySelectorAll('.tab-btn');
        tabButtons.forEach(btn => {
            if (btn.textContent.toLowerCase().includes(tabName.toLowerCase())) {
                btn.classList.add('active');
            }
        });
    }
}

// Login form submission
document.getElementById('loginForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const formData = new FormData(this);
    const data = {
        username: formData.get('username'),
        password: formData.get('password')
    };
    
    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        if (response.ok) {
            // Store user data in localStorage
            localStorage.setItem('user', JSON.stringify(result.user));
            localStorage.setItem('token', result.user.id); // Using user ID as simple token
            
            // Redirect to dashboard
            window.location.href = '/dashboard';
        } else {
            alert(result.error || 'Login failed');
        }
    } catch (error) {
        console.error('Login error:', error);
        alert('An error occurred during login');
    }
});

// Signup form submission
document.getElementById('signupForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const formData = new FormData(this);
    const data = {
        username: formData.get('username'),
        password: formData.get('password')
    };
    
    try {
        const response = await fetch('/api/signup', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });
        
        console.log('Signup response status:', response.status);
        console.log('Signup response headers:', response.headers);
        
        let result;
        try {
            result = await response.json();
            console.log('Signup response data:', result);
        } catch (jsonError) {
            console.error('JSON parsing error:', jsonError);
            const text = await response.text();
            console.log('Response text:', text);
            throw jsonError;
        }
        
        if (response.ok) {
            alert('Account created successfully! Please login.');
            // Switch to login tab
            showTab('login');
            // Clear signup form
            this.reset();
        } else {
            alert(result.error || 'Signup failed');
        }
    } catch (error) {
        console.error('Signup error:', error);
        console.error('Error name:', error.name);
        console.error('Error message:', error.message);
        console.error('Error stack:', error.stack);
        alert('An error occurred during signup');
    }
});

// Check if user is already logged in
document.addEventListener('DOMContentLoaded', async function() {
    const user = localStorage.getItem('user');
    if (user) {
        try {
            // Validate user session with server
            const userData = JSON.parse(user);
            const response = await fetch('/api/validate-session', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-User-ID': userData.id
                },
                body: JSON.stringify({ user_id: userData.id })
            });
            
            if (response.ok) {
                // User session is valid, redirect to dashboard
                window.location.href = '/dashboard';
            } else {
                // User session is invalid, clear localStorage
                localStorage.removeItem('user');
                localStorage.removeItem('token');
                console.log('Session expired, please login again');
            }
        } catch (error) {
            console.error('Session validation error:', error);
            // Clear localStorage on error
            localStorage.removeItem('user');
            localStorage.removeItem('token');
        }
    }
}); 