新たに追加・確定された要件（Template画面でのYazi風ファイル操作 `mv`/`cp`、History画面での過去メッセージの `$EDITOR` 編集、3つの基本画面状態の整理、`--stdout` / ワンショット終了オプション）を完全に網羅して再構築した `SPEC.md` です。

---

# SPEC.md: m4llama Specification

**Version:** 0.4.0-draft

**Status:** In Review / Specification Complete

**Architecture Paradigm:** Domain-Driven Design (DDD), Clean Architecture, Unix Philosophy

**Target Environment:** POSIX Terminal (Linux / macOS), `llama-server` (OpenAI-compatible local inference)

---

## 1. Overview & Vision

**m4llama** は、ローカルで稼働する `llama-server` のために設計された、ターミナルネイティブで高い拡張性を持つプロンプトエンジニアリング・コックピット兼推論クライアントである。

Go 言語で自前実装された POSIX `m4` マクロプロセッサ、Yazi / Midnight Commander (MC) スタイルの堅牢なファイル管理・コンテキスト管理 2 ペイン TUI、動的フォーム生成（`charmbracelet/huh`）、および SQLite3 による完全な対話履歴管理を統合。

アプリは以下の **3つの基本画面ステート** を中心に構成され、キーボード駆動で決定論的なワークフローを提供する：

1. **History 画面:** コンテキストの執刀台。メッセージ一覧のプレビュー、過去メッセージの `$EDITOR` による編集、Context Sniping（`$` 埋め込み位置やロール指定）の再構成。
2. **Template 画面:** 弾頭の管理・装填台。Yazi そのもののようなツリーナビゲーションおよびファイル操作（`$EDITOR` 編集、`mv`、`cp`）。
3. **Form Input 画面:** 発射台。選択されたテンプレートの注釈変数（`# @var`）を動的入力し、History で組まれた Chain を組み込んで推論リクエストを送信。

また、**ワンショット実行（stdout 出力オプション）** に対応し、一度リクエストを送信してレスポンスを標準出力へ吐き出して即座に正常終了する Unix パイプライン親和性を保持する。

### コア思想 (Core Tenets)

1. **Unix 哲学に基づくモジュール性:** マクロ展開、TUI コックピット、コンテキスト外科手術（Context Sniping）、通信（`net/http` / `curl`）、ローカル永続化（SQLite3）を徹底的に疎結合化。


2. **自己完結型ゼロ依存エンジン:** 外部の GNU `m4` のインストールを一切要求せず、内蔵の pure-Go `m4` エンジン（`pkg/m4`）で動作。


3. **Yazi-Like テンプレートマネージャー:** テンプレートを単に選ぶだけでなく、TUI 上から直接 `$EDITOR` 編集、`mv`（リネーム/移動）、`cp`（複製）が可能。
4. **過去コンテキストの直接執刀:** History 画面から過去のやり取りを `$EDITOR` で直接書き換え・調整し、DB に即時反映可能。
5. **完全なローカル再現性 & パイプライン連携:** TUI からの対話実行に加え、標準出力（stdout）への一発吐き出しオプションによる CLI ツールとしての決定論的動作を保証。

---

## 2. システムアーキテクチャ

DDD（ドメイン駆動設計）およびクリーンアーキテクチャのレイヤー分離を厳格に順守する。

```text
m4llama/
├── cmd/
│   └── m4llama/
│       └── main.go                 # サブコマンドルーティング & ホスト/ポート解決 & DI
├── pkg/
│   └── m4/                         # pure-Go POSIX m4 マクロエンジン
│       ├── engine.go               # 評価器、ダイバージョン、クォート管理
│       ├── builtins.go             # 組み込みマクロ (define, ifdef, ifelse, divert, dnl 等)
│       └── lexer.go                # トークナイザー & 多重クォート解析
├── internal/
│   ├── domain/                     # 純粋なドメインモデル
│   │   ├── entity/
│   │   │   ├── history.go          # 会話履歴 (AUTOINCREMENT int64 ID), トークン使用量
│   │   │   ├── completion.go       # 推論リクエスト/レスポンス, ロール定義
│   │   │   └── template_var.go     # テンプレート注釈変数定義
│   │   └── repository/
│   │       ├── macro_expander.go   # マクロ展開インターフェース
│   │       ├── llm_client.go       # llama-server 通信インターフェース
│   │       ├── tx_manager.go       # トランザクション抽象
│   │       └── history_repo.go     # 履歴永続化・更新インターフェース
│   ├── usecase/                    # アプリケーションロジック
│   │   ├── query_llm.go            # テンプレート展開 -> コンテキスト構築 -> 推論 -> 保存
│   │   └── history_query.go        # 履歴検索、編集更新、直近ログ取得
│   ├── infrastructure/             # 技術的詳細アダプター
│   │   ├── m4/
│   │   │   ├── expander.go         # repository.MacroExpander の pkg/m4 アダプター
│   │   │   └── parser.go           # 注釈ヘッダー (# @var) スキャナー
│   │   ├── llamaserver/
│   │   │   ├── client.go           # HTTP クライアント (/v1/chat/completions)
│   │   │   └── curl_emitter.go     # 再現用 curl コマンドジェネレーター
│   │   └── sqlite3/
│   │       ├── db.go               # DB 接続 & AUTOINCREMENT スキーマ DDL
│   │       ├── tx_manager.go       # sqlx トランザクション管理
│   │       └── history_repo.go     # SQLite3 リポジトリ実装 (Save, Update, FindByID)
│   └── ui/                         # 入出力プレゼンテーション層
│       ├── cli/                    # CLI フラグ解析、run/m4 サブコマンド
│       └── tui/
│           ├── cockpit.go          # 統合コックピット (History / Template / Form ステート管理)
│           ├── runner.go           # TUI 起動エントリーポイント
│           ├── spec_parser.go      # Context Sniping 構文解析 & 時系列昇順メッセージ生成
│           ├── tree.go             # ディレクトリツリー構築 & ファイル操作 (mv, cp)
│           ├── dynamic_form.go     # huh.Form を用いた動的変数入力
│           └── editor.go           # $EDITOR (Neovim) 連携 (一時ファイル編集 -> DB/FS反映)
├── templates/                      # プロンプトマクロテンプレート
│   └── base.m4
├── migrations/
│   └── 001_init.sql
├── Makefile
├── README.md
└── SPEC.md

```

---

## 3. インフラ & ネットワーク解決 (Portability)

ローカルポート競合や SSH トンネリング（踏み台経由）に柔軟に対応するため、エンドポイントの動的解決機構を備える。

### 優先順位



1. CLI フラグ: `--host` (`-H`), `--port` (`-p`)


2. 環境変数: `LLAMA_SERVER_URL`

3. デフォルト: `[http://127.0.0.1:8080](http://127.0.0.1:8080)`


```bash
# ポート競合時や別ポートの SSH トンネル経由
./bin/m4llama tui -p 28080
./bin/m4llama run -H 127.0.0.1 -p 18080 templates/review.m4

```

---

## 4. SQLite3 永続化設計 (AUTOINCREMENT)

履歴 ID はランダム文字列ではなく、画面上の表示インデックスおよび時系列と 1:1 で透過的に結びつく **`INTEGER PRIMARY KEY AUTOINCREMENT`** を採用する。

### スキーマ DDL (`history`)



```sql
CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_name TEXT NOT NULL,
    variables_json TEXT NOT NULL DEFAULT '{}',
    expanded_prompt TEXT NOT NULL,
    completion TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    pinned INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_pinned ON history(pinned);

```

* **ID の透過性:** DB 内の `id` がそのまま TUI 上の `[42]` と一致。


* **編集機能の担保:** 過去ログを `$EDITOR` で直接修正した際、`UPDATE history SET expanded_prompt = ?, completion = ? WHERE id = ?` により即座に内容を更新可能。

---

## 5. アプリケーションを構成する 3 つの画面ステート

アプリは状態機械として以下の 3 つの画面を遷移する。

```text
 ┌────────────────┐      Tab      ┌──────────────────┐
 │ 1. History 画面 │ <──────────> │ 2. Template 画面  │
 └───────┬────────┘               └────────┬─────────┘
         │ (Enter: 即時推論)                │ (r / Space: フォーム起動)
         │                                 ▼
         │                        ┌──────────────────┐
         └──────────────────────> │ 3. Form Input 画面│
                                  └────────┬─────────┘
                                           │ (Submit: 発射)
                                           ▼
                                 [ Inference Execution ]
                                 (常駐更新 or --stdout で終了)

```

### 画面 1: History 画面（コンテキストの執刀台）

* **左ペイン:** `[ID] プロンプト冒頭120文字（改行除去）`。


* **選択バッジ:** 現在の Chain に含まれる ID には `*`、ロール指定がある場合は `[S]`, `[U]`, `[A]` を付与。


* **右ペイン:** カーソル位置のプロンプトおよび回答の全文プレビュー。


* **キーアクション:**
* `e`: カーソル下の過去メッセージ（プロンプトおよびレスポンス）を **`$EDITOR` で直接開いて編集・修正**。保存終了後に DB レコードを更新し、プレビューをリロード。
* `Space`: カーソル行の ID を Context Chain にトグル追加 / 削除。


* `:` または `c`: 下部 Chain 入力欄へフォーカスし、手入力で `$` の挿入位置やロールを直接編集。


* `Enter`: 現在の Chain 設定のまま推論を即座に再実行。





### 画面 2: Template 画面（弾頭の管理・装填台 / Yazi スタイル）

* **左ペイン:** ワークスペース内の `.m4` ファイルとディレクトリを階層ツリー構造で表示。


* **右ペイン:** カーソル位置のテンプレート m4 ソースコードプレビュー。


* **キーアクション（Yazi風ファイル操作）:**
* `Enter`: カーソル下のテンプレートを **`$EDITOR`（Neovim）で起動・直接編集**。


* `m`: **Move / Rename (`mv`)**。プロンプトバーが開き、ファイルパスや名前を変更。
* `y` (または `c`): **Copy (`cp`)**。カーソル下のテンプレートを別名で複製。
* `r` または `Space`: テンプレートを決定し、**画面 3 (Form Input) を起動**。





### 画面 3: Form Input 画面（パラメータ入力と発射台）

* 選択されたテンプレートのヘッダー注釈（`# @var`）をパースし、型付き入力フォーム（`huh.Form`）をオーバーレイ展開。
* ユーザーが入力を完了して送信（Submit）すると：
1. `m4` マクロエンジンがテンプレートに入力変数をバインドして展開。
2. 展開されたプロンプトが、**History 画面で設定された Context Chain の `$` の位置に自動バインド**される。
3. LLM へ推論リクエストを送信。
4. 推論完了後：
* 通常モード: History の先頭に新レコードとして追加され、プレビューを更新して常駐。


* `--stdout` モード: 完了と同時にレスポンスを stdout へ出力してプロセス終了。





---

## 6. Context Sniping（コンテキスト外科手術）仕様

### 構文ルール



* 区切り文字: カンマ（`,`）


* ロール指定プレフィックス:


* `s...`: `system` ロールとして結合


* `u...`: `user` ロールとして結合


* `a...`: `assistant` ロールとして結合


* なし: 当該履歴の元ロールを展開




* トークン種別:


* **`$`**: 今回生成する新規プロンプト（テンプレート展開結果）。どこにでも埋め込み可能（例: `33~40,$,41~42`）。


* **単一ID**: `42`, `s33`, `u40`

* **範囲指定**: `33~42`, `s10~15`




### 時系列の自動昇順整列 (Chronological Reordering)

画面リストは直近が見やすい降順（DESC）で表示されるが、Chain から LLM のリクエスト配列（`[]entity.Message`）を構築する際は、**抽出された過去ログを必ず `id` 昇順（古い順）に自動整列**する。

### デフォルト Chain バインド

最新履歴が ID `42` の場合、起動時および推論完了直後には直近 10 件のシーケンスが Chain に自動セットされる。

```text
Chain: 33~42,$

```

---

## 7. ワンショット実行（--stdout オプション）仕様

CLI フラグ `--stdout`（または `-s`）を指定して起動した場合、TUI でテンプレートやフォーム入力を経由して推論を実行した直後、**結果を標準出力に出力して TUI を終了**する。

```bash
# 起動してフォーム入力・推論完了後、結果を stdout に吐いて終了
m4llama tui --stdout

# パイプで後続のツール（glow, pbcopy, jq など）に渡す
m4llama tui --stdout | glow -
m4llama tui --stdout | pbcopy

```

---

## 8. キーバインド仕様一覧

| キー | 対象画面 | 機能概要 |
| --- | --- | --- |
| `Tab` | 共通 | `History` ⇄ `Template` 画面の切り替え

 |
| `j` / `k` (`↓` / `↑`) | 共通 | リスト / ツリーのカーソル移動（右ペインプレビュー追従）

 |
| `Ctrl+d` / `Ctrl+u` | 共通 | 右ペイン（プレビュー Viewport）の半画面スクロール

 |
| `/` | 共通 | インクリメンタル grep / フィルタ入力

 |
| `e` | **History** | **カーソル下の過去メッセージを `$EDITOR` で直接編集・DB更新** |
| `Space` | **History** | カーソル行の ID を Chain にトグル追加・除外（`*` 反転）

 |
| `:` または `c` | **History** | 下部 Chain 入力欄へフォーカスし直接構文編集

 |
| `Enter` | **History** | 現在の Chain を使って即時推論実行

 |
| `Enter` | **Template** | **カーソル下のテンプレートを `$EDITOR` (Neovim) で即時編集**<br> |
| `m` | **Template** | **Yazi風 Move / Rename (`mv`)** |
| `y` (または `c`) | **Template** | **Yazi風 Copy / 複製 (`cp`)** |
| `r` / `Space` | **Template** | **Form Input 画面へ遷移（パラメータ入力へ）**<br> |
| `q` / `Ctrl+C` | 共通 | 終了

 |

---

## 9. CLI インターフェース仕様

```bash
# 1. POSIX m4 スタンドアロン実行
m4llama m4 -DFOO=bar template.m4
cat prompt.m4 | m4llama m4 -DENV=production

# 2. ヘッドレス推論実行 (完全自動化)
m4llama run -H 127.0.0.1 -p 8080 -DTASK="Refactor" templates/code.m4

# 3. Context Sniping を用いたヘッドレス実行
m4llama run -C "s33,40~42,$" -DTASK="Bugfix" templates/code.m4

# 4. curl コマンドのエクスポート（ドライラン）
m4llama run --curl templates/code.m4

# 5. Yazi/MC スタイル統合コックピット起動
m4llama tui -p 28080
m4llama tui --stdout       # 1回リクエストを投げるとレスポンスを stdout に出力して終了
m4llama                    # デフォルト起動

```
