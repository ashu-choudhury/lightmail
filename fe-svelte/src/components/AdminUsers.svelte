<script>
  import { onMount } from 'svelte';
  import { currentUser, showToast } from '../lib/stores.js';
  import { apiGetUserList, apiCreateUser, apiEditUser } from '../lib/api.js';

  let users = [];
  let loading = false;
  let showCreateModal = false;
  let creating = false;

  // New user form state
  let selectedDomain = $currentUser?.domains?.[0] || $currentUser?.domain || '';
  let localUsername = '';
  let displayName = '';
  let password = '';
  let gender = '';
  let isAdmin = 0;

  // Password change modal
  let editingUser = null;
  let newPassword = '';
  let savingPassword = false;

  $: fullEmailPreview = localUsername ? `${localUsername.trim().toLowerCase()}@${selectedDomain}` : `@${selectedDomain}`;

  function generateRandomPassword() {
    const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789!@#$%&*';
    let res = '';
    for (let i = 0; i < 12; i++) {
      res += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    password = res;
  }

  async function loadUsers() {
    loading = true;
    try {
      const res = await apiGetUserList(1, 50);
      users = res.list || [];
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      loading = false;
    }
  }

  onMount(loadUsers);

  async function handleCreateUser() {
    if (!localUsername.trim() || !displayName.trim() || !password.trim()) {
      showToast('Please fill in username, display name, and password', 'error');
      return;
    }

    creating = true;
    try {
      await apiCreateUser({
        account: localUsername.trim().toLowerCase(),
        domain: selectedDomain,
        username: displayName.trim(),
        password: password.trim(),
        isAdmin: isAdmin ? 1 : 0,
        gender: gender,
      });

      showToast(`User ${fullEmailPreview} created successfully!`, 'success');
      showCreateModal = false;
      localUsername = '';
      displayName = '';
      password = '';
      gender = '';
      isAdmin = 0;
      loadUsers();
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      creating = false;
    }
  }

  async function toggleUserDisabled(user) {
    const newStatus = user.disabled ? 0 : 1;
    try {
      await apiEditUser({
        id: user.id,
        account: user.account,
        disabled: newStatus,
      });
      user.disabled = newStatus;
      users = [...users];
      showToast(`User ${user.account} ${newStatus ? 'disabled' : 'enabled'}`, 'info');
    } catch (err) {
      showToast(err.message, 'error');
    }
  }

  async function handleChangeUserPassword() {
    if (!newPassword.trim()) {
      showToast('Password cannot be empty', 'error');
      return;
    }
    savingPassword = true;
    try {
      await apiEditUser({
        id: editingUser.id,
        account: editingUser.account,
        password: newPassword.trim(),
      });
      showToast(`Password updated for ${editingUser.account}`, 'success');
      editingUser = null;
      newPassword = '';
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      savingPassword = false;
    }
  }
</script>

<div class="admin-container">
  <div class="page-header">
    <div>
      <h2>User & Mailbox Management</h2>
      <p class="subtitle">Create and manage mailboxes and administrative privileges</p>
    </div>
    <button class="primary-btn" on:click={() => (showCreateModal = true)}>
      Create Mailbox
    </button>
  </div>

  <!-- Users Table -->
  <div class="table-card">
    {#if loading}
      <div class="loading-state">Loading users...</div>
    {:else}
      <table class="users-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Email Address</th>
            <th>Display Name</th>
            <th>Role</th>
            <th>Gender</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each users as u (u.id)}
            <tr>
              <td>{u.id}</td>
              <td class="account-cell">
                <strong>{u.account}</strong>
              </td>
              <td>{u.name}</td>
              <td>
                {#if u.is_admin}
                  <span class="role-badge admin">Administrator</span>
                {:else}
                  <span class="role-badge user">Regular User</span>
                {/if}
              </td>
              <td>{u.gender || '—'}</td>
              <td>
                <span class="status-dot" class:active={!u.disabled}></span>
                {u.disabled ? 'Disabled' : 'Active'}
              </td>
              <td class="actions-cell">
                <button
                  class="btn-text"
                  on:click={() => { editingUser = u; newPassword = ''; }}
                  title="Change Password"
                >
                  Password
                </button>
                <button
                  class="btn-text danger"
                  on:click={() => toggleUserDisabled(u)}
                >
                  {u.disabled ? 'Enable' : 'Disable'}
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <!-- Create User Modal -->
  {#if showCreateModal}
    <div
      class="modal-backdrop"
      on:click={() => (showCreateModal = false)}
      on:keydown={(e) => e.key === 'Escape' && (showCreateModal = false)}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="modal-content" on:click|stopPropagation role="document">
        <div class="modal-header">
          <h3>Create New Mailbox</h3>
          <button class="close-btn" on:click={() => (showCreateModal = false)}>✕</button>
        </div>

        <form class="modal-body" on:submit|preventDefault={handleCreateUser}>
          <!-- Domain Selector -->
          <div class="form-group">
            <label for="domain-select">Domain</label>
            <select id="domain-select" bind:value={selectedDomain}>
              {#each ($currentUser?.domains || ($currentUser?.domain ? [$currentUser.domain] : [])) as dom}
                <option value={dom}>{dom}</option>
              {/each}
            </select>
          </div>

          <!-- Username Input -->
          <div class="form-group">
            <label for="username-input">Mailbox Username (local part)</label>
            <div class="input-with-preview">
              <input
                id="username-input"
                type="text"
                placeholder="e.g. ashu, info, contact"
                bind:value={localUsername}
                required
              />
            </div>
            <div class="preview-chip">
              Full Email: <code>{fullEmailPreview}</code>
            </div>
          </div>

          <!-- Display Name -->
          <div class="form-group">
            <label for="display-name">Display Name</label>
            <input
              id="display-name"
              type="text"
              placeholder="e.g. Ashu Choudhury"
              bind:value={displayName}
              required
            />
          </div>

          <!-- Password -->
          <div class="form-group">
            <div class="label-row">
              <label for="user-password">Password</label>
              <button
                type="button"
                class="generate-btn"
                on:click={generateRandomPassword}
              >
                Generate Strong Password
              </button>
            </div>
            <input
              id="user-password"
              type="text"
              placeholder="Enter or generate password"
              bind:value={password}
              required
            />
          </div>

          <!-- Gender -->
          <div class="form-group">
            <label for="gender-select">Gender (optional)</label>
            <select id="gender-select" bind:value={gender}>
              <option value="">Prefer not to say / Unspecified</option>
              <option value="Male">Male</option>
              <option value="Female">Female</option>
              <option value="Other">Other</option>
            </select>
          </div>

          <!-- Admin Privilege Toggle -->
          <div class="form-group checkbox-group">
            <label class="switch-wrap">
              <input type="checkbox" bind:checked={isAdmin} />
              <span class="switch-label">Grant Administrator Privileges</span>
            </label>
            <small class="helper-text">
              Administrators can manage users, domains, DNS records, and server settings.
            </small>
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="secondary-btn"
              on:click={() => (showCreateModal = false)}
            >
              Cancel
            </button>
            <button
              type="submit"
              class="primary-btn"
              disabled={creating}
            >
              {creating ? 'Creating...' : 'Create Account'}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- Password Reset Modal -->
  {#if editingUser}
    <div
      class="modal-backdrop"
      on:click={() => (editingUser = null)}
      on:keydown={(e) => e.key === 'Escape' && (editingUser = null)}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div class="modal-content" on:click|stopPropagation role="document">
        <div class="modal-header">
          <h3>Reset Password: {editingUser.account}</h3>
          <button class="close-btn" on:click={() => (editingUser = null)}>✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label for="new-pass-input">New Password</label>
            <input
              id="new-pass-input"
              type="password"
              placeholder="Enter new password"
              bind:value={newPassword}
            />
          </div>
        </div>
        <div class="modal-footer">
          <button class="secondary-btn" on:click={() => (editingUser = null)}>Cancel</button>
          <button class="primary-btn" on:click={handleChangeUserPassword} disabled={savingPassword}>
            {savingPassword ? 'Saving...' : 'Update Password'}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .admin-container {
    flex: 1;
    padding: 32px 40px;
    overflow-y: auto;
    background: #f8fafd;
  }
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
  }
  .page-header h2 {
    margin: 0;
    font-size: 24px;
    font-weight: 600;
    color: #1f1f1f;
  }
  .subtitle {
    margin: 4px 0 0;
    color: #5f6368;
    font-size: 14px;
  }
  .primary-btn {
    background: #0b57d0;
    color: #fff;
    border: none;
    border-radius: 20px;
    padding: 10px 24px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  }
  .primary-btn:hover {
    background: #0842a0;
  }
  .secondary-btn {
    background: #ffffff;
    color: #444746;
    border: 1px solid #dadce0;
    border-radius: 20px;
    padding: 10px 20px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
  }
  .table-card {
    background: #ffffff;
    border-radius: 12px;
    border: 1px solid #e1e3e7;
    box-shadow: 0 1px 3px rgba(60, 64, 67, 0.05);
    overflow: hidden;
  }
  .users-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
    text-align: left;
  }
  .users-table th {
    background: #f1f3f4;
    padding: 12px 16px;
    font-weight: 600;
    color: #444746;
    border-bottom: 1px solid #e1e3e7;
  }
  .users-table td {
    padding: 14px 16px;
    border-bottom: 1px solid #f1f3f4;
    color: #202124;
  }
  .users-table tr:hover {
    background: #f8fafd;
  }
  .role-badge {
    display: inline-block;
    padding: 3px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
  }
  .role-badge.admin {
    background: #e8f0fe;
    color: #1a73e8;
  }
  .role-badge.user {
    background: #f1f3f4;
    color: #5f6368;
  }
  .status-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #dadce0;
    margin-right: 6px;
  }
  .status-dot.active {
    background: #188038;
  }
  .btn-text {
    background: none;
    border: none;
    color: #1a73e8;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    margin-right: 8px;
  }
  .btn-text:hover {
    text-decoration: underline;
  }
  .btn-text.danger {
    color: #d93025;
  }
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 500;
  }
  .modal-content {
    background: #ffffff;
    border-radius: 16px;
    width: 520px;
    max-width: 90vw;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    overflow: hidden;
  }
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 24px;
    border-bottom: 1px solid #e1e3e7;
  }
  .modal-header h3 {
    margin: 0;
    font-size: 18px;
    color: #1f1f1f;
  }
  .close-btn {
    background: none;
    border: none;
    font-size: 18px;
    color: #5f6368;
    cursor: pointer;
  }
  .modal-body {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 18px;
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
  .label-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .generate-btn {
    background: none;
    border: none;
    color: #1a73e8;
    font-size: 12px;
    cursor: pointer;
  }
  .generate-btn:hover {
    text-decoration: underline;
  }
  .form-group input,
  .form-group select {
    border: 1px solid #dadce0;
    border-radius: 8px;
    padding: 10px 14px;
    font-size: 14px;
    color: #1f1f1f;
    outline: none;
    background: #fff;
  }
  .form-group input:focus,
  .form-group select:focus {
    border-color: #1a73e8;
    box-shadow: 0 0 0 2px rgba(26, 115, 232, 0.2);
  }
  .preview-chip {
    font-size: 12px;
    color: #5f6368;
    background: #f1f3f4;
    padding: 6px 10px;
    border-radius: 6px;
    margin-top: 4px;
  }
  .preview-chip code {
    color: #1a73e8;
    font-weight: 600;
  }
  .switch-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-weight: 600;
    font-size: 14px;
    color: #202124;
  }
  .helper-text {
    color: #5f6368;
    font-size: 12px;
    margin-top: 2px;
  }
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 8px;
  }
  .loading-state {
    padding: 48px;
    text-align: center;
    color: #5f6368;
  }
</style>

