<template>
  <ElCard shadow="never">
    <template #header>
      <div style="display: flex; align-items: center; gap: 8px">
        <Bug :size="16" />
        <span>错误日志</span>
      </div>
    </template>
    <ElTable :data="items">
      <ElTableColumn prop="scope" label="范围" width="140" />
      <ElTableColumn prop="message" label="错误消息" min-width="220" />
      <ElTableColumn prop="detail" label="详情" min-width="260" />
      <ElTableColumn prop="count" label="次数" width="90" />
      <ElTableColumn label="最近出现" min-width="180">
        <template #default="{ row }">
          {{ formatTime(Number(row.last_seen_at_ms || 0)) }}
        </template>
      </ElTableColumn>
    </ElTable>
  </ElCard>
</template>


<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Bug } from "lucide-vue-next";
import { ElCard, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import { formatTime } from "@/utils";

const items = ref<Array<Record<string, unknown>>>([]);

async function load() {
  items.value = await adminAPI.errors();
}

onMounted(() => {
  void load();
});
</script>

