<script>
  import { composeState, showToast, triggerMailRefresh } from '../lib/stores.js';
  import { apiSendEmail } from '../lib/api.js';
  import { formatFileSize } from '../lib/utils.js';

  let showCc = false;
  let showBcc = false;
  let sending = false;
  let isFullscreen = false;
  let fileInput;

  /** Parse a comma/semicolon-separated address string into [{name, email}] */
  function parseAddresses(str) {
    if (!str || !str.trim()) return [];
    return str
      .split(/[,;]+/)
      .map(s => s.trim())
      .filter(Boolean)
      .map(addr => ({ name: '', email: addr }));
  }

  function close() {
    $composeState = { ...$composeState, open: false };
  }

  function toggleMinimize() {
    $composeState = { ...$composeState, minimized: !$composeState.minimized };
  }

  function toggleFullscreen() {
    isFullscreen = !isFullscreen;
  }

  function handleFileSelect(e) {
    const selected = Array.from(e.target.files);
    $composeState = { ...$composeState, files: [...$composeState.files, ...selected] };
  }

  function removeFile(index) {
    $composeState = {
      ...$composeState,
      files: $composeState.files.filter((_, i) => i !== index),
    };
  }

  async function readFileAsBase64(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = e => resolve({ name: file.name, data: e.target.result });
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  }

  async function handleSend() {
    const toList = parseAddresses($composeState.to);
    if (!toList.length) {
      showToast('Please enter at least one recipient', 'error');
      return;
    }
    if (!$composeState.subject) {
      if (!confirm('Send without a subject?')) return;
    }

    sending = true;
    try {
      // Read attachments as base64 data URLs
      const attrs = await Promise.all(
        ($composeState.files || []).map(file => readFileAsBase64(file))
      );

      const bodyHtml = $composeState.body || '';
      const bodyText = bodyHtml.replace(/<[^>]*>/g, '');

      await apiSendEmail({
        to: toList,
        cc: parseAddresses($composeState.cc),
        bcc: parseAddresses($composeState.bcc),
        subject: $composeState.subject || '(no subject)',
        html: bodyHtml,
        text: bodyText,
        attrs,
      });

      showToast('Message sent', 'success');
      $composeState = { ...$composeState, open: false };
      triggerMailRefresh();
    } catch (err) {
      showToast(err.message || 'Failed to send', 'error');
    } finally {
      sending = false;
    }
  }
</script>

{#if $composeState.open}
  <div
    class="compose-window"
    class:minimized={$composeState.minimized}
    class:fullscreen={isFullscreen}
    role="dialog"
    aria-label="Compose new email"
    aria-modal="true"
  >
    <!-- Header / Title Bar -->
    <div class="compose-header">
      <button
        class="header-title-btn"
        on:click={toggleMinimize}
        aria-label={$composeState.minimized ? 'Expand compose window' : 'Minimize compose window'}
        aria-expanded={!$composeState.minimized}
      >
        <span class="header-title">
          {$composeState.subject ? $composeState.subject : 'New Message'}
        </span>
      </button>

      <div class="header-controls" role="toolbar" aria-label="Window controls">
        <button
          class="ctrl-btn"
          on:click={toggleMinimize}
          title="Minimize"
          aria-label="Minimize"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
        </button>
        <button
          class="ctrl-btn"
          on:click={toggleFullscreen}
          title={isFullscreen ? 'Restore window' : 'Expand to full screen'}
          aria-label={isFullscreen ? 'Restore window' : 'Expand to full screen'}
        >
          {#if isFullscreen}
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M8 3v3a2 2 0 01-2 2H3m18 0h-3a2 2 0 01-2-2V3m0 18v-3a2 2 0 012-2h3M3 16h3a2 2 0 012 2v3"/>
            </svg>
          {:else}
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
            </svg>
          {/if}
        </button>
        <button
          class="ctrl-btn ctrl-close"
          on:click={close}
          title="Close and discard"
          aria-label="Close and discard"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Compose Body (hidden when minimized) -->
    {#if !$composeState.minimized}
      <div class="compose-body">

        <!-- To field -->
        <div class="field-row">
          <label class="field-label" for="compose-to">To</label>
          <input
            id="compose-to"
            type="text"
            class="field-input"
            placeholder="Recipients"
            bind:value={$composeState.to}
            autocomplete="email"
          />
          <div class="field-toggles">
            {#if !showCc}
              <button class="toggle-btn" type="button" on:click={() => (showCc = true)}>Cc</button>
            {/if}
            {#if !showBcc}
              <button class="toggle-btn" type="button" on:click={() => (showBcc = true)}>Bcc</button>
            {/if}
          </div>
        </div>

        {#if showCc}
          <div class="field-row">
            <label class="field-label" for="compose-cc">Cc</label>
            <input
              id="compose-cc"
              type="text"
              class="field-input"
              placeholder="Cc recipients"
              bind:value={$composeState.cc}
              autocomplete="email"
            />
          </div>
        {/if}

        {#if showBcc}
          <div class="field-row">
            <label class="field-label" for="compose-bcc">Bcc</label>
            <input
              id="compose-bcc"
              type="text"
              class="field-input"
              placeholder="Bcc recipients"
              bind:value={$composeState.bcc}
              autocomplete="email"
            />
          </div>
        {/if}

        <!-- Subject -->
        <div class="field-row subject-row">
          <input
            id="compose-subject"
            type="text"
            class="field-input subject-input"
            placeholder="Subject"
            bind:value={$composeState.subject}
            aria-label="Subject"
          />
        </div>

        <!-- Message body -->
        <div class="body-area">
          <textarea
            class="body-textarea"
            placeholder="Write your message here..."
            bind:value={$composeState.body}
            aria-label="Message body"
          ></textarea>
        </div>

        <!-- Attachments list -->
        {#if $composeState.files && $composeState.files.length > 0}
          <div class="attachments-bar" aria-label="Attachments">
            {#each $composeState.files as file, idx}
              <div class="file-chip">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                  <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"/>
                </svg>
                <span class="file-name">{file.name}</span>
                <span class="file-size">({formatFileSize(file.size)})</span>
                <button
                  class="remove-file"
                  on:click={() => removeFile(idx)}
                  aria-label="Remove {file.name}"
                  type="button"
                >
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                  </svg>
                </button>
              </div>
            {/each}
          </div>
        {/if}

        <!-- Footer / Actions -->
        <div class="compose-footer">
          <div class="footer-left">
            <button
              class="send-btn"
              type="button"
              on:click={handleSend}
              disabled={sending}
              aria-label={sending ? 'Sending...' : 'Send email'}
            >
              {#if sending}
                <svg class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <path d="M21 12a9 9 0 11-6.219-8.56"/>
                </svg>
                Sending...
              {:else}
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="22" y1="2" x2="11" y2="13"/>
                  <polygon points="22 2 15 22 11 13 2 9 22 2"/>
                </svg>
                Send
              {/if}
            </button>

            <button
              class="icon-btn"
              type="button"
              on:click={() => fileInput.click()}
              title="Attach files"
              aria-label="Attach files"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"/>
              </svg>
            </button>

            <input
              type="file"
              multiple
              style="display:none"
              bind:this={fileInput}
              on:change={handleFileSelect}
              aria-hidden="true"
              tabindex="-1"
            />
          </div>

          <div class="footer-right">
            <button
              class="icon-btn danger-btn"
              type="button"
              on:click={close}
              title="Discard draft"
              aria-label="Discard draft"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"/>
                <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/>
                <path d="M10 11v6M14 11v6"/>
                <path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  /* === Compose window container === */
  .compose-window {
    position: fixed;
    bottom: 0;
    right: 24px;
    width: 560px;
    height: 520px;
    background: #ffffff;
    border-radius: 12px 12px 0 0;
    box-shadow: 0 8px 40px rgba(0, 0, 0, 0.22), 0 2px 8px rgba(0, 0, 0, 0.12);
    display: flex;
    flex-direction: column;
    z-index: 300;
    overflow: hidden;
    border: 1px solid #c4c7cc;
    border-bottom: none;
    transition: width 0.2s ease, height 0.2s ease;
  }
  .compose-window.minimized {
    height: 44px;
    width: 300px;
  }
  .compose-window.fullscreen {
    top: 24px;
    bottom: 24px;
    left: 80px;
    right: 80px;
    width: auto;
    height: auto;
    border-radius: 12px;
    border: 1px solid #c4c7cc;
  }

  /* === Header === */
  .compose-header {
    display: flex;
    align-items: center;
    height: 44px;
    padding: 0 8px 0 16px;
    background: #404040;
    border-radius: 12px 12px 0 0;
    flex-shrink: 0;
    gap: 8px;
  }
  .compose-window.minimized .compose-header {
    border-radius: 12px 12px 0 0;
  }
  .header-title-btn {
    flex: 1;
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
    padding: 0;
    min-width: 0;
  }
  .header-title {
    font-size: 13px;
    font-weight: 600;
    color: #ffffff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: block;
  }
  .header-controls {
    display: flex;
    align-items: center;
    gap: 2px;
    flex-shrink: 0;
  }
  .ctrl-btn {
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.75);
    cursor: pointer;
    padding: 6px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s, color 0.15s;
  }
  .ctrl-btn:hover {
    background: rgba(255, 255, 255, 0.12);
    color: #ffffff;
  }
  .ctrl-close:hover {
    background: rgba(255, 80, 80, 0.3);
    color: #ff9999;
  }

  /* === Compose body === */
  .compose-body {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  /* === Field rows === */
  .field-row {
    display: flex;
    align-items: center;
    border-bottom: 1px solid #e8eaed;
    padding: 0 16px;
    min-height: 44px;
    gap: 0;
  }
  .field-row:focus-within {
    background: #f8f9fa;
  }
  .field-label {
    font-size: 13px;
    font-weight: 500;
    color: #5f6368;
    width: 36px;
    flex-shrink: 0;
    user-select: none;
  }
  .field-input {
    flex: 1;
    border: none;
    outline: none;
    background: transparent;
    font-size: 14px;
    color: #202124;
    padding: 10px 8px;
    font-family: inherit;
    min-width: 0;
  }
  .field-input::placeholder {
    color: #9aa0a6;
  }
  .field-toggles {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }
  .toggle-btn {
    background: none;
    border: none;
    color: #5f6368;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
    transition: background 0.15s;
  }
  .toggle-btn:hover {
    background: #e8eaed;
    color: #202124;
  }

  /* Subject row */
  .subject-row {
    padding: 0 16px;
  }
  .subject-input {
    font-weight: 500;
    font-size: 14px;
  }

  /* === Body textarea === */
  .body-area {
    flex: 1;
    display: flex;
    overflow: hidden;
    padding: 4px 0;
  }
  .body-textarea {
    flex: 1;
    border: none;
    outline: none;
    resize: none;
    font-family: inherit;
    font-size: 15px;
    line-height: 1.65;
    color: #202124;
    padding: 12px 20px;
    background: transparent;
    overflow-y: auto;
  }
  .body-textarea::placeholder {
    color: #9aa0a6;
  }

  /* === Attachments bar === */
  .attachments-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding: 8px 16px;
    border-top: 1px solid #e8eaed;
    background: #f8f9fa;
  }
  .file-chip {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: #ffffff;
    border: 1px solid #dadce0;
    border-radius: 20px;
    padding: 4px 8px 4px 10px;
    font-size: 12px;
    color: #3c4043;
  }
  .file-name {
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .file-size {
    color: #80868b;
    flex-shrink: 0;
  }
  .remove-file {
    background: none;
    border: none;
    color: #5f6368;
    cursor: pointer;
    padding: 2px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    margin-left: 2px;
  }
  .remove-file:hover {
    background: #e8eaed;
    color: #202124;
  }

  /* === Footer === */
  .compose-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 56px;
    padding: 0 16px;
    border-top: 1px solid #e8eaed;
    background: #ffffff;
    flex-shrink: 0;
  }
  .footer-left,
  .footer-right {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  /* Send button */
  .send-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    background: #0b57d0;
    color: #ffffff;
    border: none;
    border-radius: 20px;
    padding: 0 20px;
    height: 36px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    font-family: inherit;
    transition: background 0.15s, box-shadow 0.15s;
  }
  .send-btn:hover:not(:disabled) {
    background: #0842a0;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.2);
  }
  .send-btn:disabled {
    opacity: 0.65;
    cursor: default;
  }

  /* Icon buttons in footer */
  .icon-btn {
    background: none;
    border: none;
    color: #5f6368;
    cursor: pointer;
    padding: 8px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s, color 0.15s;
  }
  .icon-btn:hover {
    background: #e8eaed;
    color: #202124;
  }
  .danger-btn:hover {
    background: #fce8e6;
    color: #c5221f;
  }

  /* Spinner animation */
  .spin {
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
