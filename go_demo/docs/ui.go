package docs

func renderUI(mountPath, title string) string {
	if title == "" {
		title = "API Docs"
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width,initial-scale=1" />
<title>` + title + ` - API Docs</title>
<link
  rel="stylesheet"
  href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"
  onerror="this.onerror=null;this.href='https://unpkg.com/swagger-ui-dist@5/swagger-ui.css'"
/>
<style>
  :root{
    --page:#f5f7fa;
    --surface:#ffffff;
    --surface-soft:#f9fafb;
    --ink:#172033;
    --muted:#667085;
    --line:#d9e1ec;
    --line-soft:#edf1f6;
    --accent:#0f766e;
    --accent-soft:#e6f5f2;
    --blue:#2563eb;
    --green:#12805c;
    --amber:#b7791f;
    --red:#c2410c;
    --radius:8px;
    --shadow:0 18px 44px rgba(23,32,51,.08);
  }
  *{box-sizing:border-box}
  html,body{margin:0;padding:0;background:var(--page)}
  body{
    color:var(--ink);
    font-family:Inter,"Segoe UI","PingFang SC","Microsoft YaHei",system-ui,sans-serif;
    font-size:15px;
  }
  a{color:var(--accent);text-decoration:none}
  a:hover{text-decoration:underline;text-underline-offset:3px}
  .docs-header{
    border-bottom:1px solid var(--line);
    background:rgba(255,255,255,.92);
    backdrop-filter:saturate(160%) blur(12px);
  }
  .docs-header-inner{
    max-width:1180px;
    margin:0 auto;
    padding:24px 24px 22px;
    display:grid;
    grid-template-columns:1fr auto;
    gap:20px;
    align-items:start;
  }
  .brand-line{
    display:flex;
    align-items:center;
    gap:12px;
    min-width:0;
  }
  .brand-mark{
    width:10px;
    height:30px;
    border-radius:4px;
    background:var(--accent);
    box-shadow:0 0 0 5px var(--accent-soft);
    flex:0 0 auto;
  }
  .docs-header h1{
    margin:0;
    font-size:24px;
    line-height:1.2;
    font-weight:720;
    letter-spacing:0;
  }
  .version{
    display:inline-flex;
    align-items:center;
    height:24px;
    padding:0 9px;
    border:1px solid var(--line);
    border-radius:999px;
    background:var(--surface-soft);
    color:var(--muted);
    font-size:12px;
    font-weight:650;
    white-space:nowrap;
  }
  .intro{
    margin:10px 0 0 22px;
    max-width:820px;
    color:var(--muted);
    font-size:14px;
    line-height:1.7;
  }
  .intro strong{
    color:var(--ink);
    font-weight:680;
  }
  .langswitch{
    display:inline-flex;
    align-items:center;
    gap:3px;
    padding:3px;
    border:1px solid var(--line);
    border-radius:999px;
    background:var(--surface);
  }
  .langswitch button{
    all:unset;
    min-width:52px;
    height:28px;
    border-radius:999px;
    color:var(--muted);
    cursor:pointer;
    font-size:13px;
    font-weight:650;
    text-align:center;
  }
  .langswitch button:hover{color:var(--ink)}
  .langswitch button:focus-visible{
    outline:2px solid var(--accent);
    outline-offset:2px;
  }
  .langswitch button.active{
    background:var(--accent);
    color:#fff;
  }
  .docs-layout{
    max-width:1440px;
    margin:18px auto 56px;
    padding:0 24px;
    display:grid;
    grid-template-columns:300px minmax(0, 1fr);
    gap:24px;
    align-items:start;
  }
  .docs-sidebar,
  .docs-content{
    border:1px solid var(--line);
    border-radius:var(--radius);
    background:var(--surface);
    box-shadow:var(--shadow);
  }
  .docs-sidebar{
    position:sticky;
    top:18px;
    min-height:calc(100vh - 188px);
    max-height:calc(100vh - 36px);
    padding:0 0 18px;
    overflow:auto;
  }
  .sidebar-tabs{
    display:grid;
    grid-template-columns:1fr 1fr;
    border-bottom:1px solid var(--line);
    background:var(--surface);
  }
  .sidebar-tabs button{
    all:unset;
    height:40px;
    cursor:pointer;
    color:var(--muted);
    font-size:14px;
    font-weight:650;
    text-align:center;
    border-bottom:2px solid transparent;
  }
  .sidebar-tabs button.active{
    color:var(--accent);
    border-bottom-color:var(--accent);
  }
  .sidebar-tabs button:focus-visible{
    outline:2px solid var(--accent);
    outline-offset:-2px;
  }
  .sidebar-heading,
  .content-label{
    color:#ef4444;
    font-size:13px;
    font-weight:720;
  }
  .sidebar-heading{
    margin:16px 16px 10px;
  }
  .tag-nav{
    display:flex;
    flex-direction:column;
    gap:2px;
    padding:0 10px;
  }
  .tag-main,
  .operation-link,
  .sidebar-empty{
    display:block;
    border-radius:6px;
    color:var(--muted);
    line-height:1.35;
  }
  .tag-group{
    border-radius:7px;
  }
  .tag-main{
    width:100%;
    padding:9px 10px;
    border:1px solid transparent;
    background:transparent;
    cursor:pointer;
    font:inherit;
    font-size:14px;
    font-weight:650;
    text-align:left;
  }
  .tag-main:hover{
    border-color:var(--line);
    background:var(--surface-soft);
    color:var(--ink);
  }
  .tag-group.open > .tag-main,
  .tag-main.active{
    border-color:rgba(15,118,110,.2);
    background:var(--accent-soft);
    color:var(--accent);
  }
  .operation-nav{
    display:none;
    margin:2px 0 4px;
  }
  .tag-group.open .operation-nav{
    display:flex;
    flex-direction:column;
    gap:4px;
  }
  .operation-link{
    display:grid;
    grid-template-columns:48px minmax(0, 1fr);
    gap:8px;
    align-items:center;
    min-height:34px;
    padding:7px 10px 7px 16px;
    border:1px solid transparent;
    color:#5e6a7f;
    text-decoration:none;
  }
  .operation-link:hover{
    border-color:var(--line);
    background:#fbfcfe;
    color:var(--ink);
    text-decoration:none;
  }
  .operation-link.active{
    border-color:rgba(15,118,110,.12);
    border-right:3px solid var(--accent);
    background:var(--accent-soft);
    color:var(--accent);
  }
  .operation-method{
    display:inline-flex;
    align-items:center;
    justify-content:center;
    min-width:42px;
    height:20px;
    border-radius:4px;
    color:#fff;
    font-size:11px;
    font-weight:760;
    vertical-align:middle;
  }
  .operation-method.get{background:var(--blue)}
  .operation-method.post{background:var(--green)}
  .operation-method.put,
  .operation-method.patch{background:var(--amber)}
  .operation-method.delete{background:var(--red)}
  .operation-path{
    min-width:0;
    overflow:hidden;
    color:inherit;
    font-size:13px;
    font-weight:620;
    text-overflow:ellipsis;
    white-space:nowrap;
  }
  .operation-link .operation-real-path{
    display:none;
  }
  .sidebar-empty{
    padding:10px 16px;
    font-size:14px;
    color:var(--muted);
  }
  .docs-content{
    overflow:hidden;
    position:relative;
  }
  .content-toolbar{
    display:grid;
    grid-template-columns:1fr auto;
    gap:16px;
    align-items:center;
    min-height:72px;
    padding:0 24px;
    border-bottom:1px solid var(--line);
    background:var(--surface);
  }
  .endpoint-hero{
    padding:24px 24px 0;
    background:var(--surface);
  }
  .endpoint-title-row{
    display:grid;
    grid-template-columns:minmax(0, 1fr) auto;
    gap:16px;
    align-items:center;
    margin-bottom:14px;
  }
  .endpoint-title-row h2{
    margin:0;
    color:var(--ink);
    font-size:24px;
    line-height:1.25;
    font-weight:760;
  }
  .endpoint-refresh{
    height:34px;
    padding:0 14px;
    border:1px solid rgba(15,118,110,.35);
    border-radius:6px;
    background:var(--surface);
    color:var(--accent);
    cursor:pointer;
    font:inherit;
    font-size:13px;
    font-weight:700;
  }
  .endpoint-line{
    display:grid;
    grid-template-columns:94px minmax(0, 1fr);
    align-items:center;
    overflow:hidden;
    min-height:36px;
    border:1px solid var(--line);
    border-radius:6px;
    background:#fcfdff;
  }
  .endpoint-line-method{
    display:flex;
    align-items:center;
    justify-content:center;
    align-self:stretch;
    color:#fff;
    font-size:12px;
    font-weight:760;
  }
  .endpoint-line-method.get{background:var(--blue)}
  .endpoint-line-method.post{background:var(--green)}
  .endpoint-line-method.put,
  .endpoint-line-method.patch{background:var(--amber)}
  .endpoint-line-method.delete{background:var(--red)}
  .endpoint-line code{
    min-width:0;
    padding:0 12px;
    overflow:hidden;
    color:var(--ink);
    font-family:"SFMono-Regular",Consolas,"Liberation Mono",monospace;
    font-size:13px;
    text-overflow:ellipsis;
    white-space:nowrap;
  }
  .endpoint-tabs{
    display:flex;
    gap:22px;
    margin-top:14px;
    border-bottom:1px solid var(--line-soft);
  }
  .endpoint-tabs button{
    all:unset;
    padding:0 0 11px;
    border-bottom:2px solid transparent;
    color:var(--muted);
    cursor:pointer;
    font-size:13px;
    font-weight:650;
  }
  .endpoint-tabs button.active{
    border-bottom-color:var(--accent);
    color:var(--accent);
  }
  .docs-content.operation-focused #swagger-ui .filter,
  .docs-content.operation-focused #swagger-ui .opblock-tag,
  .docs-content.operation-focused #swagger-ui section.models{
    display:none!important;
  }
  .docs-content.operation-focused #swagger-ui .opblock-tag-section,
  .docs-content.operation-focused #swagger-ui .opblock{
    display:none;
  }
  .docs-content.operation-focused #swagger-ui .opblock-tag-section.selected,
  .docs-content.operation-focused #swagger-ui .opblock.selected{
    display:block;
  }
  .docs-content.operation-focused #swagger-ui .opblock.selected{
    margin:22px 0 0;
    border:0;
    border-radius:0;
  }
  .docs-content.operation-focused #swagger-ui .opblock.selected > .opblock-summary{
    display:none;
  }
  .docs-content.operation-focused #swagger-ui .opblock.selected .opblock-body{
    border-top:0;
    background:var(--surface);
  }
  .docs-content.operation-focused #swagger-ui .opblock.selected .opblock-section-header{
    border-radius:0;
    background:var(--surface);
    box-shadow:none;
  }
  .docs-content.operation-focused #swagger-ui .opblock.selected .try-out__btn,
  .docs-content.operation-focused #swagger-ui .opblock.selected .execute-wrapper .btn{
    border-radius:6px;
    font-weight:700;
  }
  .swagger-actions{
    display:flex;
    align-items:center;
    justify-content:flex-end;
    min-width:180px;
  }
  #swagger-ui{
    overflow:hidden;
  }
  #swagger-ui .topbar,
  #swagger-ui .information-container.wrapper{display:none!important}
  #swagger-ui .wrapper{
    max-width:none;
    padding:0 24px;
  }
  #swagger-ui .scheme-container{
    position:absolute;
    top:18px;
    right:24px;
    z-index:3;
    width:auto;
    margin:0;
    padding:0;
    border-bottom:0;
    background:transparent;
    box-shadow:none;
  }
  #swagger-ui .scheme-container .schemes{
    align-items:center;
  }
  #swagger-ui .scheme-container .wrapper{
    max-width:none;
    padding:0!important;
  }
  #swagger-ui .auth-wrapper,
  .content-toolbar .auth-wrapper{
    justify-content:flex-end;
  }
  #swagger-ui .btn.authorize,
  .content-toolbar .btn.authorize{
    height:34px;
    border-color:rgba(15,118,110,.35);
    border-radius:6px;
    color:var(--accent);
    box-shadow:none;
    font-weight:700;
  }
  #swagger-ui .btn.authorize svg,
  .content-toolbar .btn.authorize svg{fill:var(--accent)}
  #swagger-ui .filter{
    margin:24px 0 32px;
  }
  #swagger-ui .filter .operation-filter-input{
    height:42px;
    width:100%;
    border:1px solid var(--line);
    border-radius:6px;
    color:var(--ink);
    font-size:14px;
    box-shadow:none;
  }
  #swagger-ui .filter .operation-filter-input:focus{
    border-color:var(--accent);
    box-shadow:0 0 0 3px var(--accent-soft);
  }
  #swagger-ui .opblock-tag-section{
    margin:0 0 10px;
  }
  #swagger-ui .opblock-tag{
    min-height:52px;
    margin:0;
    padding:0 8px 0 12px;
    border-bottom:1px solid var(--line-soft);
    color:var(--ink);
    font-size:18px;
    font-weight:720;
    scroll-margin-top:88px;
  }
  #swagger-ui .opblock-tag:hover{
    background:#fbfcfe;
  }
  #swagger-ui .opblock-tag small{
    color:var(--muted);
    font-size:12px;
    font-weight:500;
  }
  #swagger-ui .opblock{
    overflow:hidden;
    margin:0 0 10px;
    border:1px solid var(--line);
    border-radius:var(--radius);
    background:var(--surface);
    box-shadow:none;
    scroll-margin-top:88px;
  }
  #swagger-ui .opblock .opblock-summary{
    min-height:50px;
    padding:0 12px;
    border-color:var(--line-soft);
  }
  #swagger-ui .opblock .opblock-summary-method{
    min-width:72px;
    border-radius:5px;
    font-size:12px;
    font-weight:760;
    letter-spacing:0;
    text-shadow:none;
  }
  #swagger-ui .opblock .opblock-summary-path{
    color:var(--ink);
    font-family:"SFMono-Regular",Consolas,"Liberation Mono",monospace;
    font-size:14px;
    font-weight:680;
  }
  #swagger-ui .opblock .opblock-summary-description{
    color:var(--muted);
    font-size:13px;
  }
  #swagger-ui .opblock.opblock-get{border-color:rgba(37,99,235,.22);background:#fff}
  #swagger-ui .opblock.opblock-get .opblock-summary-method{background:var(--blue)}
  #swagger-ui .opblock.opblock-post{border-color:rgba(18,128,92,.24);background:#fff}
  #swagger-ui .opblock.opblock-post .opblock-summary-method{background:var(--green)}
  #swagger-ui .opblock.opblock-put,
  #swagger-ui .opblock.opblock-patch{border-color:rgba(183,121,31,.24);background:#fff}
  #swagger-ui .opblock.opblock-put .opblock-summary-method,
  #swagger-ui .opblock.opblock-patch .opblock-summary-method{background:var(--amber)}
  #swagger-ui .opblock.opblock-delete{border-color:rgba(194,65,12,.24);background:#fff}
  #swagger-ui .opblock.opblock-delete .opblock-summary-method{background:var(--red)}
  #swagger-ui .opblock-body{
    border-top:1px solid var(--line-soft);
    background:#fcfdff;
  }
  #swagger-ui .opblock-description-wrapper,
  #swagger-ui .opblock-external-docs-wrapper,
  #swagger-ui .opblock-title_normal{
    color:var(--ink);
  }
  #swagger-ui table{
    border-collapse:separate;
    border-spacing:0;
  }
  #swagger-ui table thead tr th{
    color:var(--muted);
    font-size:12px;
    font-weight:720;
  }
  #swagger-ui .parameters-col_description,
  #swagger-ui .response-col_description{
    color:var(--ink);
  }
  #swagger-ui .model-box,
  #swagger-ui section.models{
    border-color:var(--line);
    border-radius:var(--radius);
  }
  #swagger-ui section.models h4{
    color:var(--ink);
    font-size:17px;
  }
  #loading{
    padding:80px 24px;
    color:var(--muted);
    font-size:14px;
    text-align:center;
  }
  #load-error{
    max-width:640px;
    margin:80px auto;
    padding:24px 28px;
    border:1px solid #fed7aa;
    border-radius:var(--radius);
    background:#fff7ed;
    color:#9a3412;
    line-height:1.7;
  }
  #load-error code{
    padding:2px 6px;
    border-radius:4px;
    background:#ffedd5;
    color:#7c2d12;
    font-size:.9em;
  }
  @media (max-width:760px){
    .docs-header-inner{
      grid-template-columns:1fr;
      padding:18px 16px;
    }
    .brand-line{
      align-items:flex-start;
      flex-wrap:wrap;
    }
    .brand-mark{height:26px}
    .docs-header h1{font-size:21px}
    .intro{margin-left:0;font-size:13px}
    .langswitch{justify-self:start}
    .docs-layout{
      grid-template-columns:1fr;
      margin:16px auto 32px;
      padding:0 12px;
      gap:12px;
    }
    .docs-sidebar{
      position:static;
      min-height:auto;
      padding:12px;
    }
    .docs-sidebar{max-height:260px}
    .content-toolbar{
      min-height:60px;
      padding:0 12px;
      grid-template-columns:1fr;
      gap:8px;
      padding-top:12px;
      padding-bottom:12px;
    }
    .swagger-actions{
      justify-content:flex-start;
    }
    #swagger-ui .wrapper{padding:0 12px}
    #swagger-ui .scheme-container{
      top:12px;
      right:12px;
      padding:0;
    }
    #swagger-ui .opblock .opblock-summary{
      align-items:flex-start;
      gap:8px;
      padding:10px;
    }
    #swagger-ui .opblock .opblock-summary-method{
      min-width:64px;
    }
  }
</style>
</head>
<body>
<header class="docs-header">
  <div class="docs-header-inner">
    <div>
      <div class="brand-line">
        <span class="brand-mark" aria-hidden="true"></span>
        <h1 data-i18n="title">` + title + `</h1>
        <span class="version">v1.0.0 - OpenAPI 3.0</span>
      </div>
      <p class="intro" data-i18n="intro">
        交互式接口文档。需要登录的接口请先点 Authorize 填入 Bearer JWT，再在接口卡片中使用 Try it out 调试。原始规范：
        <a id="raw-link" href="` + mountPath + `/doc.json?lang=zh" target="_blank">` + mountPath + `/doc.json?lang=zh</a>
      </p>
    </div>
    <div class="langswitch" role="tablist" aria-label="Language">
      <button type="button" data-lang="zh" role="tab">中文</button>
      <button type="button" data-lang="en" role="tab">EN</button>
    </div>
  </div>
</header>
<main class="docs-layout">
  <aside class="docs-sidebar" aria-label="API sections">
    <div class="sidebar-tabs" role="tablist" aria-label="Navigation">
      <button type="button" class="active" data-i18n="apiTab">接口</button>
      <button type="button" data-i18n="docsTab">文档</button>
    </div>
    <div class="sidebar-heading" data-i18n="sidebarTitle">标题</div>
    <nav id="tag-nav" class="tag-nav">
      <div class="sidebar-empty" data-i18n="sidebarLoading">接口加载中...</div>
    </nav>
  </aside>
  <section class="docs-content" aria-label="API details">
    <div class="content-toolbar">
      <div class="content-label" data-i18n="contentTitle">详情</div>
      <div id="swagger-actions" class="swagger-actions"></div>
    </div>
    <section id="endpoint-hero" class="endpoint-hero" hidden>
      <div class="endpoint-title-row">
        <h2 id="endpoint-title"></h2>
        <button type="button" class="endpoint-refresh" data-i18n="refreshButton">刷新</button>
      </div>
      <div class="endpoint-line">
        <span id="endpoint-method" class="endpoint-line-method"></span>
        <code id="endpoint-path"></code>
      </div>
      <div class="endpoint-tabs" role="tablist" aria-label="Endpoint views">
        <button type="button" class="active" data-view="doc" data-i18n="docTab">文档</button>
        <button type="button" data-view="json">JSON</button>
        <button type="button" data-view="typescript">TypeScript</button>
        <button type="button" data-view="debug" data-i18n="debugTab">调试</button>
      </div>
    </section>
    <div id="loading">加载 Swagger UI 中...</div>
    <div id="swagger-ui"></div>
  </section>
</main>
<script>
(function () {
  var MOUNT = '` + mountPath + `';
  function getLang() {
    var q = new URLSearchParams(location.search);
    var v = (q.get('lang') || '').toLowerCase();
    return v === 'en' ? 'en' : 'zh';
  }
  function setLang(lang) {
    var q = new URLSearchParams(location.search);
    q.set('lang', lang);
    location.search = q.toString();
  }
  var currentLang = getLang();
  var texts = {
    zh: {
      title: ` + goStringLiteral(title) + `,
      intro: '交互式接口文档。需要登录的接口请先点 <strong>Authorize</strong> 填入 Bearer JWT，再在接口卡片中使用 <strong>Try it out</strong> 调试。原始规范：<a id="raw-link" href="' + MOUNT + '/doc.json?lang=zh" target="_blank">' + MOUNT + '/doc.json?lang=zh</a>',
      loading: '加载 Swagger UI 中...',
      sidebarTitle: '标题',
      sidebarLoading: '接口加载中...',
      contentTitle: '详情',
      apiTab: '接口',
      docsTab: '文档',
      docTab: '文档',
      debugTab: '调试',
      refreshButton: '刷新',
      loadError: '<strong>无法从 CDN 加载 Swagger UI 资源。</strong><br>你可以直接访问 <code>' + MOUNT + '/doc.json</code> 查看原始 OpenAPI 3.0 JSON，或将 <code>swagger-ui-dist</code> 部署到本地静态服务。'
    },
    en: {
      title: ` + goStringLiteral(title) + `,
      intro: 'Interactive API reference. Click <strong>Authorize</strong> and paste a Bearer JWT for protected endpoints, then use <strong>Try it out</strong> on each endpoint card. Raw spec: <a id="raw-link" href="' + MOUNT + '/doc.json?lang=en" target="_blank">' + MOUNT + '/doc.json?lang=en</a>',
      loading: 'Loading Swagger UI...',
      sidebarTitle: 'Title',
      sidebarLoading: 'Loading endpoints...',
      contentTitle: 'Details',
      apiTab: 'APIs',
      docsTab: 'Docs',
      docTab: 'Docs',
      debugTab: 'Debug',
      refreshButton: 'Refresh',
      loadError: '<strong>Failed to load Swagger UI from CDN.</strong><br>You can still open <code>' + MOUNT + '/doc.json</code> to see the raw OpenAPI 3.0 JSON, or host <code>swagger-ui-dist</code> yourself.'
    }
  };

  function applyChrome() {
    var t = texts[currentLang];
    document.documentElement.setAttribute('lang', currentLang === 'en' ? 'en' : 'zh-CN');
    document.querySelectorAll('[data-i18n]').forEach(function (el) {
      var k = el.getAttribute('data-i18n');
      if (t[k] !== undefined) el.innerHTML = t[k];
    });
    var loading = document.getElementById('loading');
    if (loading) loading.textContent = t.loading;
    document.querySelectorAll('.langswitch button').forEach(function (b) {
      var active = b.getAttribute('data-lang') === currentLang;
      b.classList.toggle('active', active);
      b.setAttribute('aria-selected', active ? 'true' : 'false');
    });
  }
  applyChrome();
  document.querySelectorAll('.langswitch button').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var lang = btn.getAttribute('data-lang');
      if (lang !== currentLang) setLang(lang);
    });
  });
  function setEndpointTab(btn) {
    document.querySelectorAll('.endpoint-tabs button').forEach(function (item) {
      item.classList.toggle('active', item === btn);
    });
  }

  function enterTryItOut() {
    var selected = document.querySelector('#swagger-ui .opblock.selected');
    if (!selected) return;
    var tryButton = selected.querySelector('.try-out__btn');
    if (tryButton && !/cancel/i.test(tryButton.textContent || '')) {
      tryButton.click();
    }
    window.setTimeout(function () {
      var target = selected.querySelector('.parameters-container') || selected.querySelector('.opblock-section') || selected;
      target.scrollIntoView({
        behavior: reducedMotion() ? 'auto' : 'smooth',
        block: 'start'
      });
    }, 80);
  }

  document.querySelectorAll('.endpoint-tabs button').forEach(function (btn) {
    btn.addEventListener('click', function () {
      setEndpointTab(btn);
      if ((btn.getAttribute('data-view') || '') === 'debug') enterTryItOut();
    });
  });
  document.querySelectorAll('.sidebar-tabs button').forEach(function (btn, index) {
    btn.addEventListener('click', function () {
      document.querySelectorAll('.sidebar-tabs button').forEach(function (item) {
        item.classList.toggle('active', item === btn);
      });
      var endpointTabs = document.querySelectorAll('.endpoint-tabs button');
      if (index === 1 && endpointTabs[0]) setEndpointTab(endpointTabs[0]);
    });
  });
  var refreshButton = document.querySelector('.endpoint-refresh');
  if (refreshButton) {
    refreshButton.addEventListener('click', function () {
      location.reload();
    });
  }

  function cleanTagText(tag) {
    var clone = tag.cloneNode(true);
    clone.querySelectorAll('small, button, svg').forEach(function (el) {
      el.remove();
    });
    return clone.textContent.replace(/\s+/g, ' ').trim();
  }

  function cleanText(el) {
    return el ? el.textContent.replace(/\s+/g, ' ').trim() : '';
  }

  function methodClass(method) {
    return (method || '').toLowerCase().replace(/[^a-z]/g, '');
  }

  function reducedMotion() {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function setActiveGroup(nav, group, trigger) {
    nav.querySelectorAll('.tag-group').forEach(function (item) {
      item.classList.toggle('open', item === group);
    });
    nav.querySelectorAll('.tag-main').forEach(function (item) {
      item.classList.toggle('active', item === trigger);
    });
    nav.querySelectorAll('.operation-link').forEach(function (item) {
      item.classList.remove('active');
    });
  }

  function setEndpointHero(meta) {
    var hero = document.getElementById('endpoint-hero');
    var title = document.getElementById('endpoint-title');
    var method = document.getElementById('endpoint-method');
    var path = document.getElementById('endpoint-path');
    if (!hero || !title || !method || !path) return;

    title.textContent = meta.summary || meta.path || meta.tag;
    method.textContent = meta.method;
    method.className = 'endpoint-line-method ' + methodClass(meta.method);
    path.textContent = meta.path;
    hero.hidden = false;
  }

  function selectOperation(nav, group, trigger, link, section, opblock, meta, skipScroll) {
    setActiveGroup(nav, group, trigger);
    nav.querySelectorAll('.operation-link').forEach(function (item) {
      item.classList.toggle('active', item === link);
    });
    document.querySelectorAll('#swagger-ui .opblock-tag-section.selected, #swagger-ui .opblock.selected').forEach(function (item) {
      item.classList.remove('selected');
    });
    section.classList.add('selected');
    opblock.classList.add('selected');
    setEndpointHero(meta);
    document.querySelector('.docs-content').classList.add('operation-focused');
    var docTabButton = document.querySelector('.endpoint-tabs button[data-view="doc"]');
    if (docTabButton) setEndpointTab(docTabButton);

    var summary = opblock.querySelector('.opblock-summary');
    if (summary && !opblock.querySelector('.opblock-body')) {
      summary.click();
    }
    if (!skipScroll) {
      window.setTimeout(function () {
        document.querySelector('.docs-content').scrollIntoView({
          behavior: reducedMotion() ? 'auto' : 'smooth',
          block: 'start'
        });
      }, 80);
    }
  }

  function buildEndpointNav() {
    var nav = document.getElementById('tag-nav');
    if (!nav) return false;
    var sections = Array.prototype.slice.call(document.querySelectorAll('#swagger-ui .opblock-tag-section'));
    if (!sections.length) return false;

    var operationCount = 0;
    var firstSelection = null;
    nav.innerHTML = '';
    sections.forEach(function (section, index) {
      var tag = section.querySelector('.opblock-tag');
      if (!tag) return;
      tag.id = tag.id || 'api-tag-' + index;

      var group = document.createElement('div');
      group.className = 'tag-group';
      var groupFirst = null;

      var button = document.createElement('button');
      button.type = 'button';
      button.className = 'tag-main';
      button.textContent = cleanTagText(tag) || 'Section ' + (index + 1);
      button.addEventListener('click', function () {
        if (groupFirst) {
          selectOperation(nav, group, button, groupFirst.link, groupFirst.section, groupFirst.opblock, groupFirst.meta, false);
          return;
        }
        setActiveGroup(nav, group, button);
        tag.scrollIntoView({
          behavior: reducedMotion() ? 'auto' : 'smooth',
          block: 'start'
        });
      });

      var operationNav = document.createElement('div');
      operationNav.className = 'operation-nav';

      Array.prototype.slice.call(section.querySelectorAll('.opblock')).forEach(function (opblock, opIndex) {
        var summary = opblock.querySelector('.opblock-summary');
        var method = cleanText(opblock.querySelector('.opblock-summary-method'));
        var path = cleanText(opblock.querySelector('.opblock-summary-path')) || cleanText(opblock.querySelector('.opblock-summary-path__deprecated'));
        var description = cleanText(opblock.querySelector('.opblock-summary-description'));
        if (!summary || !path) return;

        opblock.id = opblock.id || 'api-operation-' + index + '-' + opIndex;
        var meta = {
          tag: cleanTagText(tag),
          method: method,
          path: path,
          summary: description
        };

        var opLink = document.createElement('a');
        opLink.className = 'operation-link';
        opLink.href = '#' + opblock.id;
        var methodLabel = document.createElement('span');
        methodLabel.className = 'operation-method ' + methodClass(method);
        methodLabel.textContent = method;
        var pathLabel = document.createElement('span');
        pathLabel.className = 'operation-path';
        pathLabel.textContent = description || path;
        pathLabel.title = path;
        opLink.appendChild(methodLabel);
        opLink.appendChild(pathLabel);
        opLink.addEventListener('click', function (event) {
          event.preventDefault();
          selectOperation(nav, group, button, opLink, section, opblock, meta, false);
        });
        operationNav.appendChild(opLink);
        if (!firstSelection) {
          firstSelection = {
            group: group,
            button: button,
            link: opLink,
            section: section,
            opblock: opblock,
            meta: meta
          };
        }
        if (!groupFirst) {
          groupFirst = {
            link: opLink,
            section: section,
            opblock: opblock,
            meta: meta
          };
        }
        operationCount++;
      });

      group.appendChild(button);
      group.appendChild(operationNav);
      if (index === 0) {
        group.classList.add('open');
        button.classList.add('active');
      }
      nav.appendChild(group);
    });

    if (firstSelection) {
      selectOperation(nav, firstSelection.group, firstSelection.button, firstSelection.link, firstSelection.section, firstSelection.opblock, firstSelection.meta, true);
    }
    return operationCount > 0;
  }

  function syncDocsChrome() {
    var tries = 0;
    var timer = window.setInterval(function () {
      tries++;
      var built = buildEndpointNav();
      if (built || tries > 40) {
        window.clearInterval(timer);
      }
    }, 200);
  }

  function initUI() {
    if (typeof SwaggerUIBundle === 'undefined') return false;
    var loading = document.getElementById('loading');
    if (loading) loading.remove();
    window.ui = SwaggerUIBundle({
      url: MOUNT + '/doc.json?lang=' + currentLang,
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: 'BaseLayout',
      persistAuthorization: true,
      tryItOutEnabled: true,
      docExpansion: 'full',
      defaultModelsExpandDepth: 1,
      defaultModelExpandDepth: 2,
      displayRequestDuration: true,
      filter: true
    });
    syncDocsChrome();
    return true;
  }

  function loadUI(sources) {
    var idx = 0;
    function next() {
      if (idx >= sources.length) {
        var loading = document.getElementById('loading');
        if (loading) loading.remove();
        var box = document.createElement('div');
        box.id = 'load-error';
        box.innerHTML = texts[currentLang].loadError;
        document.body.appendChild(box);
        return;
      }
      var s = document.createElement('script');
      s.src = sources[idx];
      s.onload = function () { if (!initUI()) { idx++; next(); } };
      s.onerror = function () { idx++; next(); };
      document.body.appendChild(s);
    }
    next();
  }

  loadUI([
    'https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js',
    'https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js'
  ]);
})();
</script>
</body>
</html>`
}

func goStringLiteral(s string) string {
	var b []byte
	b = append(b, '"')
	for _, r := range s {
		switch r {
		case '\\':
			b = append(b, '\\', '\\')
		case '"':
			b = append(b, '\\', '"')
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		case '\t':
			b = append(b, '\\', 't')
		default:
			if r < 0x20 {
				b = append(b, []byte{'\\', 'u', '0', '0', hex(byte(r >> 4)), hex(byte(r & 0xf))}...)
			} else {
				b = append(b, []byte(string(r))...)
			}
		}
	}
	b = append(b, '"')
	return string(b)
}

func hex(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + n - 10
}
