<template>
  <el-dropdown @command="handleCommand" trigger="click" popper-class="language-switcher-popper">
    <el-button class="language-switcher" text>
      <span class="language-switcher__icon">
        <Languages :size="15" />
      </span>
      <span class="language-switcher__label">{{ currentLangLabel }}</span>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item v-for="lang in languages" :key="lang.value" :command="lang.value" :disabled="locale === lang.value">
          <span class="language-option__label">{{ lang.label }}</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Languages } from "lucide-vue-next";

const { locale } = useI18n();

const languages = [
  { label: "简体中文", value: "zh-CN" },
  { label: "English", value: "en-US" },
];

const currentLangLabel = computed(() => {
  const lang = languages.find((l) => l.value === locale.value);
  return lang ? lang.label : "简体中文";
});

const handleCommand = (command: string) => {
  locale.value = command;
  localStorage.setItem("locale", command);
};
</script>

<style scoped>
.language-switcher {
  height: 34px;
  padding: 0 12px;
  border-radius: var(--radius-full);
  border: 1px solid var(--border-default);
  background: rgba(255, 255, 255, 0.72);
  color: var(--text-secondary);
  box-shadow: var(--shadow-xs);
  backdrop-filter: blur(10px);
  transition: all var(--transition-fast);
}

.language-switcher:hover {
  color: var(--primary-color);
  border-color: var(--border-focus);
  background: rgba(255, 255, 255, 0.94);
  box-shadow: var(--shadow-sm);
  transform: translateY(-1px);
}

.language-switcher__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
  color: var(--primary-color);
}

.language-switcher__label {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.01em;
}

.language-option {
  min-width: 112px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
}

.language-option__label {
  color: var(--text-primary);
  font-weight: 600;
}

@media (max-width: 640px) {
  .language-switcher {
    padding: 0 10px;
  }

  .language-switcher__label {
    font-size: 11px;
  }
}
</style>
