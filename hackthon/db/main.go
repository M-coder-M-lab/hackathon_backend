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

type User struct {
	ID           int       `json:"id"`
	UID          string    `json:"uid"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Firebase使ってるなら使わない想定
	CreatedAt    time.Time `json:"created_at"`
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
	connStr := fmt.Sprintf("uttc:19b-apFqu4APTx4A@tcp(34.67.141.68:3306)/hackathon?tls=custom&parseTime=true")
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

	// OPTIONS リクエストにも対応
	router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://hackthon-o9kp.vercel.app")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
	})

	// ルート登録
	router.HandleFunc("/api/login", loginHandler).Methods("POST")
	router.HandleFunc("/api/posts", getPosts).Methods("GET")
	router.HandleFunc("/api/posts", createPost).Methods("POST")
	router.HandleFunc("/api/replies", createReply).Methods("POST")
	router.HandleFunc("/api/summary/{postId}", summarizeReplies).Methods("GET")
	router.HandleFunc("/api/likes", createLike).Methods("POST")

	log.Println("サーバー起動中 :8080")
	http.ListenAndServe(":8080", router)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://hackthon-o9kp.vercel.app")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func createLike(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UID    string `json:"uid"`
		PostID int    `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "不正なリクエスト", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE uid = ?", payload.UID).Scan(&userID)
	if err != nil {
		http.Error(w, "ユーザーID取得エラー", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT IGNORE INTO likes (user_id, post_id) VALUES (?, ?)", userID, payload.PostID)
	if err != nil {
		http.Error(w, "いいね作成エラー", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "いいね登録完了"})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UID      string `json:"uid"`
		Email    string `json:"email"`
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("JSONエラー: %v", err)
		http.Error(w, "無効なリクエスト", http.StatusBadRequest)
		return
	}

	var id int
	err := db.QueryRow("SELECT id FROM users WHERE uid = ?", payload.UID).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec(`INSERT INTO users (uid, username, email, created_at) VALUES (?, ?, ?, ?)`,
			payload.UID, payload.Username, payload.Email, time.Now())
		if err != nil {
			log.Printf("ユーザーINSERT失敗: %v", err)
			http.Error(w, "ユーザー作成エラー", http.StatusInternalServerError)
			return
		}
		lastID, _ := res.LastInsertId()
		id = int(lastID)
	} else if err != nil {
		log.Printf("ユーザーSELECT失敗: %v", err)
		http.Error(w, "ユーザー検索失敗", http.StatusInternalServerError)
		return
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
		err := rows.Scan(&p.ID, &p.UserID, &p.Content, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			log.Printf("投稿読み取りエラー: %v", err)
			continue
		}

		// ★★ Replies を空スライスで初期化（← これが重要）
		p.Replies = []Reply{}

		// いいね数を取得
		err = db.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ?", p.ID).Scan(&p.Likes)
		if err != nil {
			log.Printf("いいね数読み取りエラー: %v", err)
		}

		// リプライを取得
		rpRows, err := db.Query("SELECT id, post_id, user_id, content, created_at, updated_at FROM replies WHERE post_id = ?", p.ID)
		if err != nil {
			log.Printf("リプライ取得エラー: %v", err)
		} else {
			for rpRows.Next() {
				var r Reply
				err := rpRows.Scan(&r.ID, &r.PostID, &r.UserID, &r.Content, &r.CreatedAt, &r.UpdatedAt)
				if err != nil {
					log.Printf("リプライ読み取りエラー: %v", err)
					continue
				}
				p.Replies = append(p.Replies, r)
			}
			rpRows.Close()
		}

		posts = append(posts, p)
	}

	// JSONで返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

// func getPosts(w http.ResponseWriter, r *http.Request) {
// 	rows, err := db.Query("SELECT id, user_id, content, created_at, updated_at FROM posts ORDER BY created_at DESC")
// 	if err != nil {
// 		http.Error(w, "投稿取得失敗", http.StatusInternalServerError)
// 		return
// 	}
// 	defer rows.Close()

// 	var posts []Post
// 	for rows.Next() {
// 		var p Post
// 		rows.Scan(&p.ID, &p.UserID, &p.Content, &p.CreatedAt, &p.UpdatedAt)
// 		db.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ?", p.ID).Scan(&p.Likes)
// 		rpRows, _ := db.Query("SELECT id, post_id, user_id, content, created_at, updated_at FROM replies WHERE post_id = ?", p.ID)
// 		for rpRows.Next() {
// 			var r Reply
// 			rpRows.Scan(&r.ID, &r.PostID, &r.UserID, &r.Content, &r.CreatedAt, &r.UpdatedAt)
// 			p.Replies = append(p.Replies, r)
// 		}
// 		rpRows.Close()
// 		posts = append(posts, p)
// 	}
// 	json.NewEncoder(w).Encode(posts)
// }

// func createPost(w http.ResponseWriter, r *http.Request) {
// 	var post Post
// 	json.NewDecoder(r.Body).Decode(&post)
// 	res, err := db.Exec("INSERT INTO posts (user_id, content) VALUES (?, ?)", post.UserID, post.Content)
// 	if err != nil {
// 		http.Error(w, "投稿作成エラー", http.StatusInternalServerError)
// 		return
// 	}
// 	id64, _ := res.LastInsertId()
// 	post.ID = int(id64)
// 	post.CreatedAt = time.Now()
// 	post.UpdatedAt = time.Now()
// 	json.NewEncoder(w).Encode(post)
// }

// func createPost(w http.ResponseWriter, r *http.Request) {
// 	var post Post
// 	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
// 		http.Error(w, "JSON解析エラー", http.StatusBadRequest)
// 		return
// 	}

// 	now := time.Now()
// 	res, err := db.Exec("INSERT INTO posts (user_id, content, likes, created_at, updated_at) VALUES (?, ?, 0, ?, ?)",
// 		post.UserID, post.Content, now, now)
// 	if err != nil {
// 		log.Printf("投稿作成失敗: %v", err)
// 		http.Error(w, "投稿作成失敗", http.StatusInternalServerError)
// 		return
// 	}

// 	postID, _ := res.LastInsertId()
// 	post.ID = int(postID)
// 	post.CreatedAt = now
// 	post.UpdatedAt = now
// 	post.Likes = 0

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(post)
// }
func createPost(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UID     string `json:"uid"`     // ← uid を受け取る
		Content string `json:"content"` // ← 投稿内容
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "不正なリクエスト", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE uid = ?", payload.UID).Scan(&userID)
	if err != nil {
		http.Error(w, "ユーザーID取得エラー", http.StatusInternalServerError)
		return
	}

	res, err := db.Exec("INSERT INTO posts (user_id, content) VALUES (?, ?)", userID, payload.Content)
	if err != nil {
		http.Error(w, "投稿作成エラー", http.StatusInternalServerError)
		return
	}
	id64, _ := res.LastInsertId()
	post := Post{
		ID:        int(id64),
		UserID:    userID,
		Content:   payload.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Likes:     0,
		Replies:   []Reply{},
	}
	json.NewEncoder(w).Encode(post)
}

// func createReply(w http.ResponseWriter, r *http.Request) {
// 	var reply Reply
// 	json.NewDecoder(r.Body).Decode(&reply)
// 	res, err := db.Exec("INSERT INTO replies (post_id, user_id, content) VALUES (?, ?, ?)", reply.PostID, reply.UserID, reply.Content)
// 	if err != nil {
// 		http.Error(w, "リプライ作成エラー", http.StatusInternalServerError)
// 		return
// 	}
// 	id64, _ := res.LastInsertId()
// 	reply.ID = int(id64)
// 	reply.CreatedAt = time.Now()
// 	reply.UpdatedAt = time.Now()
// 	json.NewEncoder(w).Encode(reply)
// }
// func createReply(w http.ResponseWriter, r *http.Request) {
// 	var reply Reply
// 	if err := json.NewDecoder(r.Body).Decode(&reply); err != nil {
// 		http.Error(w, "JSON解析エラー", http.StatusBadRequest)
// 		return
// 	}

// 	now := time.Now()
// 	res, err := db.Exec(`INSERT INTO replies (post_id, user_id, content, created_at, updated_at)
// 		VALUES (?, ?, ?, ?, ?)`, reply.PostID, reply.UserID, reply.Content, now, now)
// 	if err != nil {
// 		log.Printf("リプライ作成失敗: %v", err)
// 		http.Error(w, "リプライ作成エラー", http.StatusInternalServerError)
// 		return
// 	}

// 	replyID, _ := res.LastInsertId()
// 	reply.ID = int(replyID)
// 	reply.CreatedAt = now
// 	reply.UpdatedAt = now

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(reply)
// }
func createReply(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UID     string `json:"uid"`
		PostID  int    `json:"post_id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "不正なリクエスト", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE uid = ?", payload.UID).Scan(&userID)
	if err != nil {
		http.Error(w, "ユーザーID取得エラー", http.StatusInternalServerError)
		return
	}

	res, err := db.Exec("INSERT INTO replies (post_id, user_id, content) VALUES (?, ?, ?)",
		payload.PostID, userID, payload.Content)
	if err != nil {
		http.Error(w, "リプライ作成エラー", http.StatusInternalServerError)
		return
	}
	id64, _ := res.LastInsertId()
	reply := Reply{
		ID:        int(id64),
		PostID:    payload.PostID,
		UserID:    userID,
		Content:   payload.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
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
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=AIzaSyDYJCxH5qH2glxiiVlW6rzrcZE8ixeyPBI"
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

