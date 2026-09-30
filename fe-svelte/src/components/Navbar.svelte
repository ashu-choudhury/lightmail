<script>
  import { currentUser, currentFolder, searchQuery, sidebarOpen, triggerMailRefresh } from '../lib/stores.js';
  import { apiLogout } from '../lib/api.js';
  import { getInitials, stringToColor } from '../lib/utils.js';
  import Icon from './Icon.svelte';

  let localSearch = '';
  let userMenuOpen = false;

  function handleSearch(e) {
    if (e.key === 'Enter') {
      $searchQuery = localSearch;
      triggerMailRefresh();
    }
  }

  function clearSearch() {
    localSearch = '';
    $searchQuery = '';
    triggerMailRefresh();
  }

  async function handleLogout() {
    try {
      await apiLogout();
    } catch (e) {
      // ignore
    }
    $currentUser = null;
  }
</script>

<header class="navbar">
  <div class="left-section">
    <button
      class="icon-btn"
      on:click={() => ($sidebarOpen = !$sidebarOpen)}
      aria-label="Toggle Navigation Sidebar"
      title="Main menu"
    >
      <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3 18h18v-2H3v2zm0-5h18v-2H3v2zm0-7v2h18V6H3z"/>
      </svg>
    </button>
    <button class="brand" type="button" on:click={() => { $currentFolder = 'inbox'; $searchQuery = ''; }}>
      <svg class="brand-icon" viewBox="0 0 24 24" width="28" height="28" fill="#1a73e8">
        <path d="M20 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"/>
      </svg>
      <span class="brand-name">Lightmail</span>
    </button>
  </div>

  <div class="center-section">
    <div class="search-box">
      <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="#5f6368">
        <path d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
      </svg>
      <input
        type="text"
        placeholder="Search mail (press Enter to search)"
        bind:value={localSearch}
        on:keydown={handleSearch}
        aria-label="Search mail"
      />
      {#if localSearch}
        <button class="clear-btn" on:click={clearSearch} aria-label="Clear search">
          <Icon name="close" size={14} />
        </button>
      {/if}
    </div>
  </div>

  <div class="right-section">
    <div class="user-menu-container">
      <button
        class="user-pill"
        on:click={() => (userMenuOpen = !userMenuOpen)}
        aria-expanded={userMenuOpen}
        aria-label="User account menu"
      >
        <div
          class="avatar"
          style="background-color: {stringToColor($currentUser?.name || $currentUser?.account)}"
        >
          {getInitials($currentUser?.name || $currentUser?.account)}
        </div>
        <span class="user-name">{$currentUser?.name || $currentUser?.account}</span>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
          <path d="M7 10l5 5 5-5z"/>
        </svg>
      </button>

      {#if userMenuOpen}
        <div class="dropdown-backdrop" on:click={() => (userMenuOpen = false)} on:keydown={(e) => e.key === 'Escape' && (userMenuOpen = false)} role="button" tabindex="-1" aria-label="Close user menu"></div>
        <div class="dropdown-menu">
          <div class="user-info-card">
            <div
              class="avatar-large"
              style="background-color: {stringToColor($currentUser?.name || $currentUser?.account)}"
            >
              {getInitials($currentUser?.name || $currentUser?.account)}
            </div>
            <div class="user-details">
              <strong>{$currentUser?.name || 'User'}</strong>
              <small>{$currentUser?.account}</small>
              {#if $currentUser?.is_admin || $currentUser?.IsAdmin}
                <span class="badge-admin">Administrator</span>
              {/if}
            </div>
          </div>
          <hr />
          <button class="dropdown-item" on:click={() => { $currentFolder = 'settings'; userMenuOpen = false; }}>
            <Icon name="settings" size={16} />
            <span>Account Settings</span>
          </button>
          {#if $currentUser?.is_admin || $currentUser?.IsAdmin}
            <button class="dropdown-item" on:click={() => { $currentFolder = 'admin-users'; userMenuOpen = false; }}>
              <Icon name="users" size={16} />
              <span>User Management</span>
            </button>
            <button class="dropdown-item" on:click={() => { $currentFolder = 'admin-domains'; userMenuOpen = false; }}>
              <Icon name="domain" size={16} />
              <span>Domains & DNS</span>
            </button>
          {/if}
          <hr />
          <button class="dropdown-item logout" on:click={handleLogout}>
            <Icon name="logout" size={16} />
            <span>Sign Out</span>
          </button>
        </div>
      {/if}
    </div>
  </div>
</header>

<style>
  .navbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 64px;
    padding: 0 16px;
    background: #f6f8fc;
    border-bottom: 1px solid #e1e3e7;
    position: sticky;
    top: 0;
    z-index: 50;
  }
  .left-section {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 240px;
  }
  .icon-btn {
    background: none;
    border: none;
    padding: 8px;
    border-radius: 50%;
    cursor: pointer;
    color: #5f6368;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .icon-btn:hover {
    background: #e5e8ec;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    user-select: none;
    background: none;
    border: none;
    font: inherit;
    padding: 0;
  }
  .brand-name {
    font-size: 20px;
    font-weight: 600;
    color: #1f1f1f;
    letter-spacing: -0.2px;
  }
  .center-section {
    flex: 1;
    max-width: 720px;
    margin: 0 24px;
  }
  .search-box {
    display: flex;
    align-items: center;
    background: #eaf1fb;
    border-radius: 28px;
    padding: 0 16px;
    height: 48px;
    transition: background 0.2s, box-shadow 0.2s;
  }
  .search-box:focus-within {
    background: #ffffff;
    box-shadow: 0 1px 6px rgba(32, 33, 36, 0.28);
  }
  .search-box input {
    flex: 1;
    border: none;
    background: transparent;
    padding: 0 12px;
    font-size: 15px;
    color: #1f1f1f;
    outline: none;
  }
  .clear-btn {
    background: none;
    border: none;
    color: #5f6368;
    cursor: pointer;
    font-size: 14px;
    padding: 4px;
  }
  .right-section {
    display: flex;
    align-items: center;
  }
  .user-menu-container {
    position: relative;
  }
  .user-pill {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #ffffff;
    border: 1px solid #dadce0;
    border-radius: 24px;
    padding: 4px 12px 4px 4px;
    cursor: pointer;
    transition: box-shadow 0.2s;
  }
  .user-pill:hover {
    box-shadow: 0 1px 3px rgba(60, 64, 67, 0.3);
  }
  .avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    color: #ffffff;
    font-weight: 600;
    font-size: 13px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .user-name {
    font-size: 14px;
    font-weight: 500;
    color: #3c4043;
    max-width: 140px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .dropdown-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    z-index: 100;
  }
  .dropdown-menu {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    width: 280px;
    background: #ffffff;
    border-radius: 12px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
    border: 1px solid #e1e3e7;
    padding: 12px 0;
    z-index: 101;
  }
  .user-info-card {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
  }
  .avatar-large {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    color: #fff;
    font-size: 16px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .user-details {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .user-details strong {
    font-size: 14px;
    color: #202124;
  }
  .user-details small {
    font-size: 12px;
    color: #5f6368;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .badge-admin {
    display: inline-block;
    align-self: flex-start;
    margin-top: 4px;
    font-size: 10px;
    background: #e8f0fe;
    color: #1a73e8;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
  }
  hr {
    border: none;
    border-top: 1px solid #f1f3f4;
    margin: 8px 0;
  }
  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 10px 16px;
    border: none;
    background: none;
    font-size: 13px;
    font-weight: 500;
    color: #3c4043;
    cursor: pointer;
    text-align: left;
  }
  .dropdown-item:hover {
    background: #f1f3f4;
  }
  .dropdown-item.logout {
    color: #d93025;
  }
</style>
