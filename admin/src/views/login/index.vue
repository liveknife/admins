<script setup lang="ts">
import Motion from "./utils/motion";
import { useRouter } from "vue-router";
import { message } from "@/utils/message";
import { useLoginRules } from "./utils/rule";
import { onBeforeUnmount, onMounted, reactive, ref, toRaw } from "vue";
import { registerApi } from "@/api/user";
import { fetchCaptcha, type CaptchaChallenge } from "@/api/user";
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
  hue: number;
  baseX: number;
  baseY: number;
}> = [];
let mousePos = { x: -9999, y: -9999 };
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
  password: "",
  captcha: ""
});

// 图形验证码状态
const captchaImage = ref("");
const captchaId = ref("");
const captchaLoading = ref(false);

const refreshCaptcha = async () => {
  captchaLoading.value = true;
  try {
    const data = (await fetchCaptcha()) as CaptchaChallenge;
    captchaImage.value = data.image;
    captchaId.value = data.captcha_id;
    ruleForm.captcha = "";
  } catch {
    captchaImage.value = "";
    captchaId.value = "";
  } finally {
    captchaLoading.value = false;
  }
};

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
          password: ruleForm.password,
          captcha: ruleForm.captcha,
          captcha_id: captchaId.value
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
          refreshCaptcha();
        })
        .catch(() => {
          message(t("login.loginFailed"), { type: "error" });
          refreshCaptcha();
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
        password: ruleForm.password,
        captcha: ruleForm.captcha,
        captcha_id: captchaId.value
      });
      ruleForm.account = ruleForm.username;
      authMode.value = "login";
      message(t("login.registerSuccess"), { type: "success" });
      ruleFormRef.value?.clearValidate();
      refreshCaptcha();
    } catch (error) {
      message(t("login.registerFailed"), {
        type: "error"
      });
      refreshCaptcha();
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

  // 鼠标交互
  const handleMouse = (e: MouseEvent) => {
    mousePos.x = e.clientX;
    mousePos.y = e.clientY;
  };
  const handleMouseLeave = () => {
    mousePos.x = -9999;
    mousePos.y = -9999;
  };
  window.addEventListener("mousemove", handleMouse);
  window.addEventListener("mouseleave", handleMouseLeave);

  particleResize = () => {
    const dpr = window.devicePixelRatio || 1;
    canvas.width = window.innerWidth * dpr;
    canvas.height = window.innerHeight * dpr;
    canvas.style.width = `${window.innerWidth}px`;
    canvas.style.height = `${window.innerHeight}px`;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    const count = Math.min(
      240,
      Math.max(100, Math.floor((window.innerWidth * window.innerHeight) / 7000))
    );
    particles = Array.from({ length: count }, () => ({
      x: Math.random() * window.innerWidth,
      y: Math.random() * window.innerHeight,
      baseX: Math.random() * window.innerWidth,
      baseY: Math.random() * window.innerHeight,
      vx: (Math.random() - 0.5) * 0.5,
      vy: (Math.random() - 0.5) * 0.5,
      size: Math.random() * 2.4 + 0.6,
      alpha: Math.random() * 0.5 + 0.35,
      hue: Math.random() * 60 + 190
    }));
  };

  let time = 0;
  const draw = () => {
    time += 0.008;
    ctx.clearRect(0, 0, window.innerWidth, window.innerHeight);

    // 渐变背景
    const gradient = ctx.createLinearGradient(
      0, 0, window.innerWidth, window.innerHeight
    );
    gradient.addColorStop(0, "#060d1a");
    gradient.addColorStop(0.35, "#091a33");
    gradient.addColorStop(0.7, "#0c2240");
    gradient.addColorStop(1, "#081220");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, window.innerWidth, window.innerHeight);

    // 动态光晕（随时间移动）
    for (let g = 0; g < 3; g++) {
      const gx = window.innerWidth * (0.2 + g * 0.3) + Math.sin(time + g) * 80;
      const gy = window.innerHeight * (0.25 + g * 0.25) + Math.cos(time * 0.8 + g) * 60;
      const glow = ctx.createRadialGradient(gx, gy, 0, gx, gy, 280);
      const hues = [200, 170, 260];
      glow.addColorStop(0, `hsla(${hues[g]}, 85%, 55%, 0.06)`);
      glow.addColorStop(1, "transparent");
      ctx.fillStyle = glow;
      ctx.fillRect(0, 0, window.innerWidth, window.innerHeight);
    }

    // 粒子更新与绘制
    const mouseRadius = 180;
    const mouseForce = 0.045;
    const connectDist = 150;

    for (let i = 0; i < particles.length; i++) {
      const p = particles[i];

      // 缓慢漂移
      p.baseX += p.vx;
      p.baseY += p.vy;

      // 边界环绕
      if (p.baseX < -30) p.baseX = window.innerWidth + 30;
      if (p.baseX > window.innerWidth + 30) p.baseX = -30;
      if (p.baseY < -30) p.baseY = window.innerHeight + 30;
      if (p.baseY > window.innerHeight + 30) p.baseY = -30;

      p.x = p.baseX;
      p.y = p.baseY;

      // 鼠标排斥/吸引交互
      const dx = p.x - mousePos.x;
      const dy = p.y - mousePos.y;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (dist < mouseRadius && dist > 0) {
        const force = (mouseRadius - dist) / mouseRadius;
        const angle = Math.atan2(dy, dx);
        p.x += Math.cos(angle) * force * force * mouseRadius * mouseForce;
        p.y += Math.sin(angle) * force * force * mouseRadius * mouseForce;
      }

      // 呼吸效果
      const breathe = Math.sin(time * 1.5 + i * 0.15) * 0.18 + 1;
      const drawSize = p.size * breathe;

      // 绘制粒子光晕
      ctx.beginPath();
      ctx.arc(p.x, p.y, drawSize * 3, 0, Math.PI * 2);
      ctx.fillStyle = `hsla(${p.hue}, 90%, 65%, ${p.alpha * 0.12})`;
      ctx.fill();

      // 绘制粒子核心
      ctx.beginPath();
      ctx.arc(p.x, p.y, drawSize, 0, Math.PI * 2);
      ctx.fillStyle = `hsla(${p.hue}, 92%, 72%, ${p.alpha})`;
      ctx.shadowBlur = 16;
      ctx.shadowColor = `hsla(${p.hue}, 100%, 70%, 0.75)`;
      ctx.fill();
      ctx.shadowBlur = 0;

      // 粒子间连线
      for (let j = i + 1; j < particles.length; j++) {
        const next = particles[j];
        const ldx = p.x - next.x;
        const ldy = p.y - next.y;
        const ldist = Math.sqrt(ldx * ldx + ldy * ldy);
        if (ldist > connectDist) continue;

        const lineAlpha = 0.2 * (1 - ldist / connectDist);
        ctx.beginPath();
        ctx.moveTo(p.x, p.y);
        ctx.lineTo(next.x, next.y);
        const lineHue = (p.hue + next.hue) / 2;
        ctx.strokeStyle = `hsla(${lineHue}, 80%, 62%, ${lineAlpha})`;
        ctx.lineWidth = 0.8;
        ctx.stroke();
      }
    }

    // 鼠标周围高亮圈
    if (mousePos.x > -1000) {
      const ringGrad = ctx.createRadialGradient(
        mousePos.x, mousePos.y, 0,
        mousePos.x, mousePos.y, mouseRadius
      );
      ringGrad.addColorStop(0, "rgba(100, 210, 255, 0.07)");
      ringGrad.addColorStop(0.6, "rgba(80, 160, 255, 0.03)");
      ringGrad.addColorStop(1, "transparent");
      ctx.beginPath();
      ctx.arc(mousePos.x, mousePos.y, mouseRadius, 0, Math.PI * 2);
      ctx.fillStyle = ringGrad;
      ctx.fill();
    }

    animationFrame = requestAnimationFrame(draw);
  };

  particleResize();
  window.addEventListener("resize", particleResize);

  // 存储清理函数
  const origCleanup = particleResize;
  particleResize = () => {
    origCleanup();
    window.removeEventListener("mousemove", handleMouse);
    window.removeEventListener("mouseleave", handleMouseLeave);
  };

  draw();
};

onMounted(() => {
  initParticles();
  refreshCaptcha();
});

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

            <Motion :delay="authMode === 'login' ? 175 : 200">
              <el-form-item prop="captcha">
                <div class="captcha-row">
                  <el-input
                    v-model="ruleForm.captcha"
                    clearable
                    maxlength="6"
                    placeholder="请输入右侧图形验证码"
                  />
                  <button
                    type="button"
                    class="captcha-image"
                    :aria-label="'刷新验证码'"
                    :disabled="captchaLoading"
                    @click="refreshCaptcha"
                  >
                    <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
                    <span v-else>加载中</span>
                  </button>
                </div>
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

.img {
  overflow: visible !important;
}

.particle-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
}
.aurora-layer {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
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

.captcha-row {
  display: flex;
  gap: 10px;
  width: 100%;
  align-items: stretch;
}

.captcha-row :deep(.el-input) {
  flex: 1;
  min-width: 0;
}

.captcha-image {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 120px;
  padding: 0;
  overflow: hidden;
  cursor: pointer;
  background: #0f172a;
  border: 1px solid rgba(148, 163, 184, 0.35);
  border-radius: 6px;
  transition: border-color 0.2s;
}

.captcha-image:hover {
  border-color: #38bdf8;
}

.captcha-image[disabled] {
  cursor: wait;
  opacity: 0.6;
}

.captcha-image img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.captcha-image span {
  font-size: 12px;
  color: #94a3b8;
}
</style>
