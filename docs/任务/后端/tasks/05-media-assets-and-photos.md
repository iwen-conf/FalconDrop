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

## API

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/api/photos` | 查询照片列表，按 EXIF 时间分组或分页 |
| `GET` | `/api/photos/{id}` | 查询照片详情 |
| `GET` | `/api/photos/{id}/content` | 受控返回原图内容 |
| `DELETE` | `/api/photos/{id}` | 删除本地文件和数据库记录 |
| `GET` | `/api/assets` | 查询全部上传资产 |
| `GET` | `/api/assets/{id}` | 查询资产详情 |

## 处理流程

1. FTP storage hook 收到上传完成通知。
2. 计算文件内容 hash。
3. 检测 MIME 和照片类型。
4. 如果是照片，解析 EXIF 拍摄时间。
5. 生成或确认内部存储路径。
6. 按同名和 hash 规则写入 `media_assets`。
7. 写入 `transfer_events`。
8. 发布 WebSocket 事件。

## 照片识别规则

首期建议基于 MIME、扩展名和解码能力组合判断：

- `image/jpeg`
- `image/png`
- `image/heic`、`image/heif` 如果 Go 库或外部能力支持
- 其他格式可以先入 `media_assets`，但 `is_photo=false`

如果 EXIF 解析能力尚不覆盖 RAW，不应在 UI 宣称 RAW 照片支持。

## EXIF 时间规则

- 优先读取 `DateTimeOriginal`。
- 解析失败或缺失时设置 `exif_taken_at=null`。
- 必须设置 `fallback_taken_at`，建议使用上传完成时间或文件修改时间。
- 照片列表排序优先 `exif_taken_at`，为空时使用 `fallback_taken_at`。

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

## 不做项

- 不做 AI 自动修图。
- 不做外部分享链接。
- 不做对象存储。
- 不做 Android MediaStore。
