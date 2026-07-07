/**
 * 官网前端共享类型 —— 与后端 models 对齐；如果字段增加，只需要在这里补一次。
 */

export type SiteAnnouncement = {
  id: number;
  title: string;
  content: string;
  link_url: string;
};

export type SiteBanner = {
  id: number;
  title: string;
  subtitle: string;
  image_url: string;
  link_url: string;
};

export type SiteResource = {
  id: number;
  title: string;
  slug: string;
  summary: string;
  content: string;
  markdown_content: string;
  category: string;
  cover_url: string;
  link_url: string;
  tags: string;
  seo_title: string;
  seo_description: string;
  seo_keywords: string;
  is_featured: boolean;
  view_count: number;
  published_at?: string;
  updated_at?: string;
};

export type SiteTechStack = {
  id: number;
  name: string;
  category: string;
  level: number;
  icon_url: string;
  description: string;
};

export type SiteProject = {
  id: number;
  name: string;
  summary: string;
  description: string;
  cover_url: string;
  demo_url: string;
  repo_url: string;
  stack_tags: string;
  status: string;
  is_featured: boolean;
};

export type SiteTimelineEvent = {
  id: number;
  title: string;
  summary: string;
  content: string;
  phase: string;
  event_type: string;
  tags: string;
  link_url: string;
  status: string;
  is_featured: boolean;
  sort_order: number;
  happened_at?: string;
  published_at?: string;
};

export type SiteMessage = {
  id: number;
  visitor_name: string;
  content: string;
  reply: string;
  created_at: string;
};

export type SitePublicStats = {
  visit_count: number;
  article_count: number;
  message_count: number;
};

export type SiteHome = {
  announcements: SiteAnnouncement[];
  banners: SiteBanner[];
  resources: SiteResource[];
  tech_stacks: SiteTechStack[];
  projects: SiteProject[];
  timeline: SiteTimelineEvent[];
  messages: SiteMessage[];
  analytics: SitePublicStats;
};

export type KnowledgeAnswer = {
  question: string;
  answer: string;
  sources?: Array<{
    source_type: string;
    source_id: number;
    title: string;
    summary: string;
    score: number;
    url?: string;
    snippet?: string;
    highlighted_text?: string;
  }>;
  matches: SiteResource[];
  projects: SiteProject[];
  suggestions?: string[];
  query_log_id?: number;
};

export type SearchResponse = {
  items: SiteResource[];
  total: number;
  page: number;
  page_size: number;
  query: string;
};

export const emptyHome: SiteHome = {
  announcements: [],
  banners: [],
  resources: [],
  tech_stacks: [],
  projects: [],
  timeline: [],
  messages: [],
  analytics: { visit_count: 0, article_count: 0, message_count: 0 }
};

export const apiBase = import.meta.env.VITE_API_BASE_URL || "";

export const assetURL = (url?: string) => {
  if (!url) return "";
  if (/^https?:\/\//.test(url)) return url;
  return `${apiBase}${url}`;
};

export const splitTags = (value?: string) =>
  (value ?? "")
    .split(",")
    .map(item => item.trim())
    .filter(Boolean);

export const articleSlugOrID = (item: Pick<SiteResource, "id" | "slug">) =>
  encodeURIComponent(item.slug || String(item.id));

export const articleRouteHash = (item: Pick<SiteResource, "id" | "slug">) =>
  `#/articles/${articleSlugOrID(item)}`;
