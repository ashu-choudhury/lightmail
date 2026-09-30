<script>
  import { currentUser, showToast } from '../lib/stores.js';
  import { apiChangePassword } from '../lib/api.js';

  let oldPassword = '';
  let newPassword = '';
  let confirmPassword = '';
  let updating = false;

  async function handleUpdatePassword() {
    if (!oldPassword || !newPassword) {
      showToast('Please fill in current and new password', 'error');
      return;
    }
    if (newPassword !== confirmPassword) {
      showToast('New passwords do not match', 'error');
      return;
    }
    if (newPassword.length < 6) {
      showToast('New password should be at least 6 characters long', 'error');
      return;
    }

    updating = true;
    try {
      await apiChangePassword(oldPassword, newPassword);
      showToast('Password changed successfully!', 'success');
      oldPassword = '';
      newPassword = '';
      confirmPassword = '';
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      updating = false;
    }
  }
</script>

<div class="settings-container">
  <div class="page-header">
    <h2>Account Settings</h2>
    <p class="subtitle">Manage your personal profile and mailbox security</p>
  </div>

  <!-- Profile Overview Card -->
  <div class="card">
    <h3>Profile Information</h3>
    <div class="profile-grid">
      <div class="info-item">
        <span class="info-label">Account Name:</span>
        <span class="info-val">{$currentUser?.account}</span>
      </div>
      <div class="info-item">
        <span class="info-label">Display Name:</span>
        <span class="info-val">{$currentUser?.name}</span>
      </div>
      <div class="info-item">
        <span class="info-label">Domain:</span>
        <span class="info-val">{$currentUser?.domain || '—'}</span>
      </div>
      <div class="info-item">
        <span class="info-label">Account Role:</span>
        <span class="info-val">
          {$currentUser?.is_admin ? 'Administrator' : 'Standard User'}
        </span>
      </div>
    </div>
  </div>

  <!-- Change Password Card -->
  <div class="card">
    <h3>Change Password</h3>
    <form class="password-form" on:submit|preventDefault={handleUpdatePassword}>
      <div class="form-group">
        <label for="old-pwd">Current Password</label>
        <input
          id="old-pwd"
          type="password"
          placeholder="Enter current password"
          bind:value={oldPassword}
          required
        />
      </div>

      <div class="form-group">
        <label for="new-pwd">New Password</label>
        <input
          id="new-pwd"
          type="password"
          placeholder="Enter new password (min. 6 chars)"
          bind:value={newPassword}
          required
        />
      </div>

      <div class="form-group">
        <label for="confirm-pwd">Confirm New Password</label>
        <input
          id="confirm-pwd"
          type="password"
          placeholder="Re-enter new password"
          bind:value={confirmPassword}
          required
        />
      </div>

      <button type="submit" class="primary-btn" disabled={updating}>
        {updating ? 'Updating Password...' : 'Save New Password'}
      </button>
    </form>
  </div>
</div>

<style>
  .settings-container {
    flex: 1;
    padding: 32px 40px;
    overflow-y: auto;
    background: #f8fafd;
  }
  .page-header h2 {
    margin: 0;
    font-size: 24px;
    font-weight: 600;
    color: #1f1f1f;
  }
  .subtitle {
    margin: 4px 0 24px;
    color: #5f6368;
    font-size: 14px;
  }
  .card {
    background: #ffffff;
    border-radius: 12px;
    border: 1px solid #e1e3e7;
    padding: 24px;
    margin-bottom: 24px;
    max-width: 640px;
    box-shadow: 0 1px 3px rgba(60, 64, 67, 0.05);
  }
  .card h3 {
    margin: 0 0 16px;
    font-size: 16px;
    color: #1f1f1f;
  }
  .profile-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .info-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .info-label {
    font-size: 12px;
    font-weight: 600;
    color: #5f6368;
  }
  .info-val {
    font-size: 14px;
    color: #1f1f1f;
    font-weight: 500;
  }
  .password-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
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
    padding: 10px 14px;
    border: 1px solid #dadce0;
    border-radius: 8px;
    font-size: 14px;
    outline: none;
  }
  .form-group input:focus {
    border-color: #1a73e8;
    box-shadow: 0 0 0 2px rgba(26, 115, 232, 0.2);
  }
  .primary-btn {
    align-self: flex-start;
    background: #0b57d0;
    color: #fff;
    border: none;
    border-radius: 20px;
    padding: 10px 24px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    margin-top: 8px;
  }
  .primary-btn:hover {
    background: #0842a0;
  }
  .primary-btn:disabled {
    opacity: 0.6;
    cursor: default;
  }
</style>
