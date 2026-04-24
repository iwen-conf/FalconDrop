# 05 媒体资产与照片处理计划

## 依据文档

- `docs/rewrite-analysis/01-feature-map.md`
- `docs/rewrite-analysis/02-business-flows.md`
- `docs/rewrite-analysis/03-data-model.md`
- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

上传完成后将所有文件作为媒体资产入库；可识别为照片的资产进入照片 API，支持 EXIF 时间、兜底时间、预览读取和直接删除。

## 模块边界

- media service 是上传后处理的唯一入口，FTP、HTTP 或未来导入流程都应通过同一入口入库。
- `media_assets` 记录所有上传资产；照片 API 只返回 `is_photo=true`。
- `storage_path` 不出现在 API response 中；浏览器只通过媒体 id 访问内容。
- WebSocket 事件在数据库事务提交后发布，避免前端收到查不到的数据。

## API

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/api/photos` | 查询照片列表，按 EXIF 时间分组或分页 |
| `GET` | `/api/photos/{id}` | 查询照片详情 |
| `GET` | `/api/photos/{id}/content` | 受控返回原图内容 |
| `DELETE` | `/api/photos/{id}` | 删除本地文件和数据库记录 |
| `GET` | `/api/assets` | 查询全部上传资产 |
| `GET` | `/api/assets/{id}` | 查询资产详情 |

## 查询参数草案

### `GET /api/photos`

| 参数 | 说明 |
|---|---|
| `cursor` | 可选，分页游标 |
| `limit` | 可选，默认 50，最大值实现时固定 |
| `groupBy` | 可选，首期支持 `day` |
| `from` / `to` | 可选，按有效拍摄时间过滤 |

响应应包含：

```ts
type PhotosResponse = {
  items: PhotoItem[];
  nextCursor?: string;
  groups?: PhotoGroup[];
};
```

### `GET /api/assets`

| 参数 | 说明 |
|---|---|
| `cursor` | 可选，分页游标 |
| `limit` | 可选 |
| `kind` | 可选，`all`、`photo`、`non_photo` |

## 处理流程

1. FTP storage hook 收到上传完成通知。
2. 计算文件内容 hash。
3. 检测 MIME 和照片类型。
4. 如果是照片，解析 EXIF 拍摄时间。
5. 生成或确认内部存储路径。
6. 按同名和 hash 规则写入 `media_assets`。
7. 写入 `transfer_events`。
8. 发布 WebSocket 事件。

## 入库决策表

| 场景 | 文件处理 | 数据处理 | 事件 |
|---|---|---|---|
| 新文件名 + 新 hash | 移动到新内部路径 | 插入 `media_assets` | `asset-uploaded`；如照片再发 `photo-added` |
| 同名 + 同 hash | 可覆盖同一内部路径或丢弃重复临时文件 | 更新原记录 `last_seen_at`、`uploaded_at` | `asset-overwritten`；如照片更新前端 |
| 同名 + 不同 hash | 生成 hash 后缀路径 | 插入新记录 | `asset-uploaded`；如照片再发 `photo-added` |
| 非照片文件 | 正常保存 | `is_photo=false` | `asset-uploaded` |
| 处理失败 | 清理临时文件 | 写 `transfer_events.failed` | 可选 `system-status` |

## 照片识别规则

首期建议基于 MIME、扩展名和解码能力组合判断：

- `image/jpeg`
- `image/png`
- `image/heic`、`image/heif` 如果 Go 库或外部能力支持
- 其他格式可以先入 `media_assets`，但 `is_photo=false`

如果 EXIF 解析能力尚不覆盖 RAW，不应在 UI 宣称 RAW 照片支持。

## MIME 与扩展名策略

- 优先使用内容嗅探结果。
- 扩展名只作为辅助，不能单独决定 `is_photo=true`。
- `application/octet-stream` 或未知 MIME 仍可作为普通资产入库。
- HEIC/HEIF 是否进入 `is_photo=true` 取决于 Go 端实际解码或 EXIF 能力；能力不足时先作为普通资产或仅展示有限元数据，不能让 content API 崩溃。

## EXIF 时间规则

- 优先读取 `DateTimeOriginal`。
- 解析失败或缺失时设置 `exif_taken_at=null`。
- 必须设置 `fallback_taken_at`，建议使用上传完成时间或文件修改时间。
- 照片列表排序优先 `exif_taken_at`，为空时使用 `fallback_taken_at`。
- EXIF 时间统一转换为服务端可比较的 `timestamptz`；原始字符串可不入库。
- 如果照片缺少时区信息，首期按服务端本地时区或 UTC 的选择必须固定并写入实现说明。

## content API 规则

- `GET /api/photos/{id}/content` 必须鉴权。
- 根据 DB 中的 `storage_path` 找文件，不接受前端传路径。
- 文件不存在时返回 `MEDIA_FILE_MISSING`，并记录错误日志。
- 设置合适的 `Content-Type`、`Content-Length`、`ETag` 或 `Cache-Control`。
- 支持 `Range` 是 P1；P0 可先返回完整文件。

## 删除规则

直接删除数据库记录和本地文件：

1. 校验照片存在。
2. 删除本地文件。
3. 删除数据库记录或标记后物理删除。
4. 写入 `transfer_events` 的 `deleted` 事件。
5. 发布 `photo-deleted` WebSocket 事件。

需要明确失败策略：

- 文件删除失败时不删除数据库记录，并返回错误。
- 数据库删除失败时保留文件。
- 删除过程中要避免出现“数据库已删但文件仍在”的静默不一致。

## 删除一致性策略

推荐顺序：

1. 开启事务并锁定媒体记录。
2. 校验 `is_photo=true`，照片 API 不删除非照片资产。
3. 删除本地文件；如果文件已不存在，必须明确是返回 `MEDIA_FILE_MISSING` 还是按幂等删除继续处理。
4. 文件删除成功后，在事务中写 `transfer_events.deleted` 并删除 DB 记录。
5. 提交成功后发布 `photo-deleted`。

如果实现选择“DB 先标记 deleting，再删文件，再物理删除 DB”，也可以，但必须有失败恢复策略。

## WebSocket payload 草案

```ts
type AssetEvent = {
  id: string;
  originalFilename: string;
  contentHash: string;
  size: number;
  mimeType: string;
  isPhoto: boolean;
  uploadedAt: string;
};

type PhotoItem = AssetEvent & {
  contentUrl: string;
  exifTakenAt: string | null;
  fallbackTakenAt: string;
};
```

事件中 `contentUrl` 应该是 API URL，不是本地路径。

## 实施步骤

1. 实现 hash 计算。
2. 实现 MIME 和照片识别。
3. 实现 EXIF 时间解析。
4. 实现 `media.Service.HandleUploadedFile`。
5. 实现同名同 hash 覆盖逻辑。
6. 实现同名不同 hash 内部路径冲突解决。
7. 实现照片列表、详情、content 和删除 API。
8. 实现全部资产 API。
9. 编写 hash、同名规则、EXIF 兜底、删除一致性测试。
10. 编写 content API 鉴权和路径不可枚举测试。
11. 编写非照片文件入库但不出现在照片列表的测试。

## WebSocket 事件

- `asset-uploaded`
- `asset-overwritten`
- `photo-added`
- `photo-deleted`

## 验收标准

- 任意格式上传后进入 `media_assets`。
- 非照片文件不出现在 `/api/photos`。
- 照片文件可通过 `/api/photos` 查询。
- `/api/photos/{id}/content` 不暴露服务器真实路径。
- 同名同 hash 按覆盖处理。
- 同名不同 hash 保留两份记录和两份文件。
- 删除照片会删除本地文件和数据库记录。
- 非照片文件可以在 `/api/assets` 查询，但不在 `/api/photos` 查询。
- 缺失 EXIF 的照片使用 `fallback_taken_at` 排序，并能被前端识别为兜底时间。

## 不做项

- 不做 AI 自动修图。
- 不做外部分享链接。
- 不做对象存储。
- 不做 Android MediaStore。
- 不做 RAW 支持承诺，除非实际 EXIF/解码能力验证通过。
