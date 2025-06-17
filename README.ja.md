# go-jpeg-meta-web-strip

[English README](README.md)

Web配信用にJPEG画像を最適化するGoライブラリです。プライバシーに関わる情報や不要なメタデータを削除し、Webでの表示に必要な情報のみを保持することで、安全で効率的な画像配信を実現します。

**🌐 最適な用途**: コンテンツ管理システム、画像CDN、Webアプリケーション、そしてWeb上で安全かつ効率的にJPEG画像を配信する必要があるあらゆるサービス

## なぜWeb画像にこのツールが必要か？

ウェブサイトでJPEG画像を配信する際、不要なメタデータが以下の問題を引き起こします：
- **ファイルサイズが最大46%増加**し、ページの読み込みが遅くなる
- **GPS座標、カメラのシリアル番号、編集履歴**などのプライバシー情報が露出する
- **埋め込まれたサムネイルや独自のカメラデータ**で帯域幅を浪費する
- **異なるブラウザやデバイス間で一貫性のない動作**を引き起こす

このツールは、Web表示に必要な情報を保持しながら、問題のあるメタデータを賢く削除することでこれらの問題を解決します。

## 機能

- **Web最適化出力**: Web配信用にJPEG画像を準備するよう特別に設計
- **プライバシー保護**: GPS位置情報、カメラ情報、その他の個人データを削除
- **選択的メタデータ削除**: ブラックリスト方式で不要なメタデータのみを削除
- **表示に重要なデータの保持**: Web上での画像表示に影響するオリエンテーション、ICCプロファイル、DPI設定を保持
- **画像の完全性**: 処理後もピクセルデータは変更されません
- **ファイルサイズ削減**: 埋め込まれたサムネイルや不要なメタデータを削除することで、最大46%のファイルサイズ削減を実現

### 削除されるメタデータ

- EXIF サムネイル
- GPS 情報
- カメラ情報（メーカー、モデル、レンズデータ）
- メーカー独自データ
- XMP メタデータ
- IPTC メタデータ
- Photoshop IRB データ
- コメント

### 保持されるメタデータ

- オリエンテーション（画像の向き）
- ICC カラープロファイル
- DPI/解像度設定
- カラースペース情報
- ガンマ値
- 画像レンダリングに必要なデータ

## インストール

```bash
go get github.com/ideamans/go-jpeg-meta-web-strip
```

## 使い方

```go
package main

import (
    "fmt"
    "os"
    jpegmetawebstrip "github.com/ideamans/go-jpeg-meta-web-strip"
)

func main() {
    // JPEGファイルを読み込む
    jpegData, err := os.ReadFile("input.jpg")
    if err != nil {
        panic(err)
    }

    // Web配信用に不要なメタデータを削除
    cleanedData, result, err := jpegmetawebstrip.Strip(jpegData)
    if err != nil {
        panic(err)
    }

    // Web最適化されたJPEGを書き出す
    err = os.WriteFile("output.jpg", cleanedData, 0644)
    if err != nil {
        panic(err)
    }

    // 結果を表示
    fmt.Printf("削除されたメタデータ:\n")
    fmt.Printf("  EXIFサムネイル: %d バイト\n", result.Removed.ExifThumbnail)
    fmt.Printf("  GPS: %d バイト\n", result.Removed.ExifGPS)
    fmt.Printf("  カメラ情報: %d バイト\n", result.Removed.CameraInfo)
    fmt.Printf("  XMP: %d バイト\n", result.Removed.XMP)
    fmt.Printf("  IPTC: %d バイト\n", result.Removed.IPTC)
    fmt.Printf("  コメント: %d バイト\n", result.Removed.Comments)
    fmt.Printf("合計削除: %d バイト\n", result.Total)
}
```

## テストデータジェネレータ

このパッケージには、Web最適化のシナリオをテストするために、様々なメタデータの組み合わせを持つJPEGファイルを生成するテストデータジェネレータが含まれています。

### 使い方

```bash
# テストデータを生成
make data

# または直接実行
go run datacreator/cmd/main.go
```

### 生成されるテスト画像

以下のテスト画像が `testdata` ディレクトリに生成されます：

| ファイル名                     | 説明                                 | メタデータ                                       |
| ------------------------------ | ------------------------------------ | ------------------------------------------------ |
| `basic_copy.jpg`               | オリジナルの基本コピー               | 最小限のメタデータ                               |
| `with_exif_thumbnail.jpg`      | EXIF サムネイル付き JPEG             | 160x120 のサムネイル埋め込み                     |
| `with_gps.jpg`                 | GPS 情報付き JPEG                    | GPS 座標                                         |
| `with_camera_info.jpg`         | カメラ情報付き JPEG                  | メーカー、モデルタグ                             |
| `with_xmp.jpg`                 | XMP メタデータ付き JPEG              | 作成者、作成日など                               |
| `with_iptc.jpg`                | IPTC メタデータ付き JPEG             | キャプション、キーワード、著作権                 |
| `with_photoshop_irb.jpg`       | Photoshop IRB 付き JPEG              | Photoshop 固有データ                             |
| `with_comment.jpg`             | コメント付き JPEG                    | テキストコメント                                 |
| `with_orientation.jpg`         | オリエンテーション付き JPEG          | 90° 回転（保持される）                           |
| `with_icc_profile_srgb.jpg`    | sRGB ICC プロファイル付き JPEG       | カラープロファイル（保持される）                 |
| `with_icc_profile_p3.jpg`      | Display P3 ICC プロファイル付き JPEG | カラープロファイル（保持される）                 |
| `with_dpi.jpg`                 | DPI 設定付き JPEG                    | 300 DPI（保持される）                            |
| `with_colorspace.jpg`          | 特定のカラースペース付き JPEG        | sRGB カラースペース（保持される）                |
| `with_gamma.jpg`               | ガンマ値付き JPEG                    | ガンマ 2.2（保持される）                         |
| `with_quality.jpg`             | 特定の品質設定付き JPEG              | 品質 95                                          |
| `with_all_removable.jpg`       | 全削除対象メタデータ付き JPEG        | 包括的テスト                                     |
| `with_mixed_metadata.jpg`      | 混合メタデータ付き JPEG              | 削除対象＋保持対象                               |
| `with_comprehensive_mixed.jpg` | 包括的混合メタデータ                 | サムネイル、GPS、カメラ、オリエンテーション、DPI |
| `with_thumbnail_and_icc.jpg`   | サムネイルと ICC 付き JPEG           | 選択的削除のテスト                               |

### テストデータ生成の要件

- ImageMagick（`magick`コマンド）
- ExifTool（`exiftool`コマンド）- オプションですが、包括的なメタデータのために推奨

## テスト

```bash
# 全テストを実行
go test ./...

# 詳細出力付きで実行
go test -v ./...

# 特定のテストを実行
go test -v -run TestStrip

# カバレッジレポートを生成
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## テストケース

パッケージには包括的なテストが含まれています：

1. **メタデータ削除テスト**: 特定のメタデータタイプが削除されることを検証
2. **メタデータ保持テスト**: 重要なメタデータが保持されることを確認
3. **無効データ処理**: 無効な入力に対するエラー処理をテスト
4. **画像完全性テスト**: MD5 チェックサムを使用してピクセルデータが変更されていないことを検証
5. **包括的テスト**: 混合メタデータシナリオ

## 要件

- Go 1.22 以上
- 依存関係は Go モジュールで管理

## ライセンス

MIT License

Copyright (c) 2024 IdeaMans Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
