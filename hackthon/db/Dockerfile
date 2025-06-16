# ビルドステージ: Goアプリケーションをビルド
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 依存関係をダウンロードしてキャッシュする
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピーしてビルド
COPY . .
RUN go build -o /app/main .

# 最終的なイメージ: 軽量なスクラッチイメージを使用
FROM scratch

# タイムゾーンデータをコピー (タイムゾーン情報が必要な場合)
# Alpine Linuxベースの場合、tzdataをインストールしてからコピーする場合がありますが、
# scratchベースの場合は別途含める必要があります。
# 通常、Goのtimeパッケージが参照するシステムタイムゾーンデータは、
# コンテナに存在しない場合はUTCとして扱われるため、多くの場合はこれで十分です。
# 必要に応じて以下を追加:
# COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# ENV TZ=Asia/Tokyo

WORKDIR /app

# ビルドしたバイナリをコピー
COPY --from=builder /app/main .

# コンテナがリッスンするポート。Cloud Runは環境変数PORTを自動的に設定します。
# アプリケーションがリッスンするポートと一致させる必要があります。
EXPOSE 8080

# コンテナ起動時に実行するコマンド
CMD ["/app/main"]
