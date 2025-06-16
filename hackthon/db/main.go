package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/oklog/ulid"
)

// UserResForHTTPGet はHTTP GETリクエストに対するユーザー情報レスポンスの構造体です。
type UserResForHTTPGet struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// UserRequest はHTTP POSTリクエストのペイロードとして期待されるユーザー情報構造体です。
type UserRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// db はMySQLデータベース接続のためのグローバル変数です。
var db *sql.DB

// init 関数はプログラム起動時に自動的に実行され、データベース接続を初期化します。
func init() {
	// 環境変数からMySQLの接続情報を取得します。
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPwd := os.Getenv("MYSQL_PWD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")

	// 接続文字列をフォーマットします。
	// (例: user:password@tcp(host:port)/database)
	connStr := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", mysqlUser, mysqlPwd, mysqlHost, mysqlDatabase)

	var err error // err変数をここで宣言し、シャドウイングを防ぎます。
	// MySQLデータベースへの接続を開きます。
	db, err = sql.Open("mysql", connStr)
	if err != nil {
		log.Fatalf("データベース接続のオープンに失敗しました: %v\n", err)
	}

	// データベースへの接続をテストします。
	if err := db.Ping(); err != nil {
		log.Fatalf("データベースへの接続確認(Ping)に失敗しました: %v\n", err)
	}

	log.Println("データベースへの接続に成功しました。")
}

// generateULID は新しいULIDを生成して文字列として返します。
func generateULID() string {
	t := time.Now()
	// MonotonicはULIDの後半部分（エントロピー）を生成し、同じミリ秒内の連続する生成に対して単調増加を保証します。
	// rand.NewSource(t.UnixNano())は乱数ジェネレータのシードを設定します。
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

// handler はHTTPリクエストを処理するハンドラ関数です。
// GETリクエストでは名前でユーザーを検索し、POSTリクエストでは新しいユーザーを登録します。
func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// GETリクエストの処理
		// URLクエリパラメータから'name'を取得します。
		name := r.URL.Query().Get("name")
		if name == "" {
			log.Println("エラー: 名前が空です。")
			http.Error(w, "名前は必須です。", http.StatusBadRequest)
			return
		}

		// データベースから指定された名前のユーザーを検索します。
		rows, err := db.Query("SELECT id, name, age FROM user WHERE name = ?", name)
		if err != nil {
			log.Printf("データベースクエリの実行に失敗しました: %v\n", err)
			http.Error(w, "サーバーエラーが発生しました。", http.StatusInternalServerError)
			return
		}
		// 関数終了時に必ず行セットをクローズします。
		defer func() {
			if err := rows.Close(); err != nil {
				log.Printf("行セットのクローズに失敗しました: %v\n", err)
				// ここではエラーをログに記録するだけですが、より堅牢なシステムでは追加の処理が必要かもしれません。
			}
		}()

		// 取得したユーザー情報を格納するためのスライスを初期化します。
		users := make([]UserResForHTTPGet, 0)
		// 行セットを繰り返し処理し、各ユーザー情報をスキャンします。
		for rows.Next() {
			var u UserResForHTTPGet
			if err := rows.Scan(&u.Id, &u.Name, &u.Age); err != nil {
				log.Printf("行データのスキャンに失敗しました: %v\n", err)
				http.Error(w, "サーバーエラーが発生しました。", http.StatusInternalServerError)
				return
			}
			users = append(users, u)
		}

		// ユーザー情報をJSON形式にシリアライズします。
		bytes, err := json.Marshal(users)
		if err != nil {
			log.Printf("JSONのエンコードに失敗しました: %v\n", err)
			http.Error(w, "サーバーエラーが発生しました。", http.StatusInternalServerError)
			return
		}
		// レスポンスヘッダにContent-Typeを設定し、JSONを書き込みます。
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)

	case http.MethodPost:
		// POSTリクエストの処理
		var userReq UserRequest
		// リクエストボディからJSONをパースします。
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&userReq); err != nil {
			log.Printf("リクエストボディのJSONデコードに失敗しました: %v\n", err)
			http.Error(w, "不正なリクエストボディです。", http.StatusBadRequest)
			return
		}

		// 入力値のバリデーションを行います。
		if userReq.Name == "" || len(userReq.Name) > 50 {
			log.Println("エラー: 名前の形式が不正です。")
			http.Error(w, "名前は必須で、50文字以内である必要があります。", http.StatusBadRequest)
			return
		}
		if userReq.Age < 20 || userReq.Age > 80 {
			log.Println("エラー: 年齢の範囲が不正です。")
			http.Error(w, "年齢は20歳から80歳の範囲である必要があります。", http.StatusBadRequest)
			return
		}

		// トランザクションを開始します。
		tx, err := db.Begin()
		if err != nil {
			log.Printf("トランザクションの開始に失敗しました: %v\n", err)
			http.Error(w, "サーバーエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		// 新しいULIDを生成し、ユーザー情報をデータベースに挿入します。
		id := generateULID()
		_, err = tx.Exec("INSERT INTO user (id, name, age) VALUES (?, ?, ?)", id, userReq.Name, userReq.Age)
		if err != nil {
			tx.Rollback() // エラーが発生した場合はロールバックします。
			log.Printf("ユーザーの挿入に失敗しました: %v\n", err)
			http.Error(w, "サーバーエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		// トランザクションをコミットします。
		if err := tx.Commit(); err != nil {
			log.Printf("トランザクションのコミットに失敗しました: %v\n", err)
			http.Error(w, "サーバーエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		// 成功レスポンスをJSON形式で返します。
		response := map[string]string{
			"id": id,
		}
		bytes, err := json.Marshal(response)
		if err != nil {
			log.Printf("JSONのエンコードに失敗しました: %v\n", err)
			http.Error(w, "サーバーエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // ステータスコード200 OK
		w.Write(bytes)

	default:
		// サポートされていないHTTPメソッドに対する処理
		log.Printf("エラー: サポートされていないHTTPメソッドです: %s\n", r.Method)
		http.Error(w, "サポートされていないHTTPメソッドです。", http.StatusMethodNotAllowed)
	}
}

// main 関数はプログラムのエントリーポイントです。
func main() {
	// "/user"パスに対するハンドラを設定します。
	http.HandleFunc("/user", handler)

	// Ctrl+C (SIGTERM, SIGINT) シグナルでDB接続をクローズする処理を設定します。
	closeDBWithSysCall()

	// 8000番ポートでHTTPサーバーを起動し、リクエストを待ち受けます。
	log.Println("HTTPサーバーをポート8000で起動中...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("HTTPサーバーの起動に失敗しました: %v\n", err)
	}
}

// closeDBWithSysCall は、システムコール（SIGTERM, SIGINT）を受信した際に
// データベース接続をクローズするためのゴルーチンを開始します。
func closeDBWithSysCall() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT) // SIGTERMとSIGINTを監視します。
	go func() {
		s := <-sig // シグナルを受信するまでブロックします。
		log.Printf("システムコールを受信しました: %v", s)

		// データベース接続をクローズします。
		if err := db.Close(); err != nil {
			log.Fatalf("データベースのクローズに失敗しました: %v\n", err)
		}
		log.Printf("データベースのクローズに成功しました。プログラムを終了します。")
		os.Exit(0) // プログラムを正常終了します。
	}()
}
