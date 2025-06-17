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
	"os" // os パッケージをインポート
	"time"

	"github.com/go-sql-driver/mysql" // MySQLドライバをインポート
	"github.com/gorilla/mux"
)

// db 変数はグローバルで宣言
var db *sql.DB

// --- 構造体の定義 (変更なし) ---
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type Post struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Content   string    `json:"content"`
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

type Like struct {
	ID        int       `json:"id"`
	PostID    int       `json:"post_id"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func registerTLSConfig() {
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(/hackthon/db/"server-ca.pem")
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

	mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: []tls.Certificate{certs},
	})
}

func main() {
	// --- ここからデータベース接続部分の変更 ---
	// 環境変数からMySQLの接続情報を取得します。
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	// 環境変数が設定されていない場合はエラーを出すか、デフォルト値を設定します。
	if mysqlUser == "" || mysqlPwd == "" || mysqlHost == "" || mysqlDatabase == "" {
		log.Fatal("エラー: MySQL接続に必要な環境変数が設定されていません。MYSQL_USER, MYSQL_PWD, MYSQL_HOST, MYSQL_DATABASE を設定してください。")
	}
	registerTLSConfig()

	connStr := fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=custom", mysqlUser, mysqlPwd, mysqlHost, mysqlDatabase)

	// 接続文字列の作成
	// ホスト名にポート番号が含まれている可能性があるため、tcp() の形式で指定するのが一般的です。
	// 例: MYSQL_HOST="127.0.0.1:3306" の場合、そのまま使用できます。
	// もしMYSQL_HOSTが "127.0.0.1" のみで、ポートが別に設定されている場合は `host:port` の形式に調整してください。
	var err error
	db, err = sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("データベース接続エラー: %v", err)
	}
	defer db.Close()

	// データベースへのPINGで接続確認
	err = db.Ping()
	if err != nil {
		log.Fatalf("データベースPINGエラー: %v", err)
	}
	fmt.Println("MySQLデータベースに正常に接続しました！")
	// --- データベース接続部分の変更ここまで ---

	// ルーターの設定 (変更なし)
	router := mux.NewRouter()

	// CORSミドルウェア (変更なし)
	router.Use(corsMiddleware)

	// APIエンドポイントの定義 (変更なし)
	router.HandleFunc("/api/users", createUser).Methods("POST")
	router.HandleFunc("/api/posts", createPost).Methods("POST")
	router.HandleFunc("/api/replies", createReply).Methods("POST")
	router.HandleFunc("/api/likes", createLike).Methods("POST")

	fmt.Println("サーバーをポート8080で起動中...")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// CORSミドルウェア (変更なし)
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000") // Reactのポート
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// APIハンドラ関数群 (変更なし)
func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec("INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
		user.Username, user.Email, user.PasswordHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	user.ID = int(id)
	user.CreatedAt = time.Now()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	fmt.Printf("ユーザーを作成しました: %s\n", user.Username)
}

func createPost(w http.ResponseWriter, r *http.Request) {
	var post Post
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec("INSERT INTO posts (user_id, content) VALUES (?, ?)",
		post.UserID, post.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	post.ID = int(id)
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
	fmt.Printf("投稿を作成しました (UserID: %d): %s\n", post.UserID, post.Content)
}

func createReply(w http.ResponseWriter, r *http.Request) {
	var reply Reply
	err := json.NewDecoder(r.Body).Decode(&reply)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec("INSERT INTO replies (post_id, user_id, content) VALUES (?, ?, ?)",
		reply.PostID, reply.UserID, reply.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	reply.ID = int(id)
	reply.CreatedAt = time.Now()
	reply.UpdatedAt = time.Now()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
	fmt.Printf("返信を作成しました (PostID: %d, UserID: %d): %s\n", reply.PostID, reply.UserID, reply.Content)
}

func createLike(w http.ResponseWriter, r *http.Request) {
	var like Like
	err := json.NewDecoder(r.Body).Decode(&like)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := db.Exec("INSERT INTO likes (post_id, user_id) VALUES (?, ?)",
		like.PostID, like.UserID)
	if err != nil {
		mysqlErr, ok := err.(*mysql.MySQLError)
		if ok && mysqlErr.Number == 1062 {
			http.Error(w, "既にこの投稿に「いいね」しています。", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	like.ID = int(id)
	like.CreatedAt = time.Now()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(like)
	fmt.Printf("「いいね」を作成しました (PostID: %d, UserID: %d)\n", like.PostID, like.UserID)
}
