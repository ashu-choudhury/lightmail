<script>
  import { onMount } from 'svelte';
  import { showToast } from '../lib/stores.js';
  import {
    apiGetDomainList,
    apiAddDomain,
    apiDeleteDomain,
    apiCheckDomainDNS,
    apiApplyCloudflare,
    apiGetCloudflareSettings,
    apiSaveCloudflareSettings,
  } from '../lib/api.js';

  let domains = [];
  let loading = false;
  let newDomainName = '';
  let adding = false;

  // Live DNS check state by domain name
  let dnsResults = {};
  let checkingDns = {};

  // Cloudflare API state
  let cfToken = '';
  let cfEmail = '';
  let savingCf = false;
  let applyingCf = {};

  // Copy feedback
  let copiedText = '';

  async function loadDomains() {
    loading = true;
    try {
      domains = await apiGetDomainList();
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      loading = false;
    }
  }

  async function loadCloudflareSettings() {
    try {
      const res = await apiGetCloudflareSettings();
      if (res) {
        if (res.token) cfToken = res.token;
        if (res.email) cfEmail = res.email;
      }
    } catch (err) {
      console.warn('Failed to load Cloudflare settings:', err);
    }
  }

  onMount(async () => {
    await Promise.all([loadDomains(), loadCloudflareSettings()]);
  });

  async function handleAddDomain() {
    const d = newDomainName.trim().toLowerCase();
    if (!d) {
      showToast('Please enter a valid domain name', 'error');
      return;
    }

    adding = true;
    try {
      await apiAddDomain(d);
      showToast(`Domain ${d} added and DKIM keys generated!`, 'success');
      newDomainName = '';
      await loadDomains();
      // Automatically run DNS check for the new domain
      handleCheckDNS(d);
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      adding = false;
    }
  }

  async function handleDeleteDomain(domainName) {
    if (!confirm(`Are you sure you want to remove domain "${domainName}" from this mail server?`)) {
      return;
    }

    try {
      await apiDeleteDomain(domainName);
      showToast(`Domain ${domainName} removed`, 'info');
      await loadDomains();
    } catch (err) {
      showToast(err.message, 'error');
    }
  }

  async function handleCheckDNS(domainName) {
    checkingDns[domainName] = true;
    checkingDns = checkingDns;
    try {
      const res = await apiCheckDomainDNS(domainName);
      dnsResults[domainName] = res;
      dnsResults = dnsResults;
      showToast(`DNS check complete for ${domainName}`, 'info');
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      checkingDns[domainName] = false;
      checkingDns = checkingDns;
    }
  }

  async function handleSaveCloudflareSettings() {
    if (!cfToken.trim()) {
      showToast('Please enter your Cloudflare API Token or Global Key', 'error');
      return;
    }
    savingCf = true;
    try {
      await apiSaveCloudflareSettings(cfToken.trim(), cfEmail.trim());
      showToast('Cloudflare credentials saved to database permanently!', 'success');
    } catch (err) {
      showToast(`Failed to save Cloudflare settings: ${err.message}`, 'error');
    } finally {
      savingCf = false;
    }
  }

  async function handleApplyCloudflare(domainName) {
    if (!cfToken.trim()) {
      showToast('Please enter your Cloudflare API Token or Global Key first', 'error');
      return;
    }

    applyingCf[domainName] = true;
    applyingCf = applyingCf;
    try {
      const res = await apiApplyCloudflare(cfToken.trim(), domainName, cfEmail.trim());
      showToast(`Successfully created ${res.count} DNS records in Cloudflare!`, 'success');
      await loadCloudflareSettings();
      // Re-check DNS
      setTimeout(() => handleCheckDNS(domainName), 2000);
    } catch (err) {
      showToast(`Cloudflare sync failed: ${err.message}`, 'error');
    } finally {
      applyingCf[domainName] = false;
      applyingCf = applyingCf;
    }
  }

  function copyToClipboard(text, label) {
    navigator.clipboard.writeText(text);
    copiedText = text;
    showToast(`Copied ${label} to clipboard`, 'info');
    setTimeout(() => {
      if (copiedText === text) copiedText = '';
    }, 2500);
  }
</script>

<div class="domains-container">
  <div class="page-header">
    <div>
      <h2>Multi-Domain & DNS Configuration</h2>
      <p class="subtitle">Serve multiple domains, verify DKIM/SPF/DMARC records, and provision to Cloudflare</p>
    </div>
  </div>

  <!-- Add Domain Card -->
  <div class="card add-card">
    <div class="card-header">
      <h3>Add New Mail Domain</h3>
      <p>Configure a new root domain or subdomain to receive and send verified emails</p>
    </div>
    <form class="add-form" on:submit|preventDefault={handleAddDomain}>
      <input
        type="text"
        placeholder="e.g. yourdomain.com or mail.yourdomain.com"
        bind:value={newDomainName}
        required
      />
      <button type="submit" class="primary-btn" disabled={adding}>
        {adding ? 'Configuring & Generating DKIM...' : 'Add Domain'}
      </button>
    </form>
  </div>

  <!-- Cloudflare Global Token Helper -->
  <div class="card cf-token-card">
    <div class="cf-header">
      <span class="cf-icon">&#x2601;</span>
      <div>
        <strong>Cloudflare 1-Click DNS Sync</strong>
        <p>Credentials persist permanently in the SQLite database. Supports Cloudflare API Tokens (recommended) and Global API Keys.</p>
      </div>
    </div>
    <div class="cf-token-grid">
      <div class="cf-field">
        <label for="cf-token-input">API Token or Global API Key</label>
        <input
          id="cf-token-input"
          type="password"
          placeholder="Cloudflare API Token or Global API Key"
          bind:value={cfToken}
        />
      </div>
      <div class="cf-field">
        <label for="cf-email-input">Cloudflare Account Email <span class="field-hint">(Required ONLY for Global API Key)</span></label>
        <input
          id="cf-email-input"
          type="email"
          placeholder="e.g. your-email@example.com"
          bind:value={cfEmail}
        />
      </div>
      <div class="cf-actions">
        <button
          type="button"
          class="primary-btn cf-save-btn"
          on:click={handleSaveCloudflareSettings}
          disabled={savingCf}
        >
          {savingCf ? 'Saving...' : 'Save Credentials'}
        </button>
        <a
          href="https://dash.cloudflare.com/profile/api-tokens"
          target="_blank"
          rel="noreferrer"
          class="secondary-link"
        >
          Create API Token in Cloudflare ↗
        </a>
      </div>
    </div>
    <div class="cf-hint">
      <strong>Authentication Notice:</strong> If using a <strong>Custom API Token</strong> (recommended), grant <code>Zone:DNS:Edit</code> permission (leave Email blank). If using a legacy 37-character <strong>Global API Key</strong>, you must enter your Cloudflare account email.
    </div>
  </div>

  <!-- Configured Domains List -->
  {#if loading}
    <div class="loading-state">Loading configured domains...</div>
  {:else}
    <div class="domains-list">
      {#each domains as d (d.name)}
        <div class="card domain-card">
          <!-- Domain Header -->
          <div class="domain-header">
            <div class="title-row">
              <h3>{d.name}</h3>
              {#if d.primary}
                <span class="badge primary">Primary Domain</span>
              {/if}
              {#if d.dkim_ready}
                <span class="badge dkim-ready">DKIM Key Ready</span>
              {/if}
            </div>

            <div class="domain-actions">
              <button
                class="secondary-btn"
                on:click={() => handleCheckDNS(d.name)}
                disabled={checkingDns[d.name]}
              >
                {checkingDns[d.name] ? 'Checking DNS...' : 'Live DNS Verification'}
              </button>

              {#if cfToken}
                <button
                  class="cf-btn"
                  on:click={() => handleApplyCloudflare(d.name)}
                  disabled={applyingCf[d.name]}
                >
                  {applyingCf[d.name] ? 'Applying...' : 'Apply to Cloudflare'}
                </button>
              {/if}

              <a
                href="https://dash.cloudflare.com"
                target="_blank"
                rel="noreferrer"
                class="cf-link-btn"
                title="Open Cloudflare Dashboard"
              >
                Open Cloudflare ↗
              </a>

              {#if !d.primary}
                <button
                  class="danger-btn"
                  on:click={() => handleDeleteDomain(d.name)}
                  title="Remove domain"
                >
                  Remove
                </button>
              {/if}
            </div>
          </div>

          <!-- Live DNS Status Badges (if checked) -->
          {#if dnsResults[d.name]}
            {@const res = dnsResults[d.name]}
            <div class="dns-status-panel">
              <span class="panel-title">Live DNS Health Check:</span>
              <div class="status-chips">
                <span class="status-chip" class:valid={res.resolves}>
                  {res.resolves ? 'Host IP Resolves' : 'No A/AAAA Record'}
                </span>
                <span class="status-chip" class:valid={res.mx_valid}>
                  {res.mx_valid ? 'MX Routing Active' : 'MX Record Missing'}
                </span>
                <span class="status-chip" class:valid={res.spf_valid}>
                  {res.spf_valid ? 'SPF Configured' : 'SPF Missing'}
                </span>
                <span class="status-chip" class:valid={res.dkim_valid}>
                  {res.dkim_valid ? 'DKIM Published' : 'DKIM Missing'}
                </span>
                <span class="status-chip" class:valid={res.dmarc_valid}>
                  {res.dmarc_valid ? 'DMARC Active' : 'DMARC Missing'}
                </span>
              </div>
            </div>
          {/if}

          <!-- DNS Records Table -->
          <div class="records-section">
            <span class="section-title">Required DNS Records for 100% Deliverability:</span>
            <table class="records-table">
              <thead>
                <tr>
                  <th style="width: 80px">Type</th>
                  <th style="width: 220px">Host / Name</th>
                  <th>Value / Destination</th>
                  <th style="width: 70px">TTL</th>
                  <th style="width: 90px">Action</th>
                </tr>
              </thead>
              <tbody>
                {#each d.records as rec}
                  <tr>
                    <td><span class="type-tag">{rec.type}</span></td>
                    <td>
                      <code>{rec.host === '@' ? d.name : `${rec.host}.${d.name}`}</code>
                    </td>
                    <td class="value-cell">
                      <code class="value-code">{rec.value}</code>
                    </td>
                    <td>{rec.ttl || 3600}</td>
                    <td>
                      <button
                        class="copy-btn"
                        on:click={() => copyToClipboard(rec.value, `${rec.type} record`)}
                      >
                        {copiedText === rec.value ? 'Copied!' : 'Copy'}
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .domains-container {
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
    box-shadow: 0 1px 3px rgba(60, 64, 67, 0.05);
  }
  .add-card h3 {
    margin: 0 0 4px;
    font-size: 16px;
    color: #1f1f1f;
  }
  .add-card p {
    margin: 0 0 16px;
    font-size: 13px;
    color: #5f6368;
  }
  .add-form {
    display: flex;
    gap: 12px;
  }
  .add-form input {
    flex: 1;
    max-width: 480px;
    padding: 10px 14px;
    border: 1px solid #dadce0;
    border-radius: 8px;
    font-size: 14px;
    outline: none;
  }
  .add-form input:focus {
    border-color: #1a73e8;
    box-shadow: 0 0 0 2px rgba(26, 115, 232, 0.2);
  }
  .primary-btn {
    background: #0b57d0;
    color: #fff;
    border: none;
    border-radius: 8px;
    padding: 10px 20px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }
  .primary-btn:hover {
    background: #0842a0;
  }
  .secondary-btn {
    background: #ffffff;
    border: 1px solid #dadce0;
    border-radius: 6px;
    padding: 6px 14px;
    font-size: 13px;
    font-weight: 500;
    color: #444746;
    cursor: pointer;
  }
  .secondary-btn:hover {
    background: #f1f3f4;
  }
  .cf-btn {
    background: #f38020;
    color: #ffffff;
    border: none;
    border-radius: 6px;
    padding: 6px 14px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }
  .cf-btn:hover {
    background: #e56f10;
  }
  .cf-link-btn {
    display: inline-flex;
    align-items: center;
    background: #f1f3f4;
    color: #444746;
    border-radius: 6px;
    padding: 6px 12px;
    font-size: 13px;
    font-weight: 500;
    text-decoration: none;
  }
  .cf-link-btn:hover {
    background: #e8ebf0;
  }
  .danger-btn {
    background: none;
    border: 1px solid #dadce0;
    border-radius: 6px;
    padding: 6px 10px;
    cursor: pointer;
  }
  .cf-token-card {
    background: #fff8f0;
    border-color: #ffd8a8;
  }
  .cf-header {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    margin-bottom: 12px;
  }
  .cf-icon {
    font-size: 24px;
  }
  .cf-header p {
    margin: 2px 0 0;
    font-size: 12px;
    color: #5f6368;
  }
  .cf-token-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    align-items: flex-end;
  }
  .cf-field {
    display: flex;
    flex-direction: column;
    gap: 6px;
    flex: 1;
    min-width: 260px;
  }
  .cf-field label {
    font-size: 12px;
    font-weight: 600;
    color: #444746;
  }
  .field-hint {
    font-size: 11px;
    font-weight: 400;
    color: #5f6368;
  }
  .cf-field input {
    width: 100%;
    padding: 8px 12px;
    border: 1px solid #dadce0;
    border-radius: 6px;
    font-size: 13px;
    background: #fff;
    box-sizing: border-box;
  }
  .cf-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .cf-save-btn {
    padding: 8px 16px;
    font-size: 13px;
  }
  .cf-hint {
    margin-top: 14px;
    font-size: 12px;
    color: #704a00;
    line-height: 1.5;
    background: rgba(243, 128, 32, 0.08);
    padding: 8px 12px;
    border-radius: 6px;
  }
  .cf-hint code {
    background: rgba(0, 0, 0, 0.06);
    padding: 2px 5px;
    border-radius: 4px;
    font-family: monospace;
  }
  .secondary-link {
    color: #0b57d0;
    font-size: 13px;
    text-decoration: none;
    white-space: nowrap;
  }
  .secondary-link:hover {
    text-decoration: underline;
  }
  .domain-card {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
  .domain-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .title-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .title-row h3 {
    margin: 0;
    font-size: 18px;
    color: #1f1f1f;
  }
  .badge {
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 12px;
  }
  .badge.primary {
    background: #e8f0fe;
    color: #1a73e8;
  }
  .badge.dkim-ready {
    background: #e6f4ea;
    color: #137333;
  }
  .domain-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .dns-status-panel {
    background: #f8fafd;
    border-radius: 8px;
    padding: 12px 16px;
    border: 1px solid #e1e3e7;
  }
  .panel-title {
    font-size: 12px;
    font-weight: 600;
    color: #5f6368;
    display: block;
    margin-bottom: 8px;
  }
  .status-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .status-chip {
    font-size: 12px;
    font-weight: 600;
    padding: 3px 10px;
    border-radius: 12px;
    background: #fce8e6;
    color: #c5221f;
  }
  .status-chip.valid {
    background: #e6f4ea;
    color: #137333;
  }
  .records-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .section-title {
    font-size: 13px;
    font-weight: 600;
    color: #444746;
  }
  .records-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  .records-table th {
    background: #f1f3f4;
    padding: 8px 12px;
    text-align: left;
    color: #5f6368;
    font-weight: 600;
    border-bottom: 1px solid #dadce0;
  }
  .records-table td {
    padding: 10px 12px;
    border-bottom: 1px solid #f1f3f4;
    vertical-align: middle;
  }
  .type-tag {
    font-weight: 700;
    color: #1a73e8;
  }
  .value-cell {
    max-width: 420px;
  }
  .value-code {
    display: block;
    word-break: break-all;
    font-size: 12px;
    color: #3c4043;
    background: #f8f9fa;
    padding: 4px 8px;
    border-radius: 4px;
  }
  .copy-btn {
    background: #f1f3f4;
    border: 1px solid #dadce0;
    border-radius: 4px;
    padding: 4px 8px;
    font-size: 11px;
    font-weight: 600;
    color: #1f1f1f;
    cursor: pointer;
  }
  .copy-btn:hover {
    background: #e8ebf0;
  }
  .loading-state {
    padding: 48px;
    text-align: center;
    color: #5f6368;
  }
</style>



