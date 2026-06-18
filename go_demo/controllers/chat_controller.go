package controllers

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-demo/database"
	"go-demo/models"
	"go-demo/services"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────
// 常量
// ──────────────────────────────────────────────

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
const maxChatUploadSize = 25 << 20
const websocketReadTimeout = 45 * time.Second

// ──────────────────────────────────────────────
// ChatController 聊天控制器（含 WebSocket 引擎）
// ──────────────────────────────────────────────

type ChatController struct {
	db          *sql.DB
	authService *services.AuthService
	hub         *Hub
}

func NewChatController(db *sql.DB, authService *services.AuthService) *ChatController {
	return &ChatController{db: db, authService: authService, hub: NewHub()}
}

// ──────────────────────────────────────────────
// WebSocket 核心数据结构
// ──────────────────────────────────────────────

type Client struct {
	userID int64
	conn   net.Conn
	send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*Client]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[int64]map[*Client]struct{})} }

func (h *Hub) Add(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*Client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
}
func (h *Hub) Remove(c *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	cs := h.clients[c.userID]
	delete(cs, c)
	if len(cs) == 0 {
		delete(h.clients, c.userID)
		return false
	}
	return true
}
func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
func (h *Hub) SendTo(userID int64, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default:
		}
	}
}
func (h *Hub) Broadcast(payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, cs := range h.clients {
		for c := range cs {
			select {
			case c.send <- payload:
			default:
			}
		}
	}
}
func (h *Hub) Stats() (int, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users, connections := len(h.clients), 0
	for _, cs := range h.clients {
		connections += len(cs)
	}
	return users, connections
}

func (c *ChatController) WebSocketStats() (int, int) {
	return c.hub.Stats()
}

// ──────────────────────────────────────────────
// 请求/响应结构体
// ──────────────────────────────────────────────

type incomingMessage struct {
	Type        string `json:"type"`
	ToUserID    int64  `json:"to_user_id"`
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
	MediaURL    string `json:"media_url"`
	FileName    string `json:"file_name"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	Transcript  string `json:"transcript"`
	Translation string `json:"translation"`
}

type uploadResponse struct {
	URL         string `json:"url"`
	FileName    string `json:"file_name"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	MessageType string `json:"message_type"`
}

type translateRequest struct{ Text, TargetLang string }
type translateResponse struct{ TranslatedText, TargetLang string }

type wsEnvelope struct {
	Type       string              `json:"type"`
	Message    *models.ChatMessage `json:"message,omitempty"`
	UserID     int64               `json:"user_id,omitempty"`
	Online     bool                `json:"online,omitempty"`
	Timestamp  int64               `json:"timestamp,omitempty"`
	Error      string              `json:"error,omitempty"`
	ReadMsgIDs []int64             `json:"read_msg_ids,omitempty"`
}

// ──────────────────────────────────────────────
// HTTP Handler 方法
// ──────────────────────────────────────────────

func (c *ChatController) ListUsers(g *gin.Context) {
	currentID, ok := currentUserID(g)
	if !ok {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	users, err := c.listUsers(g.Request.Context())
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	for i := range users {
		users[i].Online = c.hub.IsOnline(users[i].ID)
		users[i].UnreadCount = c.unreadCount(currentID, users[i].ID)
	}
	g.JSON(http.StatusOK, gin.H{"users": users})
}

func (c *ChatController) History(g *gin.Context) {
	currentID, ok := currentUserID(g)
	if !ok {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	otherID, err := strconv.ParseInt(g.Param("id"), 10, 64)
	if err != nil || otherID <= 0 {
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	messages, err := c.history(currentID, otherID)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load history"})
		return
	}
	g.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (c *ChatController) Upload(g *gin.Context) {
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, maxChatUploadSize+1024*1024)
	fh, err := g.FormFile("file")
	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	if fh.Size <= 0 || fh.Size > maxChatUploadSize {
		g.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
		return
	}
	src, _ := fh.Open()
	defer src.Close()
	head := make([]byte, 512)
	n, _ := src.Read(head)
	src.Seek(0, io.SeekStart)
	mime := strings.TrimSpace(fh.Header.Get("Content-Type"))
	detected := http.DetectContentType(head[:n])
	if mime == "" || mime == "application/octet-stream" {
		mime = detected
	}
	name := cleanFileName(fh.Filename)
	ext := strings.ToLower(filepath.Ext(name))
	randName, _ := randomHex(16)
	dateDir := time.Now().Format("20060102")
	uploadDir := filepath.Join("uploads", "chat", dateDir)
	os.MkdirAll(uploadDir, 0755)
	diskPath := filepath.Join(uploadDir, randName+ext)
	dst, _ := os.Create(diskPath)
	defer dst.Close()
	written, _ := io.Copy(dst, src)
	url := "/" + filepath.ToSlash(filepath.Join("uploads", "chat", dateDir, randName+ext))
	g.JSON(http.StatusOK, uploadResponse{URL: url, FileName: name, MimeType: mime, FileSize: written, MessageType: messageKindFromMIME(mime)})
}

func (c *ChatController) Translate(g *gin.Context) {
	var req translateRequest
	if err := g.ShouldBindJSON(&req); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" || len([]rune(text)) > 2000 {
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid text"})
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.TargetLang))
	if target == "" {
		target = "zh"
	}
	g.JSON(http.StatusOK, translateResponse{TranslatedText: translateLocally(text, target), TargetLang: target})
}

func (c *ChatController) MarkRead(g *gin.Context) {
	currentID, ok := currentUserID(g)
	if !ok {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	fromID, err := strconv.ParseInt(g.Param("id"), 10, 64)
	if err != nil || fromID <= 0 {
		g.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// 把 fromID 发给 currentID 的未读消息全部标已读
	count, err := c.markRead(fromID, currentID)
	if err != nil {
		g.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark read"})
		return
	}
	if count > 0 {
		// 通知发送方：你发的消息已被读
		ids, _ := c.readMessageIDs(fromID, currentID)
		data, _ := json.Marshal(wsEnvelope{Type: "read", UserID: currentID, ReadMsgIDs: ids})
		c.hub.SendTo(fromID, data)
	}
	g.JSON(http.StatusOK, gin.H{"ok": true, "count": count})
}

func (c *ChatController) WebSocket(g *gin.Context) {
	token := strings.TrimSpace(g.Query("token"))
	claims, err := c.authService.ValidateAccessToken(token)
	if err != nil {
		g.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	allowed, _ := c.authService.UserHasPermission(g.Request.Context(), claims.UserID, "messages:chat")
	if !allowed {
		g.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}
	conn, err := upgradeWS(g)
	if err != nil {
		return
	}
	client := &Client{userID: claims.UserID, conn: conn, send: make(chan []byte, 32)}
	c.hub.Add(client)
	c.broadcastOnline(claims.UserID, true)
	go client.writePump()
	c.readPump(client)
	stillOnline := c.hub.Remove(client)
	close(client.send)
	conn.Close()
	if !stillOnline {
		c.broadcastOnline(claims.UserID, false)
	}
}

// ──────────────────────────────────────────────
// WebSocket 内部逻辑
// ──────────────────────────────────────────────

func (c *ChatController) readPump(client *Client) {
	client.conn.SetReadDeadline(time.Now().Add(websocketReadTimeout))
	for {
		payload, err := readTextFrame(client.conn)
		if err != nil {
			return
		}
		client.conn.SetReadDeadline(time.Now().Add(websocketReadTimeout))
		var input incomingMessage
		if json.Unmarshal(payload, &input) != nil {
			continue
		}
		if input.Type == "ping" {
			data, _ := json.Marshal(wsEnvelope{Type: "pong", Timestamp: time.Now().UnixMilli()})
			select {
			case client.send <- data:
			default:
			}
			continue
		}
		if input.Type != "message" || input.ToUserID <= 0 {
			sendSocketError(client, "消息格式不正确")
			continue
		}
		msg, err := normalizeIncomingMessage(input)
		if err != nil {
			sendSocketError(client, "消息内容不符合要求")
			continue
		}
		saved, _ := c.saveMessage(client.userID, input.ToUserID, msg)
		if saved == nil {
			sendSocketError(client, "聊天消息保存失败")
			continue
		}
		data, _ := json.Marshal(wsEnvelope{Type: "message", Message: saved})
		c.hub.SendTo(client.userID, data)
		if input.ToUserID != client.userID {
			c.hub.SendTo(input.ToUserID, data)
		}
	}
}

func sendSocketError(client *Client, text string) {
	data, _ := json.Marshal(wsEnvelope{Type: "error", Error: text})
	select {
	case client.send <- data:
	default:
	}
}

func (cl *Client) writePump() {
	for payload := range cl.send {
		writeTextFrame(cl.conn, payload)
	}
}

func (c *ChatController) broadcastOnline(userID int64, online bool) {
	data, _ := json.Marshal(wsEnvelope{Type: "online", UserID: userID, Online: online})
	c.hub.Broadcast(data)
}

// ──────────────────────────────────────────────
// 数据查询方法
// ──────────────────────────────────────────────

func (c *ChatController) listUsers(ctx interface{}) ([]models.ChatUser, error) {
	rows, err := database.Query(c.db, fmt.Sprintf(
		`SELECT u.id,u.username,u.email,u.phone,u.created_at,COALESCE(%s,'') AS roles,COALESCE(%s,'') AS permissions FROM users u LEFT JOIN user_roles ur ON ur.user_id=u.id LEFT JOIN roles r ON r.id=ur.role_id LEFT JOIN role_permissions rp ON rp.role_id=r.id LEFT JOIN permissions p ON p.id=rp.permission_id WHERE u.deleted_at IS NULL GROUP BY u.id,u.username,u.email,u.phone,u.created_at ORDER BY u.id ASC`,
		database.StringAgg(true, "r.name", ","), database.StringAgg(true, "p.code", ","),
	))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.ChatUser
	for rows.Next() {
		var u models.ChatUser
		var roles, perms string
		if rows.Scan(&u.ID, &u.Username, &u.Email, &u.Phone, &u.CreatedAt, &roles, &perms) != nil {
			return nil, err
		}
		u.Roles = splitCSV(roles)
		u.Permissions = splitCSV(perms)
		users = append(users, u)
	}
	return users, rows.Err()
}

func (c *ChatController) history(a, b int64) ([]models.ChatMessage, error) {
	left, right := a, b
	if left > right {
		left, right = right, left
	}
	rows, err := database.Query(c.db,
		`SELECT id,from_user_id,to_user_id,message_type,content,media_url,file_name,mime_type,file_size,transcript,translation,is_read,created_at FROM chat_messages WHERE ((from_user_id=$1 AND to_user_id=$2) OR (from_user_id=$3 AND to_user_id=$4)) ORDER BY id DESC LIMIT 100`,
		left, right, right, left,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if rows.Scan(&m.ID, &m.FromUserID, &m.ToUserID, &m.MessageType, &m.Content, &m.MediaURL, &m.FileName, &m.MimeType, &m.FileSize, &m.Transcript, &m.Translation, &m.IsRead, &m.CreatedAt) != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].ID < msgs[j].ID })
	return msgs, rows.Err()
}

func (c *ChatController) saveMessage(fromID, toID int64, input incomingMessage) (*models.ChatMessage, error) {
	var msgID int64
	insertSQL := database.RewriteSQL(`INSERT INTO chat_messages(from_user_id,to_user_id,message_type,content,media_url,file_name,mime_type,file_size,transcript,translation) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`)
	if database.CurrentDialect != nil && database.CurrentDialect.SupportsReturning() {
		c.db.QueryRow(insertSQL, fromID, toID, input.MessageType, input.Content, input.MediaURL, input.FileName, input.MimeType, input.FileSize, input.Transcript, input.Translation).Scan(&msgID)
	} else {
		result, _ := c.db.Exec(strings.Replace(insertSQL, " RETURNING id", "", -1), fromID, toID, input.MessageType, input.Content, input.MediaURL, input.FileName, input.MimeType, input.FileSize, input.Transcript, input.Translation)
		msgID, _ = result.LastInsertId()
	}
	var msg models.ChatMessage
	c.db.QueryRow(`SELECT id,from_user_id,to_user_id,message_type,content,media_url,file_name,mime_type,file_size,transcript,translation,is_read,created_at FROM chat_messages WHERE id=$1`, msgID).Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &msg.MessageType, &msg.Content, &msg.MediaURL, &msg.FileName, &msg.MimeType, &msg.FileSize, &msg.Transcript, &msg.Translation, &msg.IsRead, &msg.CreatedAt)
	return &msg, nil
}

// unreadCount 返回 fromID 发给 toID 的未读消息数
func (c *ChatController) unreadCount(toID, fromID int64) int64 {
	var count int64
	c.db.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE from_user_id=$1 AND to_user_id=$2 AND is_read=FALSE`, fromID, toID).Scan(&count)
	return count
}

// markRead 把 fromID 发给 toID 的未读消息全部标已读，返回影响行数
func (c *ChatController) markRead(fromID, toID int64) (int64, error) {
	result, err := c.db.Exec(`UPDATE chat_messages SET is_read=TRUE WHERE from_user_id=$1 AND to_user_id=$2 AND is_read=FALSE`, fromID, toID)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// readMessageIDs 返回 fromID 发给 toID 且已读的消息 ID 列表（最近100条）
func (c *ChatController) readMessageIDs(fromID, toID int64) ([]int64, error) {
	rows, err := database.Query(c.db, `SELECT id FROM chat_messages WHERE from_user_id=$1 AND to_user_id=$2 AND is_read=TRUE ORDER BY id DESC LIMIT 100`, fromID, toID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ──────────────────────────────────────────────
// 工具函数
// ──────────────────────────────────────────────

func normalizeIncomingMessage(input incomingMessage) (incomingMessage, error) {
	input.MessageType = strings.ToLower(strings.TrimSpace(input.MessageType))
	if input.MessageType == "" {
		input.MessageType = "text"
	}
	switch input.MessageType {
	case "text", "emoji":
		input.Content = strings.TrimSpace(input.Content)
		input.MediaURL = ""
		input.FileName = ""
		input.MimeType = ""
		input.FileSize = 0
		input.Transcript = ""
		input.Translation = ""
		if input.Content == "" || len([]rune(input.Content)) > 1000 {
			return input, errors.New("invalid content")
		}
	case "image", "video", "audio", "file":
		input.Content = strings.TrimSpace(input.Content)
		input.MediaURL = strings.TrimSpace(input.MediaURL)
		input.FileName = cleanFileName(input.FileName)
		input.MimeType = strings.TrimSpace(input.MimeType)
		input.Transcript = strings.TrimSpace(input.Transcript)
		input.Translation = strings.TrimSpace(input.Translation)
		if input.MediaURL == "" || !strings.HasPrefix(input.MediaURL, "/uploads/chat/") {
			return input, errors.New("invalid media url")
		}
	default:
		return input, errors.New("invalid message type")
	}
	return input, nil
}

func messageKindFromMIME(mime string) string {
	mime = strings.ToLower(mime)
	if strings.HasPrefix(mime, "image/") {
		return "image"
	}
	if strings.HasPrefix(mime, "video/") {
		return "video"
	}
	if strings.HasPrefix(mime, "audio/") {
		return "audio"
	}
	return "file"
}

func translateLocally(text, lang string) string {
	if strings.HasPrefix(lang, "en") {
		return replaceKnownPhrases(text, map[string]string{
			"你好": "hello", "谢谢": "thanks", "再见": "goodbye", "图片": "image", "视频": "video",
			"语音": "voice message", "文件": "file", "聊天": "chat", "发送": "send", "已连接": "connected", "未连接": "disconnected", "重连中": "reconnecting", "转文字": "transcribe", "翻译": "translate", "播放": "play", "暂停": "pause",
		})
	}
	return replaceKnownPhrases(text, map[string]string{
		"hello": "hello", "thanks": "thank you", "goodbye": "bye", "image": "image", "video": "video", "voice": "voice message", "file": "file", "chat": "chat", "send": "send", "connected": "ok", "disconnected": "off", "reconnecting": "retrying", "transcribe": "transcribe", "translate": "translate", "play": "play", "pause": "stop",
	})
}

func replaceKnownPhrases(text string, repls map[string]string) string {
	r := text
	for from, to := range repls {
		if from == "" {
			continue
		}
		r = strings.ReplaceAll(r, from, to)
		runes := []rune(from)
		runes[0] = []rune(strings.ToTitle(string(runes[0])))[0]
		r = strings.ReplaceAll(r, string(runes), to)
	}
	return r
}

func cleanFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "file"
	}
	if len([]rune(name)) > 180 {
		name = string([]rune(name)[:180])
	}
	return name
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
func splitCSV(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func currentUserID(c *gin.Context) (int64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// ──────────────────────────────────────────────
// 原生 WebSocket 协议实现
// ──────────────────────────────────────────────

func upgradeWS(c *gin.Context) (net.Conn, error) {
	if !strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		c.Status(http.StatusBadRequest)
		return nil, errors.New("not a websocket request")
	}
	key := c.GetHeader("Sec-WebSocket-Key")
	if key == "" {
		c.Status(http.StatusBadRequest)
		return nil, errors.New("missing websocket key")
	}
	hj, ok := c.Writer.(http.Hijacker)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return nil, errors.New("hijack unsupported")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", wsAccept(key))
	rw.Flush()
	return conn, nil
}

func wsAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func readTextFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 2)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}
	opcode := header[0] & 0x0f
	if opcode == 0x8 {
		return nil, io.EOF
	}
	if opcode != 0x1 {
		return nil, errors.New("unsupported frame")
	}
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7f)
	if length == 126 {
		ext := make([]byte, 2)
		_, _ = io.ReadFull(conn, ext)
		length = uint64(binary.BigEndian.Uint16(ext))
	} else if length == 127 {
		ext := make([]byte, 8)
		_, _ = io.ReadFull(conn, ext)
		length = binary.BigEndian.Uint64(ext)
	}
	if length > 8192 {
		return nil, errors.New("frame too large")
	}
	maskKey := make([]byte, 4)
	if masked {
		_, _ = io.ReadFull(conn, maskKey)
	}
	payload := make([]byte, length)
	_, _ = io.ReadFull(conn, payload)
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return payload, nil
}

func writeTextFrame(conn net.Conn, payload []byte) error {
	var header []byte
	l := len(payload)
	if l < 126 {
		header = []byte{0x81, byte(l)}
	} else if l <= 65535 {
		header = []byte{0x81, 126, 0, 0}
		binary.BigEndian.PutUint16(header[2:], uint16(l))
	} else {
		header = []byte{0x81, 127, 0, 0, 0, 0, 0, 0, 0, 0}
		binary.BigEndian.PutUint64(header[2:], uint64(l))
	}
	w := bufio.NewWriter(conn)
	w.Write(header)
	w.Write(payload)
	return w.Flush()
}
