import { computed } from "vue";
import type { FormRules } from "element-plus";
import { useI18n } from "@/i18n";

const useLoginRules = () => {
  const { t } = useI18n();

  return computed<FormRules>(() => ({
    account: [
      { required: true, message: t("login.accountRequired"), trigger: "blur" },
      { min: 2, max: 80, message: t("login.accountLength"), trigger: "blur" }
    ],
    username: [
      { required: true, message: t("login.usernameRequired"), trigger: "blur" },
      { min: 2, max: 50, message: t("login.usernameLength"), trigger: "blur" }
    ],
    phone: [
      { required: true, message: t("login.phoneRequired"), trigger: "blur" },
      { min: 5, max: 20, message: t("login.phoneLength"), trigger: "blur" }
    ],
    email: [
      { required: true, message: t("login.emailRequired"), trigger: "blur" },
      {
        type: "email",
        message: t("login.emailInvalid"),
        trigger: ["blur", "change"]
      }
    ],
    password: [
      { required: true, message: t("login.passwordRequired"), trigger: "blur" },
      { min: 6, message: t("login.passwordMin"), trigger: "blur" }
    ],
    captcha: [{ required: true, message: "请输入图形验证码", trigger: "blur" }]
  }));
};

export { useLoginRules };
