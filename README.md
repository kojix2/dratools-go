# dratools-go

[![CI](https://github.com/kojix2/dratools-go/actions/workflows/ci.yml/badge.svg)](https://github.com/kojix2/dratools-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`dratools-go` は、DDBJ Search API を使って DRA/SRA 関連 accession から
ダウンロード URL や run accession を解決するコマンドラインツールです。

このリポジトリは Ruby 版 [`dratools`](dratools-ruby/) を参考にした Go 移植版です。
Ruby 版は参照用として `dratools-ruby/` に同梱しています。

dratools は非公式のツールです。DDBJ や国立遺伝学研究所が提供する公式ツールではありません。

## インストール

Go 1.22 以上でビルドできます。

```sh
go build -o dratools ./cmd/dratools
```

## 使い方

`dratools` はサブコマンド方式です。

| コマンド | 役割 |
| --- | --- |
| `url` | ダウンロード URL を表示する |
| `get` | ファイルをダウンロードする |
| `probe` | ダウンロード URL の到達性を確認する |
| `tree` | accession から run へ辿る探索ツリーを表示する |
| `meta` | レコードのメタ情報を表示する |
| `runs` | run accession の一覧を出力する |
| `size` | ダウンロードサイズを集計する |

```sh
dratools url DRR000001
dratools url --json DRR000001 DRR000002
dratools runs PRJNA341783
dratools size --type fastq PRJNA341783
dratools get -O downloads DRR000001
```

複数の accession もまとめて渡せます。

```sh
dratools url DRR000001 DRR000002
dratools get --input list.txt -O downloads
printf 'DRR000001\nDRR000002\n' | dratools get -O downloads
```

## ダウンロード機能について

`get` は小規模な確認や手元での簡単な取得を助けるための補助的な機能です。
この Go 版では Ruby 版以上に、ダウンロード本体よりも URL 解決を主目的にしています。

大量のデータを取得する場合は、`dratools url` で URL の一覧を取得するところまでにとどめ、
実際のダウンロードは `curl`, `wget`, `aria2c`, NAS のダウンロード機能など、
用途に合った別のツールへ渡すことを強くおすすめします。

## Ruby 版との関係

Go 版は Ruby 版 `dratools` の主要コマンドと探索ロジックを引き継いでいます。
一方で、実ファイルのダウンロードは Ruby 版のように `curl`, `wget`, `aria2c` へ shell out せず、
Go ライブラリ [grab](https://github.com/cavaliergopher/grab) を使います。

`get` は `net/http` ベースで、レジューム、リトライ、md5 検証、対話端末での進捗バー表示に対応しています。
それ以外は標準ライブラリ中心の実装です。

Ruby 版から引き継いでいる主な環境変数:

- `DRATOOLS_MAX_RECURSIVE_NON_RUN_XREFS`
- `DRATOOLS_TREE_MAX_DIRECT_RUNS`
- `DRATOOLS_URL_MAX_DIRECT_RUNS`
- `DRATOOLS_SIZE_MAX_DIRECT_RUNS`

値に `unlimited` を指定すると、その制限を無効にできます。

## 開発

Ruby 版の参照実装まで取得する場合は、submodule も初期化してください。

```sh
git submodule update --init --recursive
```

```sh
go test ./...
go build -o dratools ./cmd/dratools
```

Makefile も用意しています。

```sh
make fmt
make test
make build
```

CI では Linux/macOS と Go 1.22/最新系列で、`gofmt`、テスト、ビルドを確認します。

## ライセンス

MIT License です。詳しくは [LICENSE](LICENSE) を参照してください。
