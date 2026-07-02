package docs

// renderUI 返回 Swagger UI 5.x 的 HTML。
// 资源优先从国内可用的 jsDelivr CDN 拉取，失败自动回退到 unpkg。
// 使用 Swagger UI 官方默认（浅色）主题，只在顶部叠一个自定义 hero，不改 UI 主题。
// 通过 URL query `?lang=en` 切换英文；hero 上有 "中文 / EN" 切换按钮。
func renderUI(mountPath, title string) string {
	if title == "" {
		title = "API Docs"
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8" />
<title>` + title + ` · API 文档</title>
<link
  rel="stylesheet"
  href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"
  onerror="this.onerror=null;this.href='https://unpkg.com/swagger-ui-dist@5/swagger-ui.css'"
/>
<style>
  html,body{margin:0;padding:0;background:#f7f8fb}
  body{font-family:'PingFang SC','Segoe UI',system-ui,sans-serif;color:#1e293b}
  .topbar{
    position:relative;
    background:linear-gradient(135deg,#1e3a8a 0%,#2563eb 45%,#38bdf8 100%);
    color:#fff;
    padding:36px 48px 44px;
    box-shadow:0 8px 28px rgba(30,58,138,.18);
  }
  .topbar h1{
    margin:0 0 6px;
    font-size:26px;
    letter-spacing:.5px;
    display:flex;
    align-items:center;
    gap:12px;
  }
  .topbar .version{
    display:inline-flex;
    align-items:center;
    padding:2px 10px;
    border-radius:999px;
    background:rgba(255,255,255,.18);
    font-size:12px;
    font-weight:500;
  }
  .topbar p{
    margin:0;
    opacity:.85;
    font-size:14px;
    line-height:1.7;
    max-width:820px;
  }
  .topbar a{color:#fff;text-decoration:underline;text-underline-offset:3px}
  /* 语言切换按钮 */
  .langswitch{
    position:absolute;
    top:24px;
    right:32px;
    display:inline-flex;
    background:rgba(255,255,255,.12);
    border:1px solid rgba(255,255,255,.28);
    border-radius:999px;
    padding:3px;
    backdrop-filter:blur(6px);
  }
  .langswitch button{
    all:unset;
    cursor:pointer;
    padding:6px 16px;
    border-radius:999px;
    font-size:13px;
    font-weight:500;
    color:rgba(255,255,255,.85);
    transition:all .2s;
  }
  .langswitch button:hover{color:#fff}
  .langswitch button.active{
    background:#fff;
    color:#1e3a8a;
  }
  .swagger-shell{
    max-width:1280px;
    margin:-24px auto 48px;
    padding:0 24px;
  }
  #swagger-ui{
    background:#fff;
    border-radius:14px;
    box-shadow:0 4px 24px rgba(15,23,42,.06);
    overflow:hidden;
    padding:8px 0;
  }
  #swagger-ui .topbar,
  #swagger-ui .information-container.wrapper{display:none!important}
  #swagger-ui .scheme-container{
    box-shadow:none;
    background:#f8fafc;
    border-bottom:1px solid #e2e8f0;
    padding:12px 24px;
  }
  #swagger-ui .opblock-tag{font-size:18px;border-bottom:1px solid #e2e8f0}
  #swagger-ui .opblock{margin:0 0 12px;border-radius:8px}
  #loading{text-align:center;padding:80px 24px;color:#64748b;font-size:14px}
  #load-error{
    max-width:640px;
    margin:80px auto;
    padding:24px 28px;
    background:#fff;
    border-radius:12px;
    border:1px solid #fecaca;
    color:#b91c1c;
    line-height:1.7;
  }
  #load-error code{
    background:#fef2f2;color:#dc2626;padding:2px 6px;border-radius:4px;font-size:.9em;
  }
</style>
</head>
<body>
<header class="topbar">
  <h1>
    <span>📘 <span data-i18n="title">` + title + `</span></span>
    <span class="version">v1.0.0 · OpenAPI 3.0</span>
  </h1>
  <p data-i18n="intro">
    交互式接口文档。带 <strong>🔒</strong> 图标的接口需要先点右上角 <strong>Authorize</strong> 填入 Bearer JWT，
    再点接口卡片右侧 <strong>Try it out</strong> 直接调用。原始规范：<a id="raw-link" href="` + mountPath + `/doc.json" target="_blank">` + mountPath + `/doc.json</a>
  </p>
  <div class="langswitch" role="tablist" aria-label="Language">
    <button type="button" data-lang="zh" role="tab">中文</button>
    <button type="button" data-lang="en" role="tab">EN</button>
  </div>
</header>
<div class="swagger-shell">
  <div id="loading">加载 Swagger UI 中…</div>
  <div id="swagger-ui"></div>
</div>
<script>
(function () {
  var MOUNT = '` + mountPath + `';
  // 简易 URL 语言读写 —— 保留其他 query 不动
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

  // 顶部文案本地化（hero 那一小段不走 Swagger，直接切）
  var texts = {
    zh: {
      title: ` + goStringLiteral(title) + `,
      intro: '交互式接口文档。带 <strong>🔒</strong> 图标的接口需要先点右上角 <strong>Authorize</strong> 填入 Bearer JWT，再点接口卡片右侧 <strong>Try it out</strong> 直接调用。原始规范：<a id="raw-link" href="' + MOUNT + '/doc.json?lang=zh" target="_blank">' + MOUNT + '/doc.json?lang=zh</a>',
      loading: '加载 Swagger UI 中…',
      loadError: '<strong>无法从 CDN 加载 Swagger UI 资源。</strong><br>你可以直接访问 <code>' + MOUNT + '/doc.json</code> 查看原始 OpenAPI 3.0 JSON，或将 <code>swagger-ui-dist</code> 部署到本地静态服务。'
    },
    en: {
      title: ` + goStringLiteral(title) + `,
      intro: 'Interactive API reference. Endpoints marked with <strong>🔒</strong> require an authorization step: click <strong>Authorize</strong> at the top-right and paste a Bearer JWT, then use <strong>Try it out</strong> on each card. Raw spec: <a id="raw-link" href="' + MOUNT + '/doc.json?lang=en" target="_blank">' + MOUNT + '/doc.json?lang=en</a>',
      loading: 'Loading Swagger UI…',
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
      b.classList.toggle('active', b.getAttribute('data-lang') === currentLang);
    });
  }
  applyChrome();
  document.querySelectorAll('.langswitch button').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var lang = btn.getAttribute('data-lang');
      if (lang !== currentLang) setLang(lang);
    });
  });

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
      docExpansion: 'none',
      defaultModelsExpandDepth: 1,
      defaultModelExpandDepth: 2,
      displayRequestDuration: true,
      filter: true
    });
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

// goStringLiteral 把 Go 字符串编码成 JS 字符串字面量（含引号），
// 避免 title 里出现 " ` \\ 时把 JS 玩坏。
func goStringLiteral(s string) string {
	// 用 JSON 编码保证转义正确；JSON 字符串正好是合法的 JS 字符串字面量。
	// 手动实现一个最小版避免额外依赖。
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
				b = append(b, []byte{'\\', 'u', '0', '0', hex(byte(r>>4)), hex(byte(r&0xf))}...)
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
