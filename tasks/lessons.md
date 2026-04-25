# Lessons

## 2026-04-25

- 路径越界判断不要用 `strings.HasPrefix(cleanFull, cleanRoot)` 作为唯一依据。
- 规范做法是 `filepath.Rel(root, target)` 后判断是否为 `..` 或 `../` 前缀，避免 `/data/root2` 被误判成 `/data/root` 子目录。
