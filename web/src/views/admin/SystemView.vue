<template>
  <ElCard shadow="never">
    <template #header>
      <div style="display: flex; align-items: center; gap: 8px">
        <Activity :size="16" />
        <span>系统指标</span>
      </div>
    </template>
    <ElDescriptions :column="2" border>
      <ElDescriptionsItem v-for="(value, key) in stats" :key="String(key)" :label="String(key)">
        {{ value }}
      </ElDescriptionsItem>
    </ElDescriptions>
  </ElCard>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Activity } from "lucide-vue-next";
import { ElCard, ElDescriptions, ElDescriptionsItem } from "element-plus";
import { adminAPI } from "@/api/admin";

const stats = ref<Record<string, unknown>>({});

async function load() {
  stats.value = await adminAPI.stats();
}

onMounted(() => {
  void load();
});
</script>
