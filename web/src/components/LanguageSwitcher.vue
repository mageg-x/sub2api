<template>
  <el-dropdown @command="handleCommand" trigger="click">
    <el-button type="primary" text>
      <Languages :size="20" />
      <span style="margin-left: 5px">{{ currentLangLabel }}</span>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item 
          v-for="lang in languages" 
          :key="lang.value" 
          :command="lang.value"
          :disabled="locale === lang.value"
        >
          {{ lang.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Languages } from 'lucide-vue-next'

const { locale, t } = useI18n()

const languages = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' }
]

const currentLangLabel = computed(() => {
  const lang = languages.find(l => l.value === locale.value)
  return lang ? lang.label : '简体中文'
})

const handleCommand = (command: string) => {
  locale.value = command
  localStorage.setItem('locale', command)
}
</script>
