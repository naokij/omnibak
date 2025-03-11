# 如何发布新版本

本项目使用GitHub Actions自动构建和发布二进制文件。当您创建一个新的版本标签时，GitHub Actions会自动构建多个平台的二进制文件并发布到GitHub Releases。

## 发布步骤

1. 确保所有更改已经提交并推送到GitHub：
   ```bash
   git add .
   git commit -m "准备发布 vX.Y.Z"
   git push origin main
   ```

2. 为新版本创建一个标签：
   ```bash
   git tag vX.Y.Z
   ```
   注意：标签必须以字母`v`开头，后跟版本号（例如：`v0.1.0`、`v1.2.3`）。

3. 推送标签到GitHub：
   ```bash
   git push origin vX.Y.Z
   ```

4. GitHub Actions将自动运行构建和发布流程。您可以在GitHub仓库的Actions标签页查看进度。

5. 构建完成后，新版本将自动发布在GitHub Releases页面：
   https://github.com/naokij/omnibak/releases

## 发布内容

每个发布版本包含：
- Linux (amd64和arm64)二进制文件
- macOS (Intel和Apple Silicon)二进制文件
- 所有文件的SHA256校验和
- 自动生成的发布说明（包含自上一版本以来的提交记录）

## 自定义发布说明

如果您想提供自定义的发布说明，可以在推送标签前在GitHub上手动创建Release：

1. 访问GitHub仓库的Releases页面
2. 点击"Draft a new release"
3. 输入之前创建的标签名称
4. 填写发布标题和描述
5. 勾选"This is a pre-release"选项（如果适用）
6. 点击"Save draft"保存草稿

当您推送标签时，GitHub Actions会使用这个草稿发布，并添加二进制文件。 