package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os" // osパッケージをインポート

	"github.com/go-sql-driver/mysql" // MySQLドライバをインポート
	"github.com/rs/cors"             // CORSミドルウェア
)

// Post represents a social media post.
type Post struct {
	ID        int     `json:"id"`
	UserID    int     `json:"user_id"`
	Content   string  `json:"content"`
	Likes     int     `json:"likes"`
	Replies   []Reply `json:"replies"`
	CreatedAt string  `json:"created_at"`
}

// Reply represents a reply to a post.
type Reply struct {
	ID        int    `json:"id"`
	PostID    int    `json:"post_id"`
	UserID    int    `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

var db *sql.DB

// registerTLSConfig sets up the TLS configuration for the MySQL connection.
func registerTLSConfig() {
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile("/app/server-ca.pem")
	if err != nil {
		log.Fatalf("CA証明書の読み込み失敗: %v", err)
	}

	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		log.Fatal("CA証明書を追加できませんでした")
	}

	certs, err := tls.LoadX509KeyPair("client-cert.pem", "client-key.pem")
	if err != nil {
		log.Fatalf("クライアント証明書と秘密鍵の読み込み失敗: %v", err)
	}
	log.Println("クライアント証明書と秘密鍵を読み込みました")

	err = mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs:            rootCertPool,
		Certificates:       []tls.Certificate{certs},
		InsecureSkipVerify: true, // This should be false in production if you have proper host verification
	})
	if err != nil {
		log.Fatalf("TLS設定登録失敗: %v", err)
	}
}

// initDB initializes the database connection using environment variables and TLS.
func initDB() {
	registerTLSConfig() // TLS設定を最初に登録

	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	if mysqlUser == "" || mysqlPwd == "" || mysqlHost == "" || mysqlDatabase == "" {
		log.Fatal("エラー: MySQL接続に必要な環境変数が設定されていません。MYSQL_USER, MYSQL_PWD, MYSQL_HOST, MYSQL_DATABASE を設定してください。")
	}

	// 接続文字列の作成
	// TLSを使用する場合は `tls=custom` を追加します。
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&tls=custom",
		mysqlUser,
		mysqlPwd,
		mysqlHost,
		mysqlDatabase,
	)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("データベース接続エラー: %v", err)
	}

	// データベースへのPINGで接続確認
	err = db.Ping()
	if err != nil {
		log.Fatalf("データベースPINGエラー: %v", err)
	}
	fmt.Println("MySQLデータベースに正常に接続しました！")
}

// getPostsHandler retrieves all posts with their like counts and replies.
func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
	SELECT p.id, p.user_id, p.content, p.created_at,
	(SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS like_count
	FROM posts p ORDER BY p.created_at DESC`)
	if err != nil {
		http.Error(w, "データ取得失敗: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.UserID, &post.Content, &post.CreatedAt, &post.Likes); err != nil {
			http.Error(w, "データスキャン失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		replyRows, err := db.Query("SELECT id, post_id, user_id, content, created_at FROM replies WHERE post_id = ?", post.ID)
		if err != nil {
			http.Error(w, "リプライ取得失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// replyRowsはループごとにdeferされるため、正常にクローズされます。
		// ただし、大量の投稿がある場合にパフォーマンスに影響が出る可能性があります。
		// 全ての返信を一つのクエリで取得し、アプリケーション側で関連付ける方が効率的な場合があります。
		for replyRows.Next() {
			var reply Reply
			if err := replyRows.Scan(&reply.ID, &reply.PostID, &reply.UserID, &reply.Content, &reply.CreatedAt); err == nil {
				post.Replies = append(post.Replies, reply)
			}
		}
		replyRows.Close() // ここで明示的にクローズ

		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		log.Printf("JSONエンコード失敗: %v", err)
		http.Error(w, "レスポンスの生成失敗", http.StatusInternalServerError)
	}
}

// createPostHandler creates a new post.
func createPostHandler(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		UserID  int    `json:"user_id"`
		Content string `json:"content"`
	}
	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無効な入力: "+err.Error(), http.StatusBadRequest)
		return
	}
	_, err := db.Exec("INSERT INTO posts (user_id, content) VALUES (?, ?)", req.UserID, req.Content)
	if err != nil {
		http.Error(w, "投稿作成失敗: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Printf("投稿を作成しました (UserID: %d): %s\n", req.UserID, req.Content)
}

// likePostHandler handles liking a post.
func likePostHandler(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		UserID int `json:"user_id"`
		PostID int `json:"post_id"`
	}
	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無効な入力: "+err.Error(), http.StatusBadRequest)
		return
	}
	_, err := db.Exec("INSERT IGNORE INTO likes (user_id, post_id) VALUES (?, ?)", req.UserID, req.PostID)
	if err != nil {
		mysqlErr, ok := err.(*mysql.MySQLError)
		if ok && mysqlErr.Number == 1062 { // Duplicate entry error code for MySQL
			http.Error(w, "既にこの投稿に「いいね」しています。", http.StatusConflict)
			return
		}
		http.Error(w, "いいね失敗: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Printf("「いいね」を作成しました (PostID: %d, UserID: %d)\n", req.PostID, req.UserID)
}

// replyPostHandler handles replying to a post.
func replyPostHandler(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		UserID  int    `json:"user_id"`
		PostID  int    `json:"post_id"`
		Content string `json:"content"`
	}
	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無効な入力: "+err.Error(), http.StatusBadRequest)
		return
	}
	_, err := db.Exec("INSERT INTO replies (user_id, post_id, content) VALUES (?, ?, ?)", req.UserID, req.PostID, req.Content)
	if err != nil {
		http.Error(w, "リプライ送信失敗: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Printf("返信を作成しました (PostID: %d, UserID: %d): %s\n", req.PostID, req.UserID, req.Content)
}

func main() {
	initDB()
	defer db.Close() // main関数終了時にデータベース接続をクローズ

	mux := http.NewServeMux()
	mux.HandleFunc("/posts", getPostsHandler)
	mux.HandleFunc("/posts/create", createPostHandler)
	mux.HandleFunc("/posts/like", likePostHandler)
	mux.HandleFunc("/posts/reply", replyPostHandler)

	// CORSミドルウェアの設定
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Reactのポート
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		Debug:            true, // デバッグログを有効にする
	})

	handler := c.Handler(mux)
	fmt.Println("サーバー起動中 :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
