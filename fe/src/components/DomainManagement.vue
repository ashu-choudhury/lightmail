<template>
  <div class="settings-card">
    <div class="settings-header">
      <h3>{{ lang.domain_management }}</h3>
      <p class="settings-desc">{{ lang.domain_management_desc }}</p>
    </div>

    <div class="table-container">
      <el-table :data="domains" class="modern-table" style="width: 100%">
        <el-table-column :label="lang.domain" prop="name" min-width="220" show-overflow-tooltip>
          <template #default="scope">
            <span class="domain-name">{{ scope.row.name }}</span>
            <el-tag v-if="scope.row.primary" size="small" effect="plain" class="inline-tag">
              {{ lang.primary_domain }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="lang.dkim_status" width="130">
          <template #default="scope">
            <el-tag :type="scope.row.dkim_ready ? 'success' : 'warning'" size="small" effect="plain"
                    class="inline-tag">
              {{ scope.row.dkim_ready ? lang.dkim_ready : lang.dkim_missing }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="right" width="240">
          <template #header>
            <el-button type="primary" size="small" plain @click="openCreate" class="new-btn">
              <el-icon>
                <Plus/>
              </el-icon>
              {{ lang.add_domain }}
            </el-button>
          </template>
          <template #default="scope">
            <el-button size="small" type="primary" text bg class="action-btn" @click="openRecords(scope.row)">
              {{ lang.dns_records }}
            </el-button>
            <el-button size="small" text bg class="action-btn" :disabled="scope.row.dkim_ready"
                       @click="ensureDkim(scope.row, false)">
              {{ lang.generate_dkim }}
            </el-button>
            <el-button size="small" type="danger" text bg class="action-btn" :disabled="scope.row.primary"
                       @click="removeDomain(scope.row)">
              {{ lang.del_btn }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="createDialog" :title="lang.new_domain" width="450px" class="premium-dialog">
      <div class="dialog-content">
        <el-form label-position="top">
          <el-form-item :label="lang.domain_name">
            <el-input v-model="newDomain" :placeholder="lang.domain_name_ph" @keyup.enter="submitCreate"/>
          </el-form-item>
        </el-form>
        <p class="settings-desc">{{ lang.new_domain_desc }}</p>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDialog = false">Cancel</el-button>
          <el-button type="primary" :loading="busy" @click="submitCreate">Confirm</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="recordsDialog" :title="activeDomain.name || lang.dns_records" width="680px"
               class="premium-dialog">
      <div class="dialog-content">
        <el-alert v-if="!activeDomain.dkim_ready" type="warning" show-icon :closable="false"
                  :title="lang.dkim_missing_tip" class="records-alert"/>
        <p class="settings-desc">{{ lang.records_tip }}</p>
        <el-table :data="activeDomain.records || []" size="small" class="records-table">
          <el-table-column :label="lang.type" prop="type" width="80"/>
          <el-table-column :label="lang.dns_host" prop="host" width="170" show-overflow-tooltip/>
          <el-table-column :label="lang.dns_value" prop="value" show-overflow-tooltip/>
        </el-table>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button text bg @click="copyRecords">{{ lang.copy_records }}</el-button>
          <el-button :loading="busy" @click="ensureDkim(activeDomain, true)">
            {{ lang.rotate_dkim }}
          </el-button>
          <el-button type="primary" @click="recordsDialog = false">OK</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import {reactive, ref} from 'vue'
import lang from '../i18n/i18n';
import {http} from "@/utils/axios";
import {ElMessageBox, ElNotification} from "element-plus";
import {Plus} from "@element-plus/icons-vue";

const domains = reactive([])
const createDialog = ref(false)
const recordsDialog = ref(false)
const newDomain = ref("")
const busy = ref(false)
const activeDomain = ref({name: "", dkim_ready: false, records: []})

const notify = function (res) {
  const ok = res.errorNo === 0
  ElNotification({
    title: ok ? lang.succ : lang.fail,
    message: ok ? "" : (res.errorMsg || ""),
    type: ok ? 'success' : 'error',
  })
  return ok
}

const reflushList = function () {
  http.post('/api/domain/list', {}).then(res => {
    if (!notify(res)) {
      return
    }
    domains.length = 0
    if (Array.isArray(res.data)) {
      domains.push(...res.data)
    }
  })
}

const openCreate = function () {
  newDomain.value = ""
  createDialog.value = true
}

const submitCreate = function () {
  if (newDomain.value === "") {
    return
  }
  busy.value = true
  http.post('/api/domain/add', {"domain": newDomain.value}).then(res => {
    busy.value = false
    if (!notify(res)) {
      return
    }
    createDialog.value = false
    reflushList()
    // 新增域名后直接展示需要配置的DNS记录
    openRecords(res.data)
  })
}

const removeDomain = function (row) {
  ElMessageBox.confirm(lang.del_domain_confirm, row.name, {type: "warning"}).then(() => {
    http.post('/api/domain/del', {"domain": row.name}).then(res => {
      if (notify(res)) {
        reflushList()
      }
    })
  }).catch(() => {
  })
}

const ensureDkim = function (row, rotate) {
  busy.value = true
  http.post('/api/domain/dkim', {"domain": row.name, "rotate": rotate}).then(res => {
    busy.value = false
    if (!notify(res)) {
      return
    }
    reflushList()
    recordsDialog.value = true
    activeDomain.value = res.data
  })
}

const openRecords = function (row) {
  activeDomain.value = row
  recordsDialog.value = true
}

const copyRecords = function () {
  const text = (activeDomain.value.records || [])
      .map(r => `${r.type}\t${r.host}\t${r.value}`)
      .join("\n")
  if (!navigator.clipboard) {
    ElNotification({title: lang.fail, message: text, type: 'info'})
    return
  }
  navigator.clipboard.writeText(text).then(() => {
    ElNotification({title: lang.succ, message: lang.records_copied, type: 'success'})
  }).catch(() => {
    ElNotification({title: lang.fail, message: text, type: 'info'})
  })
}

reflushList()
</script>

<style scoped>
.settings-card {
  padding: 0;
}

.settings-header {
  margin-bottom: 24px;
}

.settings-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--pm-text-primary);
  margin: 0 0 8px 0;
}

.settings-desc {
  font-size: 13px;
  color: var(--pm-text-secondary);
  margin: 0 0 12px 0;
  line-height: 1.5;
}

.table-container {
  border: 1px solid var(--pm-border-color);
  border-radius: var(--pm-radius-sm);
  overflow: hidden;
  margin-bottom: 24px;
}

.modern-table :deep(th.el-table__cell) {
  background-color: var(--pm-bg-secondary);
  color: var(--pm-text-secondary);
  font-weight: 600;
}

.domain-name {
  font-weight: 500;
  color: var(--pm-text-primary);
  margin-right: 8px;
}

.inline-tag {
  border-radius: var(--pm-radius-sm);
}

.new-btn,
.action-btn {
  border-radius: var(--pm-radius-sm);
}

.premium-dialog :deep(.el-dialog__header) {
  border-bottom: 1px solid var(--pm-border-color);
  padding-bottom: 16px;
  margin-bottom: 20px;
}

.dialog-content {
  padding: 0 8px;
}

.records-alert {
  margin-bottom: 12px;
}

.records-table {
  border: 1px solid var(--pm-border-color);
  border-radius: var(--pm-radius-sm);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}
</style>
