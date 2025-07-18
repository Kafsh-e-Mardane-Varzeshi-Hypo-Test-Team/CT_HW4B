// Authentication JavaScript

function showTab(tabName) {
    // Hide all forms
    const forms = document.querySelectorAll('.auth-form');
    forms.forEach(form => form.classList.remove('active'));
    
    // Remove active class from all tabs
    const tabs = document.querySelectorAll('.tab-btn');
    tabs.forEach(tab => tab.classList.remove('active'));
    
    // Show selected form and activate tab
    document.getElementById(tabName).classList.add('active');
    event.target.classList.add('active');
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
        
        const result = await response.json();
        
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
        alert('An error occurred during signup');
    }
});

// Check if user is already logged in
document.addEventListener('DOMContentLoaded', function() {
    const user = localStorage.getItem('user');
    if (user) {
        // User is already logged in, redirect to dashboard
        window.location.href = '/dashboard';
    }
}); 