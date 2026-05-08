<template>
  <div>
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <CircleDollarSign :size="24" />
          </div>
        </div>
        <p class="stat-label">{{ t('adminPrices.priceRecords') }}</p>
        <p class="stat-value">{{ prices.length }}</p>
        <p class="stat-helper">{{ t('adminPrices.priceCount') }}</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon active">
            <Layers :size="24" />
          </div>
        </div>
        <p class="stat-label">{{ t('adminPrices.providerCount') }}</p>
        <p class="stat-value">{{ providerCount }}</p>
        <p class="stat-helper">{{ t('adminPrices.providersConnected') }}</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon currency">
            <Coins :size="24" />
          </div>
        </div>
        <p class="stat-label">{{ t('adminPrices.enabledCurrencies') }}</p>
        <p class="stat-value">{{ new Set(prices.map((item) => item.currency)).size }}</p>
        <p class="stat-helper">{{ t('adminPrices.supportedCurrencies') }}</p>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <CircleDollarSign :size="20" />
          {{ t('adminPrices.addModelPrice') }}
        </h3>
      </div>
      <div class="card-body">
        <el-form label-position="top" class="modern-form">
          <div class="form-grid">
            <el-form-item :label="t('adminPrices.provider')">
              <el-input v-model="form.provider" :placeholder="t('adminPrices.providerPlaceholder')">
                <template #prefix><Server :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminPrices.modelName')">
              <el-input v-model="form.model" :placeholder="t('adminPrices.modelPlaceholder')">
                <template #prefix><Bot :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminPrices.inputPrice')">
              <el-input v-model.number="form.input_price" type="number" :placeholder="t('adminPrices.inputPricePlaceholder')">
                <template #prefix><ArrowDownToLine :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminPrices.outputPrice')">
              <el-input v-model.number="form.output_price" type="number" :placeholder="t('adminPrices.outputPricePlaceholder')">
                <template #prefix><ArrowUpFromLine :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminPrices.cacheCreatePrice')">
              <el-input v-model.number="form.cache_create_price" type="number" :placeholder="t('adminPrices.cacheCreatePricePlaceholder')">
                <template #prefix><Database :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminPrices.cacheReadPrice')">
              <el-input v-model.number="form.cache_read_price" type="number" :placeholder="t('adminPrices.cacheReadPricePlaceholder')">
                <template #prefix><DatabaseZap :size="16" /></template>
              </el-input>
            </el-form-item>
          </div>
          <el-button type="primary" @click="create">
            <Plus :size="16" style="margin-right: 6px" />
            {{ t('adminPrices.createPrice') }}
          </el-button>
        </el-form>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Coins :size="20" />
          {{ t('adminPrices.priceList') }}
        </h3>
        <span class="price-count">{{ prices.length }} {{ t('adminPrices.priceCountLabel') }}</span>
      </div>
      <div class="card-body">
        <el-table :data="prices" :empty-text="t('adminPrices.noPrices')" class="modern-table" :stripe="true">
          <el-table-column prop="provider" :label="t('adminPrices.provider')" width="110">
            <template #default="{ row }">
              <div class="provider-cell">
                <div class="provider-badge" :class="row.provider.toLowerCase()">
                  {{ row.provider.charAt(0) }}
                </div>
                <span>{{ row.provider }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="model" :label="t('models.model')" min-width="160">
            <template #default="{ row }">
              <code class="model-name mono">{{ row.model }}</code>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminPrices.inputPer1k')" width="110" align="right">
            <template #default="{ row }">
              <span class="price-value">{{ row.input_price }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminPrices.outputPer1k')" width="110" align="right">
            <template #default="{ row }">
              <span class="price-value">{{ row.output_price }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminPrices.cacheCreatePer1k')" width="120" align="right">
            <template #default="{ row }">
              <span class="price-value">{{ row.cache_create_price }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminPrices.cacheReadPer1k')" width="120" align="right">
            <template #default="{ row }">
              <span class="price-value">{{ row.cache_read_price }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="currency" :label="t('adminPrices.currency')" width="80">
            <template #default="{ row }">
              <el-tag type="info" size="small">{{ row.currency }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('common.status')" width="80">
            <template #default="{ row }">
              <el-tag :type="isActiveStatus(row.status) ? 'success' : 'info'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ArrowDownToLine, ArrowUpFromLine, Bot, CircleDollarSign, Coins, Database, DatabaseZap, Layers, Plus, Server } from "lucide-vue-next";
import { ElButton, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { ModelPrice } from "@/api/types";
import { isActiveStatus } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const prices = ref<ModelPrice[]>([]);
const form = reactive({
  provider: "openai",
  model: "",
  input_price: 0,
  output_price: 0,
  cache_create_price: 0,
  cache_read_price: 0,
});

const providerCount = computed(() => new Set(prices.value.map((item) => item.provider)).size);

async function load() {
  try {
    prices.value = await adminAPI.prices();
  } catch {
    prices.value = [];
  }
}

async function create() {
  if (!form.provider.trim() || !form.model.trim()) return;
  try {
    await adminAPI.createPrice({
      provider: form.provider,
      model: form.model,
      input_price: Number(form.input_price || 0),
      output_price: Number(form.output_price || 0),
      cache_create_price: Number(form.cache_create_price || 0),
      cache_read_price: Number(form.cache_read_price || 0),
    });
    form.model = "";
    form.input_price = 0;
    form.output_price = 0;
    form.cache_create_price = 0;
    form.cache_read_price = 0;
    await load();
  } catch {
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.form-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.price-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.provider-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.provider-badge {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 14px;
  color: white;
}

.provider-badge.openai {
  background: linear-gradient(135deg, #10a37f, #10b981);
}

.provider-badge.claude {
  background: linear-gradient(135deg, #d4a574, #e8c49a);
}

.provider-badge.gemini {
  background: linear-gradient(135deg, #4285f4, #667eea);
}

.provider-badge.antigravity {
  background: linear-gradient(135deg, #8b5cf6, #6d28d9);
}

.model-name {
  font-size: 13px;
  background: var(--border-light);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
}

.price-value {
  font-size: 14px;
  font-weight: 700;
  color: var(--primary-color);
  font-family: var(--font-mono, monospace);
}
</style>
