# go-sample/rest

<!-- mtoc-start -->

* [必要なツール](#必要なツール)
* [セットアップ手順](#セットアップ手順)
  * [ローカル開発環境（推奨）](#ローカル開発環境推奨)
  * [Docker環境（オプション）](#docker環境オプション)
* [ツール管理](#ツール管理)
  * [使用しているツール](#使用しているツール)
  * [ツールのインストール](#ツールのインストール)
  * [ツールの実行](#ツールの実行)
  * [Makefileコマンド](#makefileコマンド)
    * [ローカル開発用](#ローカル開発用)
    * [Docker用（オプション）](#docker用オプション)

<!-- mtoc-end -->
## 必要なツール

- Go 1.24 以上
- Docker

## セットアップ手順

### ローカル開発環境（推奨）

1. リポジトリのクローン

```bash
git clone <repository-url>
cd <repository-name>
```

2. 開発ツールのインストール

```bash
make setup  # ツールインストール + go mod tidy
```

3. サーバー起動

```bash
go run cmd/server/main.go
```

### Docker環境（オプション）

コンテナで実行したい場合は、以下のコマンドを使用します。

```bash
# Dockerイメージのビルド
make docker-build

# コンテナ起動
make docker-run
```

## ツール管理

このプロジェクトはGo 1.24の**ツールディレクティブ**機能を使用して、開発ツールの依存関係を管理しています。

### 使用しているツール

- **air** (v1.60.0) - ホットリロード
- **golangci-lint** (v1.64.5) - 静的解析
- **ogen** (v1.14.0) - OpenAPIコード生成

### ツールのインストール

ツールは`go.mod`の`tool`ディレクティブで管理されており、以下のコマンドで一括インストールできます：

```bash
go install tool
```

### ツールの実行

ツールは`go tool`コマンド経由で実行します：

```bash
# Linting
go tool golangci-lint run

# OpenAPIコード生成
go tool ogen --target internal/oas -package oas --clean api/openapi.yaml

# ホットリロード（開発時）
go tool air -c .air.toml
```

### Makefileコマンド

#### ローカル開発用

```bash
make test   # テスト実行
make lint   # 静的解析
make fmt    # コードフォーマット
make ogen   # OpenAPIコード生成
make tidy   # go mod tidy
```

#### Docker用（オプション）

```bash
make docker-build   # Dockerイメージビルド
make docker-run     # コンテナ起動
```
