<script>
  import { toasts } from '../lib/stores.js';
</script>

<div class="toast-container" aria-live="assertive" aria-atomic="true">
  {#each $toasts as toast (toast.id)}
    <div class="toast toast-{toast.type}" role="status">
      <span class="icon" aria-hidden="true">
        {#if toast.type === 'success'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        {:else if toast.type === 'error'}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="16" x2="12" y2="12"/>
            <line x1="12" y1="8" x2="12.01" y2="8"/>
          </svg>
        {/if}
      </span>
      <span class="message">{toast.message}</span>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 24px;
    left: 24px;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  }
  .toast {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 18px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    color: #fff;
    background: #323232;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
    pointer-events: auto;
    animation: slideUp 0.22s cubic-bezier(0.16, 1, 0.3, 1);
    max-width: 360px;
  }
  .toast-success { background: #188038; }
  .toast-error   { background: #c5221f; }
  .toast-info    { background: #1a73e8; }
  .icon {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }
  .message {
    flex: 1;
    line-height: 1.4;
  }
  @keyframes slideUp {
    from { opacity: 0; transform: translateY(10px); }
    to   { opacity: 1; transform: translateY(0); }
  }
</style>
