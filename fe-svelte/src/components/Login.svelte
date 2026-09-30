<script>
  import { currentUser, showToast } from '../lib/stores.js';
  import { apiLogin } from '../lib/api.js';

  let account = '';
  let password = '';
  let loading = false;

  async function handleSubmit() {
    if (!account.trim() || !password) {
      showToast('Please enter both account and password', 'error');
      return;
    }

    loading = true;
    try {
      const data = await apiLogin(account.trim(), password);
      $currentUser = data;
      showToast(`Welcome back, ${data.name || data.account}!`, 'success');
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-wrapper">
  <div class="login-card">
    <div class="brand-header">
      <svg class="brand-icon" viewBox="0 0 24 24" width="48" height="48" fill="#1a73e8">
        <path d="M20 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"/>
      </svg>
      <h1>Lightmail</h1>
      <p class="tagline">Sign in to your private mailbox</p>
    </div>

    <form class="login-form" on:submit|preventDefault={handleSubmit}>
      <div class="form-group">
        <label for="login-account">Email or Username</label>
        <input
          id="login-account"
          type="text"
          placeholder="e.g. admin or user@yourdomain.com"
          bind:value={account}
          autocomplete="username"
          required
        />
      </div>

      <div class="form-group">
        <label for="login-pwd">Password</label>
        <input
          id="login-pwd"
          type="password"
          placeholder="Enter your password"
          bind:value={password}
          autocomplete="current-password"
          required
        />
      </div>

      <button type="submit" class="submit-btn" disabled={loading}>
        {loading ? 'Signing In...' : 'Sign In'}
      </button>
    </form>

    <div class="login-footer">
      <small>Lightmail • Secure, Lightweight, Self-Hosted</small>
    </div>
  </div>
</div>

<style>
  .login-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: #f0f4f9;
    padding: 24px;
    box-sizing: border-box;
  }
  .login-card {
    width: 440px;
    max-width: 100%;
    background: #ffffff;
    border-radius: 28px;
    padding: 48px 40px;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
    border: 1px solid #e1e3e7;
    box-sizing: border-box;
  }
  .brand-header {
    text-align: center;
    margin-bottom: 32px;
  }
  .brand-icon {
    margin-bottom: 12px;
  }
  .brand-header h1 {
    margin: 0;
    font-size: 26px;
    font-weight: 600;
    color: #1f1f1f;
  }
  .tagline {
    margin: 6px 0 0;
    font-size: 14px;
    color: #5f6368;
  }
  .login-form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .form-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .form-group label {
    font-size: 13px;
    font-weight: 600;
    color: #444746;
  }
  .form-group input {
    padding: 12px 16px;
    border: 1px solid #dadce0;
    border-radius: 8px;
    font-size: 15px;
    outline: none;
    color: #1f1f1f;
  }
  .form-group input:focus {
    border-color: #1a73e8;
    box-shadow: 0 0 0 2px rgba(26, 115, 232, 0.2);
  }
  .submit-btn {
    background: #0b57d0;
    color: #ffffff;
    border: none;
    border-radius: 24px;
    height: 48px;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s, box-shadow 0.15s;
    margin-top: 10px;
  }
  .submit-btn:hover:not(:disabled) {
    background: #0842a0;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }
  .submit-btn:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .login-footer {
    text-align: center;
    margin-top: 32px;
    color: #747775;
  }
</style>
