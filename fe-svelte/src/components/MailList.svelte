<script>
  import { onMount } from 'svelte';
  import { currentFolder, selectedEmailId, searchQuery, mailRefreshTrigger, showToast } from '../lib/stores.js';
  import { apiGetEmailList, apiMarkRead, apiDeleteEmail, apiMoveEmail } from '../lib/api.js';
  import { formatEmailDate } from '../lib/utils.js';

  let emails = [];
  let loading = false;
  let page = 1;
  let pageSize = 25;
  let total = 0;
  let selectedIds = new Set();
  let starredIds = new Set();
  let focusedIndex = 0;

  $: folderTitle = getFolderTitle($currentFolder);
  $: allSelected = emails.length > 0 && selectedIds.size === emails.length;
  $: hasSelection = selectedIds.size > 0;

  function getFolderTitle(folder) {
    const titles = {
      inbox: 'Inbox',
      sent: 'Sent',
      drafts: 'Drafts',
      trash: 'Trash',
      spam: 'Spam',
    };
    return titles[folder] || 'Mailbox';
  }

  function getCacheKey(f, p, q) {
    return `lm_cache_${f}_${p}_${q || ''}`;
  }

  function restoreCache(f, p, q) {
    try {
      const raw = sessionStorage.getItem(getCacheKey(f, p, q));
      if (raw) {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed.list)) {
          emails = parsed.list;
          total = parsed.total || emails.length;
        }
      }
    } catch (_) {}
  }

  async function loadEmails() {
    restoreCache($currentFolder, page, $searchQuery);
    loading = true;
    selectedIds.clear();
    selectedIds = selectedIds;
    try {
      const res = await apiGetEmailList({
        group: $currentFolder,
        page,
        pageSize,
        search: $searchQuery,
      });
      emails = res.list || [];
      total = res.total || (res.total_page ? res.total_page * pageSize : emails.length);
      try {
        sessionStorage.setItem(getCacheKey($currentFolder, page, $searchQuery), JSON.stringify({ list: emails, total }));
      } catch (_) {}
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      loading = false;
    }
  }

  $: $currentFolder, $searchQuery, $mailRefreshTrigger, loadEmails();

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds.clear();
    } else {
      emails.forEach((e) => selectedIds.add(e.id));
    }
    selectedIds = selectedIds;
  }

  function toggleSelect(id, event) {
    event.stopPropagation();
    if (selectedIds.has(id)) {
      selectedIds.delete(id);
    } else {
      selectedIds.add(id);
    }
    selectedIds = selectedIds;
  }

  function toggleStar(id, event) {
    event.stopPropagation();
    if (starredIds.has(id)) {
      starredIds.delete(id);
    } else {
      starredIds.add(id);
    }
    starredIds = starredIds;
  }

  async function handleMarkRead(readStatus) {
    if (!hasSelection) return;
    try {
      await apiMarkRead(Array.from(selectedIds), readStatus);
      emails = emails.map((e) => (selectedIds.has(e.id) ? { ...e, is_read: readStatus ? 1 : 0 } : e));
      showToast(readStatus ? 'Marked as read' : 'Marked as unread', 'info');
      selectedIds.clear();
      selectedIds = selectedIds;
    } catch (err) {
      showToast(err.message, 'error');
    }
  }

  async function handleDelete() {
    if (!hasSelection) return;
    const ids = Array.from(selectedIds);
    try {
      if ($currentFolder === 'trash') {
        await apiDeleteEmail(ids);
        showToast('Deleted permanently', 'info');
      } else {
        await apiMoveEmail(ids, 'trash');
        showToast('Moved to Trash', 'info');
      }
      emails = emails.filter((e) => !selectedIds.has(e.id));
      selectedIds.clear();
      selectedIds = selectedIds;
    } catch (err) {
      showToast(err.message, 'error');
    }
  }

  function openEmail(id) {
    $selectedEmailId = id;
  }

  let liveAnnouncement = '';
  function announce(msg) {
    liveAnnouncement = '';
    setTimeout(() => {
      liveAnnouncement = msg;
    }, 40);
  }

  function getCheckboxAriaLabel(e) {
    if (!e) return '';
    const readState = e.is_read ? 'Read' : 'Unread';
    const sender = e.display_sender || e.from_name || e.from_address || 'Unknown Sender';
    const subj = e.subject || '(no subject)';
    const bodySnippet = (e.desc || e.text || '').replace(/\s+/g, ' ').trim().slice(0, 250);
    const date = formatEmailDate(e.send_date || e.create_time);
    return `${readState}, ${sender}, ${subj}${bodySnippet ? ', ' + bodySnippet : ''}, ${date}`;
  }

  function getRowAriaLabel(e, isSelected) {
    if (!e) return '';
    const selState = isSelected ? 'Checked' : 'Not checked';
    const readState = e.is_read ? 'Read' : 'Unread';
    const sender = e.display_sender || e.from_name || e.from_address || 'Unknown Sender';
    const subj = e.subject || '(no subject)';
    const bodySnippet = (e.desc || e.text || '').replace(/\s+/g, ' ').trim().slice(0, 250);
    const date = formatEmailDate(e.send_date || e.create_time);
    return `${selState}. ${readState}. From: ${sender}. Subject: ${subj}. Snippet: ${bodySnippet}. Date: ${date}.`;
  }

  function handleKeydown(event) {
    if ($selectedEmailId !== null) return;
    if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') return;

    if (event.key === 'j' || event.key === 'ArrowDown') {
      event.preventDefault();
      focusedIndex = Math.min(focusedIndex + 1, emails.length - 1);
      if (emails[focusedIndex]) {
        announce(getRowAriaLabel(emails[focusedIndex], selectedIds.has(emails[focusedIndex].id)));
      }
    } else if (event.key === 'k' || event.key === 'ArrowUp') {
      event.preventDefault();
      focusedIndex = Math.max(focusedIndex - 1, 0);
      if (emails[focusedIndex]) {
        announce(getRowAriaLabel(emails[focusedIndex], selectedIds.has(emails[focusedIndex].id)));
      }
    } else if (event.key === 'x' || event.key === 'X') {
      event.preventDefault();
      if (emails[focusedIndex]) {
        const cur = emails[focusedIndex];
        toggleSelect(cur.id, event);
        const isNowSelected = selectedIds.has(cur.id);
        announce(`${isNowSelected ? 'Checked' : 'Unchecked'}: ${cur.subject || '(no subject)'}`);
      }
    } else if (event.key === 't' || event.key === 'T') {
      // Toggle read / unread for focused or selected emails
      event.preventDefault();
      const targetIds = selectedIds.size > 0 ? Array.from(selectedIds) : (emails[focusedIndex] ? [emails[focusedIndex].id] : []);
      if (targetIds.length > 0) {
        const targetEmail = emails.find(e => targetIds.includes(e.id));
        const newReadStatus = targetEmail ? !targetEmail.is_read : true;
        apiMarkRead(targetIds, newReadStatus).catch(() => {});
        emails = emails.map(e => targetIds.includes(e.id) ? { ...e, is_read: newReadStatus } : e);
        const msg = newReadStatus ? 'Marked as read' : 'Marked as unread';
        showToast(msg, 'info');
        announce(msg);
      }
    } else if (event.key === 'Enter' || event.key === 'o') {
      if (emails[focusedIndex]) {
        openEmail(emails[focusedIndex].id);
      }
    } else if (event.key === '#') {
      handleDelete();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="mail-list" role="table" aria-label="{folderTitle} email list">
  <!-- Toolbar -->
  <div class="toolbar" role="toolbar" aria-label="Mail list actions">
    <input
      type="checkbox"
      class="select-all-cb"
      checked={allSelected}
      on:change={toggleSelectAll}
      aria-label="Select all emails"
      title="Select all"
    />

    <button
      class="toolbar-btn"
      on:click={loadEmails}
      title="Refresh"
      aria-label="Refresh email list"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg>
    </button>

    {#if hasSelection}
      <button
        class="toolbar-btn"
        on:click={handleDelete}
        title="Delete"
        aria-label="Delete selected emails"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/><path d="M10 11v6M14 11v6"/><path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2"/></svg>
      </button>

      <button
        class="toolbar-btn"
        on:click={() => handleMarkRead(true)}
        title="Mark as read"
        aria-label="Mark selected emails as read"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/><polyline points="22 6 12 13 2 6"/></svg>
      </button>

      <button
        class="toolbar-btn"
        on:click={() => handleMarkRead(false)}
        title="Mark as unread"
        aria-label="Mark selected emails as unread"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/><polyline points="22 6 12 13 2 6"/><line x1="2" y1="17" x2="8" y2="12"/></svg>
      </button>
    {/if}

    <span class="folder-title">{folderTitle}</span>
  </div>

  <!-- Email rows -->
  <div class="email-rows" role="rowgroup">
    {#if loading && emails.length === 0}
      <div class="loading-placeholder" aria-live="polite" aria-label="Loading emails">
        {#each Array(8) as _}
          <div class="skeleton-row"></div>
        {/each}
      </div>
    {:else if emails.length === 0}
      <div class="empty-state" aria-label="No emails in {folderTitle}">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
        <p>No messages here</p>
      </div>
    {:else}
      {#each emails as email, i (email.id)}
        {@const isSelected = selectedIds.has(email.id)}
        {@const isStarred = starredIds.has(email.id)}
        {@const isUnread = !email.is_read}
        {@const isFocused = i === focusedIndex}
        <div
          class="mail-row"
          role="row"
          class:unread={isUnread}
          class:selected={isSelected}
          class:focused={isFocused}
          aria-selected={isSelected}
        >
          <!-- Actions: checkbox + star (outside the clickable button) -->
          <div class="col-actions">
            <label class="cb-label">
              <input
                type="checkbox"
                checked={isSelected}
                on:change={(e) => toggleSelect(email.id, e)}
                aria-label={getCheckboxAriaLabel(email)}
              />
            </label>
            <button
              class="star-btn"
              on:click={(e) => toggleStar(email.id, e)}
              aria-label={isStarred ? 'Unstar email' : 'Star email'}
              title={isStarred ? 'Unstar' : 'Star'}
            >
              {#if isStarred}
                <svg width="16" height="16" viewBox="0 0 24 24" fill="#f4b400" stroke="#f4b400" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
              {:else}
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#bdc1c6" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
              {/if}
            </button>
          </div>

          <!-- Clickable row content -->
          <button
            class="row-content"
            on:click={() => openEmail(email.id)}
            on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openEmail(email.id); } }}
            aria-label={getRowAriaLabel(email, isSelected)}
          >
            <div class="col-sender">
              {email.display_sender || email.from_name || email.from_address || 'Unknown'}
            </div>
            <div class="col-content">
              <span class="subject">{email.subject || '(no subject)'}</span>
              {#if email.desc || email.text}
                <span class="snippet">{(email.desc || email.text).replace(/\s+/g, ' ').trim()}</span>
              {/if}
            </div>
            {#if email.attachments && email.attachments.length > 0}
              <div class="col-attachment" aria-label="Has attachment">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2"><path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"/></svg>
              </div>
            {/if}
            <div class="col-date">
              {formatEmailDate(email.send_date || email.create_time)}
            </div>
          </button>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Pagination -->
  {#if total > pageSize}
    <div class="pagination" role="navigation" aria-label="Pagination">
      <button
        class="page-btn"
        disabled={page <= 1}
        on:click={() => { page -= 1; loadEmails(); }}
        aria-label="Previous page"
      >
        ‹
      </button>
      <span class="page-info" aria-live="polite">
        {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, total)} of {total}
      </span>
      <button
        class="page-btn"
        disabled={page * pageSize >= total}
        on:click={() => { page += 1; loadEmails(); }}
        aria-label="Next page"
      >
        ›
      </button>
    </div>
  {/if}

  <!-- Accessible live region -->
  <div
    class="sr-only"
    aria-live="polite"
    aria-atomic="true"
  >{liveAnnouncement}</div>
</div>

<style>
  .mail-list {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #fff;
    overflow: hidden;
  }

  /* Toolbar */
  .toolbar {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 16px;
    border-bottom: 1px solid #e0e0e0;
    flex-shrink: 0;
  }

  .select-all-cb {
    width: 16px;
    height: 16px;
    cursor: pointer;
    accent-color: #1a73e8;
    flex-shrink: 0;
  }

  .toolbar-btn {
    background: none;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 6px;
    border-radius: 50%;
    cursor: pointer;
    color: #5f6368;
    transition: background 0.15s, color 0.15s;
  }

  .toolbar-btn:hover {
    background: #f1f3f4;
    color: #202124;
  }

  .folder-title {
    margin-left: auto;
    font-size: 13px;
    color: #5f6368;
    font-weight: 500;
  }

  /* Email rows container */
  .email-rows {
    flex: 1;
    overflow-y: auto;
  }

  /* Mail row */
  .mail-row {
    display: flex;
    align-items: center;
    height: 40px;
    border-bottom: 1px solid #f1f3f4;
    font-size: 14px;
    color: #202124;
    transition: box-shadow 0.15s, background 0.15s;
    user-select: none;
  }

  .mail-row:hover {
    box-shadow: inset 1px 0 0 #dadce0, inset -1px 0 0 #dadce0, 0 1px 2px rgba(60,64,67,.3), 0 1px 3px rgba(60,64,67,.15);
    z-index: 1;
    background: #f2f6fc;
  }

  .mail-row.selected {
    background: #e8f0fe;
  }

  .mail-row.selected:hover {
    background: #d2e3fc;
  }

  .mail-row.unread .col-sender,
  .mail-row.unread .subject {
    font-weight: 700;
  }

  .mail-row.focused {
    outline: 2px solid #1a73e8;
    outline-offset: -2px;
  }

  /* Col actions */
  .col-actions {
    display: flex;
    align-items: center;
    gap: 4px;
    width: 60px;
    flex-shrink: 0;
    padding-left: 16px;
  }

  .cb-label {
    display: flex;
    align-items: center;
    cursor: pointer;
  }

  .cb-label input[type="checkbox"] {
    width: 16px;
    height: 16px;
    cursor: pointer;
    accent-color: #1a73e8;
  }

  .star-btn {
    background: none;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    padding: 0;
    flex-shrink: 0;
  }

  .star-btn:hover svg {
    stroke: #f4b400;
  }

  /* Row content button */
  .row-content {
    flex: 1;
    display: flex;
    align-items: center;
    height: 100%;
    padding: 0 16px 0 4px;
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
    color: inherit;
    font: inherit;
    min-width: 0;
  }

  .row-content:focus-visible {
    outline: 2px solid #1a73e8;
    outline-offset: -2px;
  }

  /* Columns inside row-content */
  .col-sender {
    width: 180px;
    flex-shrink: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 14px;
  }

  .col-content {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    padding: 0 16px;
    display: flex;
    gap: 6px;
    min-width: 0;
  }

  .subject {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex-shrink: 0;
    max-width: 50%;
  }

  .snippet {
    color: #5f6368;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 400;
  }

  .col-attachment {
    display: flex;
    align-items: center;
    padding-right: 12px;
    flex-shrink: 0;
  }

  .col-date {
    flex-shrink: 0;
    font-size: 12px;
    color: #5f6368;
    white-space: nowrap;
  }

  .mail-row.unread .col-date {
    font-weight: 700;
    color: #202124;
  }

  /* Loading skeleton */
  .loading-placeholder {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .skeleton-row {
    height: 40px;
    background: linear-gradient(90deg, #f1f3f4 25%, #e8eaed 50%, #f1f3f4 75%);
    background-size: 200% 100%;
    animation: shimmer 1.4s infinite;
  }

  @keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }

  /* Empty state */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 200px;
    color: #5f6368;
    gap: 12px;
    font-size: 14px;
  }

  .empty-state p {
    margin: 0;
  }

  /* Pagination */
  .pagination {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    padding: 4px 16px;
    border-top: 1px solid #e0e0e0;
    gap: 8px;
    flex-shrink: 0;
  }

  .page-btn {
    background: none;
    border: none;
    font-size: 20px;
    cursor: pointer;
    padding: 2px 8px;
    border-radius: 50%;
    color: #5f6368;
    line-height: 1;
  }

  .page-btn:hover:not(:disabled) {
    background: #f1f3f4;
    color: #202124;
  }

  .page-btn:disabled {
    color: #dadce0;
    cursor: default;
  }

  .page-info {
    font-size: 12px;
    color: #5f6368;
  }

  /* Screen reader only */
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
</style>

