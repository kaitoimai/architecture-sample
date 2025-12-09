# go-sample/rest

<!-- mtoc-start -->

* [概要](#概要)
* [アーキテクチャ](#アーキテクチャ)
* [主な機能](#主な機能)
* [プロジェクト構造](#プロジェクト構造)
* [必要なツール](#必要なツール)

<!-- mtoc-end -->

## 概要

OpenAPI仕様に基づいたRESTful APIサーバーのサンプル実装です。`ogen`を使用した型安全なコード生成、ホットリロード開発環境を備えています。

## アーキテクチャ

このサービスは **API Gateway の背後で動作することを前提** としています。

* **認証（Authentication）**: API Gatewayが JWT 署名検証を実施
* **本サービスの役割**: API Gatewayから渡されたJWTトークンからユーザー情報を抽出し、ロールベースのアクセス制御（RBAC）を実施

```
[Client] --> [API Gateway (JWT認証)] --> [This Service (RBAC認可)]
```

## 主な機能

* **OpenAPI駆動開発**: `ogen`による型安全なAPIコード自動生成
* **ロールベースアクセス制御（RBAC）**: ユーザーロール（admin/user）に基づく認可
* **ホットリロード**: `air`を使用した開発時の自動リロード
* **静的解析**: `golangci-lint`による品質チェック
* **Docker対応**: マルチステージビルドによる最適化されたコンテナイメージ

## プロジェクト構造

```
.
├── api/                 # OpenAPI仕様書
│   └── openapi.yaml
├── cmd/                 # エントリーポイント
│   ├── server/         # APIサーバー
│   └── cli/            # CLIツール
├── internal/           # プライベートコード
│   ├── auth/          # 認証・認可関連の型定義
│   ├── config/        # 設定管理
│   ├── handler/       # リクエストハンドラ
│   ├── middleware/    # ミドルウェア（JWT抽出、RBAC）
│   ├── oas/           # ogen生成コード
│   ├── server/        # サーバー実装
│   └── testutil/      # テストユーティリティ
├── build/              # Dockerファイル
├── scripts/            # ユーティリティスクリプト
└── docs/               # ドキュメント
```

## 必要なツール

* Go 1.24 or higher
* Docker
