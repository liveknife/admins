<script setup lang="ts">
import { useI18n, type Locale } from "@/i18n";
import Translate from "~icons/ri/translate-2";

const { locale, localeOptions, setLocale, t } = useI18n();

const handleCommand = (value: Locale) => {
  setLocale(value);
};
</script>

<template>
  <el-dropdown trigger="click" @command="handleCommand">
    <span
      class="language-switch navbar-bg-hover"
      :title="t('language.label')"
    >
      <IconifyIconOffline :icon="Translate" />
      <span>{{ locale === "zh-CN" ? t("language.zh") : t("language.en") }}</span>
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="item in localeOptions"
          :key="item.value"
          :command="item.value"
          :disabled="item.value === locale"
        >
          {{ item.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<style scoped>
.language-switch {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  height: 48px;
  padding: 0 10px;
  font-size: 14px;
  color: #000000d9;
  cursor: pointer;
}

.language-switch svg {
  width: 16px;
  height: 16px;
}
</style>
