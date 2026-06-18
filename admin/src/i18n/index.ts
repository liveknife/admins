import { computed, ref } from "vue";

export const localeOptions = [
  { label: "简体中文", value: "zh-CN" },
  { label: "English", value: "en-US" }
] as const;

export type Locale = (typeof localeOptions)[number]["value"];
type MessageValue = string | Messages;
interface Messages {
  [key: string]: MessageValue;
}

const STORAGE_KEY = "admins-locale";
const defaultLocale: Locale = "zh-CN";

const messages: Record<Locale, Messages> = {
  "zh-CN": {
    common: {
      add: "新建",
      cancel: "取消",
      close: "关闭",
      confirm: "确定",
      copyFailed: "复制失败",
      copySuccess: "复制成功",
      delete: "删除",
      description: "描述",
      edit: "编辑",
      home: "首页",
      id: "ID",
      operation: "操作",
      refresh: "刷新",
      save: "保存",
      search: "搜索",
      tip: "提示"
    },
    language: {
      label: "语言",
      zh: "简体中文",
      en: "English"
    },
    layout: {
      logout: "退出系统",
      openSettings: "打开系统配置"
    },
    search: {
      close: "关闭",
      collect: "收藏",
      collectSortable: "收藏（可拖拽排序）",
      confirm: "确认",
      empty: "暂无搜索结果",
      history: "搜索历史",
      placeholder: "搜索菜单（支持拼音搜索）",
      switch: "切换",
      total: "共 {total} 项"
    },
    notice: {
      notice: "通知",
      message: "消息",
      todo: "待办",
      empty: "暂无消息",
      emptyNotice: "暂无通知",
      emptyTodo: "暂无待办"
    },
    settings: {
      clearCache: "清空缓存",
      clearCacheTip: "清空缓存并返回登录页",
      close: "关闭配置",
      title: "系统配置"
    },
    routes: {
      home: "首页",
      login: "登录",
      loading: "加载中...",
      system: "系统管理",
      users: "用户管理",
      roles: "角色权限",
      message: "消息管理",
      chat: "聊天",
      error: "异常页面"
    },
    login: {
      account: "用户名 / 手机号 / 邮箱",
      accountLength: "账号长度为 2-80 个字符",
      email: "邮箱",
      emailInvalid: "请输入正确的邮箱",
      login: "登录",
      loginFailed: "登录失败",
      loginSuccess: "登录成功",
      password: "密码",
      passwordMin: "密码至少 6 位",
      phone: "手机号",
      phoneLength: "手机号长度为 5-20 个字符",
      register: "注册",
      registerAccount: "注册账号",
      registerFailed: "注册失败，请检查用户名、手机号或邮箱是否已存在",
      registerSuccess: "注册成功，请登录",
      switchToLogin: "已有账号？返回登录",
      switchToRegister: "还没有账号？立即注册",
      username: "用户名",
      usernameLength: "用户名长度为 2-50 个字符",
      accountRequired: "请输入用户名 / 手机号 / 邮箱",
      usernameRequired: "请输入用户名",
      phoneRequired: "请输入手机号",
      emailRequired: "请输入邮箱",
      passwordRequired: "请输入密码"
    },
    error: {
      forbidden: "抱歉，你无权访问该页面",
      notFound: "抱歉，你访问的页面不存在",
      server: "抱歉，服务器出错了",
      backHome: "返回首页"
    },
    admin: {
      apiPrefix: "Go 后端",
      createRole: "新建角色",
      createdAt: "创建时间",
      deactivate: "注销",
      deactivated: "已注销",
      normal: "正常",
      password: "密码",
      passwordLength: "密码长度为 6-72 位",
      passwordReadNotice: "该操作仅管理员可用，密码来自服务端加密存储。",
      permissions: "权限",
      role: "角色",
      roleCreateFailed: "角色创建失败",
      roleCreated: "角色已创建",
      roleDeleteConfirm: "确认删除角色 {name}？",
      roleDeleteFailed: "角色删除失败",
      roleDeleted: "角色已删除",
      roleEdit: "编辑角色",
      roleLoadFailed: "角色权限加载失败",
      roleName: "角色名称",
      roleNameLength: "角色名称长度为 2-50 个字符",
      roleNameRequired: "请输入角色名称",
      roleNew: "新建角色",
      rolePlaceholder: "请选择角色",
      roleUpdateFailed: "角色更新失败",
      roleUpdated: "角色已更新",
      setRole: "设置角色",
      setUserRole: "设置用户角色",
      status: "状态",
      user: "用户",
      userDeactivateConfirm: "确认注销用户 {name}？注销后该用户将无法登录。",
      userDeactivateFailed: "用户注销失败",
      userDeactivated: "用户已注销",
      userListLoadFailed: "用户列表加载失败",
      userRoleUpdated: "用户角色已更新",
      userRoleUpdateFailed: "用户角色更新失败",
      username: "用户名",
      phone: "手机号",
      email: "邮箱",
      resetPassword: "重置密码",
      newPassword: "新密码",
      newPasswordRequired: "请输入新密码",
      passwordReset: "密码已重置",
      passwordResetFailed: "密码重置失败",
      viewPassword: "查看密码",
      viewPasswordConfirm: "确认查看 {name} 的明文密码？",
      viewPasswordFailed: "密码查看失败",
      sensitiveAction: "敏感操作",
      closeConfirm: "确认关闭当前窗口？",
      descriptionPlaceholder: "请输入描述",
      permissionPlaceholder: "请选择权限",
      createUser: "新建用户",
      editUser: "编辑用户",
      create: "创建",
      usernamePlaceholder: "请输入用户名（2-50位）",
      emailPlaceholder: "请输入邮箱地址",
      phoneOptional: "手机号（可选）",
      passwordPlaceholder: "请输入密码（6-72位）",
      roleHint: "不选择则默认分配 user 角色",
      userCreated: "用户已创建",
      userCreateFailed: "用户创建失败",
      userUpdated: "用户信息已更新",
      userUpdateFailed: "用户更新失败",
      usernameRequired: "请输入用户名",
      emailRequired: "请输入邮箱地址",
      passwordRequired: "请输入密码"
    },
    chat: {
      audio: "语音",
      chat: "聊天",
      connected: "已连接",
      disconnected: "未连接",
      empty: "暂无消息",
      emptyVoice: "录音内容为空",
      emoji: "表情",
      file: "文件",
      fileSendFailed: "文件发送失败",
      image: "图片",
      messageFailed: "聊天消息处理失败",
      messageSendFailed: "消息发送失败",
      micDenied: "无法获取麦克风权限",
      noTranslatableText: "没有可翻译的文字",
      noTranscript: "暂无可识别文字",
      offline: "离线",
      online: "在线",
      pageDesc: "系统用户实时通信",
      pause: "暂停",
      placeholder: "输入消息，Enter 发送，Shift + Enter 换行",
      play: "播放",
      record: "按下录音",
      recording: "正在录音",
      reconnecting: "重连中",
      refresh: "刷新",
      send: "发送",
      socketNotReady: "聊天连接未就绪",
      speechUnavailable: "语音识别暂不可用，录音仍会发送",
      stopRecord: "停止录音",
      transcribe: "转文字",
      transcript: "文字",
      translate: "翻译",
      translateFailed: "翻译失败",
      translation: "译文",
      unsupportedRecorder: "当前浏览器不支持录音",
      unsupportedRecorderCodec: "当前浏览器不支持录音编码",
      userLoadFailed: "用户列表加载失败",
      video: "视频",
      voiceReady: "语音消息",
      voiceSendFailed: "语音发送失败",
      voiceSent: "语音已发送",
      historyLoadFailed: "聊天记录加载失败",
      me: "我",
      read: "已读",
      unread: "未读"
    },
    permission: {
      adminRole: "管理员角色",
      authList: "当前拥有的 code 列表：",
      commonRole: "普通角色",
      componentMode: "组件方式判断权限",
      currentRole: "当前角色：{name}",
      directiveMode: "指令方式判断权限（该方式不能动态修改权限）",
      functionMode: "函数方式判断权限",
      hasCode: "拥有 code：{code} 权限可见",
      pageDesc:
        "模拟后台根据不同角色返回对应路由，观察左侧菜单变化。管理员可查看系统管理菜单，普通角色不可查看系统管理菜单。",
      superAdminTip: "*:*:* 代表拥有全部按钮级别权限"
    },
    tableBar: {
      columnDisplay: "列展示",
      columnSettings: "列设置",
      compact: "紧凑",
      default: "默认",
      density: "密度",
      expand: "展开",
      fullscreen: "全屏",
      list: "列表",
      loose: "宽松",
      refresh: "刷新",
      reset: "重置",
      fold: "折叠",
      exitFullscreen: "退出全屏"
    },
    tags: {
      reload: "重新加载",
      closeCurrent: "关闭当前标签页",
      closeLeft: "关闭左侧标签页",
      closeRight: "关闭右侧标签页",
      closeOther: "关闭其他标签页",
      closeAll: "关闭全部标签页",
      contentFullscreen: "内容区全屏",
      contentExitFullscreen: "内容区退出全屏"
    },
    welcome: {
      title: "Admins 管理系统"
    }
  },
  "en-US": {
    common: {
      add: "Create",
      cancel: "Cancel",
      close: "Close",
      confirm: "OK",
      copyFailed: "Copy failed",
      copySuccess: "Copied",
      delete: "Delete",
      description: "Description",
      edit: "Edit",
      home: "Home",
      id: "ID",
      operation: "Actions",
      refresh: "Refresh",
      save: "Save",
      search: "Search",
      tip: "Tip"
    },
    language: {
      label: "Language",
      zh: "简体中文",
      en: "English"
    },
    layout: {
      logout: "Log out",
      openSettings: "Open system settings"
    },
    search: {
      close: "Close",
      collect: "Favorites",
      collectSortable: "Favorites (drag to sort)",
      confirm: "Confirm",
      empty: "No results",
      history: "Search history",
      placeholder: "Search menus (pinyin supported)",
      switch: "Switch",
      total: "{total} item(s)"
    },
    notice: {
      notice: "Notifications",
      message: "Messages",
      todo: "Todo",
      empty: "No messages",
      emptyNotice: "No notifications",
      emptyTodo: "No todo items"
    },
    settings: {
      clearCache: "Clear cache",
      clearCacheTip: "Clear cache and return to login",
      close: "Close settings",
      title: "System Settings"
    },
    routes: {
      home: "Home",
      login: "Login",
      loading: "Loading...",
      system: "System",
      users: "Users",
      roles: "Roles & Permissions",
      message: "Messages",
      chat: "Chat",
      error: "Error Pages"
    },
    login: {
      account: "Username / phone / email",
      accountLength: "Account must be 2-80 characters",
      email: "Email",
      emailInvalid: "Please enter a valid email",
      login: "Login",
      loginFailed: "Login failed",
      loginSuccess: "Logged in",
      password: "Password",
      passwordMin: "Password must be at least 6 characters",
      phone: "Phone",
      phoneLength: "Phone must be 5-20 characters",
      register: "Register",
      registerAccount: "Create Account",
      registerFailed: "Registration failed. Check whether the username, phone, or email already exists.",
      registerSuccess: "Registered. Please log in",
      switchToLogin: "Already have an account? Back to login",
      switchToRegister: "No account yet? Register now",
      username: "Username",
      usernameLength: "Username must be 2-50 characters",
      accountRequired: "Please enter username / phone / email",
      usernameRequired: "Please enter username",
      phoneRequired: "Please enter phone",
      emailRequired: "Please enter email",
      passwordRequired: "Please enter password"
    },
    error: {
      forbidden: "Sorry, you do not have access to this page",
      notFound: "Sorry, the page you visited does not exist",
      server: "Sorry, the server encountered an error",
      backHome: "Back Home"
    },
    admin: {
      apiPrefix: "Go backend",
      createRole: "Create Role",
      createdAt: "Created At",
      deactivate: "Deactivate",
      deactivated: "Deactivated",
      normal: "Normal",
      password: "Password",
      passwordLength: "Password must be 6-72 characters",
      passwordReadNotice: "This action is only available to administrators. The password is encrypted and stored on the server.",
      permissions: "Permissions",
      role: "Role",
      roleCreateFailed: "Failed to create role",
      roleCreated: "Role created",
      roleDeleteConfirm: "Delete role {name}?",
      roleDeleteFailed: "Failed to delete role",
      roleDeleted: "Role deleted",
      roleEdit: "Edit Role",
      roleLoadFailed: "Failed to load roles and permissions",
      roleName: "Role Name",
      roleNameLength: "Role name must be 2-50 characters",
      roleNameRequired: "Please enter role name",
      roleNew: "Create Role",
      rolePlaceholder: "Select roles",
      roleUpdateFailed: "Failed to update role",
      roleUpdated: "Role updated",
      setRole: "Set Role",
      setUserRole: "Set User Roles",
      status: "Status",
      user: "User",
      userDeactivateConfirm: "Deactivate user {name}? The user will no longer be able to log in.",
      userDeactivateFailed: "Failed to deactivate user",
      userDeactivated: "User deactivated",
      userListLoadFailed: "Failed to load users",
      userRoleUpdated: "User roles updated",
      userRoleUpdateFailed: "Failed to update user roles",
      username: "Username",
      phone: "Phone",
      email: "Email",
      resetPassword: "Reset Password",
      newPassword: "New Password",
      newPasswordRequired: "Please enter new password",
      passwordReset: "Password reset",
      passwordResetFailed: "Failed to reset password",
      viewPassword: "View Password",
      viewPasswordConfirm: "View plaintext password for {name}?",
      viewPasswordFailed: "Failed to view password",
      sensitiveAction: "Sensitive Action",
      closeConfirm: "Close this dialog?",
      descriptionPlaceholder: "Enter description",
      permissionPlaceholder: "Select permissions",
      createUser: "Create User",
      editUser: "Edit User",
      create: "Create",
      usernamePlaceholder: "Enter username (2-50 chars)",
      emailPlaceholder: "Enter email address",
      phoneOptional: "Phone (optional)",
      passwordPlaceholder: "Enter password (6-72 chars)",
      roleHint: "Defaults to 'user' role if none selected",
      userCreated: "User created",
      userCreateFailed: "Failed to create user",
      userUpdated: "User info updated",
      userUpdateFailed: "Failed to update user",
      usernameRequired: "Please enter a username",
      emailRequired: "Please enter an email address",
      passwordRequired: "Please enter a password"
    },
    chat: {
      audio: "Voice",
      chat: "Chat",
      connected: "Connected",
      disconnected: "Disconnected",
      empty: "No messages",
      emptyVoice: "Recording is empty",
      emoji: "Emoji",
      file: "File",
      fileSendFailed: "Failed to send file",
      image: "Image",
      messageFailed: "Failed to process chat message",
      messageSendFailed: "Failed to send message",
      micDenied: "Unable to access microphone",
      noTranslatableText: "No text to translate",
      noTranscript: "No recognizable text",
      offline: "Offline",
      online: "Online",
      pageDesc: "Real-time messaging between system users",
      pause: "Pause",
      placeholder: "Type a message. Enter to send, Shift + Enter for a new line",
      play: "Play",
      record: "Hold to record",
      recording: "Recording",
      reconnecting: "Reconnecting",
      refresh: "Refresh",
      send: "Send",
      socketNotReady: "Chat connection is not ready",
      speechUnavailable: "Speech recognition is unavailable. The recording will still be sent",
      stopRecord: "Stop recording",
      transcribe: "Transcribe",
      transcript: "Transcript",
      translate: "Translate",
      translateFailed: "Translation failed",
      translation: "Translation",
      unsupportedRecorder: "Recording is not supported by this browser",
      unsupportedRecorderCodec: "Recording codec is not supported by this browser",
      userLoadFailed: "Failed to load users",
      video: "Video",
      voiceReady: "Voice message",
      voiceSendFailed: "Failed to send voice message",
      voiceSent: "Voice message sent",
      historyLoadFailed: "Failed to load chat history",
      me: "Me",
      read: "Read",
      unread: "Unread"
    },
    permission: {
      adminRole: "Admin role",
      authList: "Current code list: ",
      commonRole: "Common role",
      componentMode: "Component permission check",
      currentRole: "Current role: {name}",
      directiveMode: "Directive permission check (permissions cannot be updated dynamically)",
      functionMode: "Function permission check",
      hasCode: "Visible with code: {code}",
      pageDesc:
        "Simulate backend routes by role and observe sidebar changes. Admin can see the system menu; common users cannot.",
      superAdminTip: "*:*:* grants all button-level permissions"
    },
    tableBar: {
      columnDisplay: "Columns",
      columnSettings: "Column settings",
      compact: "Compact",
      default: "Default",
      density: "Density",
      expand: "Expand",
      fullscreen: "Fullscreen",
      list: "List",
      loose: "Loose",
      refresh: "Refresh",
      reset: "Reset",
      fold: "Fold",
      exitFullscreen: "Exit fullscreen"
    },
    tags: {
      reload: "Reload",
      closeCurrent: "Close current tab",
      closeLeft: "Close left tabs",
      closeRight: "Close right tabs",
      closeOther: "Close other tabs",
      closeAll: "Close all tabs",
      contentFullscreen: "Content fullscreen",
      contentExitFullscreen: "Exit content fullscreen"
    },
    welcome: {
      title: "Admins Management System"
    }
  }
};

const normalizeLocale = (value: string | null): Locale => {
  return value === "en-US" || value === "zh-CN" ? value : defaultLocale;
};

export const currentLocale = ref<Locale>(
  normalizeLocale(window.localStorage.getItem(STORAGE_KEY))
);

export const currentLocaleOption = computed(() =>
  localeOptions.find(item => item.value === currentLocale.value)
);

export function setLocale(locale: Locale) {
  currentLocale.value = locale;
  window.localStorage.setItem(STORAGE_KEY, locale);
  document.documentElement.lang = locale;
}

export function t(key: string, params: Record<string, string | number> = {}) {
  const segments = key.split(".");
  let value: MessageValue | undefined = messages[currentLocale.value];

  for (const segment of segments) {
    if (typeof value !== "object") break;
    value = value[segment];
  }

  const template = typeof value === "string" ? value : key;
  return Object.entries(params).reduce(
    (text, [name, paramValue]) =>
      text.replace(new RegExp(`\\{${name}\\}`, "g"), String(paramValue)),
    template
  );
}

export function useI18n() {
  return {
    locale: currentLocale,
    localeOptions,
    currentLocaleOption,
    setLocale,
    t
  };
}

document.documentElement.lang = currentLocale.value;
