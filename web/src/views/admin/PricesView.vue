<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">价格记录</p>
        <p class="stat-value">{{ prices.length }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">Provider 数</p>
        <p class="stat-value">{{ providerCount }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">启用币种</p>
        <p class="stat-value">
          {{ new Set(prices.map((item) => item.currency)).size }}
        </p>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <CircleDollarSign :size="16" />
          <span>添加模型价格</span>
        </div>
      </template>
      <ElForm label-position="top">
        <div style="display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px">
          <ElFormItem label="Provider">
            <ElInput v-model="form.provider" placeholder="openai / claude / gemini / antigravity" />
          </ElFormItem>
          <ElFormItem label="模型名">
            <ElInput v-model="form.model" placeholder="如 gpt-4o-mini" />
          </ElFormItem>
          <ElFormItem label="输入单价 / 1k">
            <ElInput v-model.number="form.input_price" type="number" />
          </ElFormItem>
          <ElFormItem label="输出单价 / 1k">
            <ElInput v-model.number="form.output_price" type="number" />
          </ElFormItem>
        </div>
        <ElButton type="primary" @click="create">创建价格</ElButton>
      </ElForm>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <span>价格表</span>
      </template>
      <ElTable :data="prices" empty-text="暂无价格记录">
        <ElTableColumn prop="provider" label="Provider" width="120" />
        <ElTableColumn prop="model" label="模型" min-width="220" />
        <ElTableColumn prop="input_price" label="输入单价 / 1k" min-width="140" />
        <ElTableColumn prop="output_price" label="输出单价 / 1k" min-width="140" />
        <ElTableColumn prop="currency" label="货币" width="100" />
        <ElTableColumn label="状态" width="110">
          <template #default="{ row }">
            <ElTag :type="isActiveStatus(row.status) ? 'success' : 'info'">{{ row.status }}</ElTag>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CircleDollarSign } from "lucide-vue-next";
import { ElButton, ElCard, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { ModelPrice } from "@/api/types";
import { isActiveStatus } from "@/utils";

const prices = ref<ModelPrice[]>([]);
const form = reactive({
  provider: "openai",
  model: "",
  input_price: 0,
  output_price: 0,
});

const providerCount = computed(() => new Set(prices.value.map((item) => item.provider)).size);

async function load() {
  prices.value = await adminAPI.prices();
}

async function create() {
  await adminAPI.createPrice({
    provider: form.provider,
    model: form.model,
    input_price: Number(form.input_price || 0),
    output_price: Number(form.output_price || 0),
  });
  form.model = "";
  form.input_price = 0;
  form.output_price = 0;
  await load();
}

onMounted(() => {
  void load();
});
</script>


