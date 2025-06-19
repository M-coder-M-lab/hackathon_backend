package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var db *sql.DB

// --- Structs ---
type User struct {
	ID        int       `json:"id"`
	UID       string    `json:"uid"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Post struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Content   string    `json:"content"`
	Likes     int       `json:"likes"`
	Replies   []Reply   `json:"replies"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Reply struct {
	ID        int       `json:"id"`
	PostID    int       `json:"post_id"`
	UserID    int       `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func registerTLSConfig() {
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile("/app/server-ca.pem")
	if err != nil {
		log.Fatal(err)
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		log.Fatal("CA証明書を追加できませんでした")
	}
	certs, err := tls.LoadX509KeyPair("client-cert.pem", "client-key.pem")
	if err != nil {
		log.Fatal(err)
	}
	err = mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs:            rootCertPool,
		Certificates:       []tls.Certificate{certs},
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Fatalf("TLS設定登録失敗: %v", err)
	}
}

func main() {
	registerTLSConfig()
	connStr := fmt.Sprintf("uttc:19b-apFqu4APTx4A@tcp(34.67.141.68:3306)/hackathon?tls=custom")
	var err error
	db, err = sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("データベース接続エラー: %v", err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatalf("DB接続失敗: %v", err)
	}
	fmt.Println("MySQLに接続成功")

	router := mux.NewRouter()
	router.Use(corsMiddleware)

	router.HandleFunc("/api/login", loginHandler).Methods("POST")
	router.HandleFunc("/api/posts", getPosts).Methods("GET")
	router.HandleFunc("/api/posts", createPost).Methods("POST")
	router.HandleFunc("/api/replies", createReply).Methods("POST")
	router.HandleFunc("/api/summary/{postId}", summarizeReplies).Methods("GET")

	log.Println("サーバー起動中 :8080")
	http.ListenAndServe(":8080", router)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UID      string `json:"uid"`
		Email    string `json:"email"`
		Username string `json:"username"`
	}
	json.NewDecoder(r.Body).Decode(&payload)
	var id int
	err := db.QueryRow("SELECT id FROM users WHERE uid = ?", payload.UID).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO users (uid, email, username) VALUES (?, ?, ?)", payload.UID, payload.Email, payload.Username)
		if err != nil {
			http.Error(w, "ユーザー作成エラー", http.StatusInternalServerError)
			return
		}
		id64, _ := res.LastInsertId()
		id = int(id64)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]int{"user_id": id})
}

func getPosts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, user_id, content, created_at, updated_at FROM posts ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "投稿取得失敗", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.UserID, &p.Content, &p.CreatedAt, &p.UpdatedAt)
		db.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ?", p.ID).Scan(&p.Likes)
		rpRows, _ := db.Query("SELECT id, post_id, user_id, content, created_at, updated_at FROM replies WHERE post_id = ?", p.ID)
		for rpRows.Next() {
			var r Reply
			rpRows.Scan(&r.ID, &r.PostID, &r.UserID, &r.Content, &r.CreatedAt, &r.UpdatedAt)
			p.Replies = append(p.Replies, r)
		}
		rpRows.Close()
		posts = append(posts, p)
	}
	json.NewEncoder(w).Encode(posts)
}

func createPost(w http.ResponseWriter, r *http.Request) {
	var post Post
	json.NewDecoder(r.Body).Decode(&post)
	res, err := db.Exec("INSERT INTO posts (user_id, content) VALUES (?, ?)", post.UserID, post.Content)
	if err != nil {
		http.Error(w, "投稿作成エラー", http.StatusInternalServerError)
		return
	}
	id64, _ := res.LastInsertId()
	post.ID = int(id64)
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()
	json.NewEncoder(w).Encode(post)
}

func createReply(w http.ResponseWriter, r *http.Request) {
	var reply Reply
	json.NewDecoder(r.Body).Decode(&reply)
	res, err := db.Exec("INSERT INTO replies (post_id, user_id, content) VALUES (?, ?, ?)", reply.PostID, reply.UserID, reply.Content)
	if err != nil {
		http.Error(w, "リプライ作成エラー", http.StatusInternalServerError)
		return
	}
	id64, _ := res.LastInsertId()
	reply.ID = int(id64)
	reply.CreatedAt = time.Now()
	reply.UpdatedAt = time.Now()
	json.NewEncoder(w).Encode(reply)
}

func summarizeReplies(w http.ResponseWriter, r *http.Request) {
	postID := mux.Vars(r)["postId"]
	replies, _ := db.Query("SELECT content FROM replies WHERE post_id = ?", postID)
	var all string
	for replies.Next() {
		var content string
		replies.Scan(&content)
		all += content + "\n"
	}
	summary := callGeminiAPI(all)
	json.NewEncoder(w).Encode(map[string]string{"summary": summary})
}

func callGeminiAPI(text string) string {
	// Gemini APIのエンドポイントとAPIキーを設定
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=" + "AIzaSyDYJCxH5qH2glxiiVlW6rzrcZE8ixeyPBI"

	// リクエストのペイロードを JSON で作成
	payload := []byte(fmt.Sprintf(`{
		"contents": [{
			"parts": [{"text": "次のリプライ群を要約してください:\n%s"}]
		}]
	}`, text))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("リクエスト作成失敗: %v", err)
		return "要約エラー"
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Gemini API 呼び出し失敗: %v", err)
		return "要約エラー"
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	// JSONからレスポンスを抽出（簡易的）
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	json.Unmarshal(body, &result)

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text
	}
	return "要約結果なし"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b

}
