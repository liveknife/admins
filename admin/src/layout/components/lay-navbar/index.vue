<script setup lang="ts">
import { useNav } from "@/layout/hooks/useNav";
import LaySearch from "../lay-search/index.vue";
import LayNotice from "../lay-notice/index.vue";
import LayNavMix from "../lay-sidebar/NavMix.vue";
import LanguageSwitch from "@/components/LanguageSwitch/index.vue";
import LaySidebarFullScreen from "../lay-sidebar/components/SidebarFullScreen.vue";
import LaySidebarBreadCrumb from "../lay-sidebar/components/SidebarBreadCrumb.vue";
import LaySidebarTopCollapse from "../lay-sidebar/components/SidebarTopCollapse.vue";
import { useI18n } from "@/i18n";

import LogoutCircleRLine from "~icons/ri/logout-circle-r-line";
import UserSettings from "~icons/ri/user-settings-line";
import Setting from "~icons/ri/settings-3-line";
import { useRouter } from "vue-router";

const {
  layout,
  device,
  logout,
  onPanel,
  pureApp,
  username,
  userAvatar,
  avatarsStyle,
  toggleSideBar
} = useNav();
const { t } = useI18n();
const router = useRouter();
</script>

<template>
  <div class="navbar">
    <LaySidebarTopCollapse
      v-if="device === 'mobile'"
      class="hamburger-container"
      :is-active="pureApp.sidebar.opened"
      @toggleClick="toggleSideBar"
    />

    <LaySidebarBreadCrumb
      v-if="layout !== 'mix' && device !== 'mobile'"
      class="breadcrumb-container"
    />

    <LayNavMix v-if="layout === 'mix'" />

    <div v-if="layout === 'vertical'" class="vertical-header-right">
      <!-- 菜单搜索 -->
      <LaySearch id="header-search" />
      <!-- 全屏 -->
      <LaySidebarFullScreen id="full-screen" />
      <!-- 消息通知 -->
      <LayNotice id="header-notice" />
      <LanguageSwitch />
      <!-- 退出登录 -->
      <el-dropdown trigger="click">
        <span class="el-dropdown-link navbar-bg-hover select-none">
          <img :src="userAvatar" :style="avatarsStyle" />
          <p v-if="username" class="dark:text-white">{{ username }}</p>
        </span>
        <template #dropdown>
          <el-dropdown-menu class="logout">
            <el-dropdown-item @click="router.push('/profile')">
              <IconifyIconOffline :icon="UserSettings" style="margin: 5px" />
              个人中心
            </el-dropdown-item>
            <el-dropdown-item @click="logout">
              <IconifyIconOffline
                :icon="LogoutCircleRLine"
                style="margin: 5px"
              />
              {{ t("layout.logout") }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <span
        class="set-icon navbar-bg-hover"
        :title="t('layout.openSettings')"
        @click="onPanel"
      >
        <IconifyIconOffline :icon="Setting" />
      </span>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.navbar {
  width: 100%;
  height: 52px;
  overflow: hidden;
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border-soft);
  box-shadow: none !important;

  .hamburger-container {
    float: left;
    height: 100%;
    line-height: 52px;
    cursor: pointer;
  }

  .vertical-header-right {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    min-width: 280px;
    height: 52px;
    color: var(--app-text-secondary);
    gap: 2px;

    .el-dropdown-link {
      display: flex;
      align-items: center;
      justify-content: space-around;
      height: 52px;
      padding: 10px 12px;
      color: var(--app-text-secondary);
      cursor: pointer;
      border-radius: 6px;
      transition: background 0.18s;

      &:hover {
        background: var(--app-surface-soft);
      }

      p {
        font-size: 13.5px;
        font-weight: 500;
        margin-left: 6px;
      }

      img {
        width: 26px;
        height: 26px;
        border-radius: 50%;
        border: 1.5px solid var(--app-border);
        object-fit: cover;
      }
    }
  }

  .breadcrumb-container {
    float: left;
    margin-left: 16px;
  }
}

.logout {
  width: 130px;
  border-radius: 8px !important;
  overflow: hidden;

  ::v-deep(.el-dropdown-menu__item) {
    display: inline-flex;
    flex-wrap: wrap;
    min-width: 100%;
    font-size: 13.5px;
    padding: 8px 12px;
  }
}
</style>
