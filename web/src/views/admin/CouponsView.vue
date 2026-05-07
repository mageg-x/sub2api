<template>
  <div style="display: grid; gap: 18px">
    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <BadgePercent :size="16" />
          <span>创建兑换码</span>
        </div>
      </template>
      <ElForm label-position="top">
        <ElFormItem label="兑换码">
          <ElInput v-model="form.code" />
        </ElFormItem>
        <ElFormItem label="金额">
          <ElInput v-model.number="form.amount" />
        </ElFormItem>
        <ElFormItem label="最大使用次数">
          <ElInput v-model.number="form.max_uses" />
        </ElFormItem>
        <ElButton type="primary" @click="create">创建兑换码</ElButton>
      </ElForm>
    </ElCard>

    <ElCard shadow="never">
      <ElTable :data="items">
        <ElTableColumn prop="code" label="兑换码" min-width="160" />
        <ElTableColumn prop="amount" label="金额" width="120" />
        <ElTableColumn prop="used_count" label="已使用" width="100" />
        <ElTableColumn label="状态" width="120">
          <template #default="{ row }">
            <ElTag>{{ row.status }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="过期时间" min-width="180">
          <template #default="{ row }">
            {{ formatTime(Number(row.expires_at_ms || 0)) }}
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { BadgePercent } from "lucide-vue-next";
import { ElButton, ElCard, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Coupon } from "@/api/types";
import { formatTime } from "@/utils";

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


