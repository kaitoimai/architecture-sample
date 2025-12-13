# Terraformガイドライン

<!-- mtoc-start -->

* [モジュール構成と配置の要件](#モジュール構成と配置の要件)
  * [Stateの完全な環境分離](#stateの完全な環境分離)
  * [ルートモジュールのコンポジション化](#ルートモジュールのコンポジション化)
  * [実行権限の単一化](#実行権限の単一化)
  * [モジュールの役割定義](#モジュールの役割定義)
* [設計指針](#設計指針)
* [ディレクトリ構造と主要リソース](#ディレクトリ構造と主要リソース)
  * [ネットワーキング](#ネットワーキング)
  * [コンピューティング](#コンピューティング)
  * [ストレージ、データベース](#ストレージデータベース)
  * [セキュリティ](#セキュリティ)
  * [監視、可観測性](#監視可観測性)
* [カテゴリ定義と配置ルール](#カテゴリ定義と配置ルール)
  * [network](#network)
  * [data](#data)
  * [app](#app)
  * [ops](#ops)
* [実装のガイドライン](#実装のガイドライン)
  * [禁止事項: Terraform Moduleの利用](#禁止事項-terraform-moduleの利用)
  * [禁止事項: 環境差分による条件分岐](#禁止事項-環境差分による条件分岐)
  * [モジュールの粒度](#モジュールの粒度)

<!-- mtoc-end -->
## モジュール構成と配置の要件

### Stateの完全な環境分離

* 要件: 各環境（Develop/Staging/Production等）は、それぞれ独立したバックエンド設定（tfstate格納先）を持つこと。
* 目的: 環境ごとのStateファイルを物理的に分けることで、Stagingへの操作が誤ってProductionの状態を破壊するリスクを排除すること。

### ルートモジュールのコンポジション化

* 要件: 各環境のルートモジュール（`environments/*/main.tf`）にはリソース定義（`resource`ブロック）を直接記述せず、必ず「論理的な単位で分割されたモジュール」の呼び出しのみを行うこと。
* 目的: ルートモジュールを「何のリソースを使っているか」を示すマニフェストとして機能させ、巨大な `main.tf` の発生を防ぐこと。

### 実行権限の単一化

* 要件: 1つの `terraform plan&apply` など実行において、複数の認証情報（異なるService Account Keyの使い分けなど）を要求しない構成とすること。
* 目的: Terraformを操作できる認証情報が複数存在することによるセキュリティリスクを排除すること。認証フローをシンプルにし、ローカル実行およびCI/CDパイプラインでの実行難易度を下げること。

### モジュールの役割定義

* 要件: 「複数のモジュールから共通利用される部品」と、「各環境から呼び出される構成単位」の役割を混同させないこと。
* 意図: 再利用性を高めるべきコードと、特定の環境構成に特化したコードを明確に区別し、変更時の影響範囲を予測可能にすること。

## 設計指針

以下の原則に基づいて構成する。

1. 透明性
      * システムのアーキテクチャと依存関係が把握できること。
      * 高度な抽象化よりも、具体的な記述を優先すること。
2. コードとリソースの1:1対応
      * コードの状態がリソースの状態と直結していること。

## ディレクトリ構造と主要リソース

```
.
├── README.md
├── environments
│   └── develop
│       ├── backend.tf      # GCS backend configuration
│       ├── main.tf         # Main entry point with provider and module calls
│       ├── variables.tf    # Environment-specific variables
│       └── version.tf      # Terraform version constraints
└── modules
    ├── network/            # Network layer
    │   ├── vpc/            # VPC, Subnet, AlloyDB Peering
    │   ├── lb/             # Load Balancer & SSL, Cloud Armor
    │   └── access_context_manager/  # Access Context Manager policies
    ├── data/               # Data layer
    │   ├── alloydb/        # AlloyDB Cluster & Instance
    │   ├── gcs/            # Cloud Storage Buckets
    │   ├── registry/       # Artifact Registry (Docker images)
    │   └── secret-manager/ # Secret Manager references
    ├── app/                # Application layer
    │   ├── api-server/     # Cloud Run API Service
    │   ├── worker/         # Cloud Run Async Worker
    │   ├── db-migration/   # Cloud Run Migration Job
    │   ├── bastion/        # GCE Bastion & NAT
    │   └── messaging/      # Pub/Sub, Cloud Tasks, Eventarc
    └── ops/                # Operations layer
        ├── cloud-build/    # CI/CD Build Triggers
        └── monitoring/     # Alerts & Logging

```

### ネットワーキング

* プライベートサブネットを持つVPC
* AlloyDB用のVPCピアリング
* SSL/TLS終端を持つCloud Load Balancer（Cloud Armorによるアクセス制限付き）

### コンピューティング

* APIサーバーとバックグラウンドジョブ用のCloud Runサービス
* DBアクセス用の踏み台サーバー

### ストレージ、データベース

* AlloyDBクラスタ
* 静的コンテンツとデータファイル用のCloud Storageバケット
* Dockerイメージ用のArtifact Registry

### セキュリティ

* 機密設定のためのSecret Manager
* 最小権限のIAMサービスアカウント
* 可能な限りVPC内部通信を使用

### 監視、可観測性

* Cloud Monitoringアラート
* 外部監視サービスへの構造化ロギング
* イベント駆動型通知のためのPub/Sub

## カテゴリ定義と配置ルール

リソースを新規追加する際は、以下の基準に従って配置場所を決定すること。

### network

* 定義: システム全体の土台。これがないと何も始まらない。
* 特徴: 変更頻度は極めて低い。作り直しは困難。

### data

* 定義: 「ステート（データ）」を持つリソース。
* 特徴: `terraform destroy` するとデータロストが発生し、取り返しがつかないもの。

### app

* 定義: 計算リソース、またはビジネスロジックを処理するリソース。
* 特徴: ステートレスであり、削除しても再デプロイで復旧可能。

### ops

* 定義: アプリケーションやインフラを「管理・運用」するためのツール群。
* 特徴: システムの動作そのものには直接関与しない横断的なきのう。

## 実装のガイドライン

### 禁止事項: Terraform Moduleの利用

* 設計指針に反するため。
  * 過去に拝見した中で、個人的に最も詳細に言語化していると思った記事。
  * [なぜインフラコードのモジュール化は難しいのか - アプリケーションコードとの本質的な違いから考える](https://speakerdeck.com/mizzy/yapc-fukuoka-2025)

### 禁止事項: 環境差分による条件分岐

モジュール内部で `var.env` などを判定してリソースを作ったり作らなかったりするロジックは禁止とする。

* Bad: `modules/app/api-server/main.tf` 内で `if env == 'prod'` を書く。
* Good: `environments/producion/main.tf` で必要なモジュールだけを呼び出す。不要な環境ではモジュール定義自体を書かない。

### モジュールの粒度

* `modules/network` や `modules/app` 自体には `main.tf` を置かない（これらはディレクトリ）。
* 末端のディレクトリ（例: `modules/app/batch`）が1つの Terraform Module となる。
