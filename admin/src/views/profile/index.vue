<script setup lang="ts">
/**
 * 个人中心：基础资料 + 修改密码 + 头像上传
 * 复用后端已有的 /api/v1/me、/me/password、/me/avatar 接口。
 */
import { onMounted, reactive, ref, computed } from "vue";
import type { FormInstance, FormRules, UploadRawFile } from "element-plus";
import { message } from "@/utils/message";
import {
  getMe,
  updateProfileApi,
  changeMyPasswordApi,
  uploadAvatarApi,
  type GoUser
} from "@/api/user";
import { useUserStoreHook } from "@/store/modules/user";

defineOptions({ name: "Profile" });

const activeTab = ref<"profile" | "password">("profile");

// —— 基础资料 ——
const profileForm = reactive({
  username: "",
  email: "",
  phone: "",
  avatar_url: ""
});
const profileLoading = ref(false);
const profileSaving = ref(false);
const profileRef = ref<FormInstance>();

const profileRules: FormRules = {
  username: [
    { required: true, message: "请输入用户名", trigger: "blur" },
    { min: 2, max: 50, message: "长度 2~50", trigger: "blur" }
  ],
  email: [
    { required: true, message: "请输入邮箱", trigger: "blur" },
    { type: "email", message: "邮箱格式不合法", trigger: ["blur", "change"] }
  ]
};

const currentUser = ref<GoUser | null>(null);
const roleText = computed(() => currentUser.value?.roles?.join("、") || "—");

const apiBase = import.meta.env.VITE_API_BASE_URL || "";
const avatarSrc = computed(() => {
  const url = profileForm.avatar_url;
  if (!url) return "";
  return /^https?:\/\//.test(url) ? url : `${apiBase}${url}`;
});

const loadProfile = async () => {
  profileLoading.value = true;
  try {
    const { data } = await getMe();
    currentUser.value = data;
    profileForm.username = data.username;
    profileForm.email = data.email;
    profileForm.phone = data.phone || "";
    profileForm.avatar_url = data.avatar_url || "";
  } catch {
    message("加载个人信息失败", { type: "error" });
  } finally {
    profileLoading.value = false;
  }
};

const saveProfile = () => {
  profileRef.value?.validate(async valid => {
    if (!valid) return;
    profileSaving.value = true;
    try {
      await updateProfileApi({
        username: profileForm.username,
        email: profileForm.email,
        phone: profileForm.phone,
        avatar_url: profileForm.avatar_url
      });
      // 同步到全局 store
      const fullUrl = profileForm.avatar_url
        ? (/^https?:\/\//.test(profileForm.avatar_url) ? profileForm.avatar_url : `${apiBase}${profileForm.avatar_url}`)
        : "";
      useUserStoreHook().SET_AVATAR(fullUrl);
      message("资料已更新", { type: "success" });
      loadProfile();
    } catch (err: any) {
      message(err?.response?.data?.error || "保存失败", { type: "error" });
    } finally {
      profileSaving.value = false;
    }
  });
};

// —— 头像上传 ——
const uploading = ref(false);
const beforeAvatarUpload = (file: UploadRawFile) => {
  const okType = ["image/jpeg", "image/png", "image/webp", "image/gif"].includes(
    file.type
  );
  if (!okType) {
    message("仅支持 JPG / PNG / WEBP / GIF", { type: "warning" });
    return false;
  }
  if (file.size > 5 * 1024 * 1024) {
    message("头像最大 5MB", { type: "warning" });
    return false;
  }
  return true;
};

const onAvatarPicked = async (raw: { file: File }) => {
  uploading.value = true;
  try {
    const { avatar_url } = await uploadAvatarApi(raw.file);
    profileForm.avatar_url = avatar_url;
    // 同步到全局 store，让导航栏头像立即更新
    const fullUrl = /^https?:\/\//.test(avatar_url) ? avatar_url : `${apiBase}${avatar_url}`;
    useUserStoreHook().SET_AVATAR(fullUrl);
    message("头像已更新", { type: "success" });
  } catch (err: any) {
    message(err?.response?.data?.error || "上传失败", { type: "error" });
  } finally {
    uploading.value = false;
  }
};

// —— 修改密码 ——
const passwordForm = reactive({
  old_password: "",
  new_password: "",
  confirm_password: ""
});
const passwordSaving = ref(false);
const passwordRef = ref<FormInstance>();

const passwordRules: FormRules = {
  old_password: [
    { required: true, message: "请输入原密码", trigger: "blur" }
  ],
  new_password: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { min: 6, max: 72, message: "长度 6~72", trigger: "blur" }
  ],
  confirm_password: [
    { required: true, message: "请再次输入", trigger: "blur" },
    {
      validator: (_r, value, cb) => {
        if (value !== passwordForm.new_password) {
          cb(new Error("两次输入不一致"));
        } else {
          cb();
        }
      },
      trigger: "blur"
    }
  ]
};

const changePassword = () => {
  passwordRef.value?.validate(async valid => {
    if (!valid) return;
    passwordSaving.value = true;
    try {
      await changeMyPasswordApi({
        old_password: passwordForm.old_password,
        new_password: passwordForm.new_password
      });
      message("密码已修改，请重新登录", { type: "success" });
      passwordForm.old_password = "";
      passwordForm.new_password = "";
      passwordForm.confirm_password = "";
    } catch (err: any) {
      message(err?.response?.data?.error || "修改失败", { type: "error" });
    } finally {
      passwordSaving.value = false;
    }
  });
};

onMounted(loadProfile);
</script>

<template>
  <div class="profile-page" v-loading="profileLoading">
    <el-card shadow="never" class="profile-card">
      <template #header>
        <div class="profile-header">
          <h3>个人中心</h3>
          <span v-if="currentUser">
            {{ currentUser.username }} · 角色 {{ roleText }}
          </span>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="基础资料" name="profile">
          <div class="profile-grid">
            <div class="avatar-block">
              <div class="avatar-preview">
                <img v-if="avatarSrc" :src="avatarSrc" alt="头像" />
                <span v-else>无头像</span>
              </div>
              <el-upload
                :show-file-list="false"
                :before-upload="beforeAvatarUpload"
                :http-request="onAvatarPicked"
                accept="image/*"
              >
                <el-button :loading="uploading" type="primary" plain>
                  {{ uploading ? "上传中" : "更换头像" }}
                </el-button>
              </el-upload>
              <p class="hint">
                建议 400×400 像素，仅支持 JPG / PNG / WEBP / GIF，5 MB 以内
              </p>
            </div>

            <el-form
              ref="profileRef"
              :model="profileForm"
              :rules="profileRules"
              label-width="80px"
              class="profile-form"
            >
              <el-form-item label="用户名" prop="username">
                <el-input v-model="profileForm.username" />
              </el-form-item>
              <el-form-item label="邮箱" prop="email">
                <el-input v-model="profileForm.email" />
              </el-form-item>
              <el-form-item label="手机号" prop="phone">
                <el-input v-model="profileForm.phone" placeholder="选填" />
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="profileSaving"
                  @click="saveProfile"
                >
                  保存
                </el-button>
                <el-button @click="loadProfile">重置</el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <el-tab-pane label="修改密码" name="password">
          <el-form
            ref="passwordRef"
            :model="passwordForm"
            :rules="passwordRules"
            label-width="100px"
            class="password-form"
          >
            <el-form-item label="原密码" prop="old_password">
              <el-input
                v-model.trim="passwordForm.old_password"
                type="password"
                show-password
                autocomplete="current-password"
              />
            </el-form-item>
            <el-form-item label="新密码" prop="new_password">
              <el-input
                v-model.trim="passwordForm.new_password"
                type="password"
                show-password
                autocomplete="new-password"
              />
            </el-form-item>
            <el-form-item label="确认新密码" prop="confirm_password">
              <el-input
                v-model.trim="passwordForm.confirm_password"
                type="password"
                show-password
                autocomplete="new-password"
              />
            </el-form-item>
            <el-form-item>
              <el-button
                type="primary"
                :loading="passwordSaving"
                @click="changePassword"
              >
                修改密码
              </el-button>
              <span class="hint">修改成功后此令牌立即失效，请重新登录</span>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<style scoped lang="scss">
.profile-page {
  padding: 24px;
}

.profile-card {
  max-width: 960px;
  margin: 0 auto;
  border-radius: 12px;
}

.profile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  color: #64748b;

  h3 {
    margin: 0;
    font-size: 18px;
    color: #0f172a;
  }
}

.profile-grid {
  display: grid;
  gap: 32px;
  grid-template-columns: 260px 1fr;
  align-items: start;

  @media (max-width: 720px) {
    grid-template-columns: 1fr;
  }
}

.avatar-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}

.avatar-preview {
  width: 128px;
  height: 128px;
  overflow: hidden;
  background: #f1f5f9;
  border: 2px dashed #cbd5f5;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.profile-form,
.password-form {
  max-width: 480px;
}

.hint {
  font-size: 12px;
  color: #94a3b8;
  margin: 0;
}
</style>
