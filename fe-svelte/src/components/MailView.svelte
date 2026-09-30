<script>
  import { onMount } from 'svelte';
  import { selectedEmailId, currentFolder, composeState, showToast } from '../lib/stores.js';
  import { apiGetEmailDetail, apiMarkRead, apiDeleteEmail, apiMoveEmail } from '../lib/api.js';
  import { formatFullDateTime, formatFileSize, getInitials, stringToColor } from '../lib/utils.js';

  let email = null;
  let loading = true;
  let showDetails = false;

  async function loadDetail() {
    if (!$selectedEmailId) return;
    loading = true;
    try {
      email = await apiGetEmailDetail($selectedEmailId);
      // Mark as read automatically when opened
      if (!email.is_read) {
        apiMarkRead([$selectedEmailId], true).catch(() => {});
      }
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      loading = false;
    }
  }

  $: $selectedEmailId, loadDetail();

  function goBack() {
    $selectedEmailId = null;
  }

  async function handleDelete() {
    try {
      if ($currentFolder === 'trash') {
        await apiDeleteEmail([email.id]);
        showToast('Deleted permanently', 'info');
      } else {
        await apiMoveEmail([email.id], 'trash');
        showToast('Moved to Trash', 'info');
      }
      goBack();
    } catch (err) {
      showToast(err.message, 'error');
    }
  }

  function handleReply() {
    $composeState = {
      open: true,
      minimized: false,
      to: email.from_address || email.display_sender,
      cc: '',
      bcc: '',
      subject: email.subject.startsWith('Re:') ? email.subject : `Re: ${email.subject}`,
      body: `<br><br><hr>On ${formatFullDateTime(email.send_date)}, ${email.display_sender} wrote:<br><blockquote>${email.html || email.text || email.desc || ''}</blockquote>`,
      files: [],
      replyToId: email.id,
    };
  }

  function handleForward() {
    $composeState = {
      open: true,
      minimized: false,
      to: '',
      cc: '',
      bcc: '',
      subject: email.subject.startsWith('Fwd:') ? email.subject : `Fwd: ${email.subject}`,
      body: `<br><br>---------- Forwarded message ---------<br>From: ${email.display_sender} &lt;${email.from_address}&gt;<br>Date: ${formatFullDateTime(email.send_date)}<br>Subject: ${email.subject}<br>To: ${email.to_display || email.to}<br><br>${email.html || email.text || email.desc || ''}`,
      files: [],
      replyToId: null,
    };
  }
</script>

<div class="mail-view-container">
  <!-- Top Navigation Toolbar -->
  <div class="view-toolbar">
    <button class="tool-btn" on:click={goBack} title="Back to list"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg></button>
    <button class="tool-btn" on:click={handleDelete} title="Delete message" aria-label="Delete message"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/><path d="M10 11v6M14 11v6"/><path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2"/></svg></button>
    <button class="tool-btn" on:click={handleReply} title="Reply" aria-label="Reply"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 17 4 12 9 7"/><path d="M20 18v-2a4 4 0 00-4-4H4"/></svg></button>
    <button class="tool-btn" on:click={handleForward} title="Forward" aria-label="Forward"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 17 20 12 15 7"/><path d="M4 18v-2a4 4 0 014-4h12"/></svg></button>
  </div>

  {#if loading}
    <div class="state-container">
      <div class="spinner"></div>
      <span>Loading email...</span>
    </div>
  {:else if email}
    <div class="content-scroll">
      <!-- Subject Header -->
      <div class="subject-row">
        <h2>{email.subject || '(no subject)'}</h2>
      </div>

      <!-- Sender Info Bar -->
      <div class="sender-card">
        <div
          class="avatar"
          style="background-color: {stringToColor(email.display_sender || email.from_name || email.from_address)}"
        >
          {getInitials(email.display_sender || email.from_name || email.from_address)}
        </div>

        <div class="sender-meta">
          <div class="sender-line">
            <span class="sender-name">{email.display_sender || email.from_name || email.from_address}</span>
            {#if email.from_address}
              <span class="sender-address">&lt;{email.from_address}&gt;</span>
            {/if}
            <span class="date-stamp">{formatFullDateTime(email.send_date || email.create_time)}</span>
          </div>

          <div class="recipient-line">
            <button
              type="button"
              class="to-chip"
              on:click={() => (showDetails = !showDetails)}
              aria-expanded={showDetails}
            >
              to {email.to_display || email.to || 'me'}
              <span class="dropdown-arrow">&#x25BE;</span>
            </button>

            <!-- Verification Security Badges -->
            <div class="badges-row">
              {#if email.dkim_check === 1}
                <span class="badge badge-success" title="Cryptographically signed by sender domain">
                  DKIM Verified
                </span>
              {/if}
              {#if email.spf_check === 1}
                <span class="badge badge-success" title="Sender IP verified by SPF record">
                  SPF Pass
                </span>
              {/if}
            </div>
          </div>

          {#if showDetails}
            <div class="details-dropdown">
              <div><strong>From:</strong> {email.display_sender} {#if email.from_address}&lt;{email.from_address}&gt;{/if}</div>
              <div><strong>To:</strong> {email.to_display || email.to}</div>
              <div><strong>Date:</strong> {formatFullDateTime(email.send_date || email.create_time)}</div>
              <div><strong>Subject:</strong> {email.subject}</div>
            </div>
          {/if}
        </div>
      </div>

      <!-- Attachments Section -->
      {#if email.attachments && email.attachments.length > 0}
        <div class="attachments-box">
          <span class="attachments-label">{email.attachments.length} Attachment(s)</span>
          <div class="attachment-chips">
            {#each email.attachments as att}
              <a
                class="att-chip"
                href={`/attachments/download/${att.id || att.Index || ''}`}
                target="_blank"
                rel="noreferrer"
                download
              >
                <span class="att-name">{att.filename || att.Filename || 'attachment'}</span>
                <span class="att-size">({formatFileSize(att.size || 0)})</span>
                <svg class="att-dl" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
              </a>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Email Body -->
      <div class="email-body">
        {#if email.html}
          <iframe
            title="Email Content"
            class="email-frame"
            srcdoc={email.html}
            sandbox="allow-popups allow-popups-to-escape-sandbox"
          ></iframe>
        {:else}
          <pre class="plain-text">{email.text || ''}</pre>
        {/if}
      </div>

      <!-- Action Buttons -->
      <div class="bottom-actions">
        <button class="action-btn" on:click={handleReply}>
          Reply
        </button>
        <button class="action-btn" on:click={handleForward}>
          Forward
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .mail-view-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: calc(100vh - 64px);
    background: #ffffff;
    border-radius: 16px 16px 0 0;
    overflow: hidden;
    margin-right: 12px;
  }
  .view-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 48px;
    padding: 0 16px;
    border-bottom: 1px solid #e1e3e7;
    background: #ffffff;
  }
  .tool-btn {
    background: none;
    border: none;
    border-radius: 50%;
    width: 36px;
    height: 36px;
    font-size: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #444746;
    cursor: pointer;
  }
  .tool-btn:hover {
    background: #e8ebf0;
  }
  .content-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
  }
  .subject-row h2 {
    font-size: 22px;
    font-weight: 500;
    color: #202124;
    margin: 0 0 20px 0;
  }
  .sender-card {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    margin-bottom: 24px;
  }
  .avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    color: #ffffff;
    font-size: 15px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .sender-meta {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .sender-line {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .sender-name {
    font-weight: 600;
    color: #202124;
    font-size: 15px;
  }
  .sender-address {
    color: #5f6368;
    font-size: 13px;
  }
  .date-stamp {
    margin-left: auto;
    color: #5f6368;
    font-size: 12px;
  }
  .recipient-line {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 13px;
    color: #5f6368;
  }
  .to-chip {
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
    border-radius: 4px;
    padding: 2px 6px;
  }
  .to-chip:hover {
    background: #f1f3f4;
  }
  .dropdown-arrow {
    font-size: 8px;
  }
  .badges-row {
    display: flex;
    gap: 8px;
  }
  .badge {
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 12px;
  }
  .badge-success {
    background: #e6f4ea;
    color: #137333;
  }
  .details-dropdown {
    margin-top: 8px;
    padding: 12px;
    background: #f8f9fa;
    border: 1px solid #dadce0;
    border-radius: 8px;
    font-size: 12px;
    line-height: 1.6;
    color: #3c4043;
  }
  .attachments-box {
    margin-bottom: 24px;
    padding: 12px 16px;
    background: #f8f9fa;
    border-radius: 12px;
    border: 1px solid #e1e3e7;
  }
  .attachments-label {
    font-size: 13px;
    font-weight: 600;
    color: #444746;
    display: block;
    margin-bottom: 8px;
  }
  .attachment-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .att-chip {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #ffffff;
    border: 1px solid #dadce0;
    border-radius: 8px;
    padding: 6px 12px;
    font-size: 13px;
    color: #1a73e8;
    text-decoration: none;
    transition: background 0.15s, box-shadow 0.15s;
  }
  .att-chip:hover {
    background: #f1f3f4;
    box-shadow: 0 1px 3px rgba(60, 64, 67, 0.3);
  }
  .att-size {
    color: #5f6368;
    font-size: 12px;
  }
  .email-body {
    min-height: 240px;
    line-height: 1.6;
    color: #202124;
  }
  .email-frame {
    width: 100%;
    min-height: 480px;
    border: none;
  }
  .plain-text {
    font-family: inherit;
    white-space: pre-wrap;
    word-break: break-word;
    font-size: 14px;
    line-height: 1.6;
  }
  .bottom-actions {
    display: flex;
    gap: 12px;
    margin-top: 32px;
    padding-top: 24px;
    border-top: 1px solid #e1e3e7;
  }
  .action-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 24px;
    background: #ffffff;
    border: 1px solid #747775;
    border-radius: 20px;
    font-size: 14px;
    font-weight: 500;
    color: #444746;
    cursor: pointer;
    transition: background 0.15s;
  }
  .action-btn:hover {
    background: #f1f3f4;
  }
  .state-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 320px;
    gap: 12px;
    color: #5f6368;
  }
  .spinner {
    width: 28px;
    height: 28px;
    border: 3px solid #e1e3e7;
    border-top-color: #1a73e8;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>



