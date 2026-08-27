# BENZHI_README

## 项目说明
- 项目：benzhi-project-6a7907ea-a6eb-4ce6-8eaf-87e9b933f3ab
- 项目用途：已完整实现水下考古饱水木材浸渍阶段转换资格服务，覆盖冻结基线、规则观测、偏差恢复、独立复核、SQLite 持久幂等、摘要链审计和规范证书封存校验。
- Go 工具链：`golang:1.22`
- 前端工具链：无

## 项目描述
- 项目名称：timber-stage-qualifier
- 项目介绍：面向水下考古保护实验室的饱水木材浸渍阶段转换资格服务，围绕一个处理批次完成基线冻结、过程观测、偏差处置、恢复验证、独立复核和不可变资格证书封存。
- 项目概述：面向水下考古保护实验室的饱水木材浸渍阶段转换资格服务，围绕一个处理批次完成基线冻结、过程观测、偏差处置、恢复验证、独立复核和不可变资格证书封存。
- 核心工作流：保护员建立饱水木材处理批次并冻结浸渍基线，按阶段提交浓度、温度与质量观测；规则判定异常时批次自动暂停，保护员登记纠正措施并完成连续恢复观测，随后由未参与操作的复核员独立裁定，合格批次生成可验证的阶段转换证书并进入只读封存状态，不合格批次以拒绝结论封存。
- 对外接口：仅提供版本化 JSON HTTP API，覆盖批次命令、状态查询、观测提交、偏差恢复、复核裁定、证书下载和审计校验；服务支持 -addr=127.0.0.1:<port>，也读取 PORT 并绑定 127.0.0.1:<PORT>，默认监听 127.0.0.1:19081，绝不默认绑定 0.0.0.0。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...

cd /app && GOTOOLCHAIN=local go run ./cmd/server -self-check -addr=127.0.0.1:19081

cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh

./build_benzhi_docker.sh benzhi-project-6a7907ea-a6eb-4ce6-8eaf-87e9b933f3ab-amd64 linux/amd64

./build_benzhi_docker.sh benzhi-project-6a7907ea-a6eb-4ce6-8eaf-87e9b933f3ab-arm64 linux/arm64

docker run -it benzhi-project-6a7907ea-a6eb-4ce6-8eaf-87e9b933f3ab-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -self-check -addr=127.0.0.1:19081`
