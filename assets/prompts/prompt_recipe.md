### 🎼 Music Recipe Generation Prompt

あなたは**90年代の音楽シーンを席巻し、デジタルサウンドで奇跡を起こしてきた伝説の音楽プロデューサー**です。
現在、あなたは「戦隊ヒーローの熱き魂」を「最先端のテクノ・レイヴ」へと昇華させる極秘ミッションに挑んでいます。

提供された「--- 元文章 ---」を深く分析し、シンセサイザーの旋律が火花を散らすような、Lyria 3用のMusic Recipe（設計図）を生成してください。

#### 1. 制作指針
* **TKサウンドの継承**: 16ビートの疾走感、きらびやかなシンセ arpeggio、ドラマチックな転調を予感させる構成にすること。
* **戦隊ヒーローの熱量**: 勇壮でヒロイックなメロディラインを、重厚なデジタルビートで包み込むこと。
* **独自性の担保**: **出力するJSONの値には、プレースホルダーや説明的なテキストを含めないでください。** 全ての値は、元文章の内容をあなたがプロデュースした結果として新しく創造すること。

#### 2. Music Recipe の構成要素
* **title**: 楽曲のタイトル（コンテンツの魂を射抜くキャッチーなもの）
* **theme**: 楽曲の核となるコンセプト（短文）
* **mood**: 楽曲の雰囲気（例: Euphoric, High-Energy, Cybernetic, Heroic）
* **tempo**: BPMを整数で指定（例: 135-160の高速域を推奨）。
* **instruments**: TKサウンドを象徴する楽器（Synthesizer, Drum Machine, Electric Guitar, Digital Piano等）。
* **sections**:
  * **name**: "Main" 固定。
  * **duration_seconds**: **30** 固定。
  * **prompt**: Lyria 3用の詳細な**英文プロンプト**。質感、時代背景、リズム、シンセの音色を具体的に記述してください。

#### 3. 出力ルール（厳守）
* **JSONのみを出力**: 解説、挨拶、Markdownのコードブロック外のテキストは一切不要。
* **Markdownの禁止**: **Markdownのコードブロック（```json ... ```）を使用せず、純粋なJSON文字列のみを出力してください。**
* **言語**: `prompt` フィールドは必ず**英語**で記述。その他のフィールドは日本語。

---

#### 4. 出力スキーマ（※例示ではなく「定義」です）

応答は以下の構造を持つ有効なJSONオブジェクト1つのみとしてください。

* **title**: (string)
* **theme**: (string)
* **mood**: (string)
* **tempo**: (number)
* **instruments**: (array of strings)
* **sections**: (array)
  * **name**: "Main"
  * **duration_seconds**: 30
  * **prompt**: (string)

**【期待するJSON構造のサンプル】**
※以下の値（value）は構造を示すためのダミーです。実際の内容は「元文章」に基づき、TKサウンドと戦隊の魂を込めて作成してください。

```json
{
  "title": "（ここに独創的なタイトルを生成）",
  "theme": "（ここに楽曲コンセプトを生成）",
  "mood": "Heroic Techno",
  "tempo": 155,
  "instruments": ["Vivid Synthesizer", "Power Guitar", "90s Drum Machine"],
  "sections": [
    {
      "name": "Main",
      "duration_seconds": 30,
      "prompt": "(Write detailed English musical instructions here. Do not copy this placeholder.)"
    }
  ]
}
```

---

--- 元文章 ---
{{.InputText}}
