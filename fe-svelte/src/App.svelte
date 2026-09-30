<script>
  import { onMount } from 'svelte';
  import { currentUser, currentFolder, selectedEmailId } from './lib/stores.js';
  import { apiGetUserInfo } from './lib/api.js';

  import Toast from './components/Toast.svelte';
  import Login from './components/Login.svelte';
  import Navbar from './components/Navbar.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import MailList from './components/MailList.svelte';
  import MailView from './components/MailView.svelte';
  import Compose from './components/Compose.svelte';
  import AdminUsers from './components/AdminUsers.svelte';
  import AdminDomains from './components/AdminDomains.svelte';
  import Settings from './components/Settings.svelte';

  // Check login session on mount
  onMount(async () => {
    if ($currentUser) {
      try {
        const info = await apiGetUserInfo();
        $currentUser = {
          ...$currentUser,
          ...info,
          is_admin: info.is_admin === true || info.is_admin === 1 || info.IsAdmin === 1 || $currentUser.is_admin === true || $currentUser.is_admin === 1,
        };
      } catch (e) {
        // session expired
        $currentUser = null;
      }
    }
  });
</script>

<Toast />

{#if !$currentUser}
  <Login />
{:else}
  <div class="app-layout">
    <Navbar />
    <div class="app-body">
      <Sidebar />
      <main class="main-content">
        {#if $selectedEmailId}
          <MailView />
        {:else if $currentFolder === 'settings'}
          <Settings />
        {:else if $currentFolder === 'admin-users'}
          <AdminUsers />
        {:else if $currentFolder === 'admin-domains'}
          <AdminDomains />
        {:else}
          <MailList />
        {/if}
      </main>
    </div>
    <Compose />
  </div>
{/if}

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
    color: #1f1f1f;
    background: #f6f8fc;
    overflow: hidden;
  }
  :global(*) {
    box-sizing: border-box;
  }
  .app-layout {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }
  .app-body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }
  .main-content {
    flex: 1;
    display: flex;
    overflow: hidden;
  }
</style>
