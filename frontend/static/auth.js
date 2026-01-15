// Auth module - handles login, registration, and session management

const API_BASE = 'http://localhost:8080';

// Tab switching
const tabButtons = document.querySelectorAll('.tab-button');
const tabContents = document.querySelectorAll('.tab-content');

tabButtons.forEach(button => {
  button.addEventListener('click', () => {
    const tabName = button.dataset.tab;

    // Update active states
    tabButtons.forEach(btn => btn.classList.remove('active'));
    tabContents.forEach(content => content.classList.remove('active'));

    button.classList.add('active');
    document.getElementById(`${tabName}-tab`).classList.add('active');

    // Clear error message when switching tabs
    hideError();
  });
});

// Error handling
function showError(message) {
  const errorEl = document.getElementById('error-message');
  errorEl.textContent = message;
  errorEl.classList.add('show');
}

function hideError() {
  const errorEl = document.getElementById('error-message');
  errorEl.classList.remove('show');
}

// Login form handler
document.getElementById('login-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  hideError();

  const email = document.getElementById('login-email').value.trim();
  const password = document.getElementById('login-password').value;

  const submitBtn = document.getElementById('login-submit');
  submitBtn.disabled = true;
  submitBtn.setAttribute('aria-busy', 'true');
  submitBtn.textContent = 'Logging in...';

  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.AuthService/Login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        email,
        password,
      }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Login failed');
    }

    const data = await response.json();

    // Store auth token
    localStorage.setItem('auth_token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));

    // Redirect to main app
    window.location.href = '/index.html';

  } catch (error) {
    console.error('Login error:', error);
    showError(error.message || 'Login failed. Please check your credentials.');
  } finally {
    submitBtn.disabled = false;
    submitBtn.removeAttribute('aria-busy');
    submitBtn.textContent = 'Login';
  }
});

// Register form handler
document.getElementById('register-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  hideError();

  const displayName = document.getElementById('register-name').value.trim();
  const email = document.getElementById('register-email').value.trim();
  const password = document.getElementById('register-password').value;

  // Client-side validation
  if (password.length < 8) {
    showError('Password must be at least 8 characters long');
    return;
  }

  const submitBtn = document.getElementById('register-submit');
  submitBtn.disabled = true;
  submitBtn.setAttribute('aria-busy', 'true');
  submitBtn.textContent = 'Creating account...';

  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.AuthService/Register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        email,
        password,
        display_name: displayName,
      }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Registration failed');
    }

    const data = await response.json();

    // Store auth token
    localStorage.setItem('auth_token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));

    // Redirect to main app
    window.location.href = '/index.html';

  } catch (error) {
    console.error('Registration error:', error);
    showError(error.message || 'Registration failed. Please try again.');
  } finally {
    submitBtn.disabled = false;
    submitBtn.removeAttribute('aria-busy');
    submitBtn.textContent = 'Create Account';
  }
});

// Check if already logged in
const token = localStorage.getItem('auth_token');
if (token) {
  // Verify token is still valid (optional)
  // For now, just redirect to main app
  window.location.href = '/index.html';
}
