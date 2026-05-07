<template>
  <div style="display: grid; gap: 18px">
    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <Megaphone :size="16" />
          <span>发布公告</span>
        </div>
      </template>
      <ElForm label-position="top">
        <ElFormItem label="标题">
          <ElInput v-model="form.title" />
        </ElFormItem>
        <ElFormItem label="内容">
          <ElInput v-model="form.content" type="textarea" :rows="4" />
        </ElFormItem>
        <ElButton type="primary" @click="create">发布公告</ElButton>
      </ElForm>
    </ElCard>

    <ElCard shadow="never">
      <ElTable :data="items">
        <ElTableColumn prop="title" label="标题" min-width="220" />
        <ElTableColumn label="状态" width="120">
          <template #default="{ row }">
            <ElTag>{{ row.status }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="发布时间" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.published_at_ms) }}
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElButton, ElCard, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { Megaphone } from "lucide-vue-next";
import { adminAPI } from "@/api/admin";
import type { Announcement } from "@/api/types";
import { formatTime } from "@/utils";

const items = ref<Announcement[]>([]);
const form = reactive({
  title: "",
  content: "",
  status: "published",
});

async function load() {
  items.value = await adminAPI.announcements();
}

async function create() {
  await adminAPI.createAnnouncement(form);
  form.title = "";
  form.content = "";
  form.status = "published";
  await load();
}

onMounted(() => {
  void load();
});
</script>


