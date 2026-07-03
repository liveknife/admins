<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";

defineOptions({ name: "LoginIllustration3D" });

const canvasRef = ref<HTMLCanvasElement>();
const cityName = ref("定位中...");
let rafId = 0;

/* ── IP 地理定位 ── */
interface GeoInfo {
  city: string;
  lat: number;
  lon: number;
}

async function fetchGeoLocation(): Promise<GeoInfo> {
  try {
    const res = await fetch(
      "http://ip-api.com/json/?lang=zh-CN&fields=status,city,lat,lon",
      { signal: AbortSignal.timeout(5000) }
    );
    const data = await res.json();
    if (data.status === "success") {
      return { city: data.city || "未知位置", lat: data.lat ?? 30, lon: data.lon ?? 105 };
    }
  } catch { /* 静默 */ }
  return { city: "未知位置", lat: 30, lon: 105 };
}

/* ── 噪声函数（基于坐标种子生成确定性地形） ── */
function createNoise(seed: number) {
  const perm = new Uint8Array(512);
  const p = new Uint8Array(256);
  for (let i = 0; i < 256; i++) p[i] = i;
  let s = seed | 0;
  for (let i = 255; i > 0; i--) {
    s = (s * 1664525 + 1013904223) & 0xffffffff;
    const j = ((s >>> 16) & 0x7fff) % (i + 1);
    [p[i], p[j]] = [p[j], p[i]];
  }
  for (let i = 0; i < 512; i++) perm[i] = p[i & 255];

  const fade = (t: number) => t * t * t * (t * (t * 6 - 15) + 10);
  const lerp = (a: number, b: number, t: number) => a + t * (b - a);
  const grad = (h: number, x: number, y: number) => {
    const v = h & 1 ? y : x;
    return (h & 1 ? -v : v) + (h & 2 ? (h & 3 ? -x : -y) : 0);
  };

  return (x: number, y: number): number => {
    const X = Math.floor(x) & 255,
      Y = Math.floor(y) & 255;
    const xf = x - Math.floor(x),
      yf = y - Math.floor(y);
    const u = fade(xf),
      v = fade(yf);
    const A = perm[X] + Y,
      B = perm[X + 1] + Y;
    return lerp(
      lerp(grad(perm[A], xf, yf), grad(perm[B], xf - 1, yf), u),
      lerp(grad(perm[A + 1], xf, yf - 1), grad(perm[B + 1], xf - 1, yf - 1), u),
      v
    );
  };
}

function fbm(n: (x: number, y: number) => number, x: number, y: number): number {
  let v = 0,
    a = 1,
    f = 1,
    m = 0;
  for (let i = 0; i < 5; i++) {
    v += a * n(x * f, y * f);
    m += a;
    a *= 0.5;
    f *= 2.1;
  }
  return v / m;
}

onMounted(async () => {
  const canvas = canvasRef.value;
  const ctx = canvas?.getContext("2d");
  if (!canvas || !ctx) return;

  // ── 获取地理位置 ──
  const geo = await fetchGeoLocation();
  cityName.value = geo.city;

  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  const W = 320,
    H = 400;
  canvas.width = W * dpr;
  canvas.height = H * dpr;
  canvas.style.width = `${W}px`;
  canvas.style.height = `${H}px`;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  // ── 地形网格参数 ──
  const GRID = 24;
  const noiseSeed = ((geo.lat * 1000) | 0) ^ (((geo.lon * 1000) | 0) << 16);
  const noise = createNoise(noiseSeed);
  const offsetX = (geo.lon % 7) * 0.6;
  const offsetY = (geo.lat % 7) * 0.6;

  // 生成高度图
  const heights: number[][] = [];
  for (let gy = 0; gy <= GRID; gy++) {
    heights[gy] = [];
    for (let gx = 0; gx <= GRID; gx++) {
      const nx = (gx / GRID) * 2.8 + offsetX;
      const ny = (gy / GRID) * 2.8 + offsetY;
      let h = fbm(noise, nx, ny);
      // 距离中心越远高度略降（盆地效果）
      const dcx = (gx - GRID / 2) / (GRID / 2);
      const dcy = (gy - GRID / 2) / (GRID / 2);
      const distCenter = Math.sqrt(dcx * dcx + dcy * dcy);
      h -= distCenter * 0.15;
      heights[gy][gx] = h;
    }
  }

  // 归一化到 0~1
  let minH = Infinity,
    maxH = -Infinity;
  for (const row of heights)
    for (const h of row) {
      if (h < minH) minH = h;
      if (h > maxH) maxH = h;
    }
  const range = maxH - minH || 1;
  for (let gy = 0; gy <= GRID; gy++)
    for (let gx = 0; gx <= GRID; gx++) {
      heights[gy][gx] = (heights[gy][gx] - minH) / range;
    }

  // ── 鼠标交互 ──
  let mouseX = 0,
    mouseY = 0;
  const handleMouse = (e: MouseEvent) => {
    const rect = canvas.getBoundingClientRect();
    mouseX = ((e.clientX - rect.left) / rect.width) * 2 - 1;
    mouseY = ((e.clientY - rect.top) / rect.height) * 2 - 1;
  };
  const resetMouse = () => {
    mouseX = 0;
    mouseY = 0;
  };
  canvas.addEventListener("mousemove", handleMouse);
  canvas.addEventListener("mouseleave", resetMouse);

  const cx = W / 2,
    cy = H / 2 + 25;
  const baseScale = 115;
  const fov = 300;
  const heightAmp = 55; // 高度放大系数

  // 投影
  const project = (x: number, y: number, z: number): { x: number; y: number; z: number } => {
    const zz = z + fov;
    return {
      x: cx + (x / zz) * fov * baseScale,
      y: cy + (y / zz) * fov * baseScale,
      z
    };
  };

  const rotateX = (v: number[], ang: number): number[] => {
    const c = Math.cos(ang),
      s = Math.sin(ang);
    return [v[0], c * v[1] - s * v[2], s * v[1] + c * v[2]];
  };
  const rotateY = (v: number[], ang: number): number[] => {
    const c = Math.cos(ang),
      s = Math.sin(ang);
    return [c * v[0] - s * v[2], v[1], s * v[0] + c * v[2]];
  };
  const rotateZ = (v: number[], ang: number): number[] => {
    const c = Math.cos(ang),
      s = Math.sin(ang);
    return [c * v[0] - s * v[1], s * v[0] + c * v[1], v[2]];
  };

  let time = 0;

  const draw = () => {
    time += 0.008;
    ctx.clearRect(0, 0, W, H);

    // 纯鼠标控制旋转，无自动旋转
    const rotX = mouseY * 0.5 + 0.5;
    const rotY = mouseX * 0.7 + 0.45;
    const rotZ = mouseX * 0.15;

    // 网格坐标映射：将网格归一化到 [-1, 1]
    const gridTo3d = (gx: number, gy: number, h: number): number[] => [
      (gx / GRID) * 2 - 1,
      h * 1.2 - 0.3, // Y轴为高度
      (gy / GRID) * 2 - 1
    ];

    // 计算所有顶点的投影
    const proj: { x: number; y: number; z: number }[][] = [];
    for (let gy = 0; gy <= GRID; gy++) {
      proj[gy] = [];
      for (let gx = 0; gx <= GRID; gx++) {
        const raw = gridTo3d(gx, gy, heights[gy][gx]);
        const r = rotateZ(rotateY(rotateX(raw, rotX), rotY), rotZ);
        proj[gy][gx] = project(r[0], r[1], r[2]);
      }
    }

    // 收集四边形面并按深度排序
    interface Face {
      pts: { x: number; y: number; z: number }[];
      avgZ: number;
      avgH: number;
      gx: number;
      gy: number;
    }
    const faces: Face[] = [];
    for (let gy = 0; gy < GRID; gy++) {
      for (let gx = 0; gx < GRID; gx++) {
        const p00 = proj[gy][gx];
        const p10 = proj[gy][gx + 1];
        const p01 = proj[gy + 1][gx];
        const p11 = proj[gy + 1][gx + 1];
        const avgZ = (p00.z + p10.z + p01.z + p11.z) / 4;
        const avgH =
          (heights[gy][gx] +
            heights[gy][gx + 1] +
            heights[gy + 1][gx] +
            heights[gy + 1][gx + 1]) /
          4;
        faces.push({ pts: [p00, p10, p01, p11], avgZ, avgH, gx, gy });
      }
    }
    faces.sort((a, b) => a.avgZ - b.avgZ);

    // 绘制地形面
    for (const f of faces) {
      const [p00, p10, p01, p11] = f.pts;

      // 面颜色基于高度
      const h = f.avgH;
      let color: string;
      if (h < 0.28) {
        // 深谷 — 深蓝/紫
        const t = h / 0.28;
        color = `rgba(${40 + t * 20}, ${50 + t * 60}, ${140 + t * 80}, 0.92)`;
      } else if (h < 0.5) {
        // 平原/低地 — 青绿
        const t = (h - 0.28) / 0.22;
        color = `rgba(${60 + t * 80}, ${110 + t * 70}, ${220 - t * 60}, 0.88)`;
      } else if (h < 0.72) {
        // 山坡 — 绿/黄
        const t = (h - 0.5) / 0.22;
        color = `rgba(${140 + t * 90}, ${180 + t * 40}, ${160 - t * 60}, 0.84)`;
      } else {
        // 山峰 — 橙/白
        const t = (h - 0.72) / 0.28;
        color = `rgba(${230 + t * 25}, ${200 + t * 55}, ${100 + t * 155}, 0.9)`;
      }

      // 绘制填充四边形
      ctx.beginPath();
      ctx.moveTo(p00.x, p00.y);
      ctx.lineTo(p10.x, p10.y);
      ctx.lineTo(p11.x, p11.y);
      ctx.lineTo(p01.x, p01.y);
      ctx.closePath();
      ctx.fillStyle = color;
      ctx.fill();
    }

    // 绘制线框网格（第二遍，覆盖在面上）
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    for (let gy = 0; gy <= GRID; gy++) {
      for (let gx = 0; gx < GRID; gx++) {
        const pa = proj[gy][gx],
          pb = proj[gy][gx + 1];
        const alpha = 0.12 + ((pa.z + pb.z) / 2 + 2) * 0.08;
        ctx.beginPath();
        ctx.moveTo(pa.x, pa.y);
        ctx.lineTo(pb.x, pb.y);
        ctx.strokeStyle = `rgba(120, 210, 255, ${alpha})`;
        ctx.lineWidth = 0.6;
        ctx.stroke();
      }
    }
    for (let gx = 0; gx <= GRID; gx++) {
      for (let gy = 0; gy < GRID; gy++) {
        const pa = proj[gy][gx],
          pb = proj[gy + 1][gx];
        const alpha = 0.12 + ((pa.z + pb.z) / 2 + 2) * 0.08;
        ctx.beginPath();
        ctx.moveTo(pa.x, pa.y);
        ctx.lineTo(pb.x, pb.y);
        ctx.strokeStyle = `rgba(120, 210, 255, ${alpha})`;
        ctx.lineWidth = 0.6;
        ctx.stroke();
      }
    }

    // 绘制山顶光点（高亮点）
    const peaks: { x: number; y: number; z: number; h: number }[] = [];
    for (let gy = 0; gy <= GRID; gy += 3) {
      for (let gx = 0; gx <= GRID; gx += 3) {
        const h = heights[gy][gx];
        if (h > 0.65) peaks.push({ ...proj[gy][gx], h });
      }
    }
    peaks.sort((a, b) => a.z - b.z);
    for (const pk of peaks) {
      const depthF = Math.max(0.4, Math.min(1.2, (pk.z + 2) / 3));
      const r = 2 * depthF * (pk.h - 0.55) * 3;

      const glow = ctx.createRadialGradient(pk.x, pk.y, 0, pk.x, pk.y, r * 6);
      glow.addColorStop(0, `rgba(180, 240, 255, ${0.4 * depthF})`);
      glow.addColorStop(1, "transparent");
      ctx.fillStyle = glow;
      ctx.fillRect(pk.x - r * 6, pk.y - r * 6, r * 12, r * 12);

      ctx.beginPath();
      ctx.arc(pk.x, pk.y, Math.max(0.8, r), 0, Math.PI * 2);
      ctx.fillStyle = `rgba(230, 250, 255, ${depthF * 0.85})`;
      ctx.fill();
    }

    // 底部城市名已移至 HTML 覆盖层，避免被 3D 模型遮挡

    rafId = requestAnimationFrame(draw);
  };

  draw();

  onBeforeUnmount(() => {
    cancelAnimationFrame(rafId);
    canvas.removeEventListener("mousemove", handleMouse);
    canvas.removeEventListener("mouseleave", resetMouse);
  });
});
</script>

<template>
  <div class="terrain-wrap">
    <canvas ref="canvasRef" class="terrain-3d" aria-label="3D terrain map of your location" />
    <div class="city-label">{{ cityName }}</div>
  </div>
</template>

<style scoped>
.terrain-wrap {
  position: relative;
  width: 380px;
  height: 440px;
  overflow: visible;
}

.terrain-3d {
  display: block;
  width: 100%;
  height: 100%;
  cursor: grab;
}

.terrain-3d:active {
  cursor: grabbing;
}

.city-label {
  position: absolute;
  bottom: 18px;
  left: 0;
  right: 0;
  text-align: center;
  font: 600 14px "Inter", "SF Pro Display", system-ui, sans-serif;
  color: rgba(160, 220, 255, 0.85);
  pointer-events: none;
  z-index: 20;
  letter-spacing: 2px;
  text-shadow:
    0 0 10px rgba(100, 200, 255, 0.6),
    0 2px 12px rgba(4, 12, 32, 0.9),
    0 1px 3px rgba(0, 0, 0, 0.8);
}
</style>
