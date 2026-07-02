/**
 * 极简 hash 路由 —— 单文件、无依赖。
 * 支持 /、/articles/:slug、/search、/search?q=... 三条路径。
 * 使用 hash（#/articles/xxx）避免服务端配置。
 */
import { useEffect, useState } from "react";

export type Route =
  | { name: "home" }
  | { name: "article"; slug: string }
  | { name: "search"; q: string; category: string; tag: string; page: number };

const decode = (value: string) => {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
};

const parseHash = (hash: string): Route => {
  const raw = (hash || "").replace(/^#/, "");
  const [pathPart, queryPart] = raw.split("?");
  const params = new URLSearchParams(queryPart ?? "");
  const path = (pathPart || "/").replace(/\/$/, "") || "/";

  if (path === "/" || path === "") return { name: "home" };
  if (path === "/search") {
    return {
      name: "search",
      q: params.get("q") ?? "",
      category: params.get("category") ?? "",
      tag: params.get("tag") ?? "",
      page: Math.max(1, Number(params.get("page") || 1))
    };
  }
  if (path.startsWith("/articles/")) {
    return { name: "article", slug: decode(path.slice("/articles/".length)) };
  }
  return { name: "home" };
};

export const useHashRoute = (): Route => {
  const [route, setRoute] = useState<Route>(() => parseHash(window.location.hash));
  useEffect(() => {
    const onChange = () => setRoute(parseHash(window.location.hash));
    window.addEventListener("hashchange", onChange);
    return () => window.removeEventListener("hashchange", onChange);
  }, []);
  return route;
};

/** 编程式跳转 */
export const navigate = (target: string) => {
  window.location.hash = target.startsWith("#") ? target : `#${target}`;
  window.scrollTo({ top: 0, behavior: "instant" as ScrollBehavior });
};

export const searchHash = (q: string, extras: Record<string, string | number> = {}) => {
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  Object.entries(extras).forEach(([k, v]) => {
    if (v !== "" && v !== undefined && v !== null && String(v) !== "0") {
      params.set(k, String(v));
    }
  });
  const qs = params.toString();
  return `#/search${qs ? `?${qs}` : ""}`;
};
