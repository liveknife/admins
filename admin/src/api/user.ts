import { http } from "@/utils/http";
import { encryptPassword } from "@/utils/passwordCrypto";

export type GoUser = {
  id: number;
  username: string;
  email: string;
  phone: string;
  roles: string[];
  permissions: string[];
  created_at: string;
  deleted_at?: string;
};

type GoTokenPair = {
  access_token: string;
  refresh_token: string;
  access_token_expires_in: number;
  refresh_token_expires_in: number;
  token_type: string;
};

type GoAuthResponse = {
  user: GoUser;
  tokens: GoTokenPair;
};

type GoUserResponse = {
  user: GoUser;
};

export type UserResult = {
  success: boolean;
  data: {
    avatar: string;
    username: string;
    nickname: string;
    roles: Array<string>;
    permissions: Array<string>;
    accessToken: string;
    refreshToken: string;
    expires: Date;
    email?: string;
    id?: number;
  };
};

export type RefreshTokenResult = UserResult;

export type MeResult = {
  success: boolean;
  data: GoUser;
};

const toExpiresDate = (expiresInSeconds: number) => {
  return new Date(Date.now() + expiresInSeconds * 1000);
};

const normalizeAuthResponse = (response: GoAuthResponse): UserResult => {
  const { user, tokens } = response;

  return {
    success: true,
    data: {
      id: user.id,
      email: user.email,
      avatar: "",
      username: user.username,
      nickname: user.username,
      roles: user.roles ?? [],
      permissions: user.permissions ?? [],
      accessToken: tokens.access_token,
      refreshToken: tokens.refresh_token,
      expires: toExpiresDate(tokens.access_token_expires_in)
    }
  };
};

export const getLogin = async (data?: {
  account?: string;
  email?: string;
  password?: string;
}) => {
  const passwordEncrypted = await encryptPassword(data?.password ?? "");
  return http
    .request<GoAuthResponse>("post", "/api/v1/login", {
      data: {
        account: data?.account ?? data?.email,
        password_encrypted: passwordEncrypted
      }
    })
    .then(normalizeAuthResponse);
};

export const refreshTokenApi = (data?: { refreshToken?: string }) => {
  return http
    .request<GoAuthResponse>("post", "/api/v1/refresh-token", {
      data: { refresh_token: data?.refreshToken }
    })
    .then(normalizeAuthResponse);
};

export const registerApi = async (data?: {
  username?: string;
  email?: string;
  phone?: string;
  password?: string;
}) => {
  const passwordEncrypted = await encryptPassword(data?.password ?? "");
  return http.request<GoUserResponse>("post", "/api/v1/register", {
    data: {
      username: data?.username,
      email: data?.email,
      phone: data?.phone,
      password_encrypted: passwordEncrypted
    }
  });
};

export const forgotPasswordApi = (data?: { email?: string }) => {
  return http.request<{ message: string; reset_token: string }>(
    "post",
    "/api/v1/forgot-password",
    { data }
  );
};

export const resetPasswordApi = async (data?: {
  token?: string;
  new_password?: string;
}) => {
  const passwordEncrypted = await encryptPassword(data?.new_password ?? "");
  return http.request<{ message: string }>("post", "/api/v1/reset-password", {
    data: {
      token: data?.token,
      new_password_encrypted: passwordEncrypted
    }
  });
};

export const getMe = () => {
  return http.request<GoUserResponse>("get", "/api/v1/me").then(response => ({
    success: true,
    data: response.user
  }));
};
