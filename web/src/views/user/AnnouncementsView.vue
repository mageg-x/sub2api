<template>
  <ElCard shadow="never">
    <template #header>
      <div style="display: flex; align-items: center; gap: 8px">
        <Bell :size="16" />
        <span>平台公告</span>
      </div>
    </template>
    <ElEmpty v-if="items.length === 0" description="暂无公告" />
    <ElTimeline v-else>
      <ElTimelineItem v-for="item in items" :key="item.id" :timestamp="formatTime(item.published_at_ms)" placement="top">
        <ElCard shadow="never">
          <h3 style="margin: 0 0 8px">{{ item.title }}</h3>
          <div>{{ item.content }}</div>
        </ElCard>
      </ElTimelineItem>
    </ElTimeline>
  </ElCard>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Bell } from "lucide-vue-next";
import { ElCard, ElEmpty, ElTimeline, ElTimelineItem } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Announcement } from "@/api/types";
import { formatTime } from "@/utils";

const items = ref<Announcement[]>([]);

async function load() {
  items.value = await adminAPI.announcements();
}

onMounted(() => {
  void load();
});
</script>
