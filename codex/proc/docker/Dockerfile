# 使用基于 Debian 的 Node.js 官方轻量镜像
FROM node:26.7-bookworm-slim

# 设置环境变量 ENV
ENV TZ=Asia/Shanghai

# 1. 安装 Python3、Git 以及基础构建/工具链
RUN apt-get update && apt-get install -y --no-install-recommends \
    bubblewrap \
    python3 \
    python3-pip \
    python3-venv \
    git \
    curl \
    ca-certificates \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN npm config set registry https://registry.npmmirror.com

# 4. 如果选择将 Codex 打包进镜像 (如果不打包，可以在运行容器时通过 Volume 挂载)
RUN npm install -g @openai/codex

# 2. 设置全局环境变量
ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    CODEX_HOME=/root/.codex

# 3. 创建必要的目录（用于挂载 Socket 及 Workspace）
RUN mkdir -p /var/run/codex /workspace/.codex /root/.codex

# 4. 将 .codex 声明为 Volume（可选，提示 Docker 这是一个数据卷）
VOLUME ["/root/.codex", "/workspace"]

WORKDIR /workspace

# 默认指令 (实际会被 Go 网关启动容器时的 Cmd 覆盖)
CMD ["codex", "app-server","--listen","ws://0.0.0.0:4500"]