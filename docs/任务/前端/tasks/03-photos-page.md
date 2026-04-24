# 03 照片页面计划

## 依据文档

- `docs/rewrite-analysis/01-feature-map.md`
- `docs/rewrite-analysis/02-business-flows.md`
- `docs/rewrite-analysis/03-data-model.md`
- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/05-page-map.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

展示 FTP 上传后被后端识别为照片的资产，支持按 EXIF 时间分类、预览和删除。非照片文件允许上传和入库，但不进入照片页。

## 页面定位

- `/photos` 是默认主工作页。
- 页面服务现场查看和确认相机 FTP 图传结果，优先保证新照片出现、时间分组清晰、预览和删除可靠。
- 不做原 Android 图库的长按选择、分享和系统删除确认。

## API 契约

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/api/photos` | 获取照片分页或分组列表 |
| `GET` | `/api/photos/{id}` | 获取照片详情 |
| `GET` | `/api/photos/{id}/content` | 受控读取原图内容 |
| `DELETE` | `/api/photos/{id}` | 删除照片数据库记录和本地文件 |
| `GET` | `/api/ws` | 接收照片新增、覆盖、删除事件 |

## 页面结构

- 顶部：页面标题、当前照片数量、连接状态提示。
- 左侧或上方：时间分组导航，按 EXIF 日期聚合。
- 主区域：照片网格，显示缩略图或原图降级预览、原文件名、EXIF 时间、上传时间。
- 预览层：大图、文件名、大小、hash、EXIF 时间、上传时间、删除入口。
- 空状态：提示“暂无照片，请通过 FTP 上传照片文件”。

## 数据模型

```ts
type PhotoItem = {
  id: string;
  originalFilename: string;
  contentUrl: string;
  size: number;
  contentHash: string;
  mimeType: string;
  exifTakenAt: string | null;
  fallbackTakenAt: string;
  uploadedAt: string;
  groupKey: string;
};
```

## 组件拆分

| 组件 | 职责 |
|---|---|
| `PhotosPage` | 页面级 query、事件订阅、布局 |
| `PhotoGroupNav` | 时间分组导航和数量 |
| `PhotoGrid` | 稳定网格、loading、empty、error |
| `PhotoCard` | 单张照片缩略展示、文件名、时间状态 |
| `PhotoPreviewDialog` | 大图预览、元数据、删除入口 |
| `DeletePhotoDialog` | 删除确认 |
| `PhotoConnectionBanner` | WebSocket 离线或重连提示 |

组件使用 shadcn/ui 的 Dialog、Button、Tooltip、Skeleton、Alert 等基础组件；图标使用 lucide-react。

## 分组规则

1. 优先使用 `exifTakenAt` 作为排序和分组时间。
2. 缺失 EXIF 时间时使用后端提供的 `fallbackTakenAt`。
3. UI 需要明确标注“无 EXIF 时间，按上传时间归类”或等价中文提示。
4. 同名不同 hash 照片必须能同时展示，不能以前端文件名去重。
5. 分组 key 建议使用本地展示日期 `YYYY-MM-DD`，但排序基于 ISO 时间。
6. 同一天内按有效时间倒序，时间相同再按 `uploadedAt` 倒序。

## 网格与预览规则

- 网格项尺寸稳定，图片加载完成前使用 skeleton，避免布局跳动。
- `contentUrl` 使用 `/api/photos/{id}/content`，不拼接服务器路径。
- 图片加载失败时显示错误占位和文件名。
- 预览层展示文件名、大小、MIME、hash、EXIF 时间、上传时间。
- hash 可显示前 12 位并提供复制按钮。
- 删除按钮使用危险样式，并要求二次确认。
- 大图预览不需要实现复杂缩放；如后续需要，作为 P1。

## WebSocket 增量规则

| 事件 | 前端行为 |
|---|---|
| `photo-added` | 将新照片插入对应时间分组，必要时刷新当前页 |
| `asset-overwritten` | 更新对应照片的 hash、上传时间和预览资源 |
| `photo-deleted` | 从列表和预览层移除对应照片 |
| `system-status` | 更新页面连接状态提示 |

页面加载或 WebSocket 重连后必须重新调用 `/api/photos` 获取快照，不能只依赖历史事件。

## TanStack Query 策略

- Query key：`['photos', filters]`。
- 页面首次进入拉取 REST 快照。
- `photo-added` 到达时可乐观插入当前缓存；如果分页边界复杂，直接 invalidate。
- `asset-overwritten` 到达时更新匹配 id 或 hash 的条目；找不到则 invalidate。
- `photo-deleted` 到达时移除 id，并关闭正在预览的同一照片。
- WebSocket 重连成功后统一 invalidate `['photos']`。

## 删除流程

1. 用户点击删除。
2. 显示中文确认文案，说明会同时删除本地文件和数据库记录。
3. 调用 `DELETE /api/photos/{id}`。
4. 成功后关闭预览层并从列表移除。
5. 失败时展示中文错误，并保留当前列表状态。

## 空状态与错误状态

| 状态 | UI |
|---|---|
| loading | Skeleton 网格 |
| empty | “暂无照片，请通过 FTP 上传照片文件” |
| offline | 顶部提示 WebSocket 已断开，页面仍显示最近一次 REST 快照 |
| 401 | 交给全局会话处理 |
| `MEDIA_NOT_FOUND` | 刷新照片列表，提示照片已不存在 |
| content 图片加载失败 | 单图错误占位，不影响整页 |

## 实施步骤

1. 建立 `features/photos/photosApi.ts`。
2. 建立 `features/photos/usePhotosQuery.ts`，封装分页、分组和刷新。
3. 建立 `features/photos/usePhotoEvents.ts`，处理 WebSocket 增量和重连补偿。
4. 实现照片网格和时间分组导航。
5. 实现照片预览层和删除确认。
6. 补齐 loading、empty、error、offline 状态。
7. 为分组、同名照片、删除失败、WebSocket 事件写单元测试。
8. 使用 Framer Motion 给预览层打开/关闭和删除移除做轻量动画。
9. 使用 React Testing Library 覆盖“同名不同 hash 不去重”和“无 EXIF 兜底提示”。

## 验收标准

- FTP 上传照片后，照片页通过 WebSocket 实时出现新照片。
- 刷新页面后照片仍从 PostgreSQL 快照恢复。
- 照片按 EXIF 时间分组展示。
- 无 EXIF 时间照片有明确兜底时间提示。
- 同名不同 hash 的照片不会互相覆盖 UI。
- 删除照片后列表移除，重新刷新也不再出现。
- WebSocket 断开时页面有状态提示，重连后会拉取 REST 快照。
- 图片 URL 不暴露服务器真实路径。

## 不做项

- 不展示非照片文件。
- 不做分享链接。
- 不做 AI 修图入口。
- 不做 Android MediaStore 选择模式。
- 不做 Windows 本地预览窗口。
