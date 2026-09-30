<script>
  import { currentFolder, selectedEmailId, sidebarOpen, composeState, currentUser } from '../lib/stores.js';
  import Icon from './Icon.svelte';

  export let unreadCount = 0;

  function openCompose() {
    $composeState = {
      open: true,
      minimized: false,
      to: '',
      cc: '',
      bcc: '',
      subject: '',
      body: '',
      files: [],
      replyToId: null,
    };
  }

  function setFolder(folder) {
    $currentFolder = folder;
    $selectedEmailId = null;
  }
</script>

<aside class="sidebar" class:collapsed={!$sidebarOpen}>
  <div class="compose-container">
    <button class="compose-btn" on:click={openCompose} aria-label="Compose new email">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 20h9"/>
        <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/>
      </svg>
      <span class="compose-text">Compose</span>
    </button>
  </div>

  <nav class="nav-list" aria-label="Email folders">
    <button
      class="nav-item"
      class:active={$currentFolder === 'inbox'}
      on:click={() => setFolder('inbox')}
    >
      <span class="icon"><Icon name="inbox" size={18} /></span>
      <span class="label">Inbox</span>
      {#if unreadCount > 0}
        <span class="badge">{unreadCount}</span>
      {/if}
    </button>

    <button
      class="nav-item"
      class:active={$currentFolder === 'sent'}
      on:click={() => setFolder('sent')}
    >
      <span class="icon"><Icon name="sent" size={18} /></span>
      <span class="label">Sent</span>
    </button>

    <button
      class="nav-item"
      class:active={$currentFolder === 'drafts'}
      on:click={() => setFolder('drafts')}
    >
      <span class="icon"><Icon name="drafts" size={18} /></span>
      <span class="label">Drafts</span>
    </button>

    <button
      class="nav-item"
      class:active={$currentFolder === 'trash'}
      on:click={() => setFolder('trash')}
    >
      <span class="icon"><Icon name="trash" size={18} /></span>
      <span class="label">Trash</span>
    </button>

    <button
      class="nav-item"
      class:active={$currentFolder === 'spam'}
      on:click={() => setFolder('spam')}
    >
      <span class="icon"><Icon name="spam" size={18} /></span>
      <span class="label">Spam</span>
    </button>

    {#if $currentUser?.is_admin || $currentUser?.IsAdmin}
      <div class="section-divider">ADMINISTRATION</div>

      <button
        class="nav-item admin-item"
        class:active={$currentFolder === 'admin-users'}
        on:click={() => setFolder('admin-users')}
      >
        <span class="icon"><Icon name="users" size={18} /></span>
        <span class="label">User Accounts</span>
      </button>

      <button
        class="nav-item admin-item"
        class:active={$currentFolder === 'admin-domains'}
        on:click={() => setFolder('admin-domains')}
      >
        <span class="icon"><Icon name="domain" size={18} /></span>
        <span class="label">Domains & Cloudflare</span>
      </button>
    {/if}

    <div class="section-divider">PREFERENCES</div>

    <button
      class="nav-item"
      class:active={$currentFolder === 'settings'}
      on:click={() => setFolder('settings')}
    >
      <span class="icon"><Icon name="settings" size={18} /></span>
      <span class="label">Settings</span>
    </button>
  </nav>

  <div class="sidebar-footer">
    <small>Lightmail • Pure SQLite</small>
  </div>
</aside>

<style>
  .sidebar {
    width: 256px;
    height: calc(100vh - 64px);
    background: #f6f8fc;
    display: flex;
    flex-direction: column;
    padding: 16px 12px;
    box-sizing: border-box;
    transition: width 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    flex-shrink: 0;
    overflow-y: auto;
  }
  .sidebar.collapsed {
    width: 72px;
    padding: 16px 8px;
  }
  .compose-container {
    margin-bottom: 16px;
  }
  .compose-btn {
    display: flex;
    align-items: center;
    gap: 12px;
    background: #c2e7ff;
    color: #001d35;
    border: none;
    border-radius: 16px;
    padding: 0 24px;
    height: 56px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
    transition: box-shadow 0.2s, background 0.2s;
    width: 100%;
    justify-content: flex-start;
  }
  .compose-btn:hover {
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.18);
    background: #b3def7;
  }
  .sidebar.collapsed .compose-btn {
    padding: 0;
    width: 56px;
    height: 56px;
    border-radius: 16px;
    justify-content: center;
    margin: 0 auto;
  }
  .sidebar.collapsed .compose-text {
    display: none;
  }
  .nav-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
  }
  .nav-item {
    display: flex;
    align-items: center;
    gap: 16px;
    height: 40px;
    padding: 0 16px;
    border: none;
    background: transparent;
    border-radius: 20px;
    color: #444746;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s, color 0.15s;
    user-select: none;
  }
  .nav-item:hover {
    background: #e8ebf0;
    color: #1f1f1f;
  }
  .nav-item.active {
    background: #d3e3fd;
    color: #041e49;
    font-weight: 600;
  }
  .icon {
    font-size: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
  }
  .label {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .badge {
    font-size: 12px;
    font-weight: 700;
    color: #1f1f1f;
  }
  .sidebar.collapsed .label,
  .sidebar.collapsed .badge,
  .sidebar.collapsed .section-divider {
    display: none;
  }
  .sidebar.collapsed .nav-item {
    justify-content: center;
    padding: 0;
    width: 48px;
    margin: 0 auto;
    border-radius: 50%;
  }
  .section-divider {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.8px;
    color: #747775;
    padding: 16px 16px 6px;
  }
  .admin-item {
    color: #1a73e8;
  }
  .admin-item.active {
    background: #e8f0fe;
    color: #1a73e8;
  }
  .sidebar-footer {
    padding: 12px 16px;
    color: #747775;
    border-top: 1px solid #e1e3e7;
    margin-top: auto;
  }
  .sidebar.collapsed .sidebar-footer {
    display: none;
  }
</style>
