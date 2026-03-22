# Markdown CSS Test Cases

このファイルは、ブログの Markdown 表示用 CSS をまとめて確認するためのサンプルです。

---

## Paragraphs

通常の段落です。これは **太字**、*イタリック*、***太字イタリック***、~~打ち消し~~、`inline code`、[リンク](https://example.com) を含みます。

改行オプションが有効なので、この行のあとに  
明示的な改行があります。

空行を入れた次の段落です。

## Headings

# H1 Heading
## H2 Heading
### H3 Heading
#### H4 Heading
##### H5 Heading
###### H6 Heading

## Lists

### Unordered List

- Item A
- Item B
- Item C

### Nested Unordered List

- Parent 1
  - Child 1
  - Child 2
    - Grandchild 1
- Parent 2

### Ordered List

1. First
2. Second
3. Third

### Mixed List

1. Ordered parent
   - Unordered child
   - Another child
2. Ordered parent 2

### Task List

- [ ] Open item
- [x] Done item
- [ ] Another open item

## Blockquotes

> これは通常の引用です。
>
> 複数段落の引用も確認します。

> ### Quote With Heading
>
> - 引用内のリスト
> - 2つ目
>
> `quoted inline code`

## GitHub Alerts

> [!NOTE]
> NOTE の表示確認です。

> [!TIP]
> TIP の表示確認です。

> [!IMPORTANT]
> IMPORTANT の表示確認です。

> [!WARNING]
> WARNING の表示確認です。

> [!CAUTION]
> CAUTION の表示確認です。

## Code

### Fenced Code Block

```ts
type User = {
  id: string;
  name: string;
};

const users: User[] = [
  { id: "1", name: "Alice" },
  { id: "2", name: "Bob" },
];

console.log(users.map((user) => user.name).join(", "));
```

### Bash Code Block

```bash
curl -I https://example.com
rg --files
docker compose up --build
```

### Plain Code Block

```
no language
just plain text
with multiple lines
```

## Tables

| Column A | Column B | Column C |
| --- | ---: | :---: |
| left | right | center |
| long text sample | 12345 | ok |
| `inline code` | **bold** | [link](https://example.com) |

## Horizontal Rule

---

## Images

![Sample wide image](https://images.unsplash.com/photo-1500530855697-b586d89ba3ee?auto=format&fit=crop&w=1200&q=80)

Inline image in paragraph:
![Small sample](https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=600&q=80)

## Definition-Like Content

Term 1  
説明文のような2行構成です。

Term 2  
2つ目の説明文です。

## Escaping

\*これはエスケープされたアスタリスクです\*  
\# これは見出しにならない文字列です。

## HTML Inside Markdown

<div>
  <strong>Raw HTML block</strong> の表示確認です。
</div>

<p><mark>HTML の mark 要素</mark> も混ぜておきます。</p>

## Long Content

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer posuere, mauris vel varius commodo, purus velit tempus libero, nec pharetra felis libero at sapien. Morbi suscipit, velit eu consequat aliquet, augue lorem malesuada mauris, sit amet molestie felis velit ac odio.

Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.

## Edge Cases

- [Link only](https://example.com)
- `code only`
- **bold only**
- *italic only*
- ~~deleted only~~

> [!NOTE]
> 最後のアラート。CSS の余白と最後の子要素の挙動確認用です。
