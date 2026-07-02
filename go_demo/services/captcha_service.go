package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	mrand "math/rand"
	"strings"
	"sync"
	"time"

	"go-demo/database"
)

// CaptchaService 提供图形验证码的生成与校验。
// 使用 Redis 存储 (captcha_id → 大写字符串) 与 TTL；Redis 掉线时降级到进程内 map。
type CaptchaService struct {
	ttl time.Duration
}

var ErrCaptchaInvalid = errors.New("captcha is invalid or expired")

const (
	captchaAlphabet    = "23456789ABCDEFGHJKMNPQRSTUVWXYZ" // 去掉易混淆的 0/O/1/I/L
	captchaImageWidth  = 160
	captchaImageHeight = 56
	captchaLength      = 4
	captchaGlyphScale  = 4 // 5×7 位图 × 4 = 20×28 像素字符
)

func NewCaptchaService() *CaptchaService {
	return &CaptchaService{ttl: 3 * time.Minute}
}

// CaptchaChallenge 表示一次验证码挑战，返回给前端
type CaptchaChallenge struct {
	ID        string `json:"captcha_id"`
	Image     string `json:"image"` // data URI (image/png;base64,...)
	ExpiresIn int64  `json:"expires_in"`
}

// Generate 生成一次验证码并存入存储。
func (s *CaptchaService) Generate(ctx context.Context) (*CaptchaChallenge, error) {
	code := randomCode(captchaLength)
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	pngBytes, err := renderPNG(code)
	if err != nil {
		return nil, err
	}
	if err := captchaStore.set(ctx, id, code, s.ttl); err != nil {
		return nil, err
	}
	return &CaptchaChallenge{
		ID:        id,
		Image:     "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes),
		ExpiresIn: int64(s.ttl.Seconds()),
	}, nil
}

// Verify 校验一次输入。无论对错都会消费掉这次挑战，避免暴力尝试。
func (s *CaptchaService) Verify(ctx context.Context, id, answer string) error {
	id = strings.TrimSpace(id)
	answer = strings.TrimSpace(answer)
	if id == "" || answer == "" {
		return ErrCaptchaInvalid
	}
	expected, ok := captchaStore.take(ctx, id)
	if !ok {
		return ErrCaptchaInvalid
	}
	if !strings.EqualFold(expected, answer) {
		return ErrCaptchaInvalid
	}
	return nil
}

// ──────────────────────────────────────────────
// 存储：Redis 优先 + 进程内保底
// ──────────────────────────────────────────────

type captchaStorage struct {
	mu   sync.Mutex
	data map[string]captchaEntry
}

type captchaEntry struct {
	code      string
	expiresAt time.Time
}

var captchaStore = &captchaStorage{data: map[string]captchaEntry{}}

func (s *captchaStorage) redisKey(id string) string { return "captcha:" + id }

func (s *captchaStorage) set(ctx context.Context, id, code string, ttl time.Duration) error {
	if database.RedisClient != nil {
		reqCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		if err := database.RedisClient.Set(reqCtx, s.redisKey(id), code, ttl).Err(); err == nil {
			return nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = captchaEntry{code: code, expiresAt: time.Now().Add(ttl)}
	if len(s.data) > 4096 {
		now := time.Now()
		for k, v := range s.data {
			if v.expiresAt.Before(now) {
				delete(s.data, k)
			}
		}
	}
	return nil
}

func (s *captchaStorage) take(ctx context.Context, id string) (string, bool) {
	if database.RedisClient != nil {
		reqCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		val, err := database.RedisClient.GetDel(reqCtx, s.redisKey(id)).Result()
		if err == nil {
			return val, true
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[id]
	if !ok {
		return "", false
	}
	delete(s.data, id)
	if entry.expiresAt.Before(time.Now()) {
		return "", false
	}
	return entry.code, true
}

// ──────────────────────────────────────────────
// 生成随机码 / 渲染 PNG
// ──────────────────────────────────────────────

func randomCode(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "AAAA"
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = captchaAlphabet[int(b)%len(captchaAlphabet)]
	}
	return string(out)
}

func randomID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// renderPNG 生成简易验证码 PNG：
//   - 白底 + 深蓝字符（保证清晰）
//   - 每个字符独占一格，仅做垂直/水平微位移；不旋转，避免字符互相咬合
//   - 干扰点小、干扰线细，不覆盖字符主要笔画
func renderPNG(code string) ([]byte, error) {
	rng := mrand.New(mrand.NewSource(time.Now().UnixNano() ^ int64(hashString(code))))
	w, h := captchaImageWidth, captchaImageHeight
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// 背景：浅色渐变
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(240 + rng.Intn(15)),
				G: uint8(244 + rng.Intn(11)),
				B: uint8(252 + rng.Intn(3)),
				A: 255,
			})
		}
	}

	// 少量小干扰点
	for i := 0; i < 60; i++ {
		x := rng.Intn(w)
		y := rng.Intn(h)
		img.Set(x, y, color.RGBA{
			R: uint8(140 + rng.Intn(80)),
			G: uint8(140 + rng.Intn(80)),
			B: uint8(180 + rng.Intn(60)),
			A: 200,
		})
	}

	// 2 条细干扰线
	for i := 0; i < 2; i++ {
		x1, y1 := rng.Intn(w/3), rng.Intn(h)
		x2, y2 := w-1-rng.Intn(w/3), rng.Intn(h)
		lineColor := color.RGBA{
			R: uint8(100 + rng.Intn(80)),
			G: uint8(130 + rng.Intn(80)),
			B: uint8(180 + rng.Intn(50)),
			A: 200,
		}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}

	// 字符：均匀分格，每格中心 ± 少量抖动
	chars := []rune(code)
	slot := w / len(chars)
	charW := 5 * captchaGlyphScale
	charH := 7 * captchaGlyphScale
	palettes := []color.RGBA{
		{R: 30, G: 64, B: 175, A: 255},
		{R: 79, G: 70, B: 229, A: 255},
		{R: 15, G: 118, B: 110, A: 255},
		{R: 190, G: 24, B: 93, A: 255},
	}
	for i, r := range chars {
		// 每格中心 x
		centerX := slot*i + slot/2
		// 允许 ±(slot/2 - charW/2 - 2) 内水平抖动，保证字符不重叠
		maxJitterX := slot/2 - charW/2 - 2
		if maxJitterX < 0 {
			maxJitterX = 0
		}
		jitterX := 0
		if maxJitterX > 0 {
			jitterX = rng.Intn(maxJitterX*2+1) - maxJitterX
		}
		originX := centerX + jitterX - charW/2
		originY := (h-charH)/2 + rng.Intn(5) - 2
		c := palettes[rng.Intn(len(palettes))]
		drawGlyph(img, r, originX, originY, c)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// hashString 生成一个稳定的种子扰动
func hashString(s string) int {
	sum := 0
	for _, b := range []byte(s) {
		sum = sum*131 + int(b)
	}
	return sum
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := abs(x2 - x1)
	dy := -abs(y2 - y1)
	sx, sy := 1, 1
	if x1 >= x2 {
		sx = -1
	}
	if y1 >= y2 {
		sy = -1
	}
	err := dx + dy
	for {
		img.Set(x1, y1, c)
		if x1 == x2 && y1 == y2 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// drawGlyph 把 5×7 位图字符按 captchaGlyphScale 放大画到 (originX, originY)。
// 不做旋转，保持字形清晰。位图第 1 位对应左侧像素。
func drawGlyph(img *image.RGBA, r rune, originX, originY int, c color.RGBA) {
	glyph, ok := glyphBitmap5x7[r]
	if !ok {
		return
	}
	scale := captchaGlyphScale
	for y := 0; y < 7; y++ {
		row := glyph[y]
		for x := 0; x < 5; x++ {
			if (row>>(4-x))&1 == 0 {
				continue
			}
			// 把每个位图像素放大到 scale×scale 的方块
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					px := originX + x*scale + dx
					py := originY + y*scale + dy
					if px < 0 || py < 0 || px >= img.Bounds().Dx() || py >= img.Bounds().Dy() {
						continue
					}
					img.Set(px, py, c)
				}
			}
		}
	}
}

// 5×7 位图字符表：仅覆盖大写字母 + 数字（去掉了 0,1,I,L,O）。
// 每字符 7 行 × 5 位，位图第 1 位对应左侧像素。
var glyphBitmap5x7 = map[rune][7]uint8{
	'2': {0x0E, 0x11, 0x01, 0x02, 0x04, 0x08, 0x1F},
	'3': {0x1E, 0x01, 0x01, 0x0E, 0x01, 0x01, 0x1E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	'A': {0x0E, 0x11, 0x11, 0x11, 0x1F, 0x11, 0x11},
	'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D': {0x1E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1E},
	'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0E},
	'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'J': {0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
	'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'M': {0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11},
	'N': {0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
	'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'Q': {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S': {0x0F, 0x10, 0x10, 0x0E, 0x01, 0x01, 0x1E},
	'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V': {0x11, 0x11, 0x11, 0x11, 0x11, 0x0A, 0x04},
	'W': {0x11, 0x11, 0x11, 0x15, 0x15, 0x15, 0x0A},
	'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'Z': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
}
