<script setup lang="ts">
import Motion from "./utils/motion";
import { useRouter } from "vue-router";
import { message } from "@/utils/message";
import { useLoginRules } from "./utils/rule";
import { onBeforeUnmount, onMounted, reactive, ref, toRaw } from "vue";
import { registerApi } from "@/api/user";
import { debounce } from "@pureadmin/utils";
import { useNav } from "@/layout/hooks/useNav";
import { useEventListener } from "@vueuse/core";
import type { FormInstance } from "element-plus";
import { useLayout } from "@/layout/hooks/useLayout";
import { useUserStoreHook } from "@/store/modules/user";
import { initRouter, getTopMenu } from "@/router/utils";
import { avatar, illustration } from "./utils/static";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import { useDataThemeChange } from "@/layout/hooks/useDataThemeChange";
import { useI18n } from "@/i18n";
// import LanguageSwitch from "@/components/LanguageSwitch/index.vue";

import dayIcon from "@/assets/svg/day.svg?component";
import darkIcon from "@/assets/svg/dark.svg?component";
import Lock from "~icons/ri/lock-fill";
import Mail from "~icons/ri/mail-fill";
import Phone from "~icons/ri/phone-fill";
import User from "~icons/ri/user-fill";

defineOptions({
  name: "Login"
});

const router = useRouter();
const loading = ref(false);
const disabled = ref(false);
const authMode = ref<"login" | "register">("login");
const ruleFormRef = ref<FormInstance>();
const particleCanvasRef = ref<HTMLCanvasElement>();
let animationFrame = 0;
let particles: Array<{
  x: number;
  y: number;
  vx: number;
  vy: number;
  size: number;
  alpha: number;
}> = [];
let particleResize: (() => void) | undefined;

const { initStorage } = useLayout();
initStorage();

const { dataTheme, overallStyle, dataThemeChange } = useDataThemeChange();
dataThemeChange(overallStyle.value);
const { title } = useNav();
const { t } = useI18n();
const loginRules = useLoginRules();

const ruleForm = reactive({
  account: "",
  username: "",
  phone: "",
  email: "",
  password: ""
});

const toggleMode = () => {
  authMode.value = authMode.value === "login" ? "register" : "login";
  ruleFormRef.value?.clearValidate();
};

const onLogin = async (formEl: FormInstance | undefined) => {
  if (!formEl) return;
  await formEl.validate(valid => {
    if (valid) {
      loading.value = true;
      useUserStoreHook()
        .loginByUsername({
          account: ruleForm.account,
          password: ruleForm.password
        })
        .then(res => {
          if (res.success) {
            return initRouter().then(() => {
              disabled.value = true;
              router
                .push(getTopMenu(true).path)
                .then(() => {
                  message(t("login.loginSuccess"), { type: "success" });
                })
                .finally(() => (disabled.value = false));
            });
          }
          message(t("login.loginFailed"), { type: "error" });
        })
        .catch(() => {
          message(t("login.loginFailed"), { type: "error" });
        })
        .finally(() => (loading.value = false));
    }
  });
};

const onRegister = async (formEl: FormInstance | undefined) => {
  if (!formEl) return;
  await formEl.validate(async valid => {
    if (!valid) return;
    loading.value = true;
    try {
      await registerApi({
        username: ruleForm.username,
        email: ruleForm.email,
        phone: ruleForm.phone,
        password: ruleForm.password
      });
      ruleForm.account = ruleForm.username;
      authMode.value = "login";
      message(t("login.registerSuccess"), { type: "success" });
      ruleFormRef.value?.clearValidate();
    } catch (error) {
      message(t("login.registerFailed"), {
        type: "error"
      });
    } finally {
      loading.value = false;
    }
  });
};

const submit = (formEl: FormInstance | undefined) => {
  return authMode.value === "login" ? onLogin(formEl) : onRegister(formEl);
};

const immediateDebounce: any = debounce(
  formRef => submit(formRef),
  1000,
  true
);

useEventListener(document, "keydown", ({ code }) => {
  if (
    ["Enter", "NumpadEnter"].includes(code) &&
    !disabled.value &&
    !loading.value
  )
    immediateDebounce(ruleFormRef.value);
});

const initParticles = () => {
  const canvas = particleCanvasRef.value;
  const ctx = canvas?.getContext("2d");
  if (!canvas || !ctx) return;

  particleResize = () => {
    const dpr = window.devicePixelRatio || 1;
    canvas.width = window.innerWidth * dpr;
    canvas.height = window.innerHeight * dpr;
    canvas.style.width = `${window.innerWidth}px`;
    canvas.style.height = `${window.innerHeight}px`;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    const count = Math.min(
      110,
      Math.max(48, Math.floor((window.innerWidth * window.innerHeight) / 15000))
    );
    particles = Array.from({ length: count }, () => ({
      x: Math.random() * window.innerWidth,
      y: Math.random() * window.innerHeight,
      vx: (Math.random() - 0.5) * 0.45,
      vy: (Math.random() - 0.5) * 0.45,
      size: Math.random() * 2.1 + 0.8,
      alpha: Math.random() * 0.45 + 0.35
    }));
  };

  const draw = () => {
    ctx.clearRect(0, 0, window.innerWidth, window.innerHeight);

    const gradient = ctx.createLinearGradient(
      0,
      0,
      window.innerWidth,
      window.innerHeight
    );
    gradient.addColorStop(0, "#08111f");
    gradient.addColorStop(0.45, "#0b2445");
    gradient.addColorStop(1, "#101827");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, window.innerWidth, window.innerHeight);

    for (let i = 0; i < particles.length; i++) {
      const p = particles[i];
      p.x += p.vx;
      p.y += p.vy;

      if (p.x < -20) p.x = window.innerWidth + 20;
      if (p.x > window.innerWidth + 20) p.x = -20;
      if (p.y < -20) p.y = window.innerHeight + 20;
      if (p.y > window.innerHeight + 20) p.y = -20;

      ctx.beginPath();
      ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
      ctx.fillStyle = `rgba(93, 213, 255, ${p.alpha})`;
      ctx.shadowBlur = 14;
      ctx.shadowColor = "rgba(93, 213, 255, 0.72)";
      ctx.fill();
      ctx.shadowBlur = 0;

      for (let j = i + 1; j < particles.length; j++) {
        const next = particles[j];
        const dx = p.x - next.x;
        const dy = p.y - next.y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        if (distance > 145) continue;
        ctx.beginPath();
        ctx.moveTo(p.x, p.y);
        ctx.lineTo(next.x, next.y);
        ctx.strokeStyle = `rgba(74, 144, 255, ${0.18 * (1 - distance / 145)})`;
        ctx.lineWidth = 1;
        ctx.stroke();
      }
    }

    animationFrame = requestAnimationFrame(draw);
  };

  particleResize();
  window.addEventListener("resize", particleResize);
  draw();
};

onMounted(initParticles);

onBeforeUnmount(() => {
  cancelAnimationFrame(animationFrame);
  if (particleResize) window.removeEventListener("resize", particleResize);
});
</script>

<template>
  <div class="login-page select-none">
    <canvas ref="particleCanvasRef" class="particle-bg" />
    <div class="aurora-layer" />
    <div class="login-tools absolute right-5 top-3 z-2">
      <!-- <LanguageSwitch /> -->
      <!-- <el-switch
        v-model="dataTheme"
        inline-prompt
        :active-icon="dayIcon"
        :inactive-icon="darkIcon"
        @change="dataThemeChange"
      /> -->
    </div>
    <div class="login-container">
      <div class="img">
        <component :is="toRaw(illustration)" />
      </div>
      <div class="login-box">
        <div class="login-form">
          <avatar class="avatar" />
          <Motion>
            <h2 class="outline-hidden">
              {{ authMode === "login" ? title : t("login.registerAccount") }}
            </h2>
          </Motion>

          <el-form
            ref="ruleFormRef"
            :model="ruleForm"
            :rules="loginRules"
            size="large"
          >
            <Motion v-if="authMode === 'login'" :delay="100">
              <el-form-item prop="account">
                <el-input
                  v-model="ruleForm.account"
                  clearable
                  :placeholder="t('login.account')"
                  :prefix-icon="useRenderIcon(User)"
                />
              </el-form-item>
            </Motion>

            <template v-else>
              <Motion :delay="100">
                <el-form-item prop="username">
                  <el-input
                    v-model="ruleForm.username"
                    clearable
                    :placeholder="t('login.username')"
                    :prefix-icon="useRenderIcon(User)"
                  />
                </el-form-item>
              </Motion>

              <Motion :delay="125">
                <el-form-item prop="phone">
                  <el-input
                    v-model="ruleForm.phone"
                    clearable
                    :placeholder="t('login.phone')"
                    :prefix-icon="useRenderIcon(Phone)"
                  />
                </el-form-item>
              </Motion>

              <Motion :delay="150">
                <el-form-item prop="email">
                  <el-input
                    v-model="ruleForm.email"
                    clearable
                    :placeholder="t('login.email')"
                    :prefix-icon="useRenderIcon(Mail)"
                  />
                </el-form-item>
              </Motion>
            </template>

            <Motion :delay="authMode === 'login' ? 150 : 175">
              <el-form-item prop="password">
                <el-input
                  v-model.trim="ruleForm.password"
                  clearable
                  show-password
                  :placeholder="t('login.password')"
                  :prefix-icon="useRenderIcon(Lock)"
                  @input="ruleForm.password = ruleForm.password.replace(/\s+/g, '')"
                />
              </el-form-item>
            </Motion>

            <Motion :delay="250">
              <el-button
                class="w-full mt-4!"
                size="default"
                type="primary"
                :loading="loading"
                :disabled="disabled"
                @click="submit(ruleFormRef)"
              >
                {{ authMode === "login" ? t("login.login") : t("login.register") }}
              </el-button>
            </Motion>

            <Motion :delay="300">
              <el-button class="auth-switch w-full mt-3!" text @click="toggleMode">
                {{
                  authMode === "login"
                    ? t("login.switchToRegister")
                    : t("login.switchToLogin")
                }}
              </el-button>
            </Motion>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url("@/style/login.css");
</style>

<style lang="scss" scoped>
:deep(.el-input-group__append, .el-input-group__prepend) {
  padding: 0;
}

.login-page {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  color: #eaf6ff;
}

.particle-bg,
.aurora-layer {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
}

.aurora-layer {
  background:
    radial-gradient(circle at 18% 18%, rgb(43 164 255 / 34%), transparent 28%),
    radial-gradient(circle at 70% 28%, rgb(20 184 166 / 22%), transparent 30%),
    radial-gradient(circle at 62% 80%, rgb(124 58 237 / 20%), transparent 34%);
  filter: blur(2px);
}

.login-page :deep(.login-container),
.login-page > .login-tools {
  position: relative;
  z-index: 1;
}

.login-tools {
  display: flex;
  gap: 10px;
  align-items: center;
}

.login-tools :deep(.language-switch) {
  height: 32px;
  color: #eaf6ff;
  border-radius: 6px;
}

.auth-switch {
  color: #38bdf8;
  font-weight: 600;
}

.auth-switch:hover {
  color: #60a5fa;
}
</style>
