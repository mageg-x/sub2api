<template>
  <div>
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <Ticket :size="24" />
          </div>
        </div>
        <p class="stat-label">兑换码总数</p>
        <p class="stat-value">{{ items.length }}</p>
        <p class="stat-helper">已创建兑换码</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon active">
            <CheckCircle :size="24" />
          </div>
        </div>
        <p class="stat-label">已激活</p>
        <p class="stat-value">{{ items.filter((item) => item.status === "active").length }}</p>
        <p class="stat-helper">可使用的</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon used">
            <Gift :size="24" />
          </div>
        </div>
        <p class="stat-label">已兑换</p>
        <p class="stat-value">{{ items.reduce((sum, item) => sum + item.used_count, 0) }}</p>
        <p class="stat-helper">累计使用次数</p>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Ticket :size="20" />
          创建兑换码
        </h3>
      </div>
      <div class="card-body">
        <el-form label-position="top" class="modern-form">
          <div class="form-grid">
            <el-form-item label="兑换码">
              <el-input v-model="form.code" placeholder="留空则自动生成">
                <template #prefix><Ticket :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item label="金额">
              <el-input v-model.number="form.amount" type="number" placeholder="充值金额（分）">
                <template #prefix><CircleDollarSign :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item label="最大使用次数">
              <el-input v-model.number="form.max_uses" type="number" placeholder="默认 1">
                <template #prefix><Hash :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item label="过期时间">
              <el-input v-model.number="form.expires_at_ms" type="number" placeholder="0 = 不过期(ms)">
                <template #prefix><Clock :size="16" /></template>
              </el-input>
            </el-form-item>
          </div>
          <el-button type="primary" @click="create">
            <Ticket :size="16" style="margin-right: 6px" />
            创建兑换码
          </el-button>
        </el-form>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Gift :size="20" />
          兑换码列表
        </h3>
        <span class="coupon-count">{{ items.length }} 个</span>
      </div>
      <div class="card-body">
        <el-table :data="items" empty-text="暂无兑换码" class="modern-table" :stripe="true">
          <el-table-column prop="code" label="兑换码" min-width="160">
            <template #default="{ row }">
              <code class="code-value mono">{{ row.code }}</code>
            </template>
          </el-table-column>
          <el-table-column label="金额" width="100">
            <template #default="{ row }">
              <span class="amount-value">{{ formatCurrency(row.amount) }} 元</span>
            </template>
          </el-table-column>
          <el-table-column label="已使用" width="80">
            <template #default="{ row }">
              <span class="used-count">{{ row.used_count }} / {{ row.max_uses }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="过期时间" width="150">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(Number(row.expires_at_ms || 0)) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { CheckCircle, CircleDollarSign, Clock, Gift, Hash, Ticket } from "lucide-vue-next";
import { ElButton, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Coupon } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";

const items = ref<Coupon[]>([]);
const form = reactive({
  code: "",
  kind: "balance",
  amount: 10000,
  max_uses: 1,
  expires_at_ms: 0,
});

async function load() {
  items.value = await adminAPI.coupons();
}

async function create() {
  await adminAPI.createCoupon(form);
  form.code = "";
  await load();
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.coupon-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.code-value {
  font-size: 13px;
  background: var(--border-light);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
}

.amount-value {
  font-size: 14px;
  font-weight: 700;
  color: var(--success-color);
}

.used-count {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}
</style>
