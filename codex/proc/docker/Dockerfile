# 使用基于 Debian 的 Node.js 官方轻量镜像
FROM node:26.7-bookworm-slim

# 设置环境变量 ENV
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Noto CJK 字体包较大，构建时从镜像源下载不稳定，预先放入构建上下文再复制。
COPY assets/fonts-noto-cjk.deb /tmp/fonts-noto-cjk.deb
COPY assets/pandoc.deb /tmp/pandoc.deb

# 1. 安装 Python、文档解析、格式转换和基础工具链
RUN sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list.d/debian.sources

RUN apt-get update && apt-get install -y --no-install-recommends \
    bubblewrap \
    python3 \
    python3-pip \
    python3-venv \
    git \
    curl \
    ca-certificates \
    build-essential \
    jq \
    unzip \
    zip \
    file \
    ripgrep \
    sqlite3 \
    liblua5.3-0 \
    libyaml-0-2 \
    poppler-utils \
    ghostscript \
    pandoc-data \
    antiword \
    catdoc \
    libreoffice-writer \
    libreoffice-calc \
    libreoffice-impress \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-chi-sim \
    tesseract-ocr-chi-tra \
    && dpkg -i /tmp/fonts-noto-cjk.deb /tmp/pandoc.deb \
    && fc-cache -f \
    && rm -f /tmp/fonts-noto-cjk.deb /tmp/pandoc.deb \
    && rm -rf /var/lib/apt/lists/*

RUN npm config set registry https://registry.npmmirror.com

# 4. 如果选择将 Codex 打包进镜像 (如果不打包，可以在运行容器时通过 Volume 挂载)
RUN npm install -g @openai/codex

# 2. 预装文档分析依赖。系统 Python 受 PEP 668 管理，统一放进独立 venv。
RUN python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install --no-cache-dir --upgrade pip \
    && /opt/venv/bin/pip install --no-cache-dir -i https://pypi.tuna.tsinghua.edu.cn/simple \
        openpyxl xlrd xlsxwriter \
        python-docx python-pptx odfpy \
        pypdf PyPDF2 pdfplumber PyMuPDF \
        pandas numpy matplotlib seaborn pillow \
        pytesseract tabulate beautifulsoup4 lxml \
    && /opt/venv/bin/python -c "import openpyxl, docx, pptx, pypdf, pdfplumber, fitz, pandas, numpy, matplotlib, PIL"

# 3. 设置全局环境变量
ENV PATH="/opt/venv/bin:${PATH}" \
    VIRTUAL_ENV=/opt/venv \
    PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    CODEX_HOME=/root/.codex

# 4. 创建必要的目录（用于挂载 Socket 及 Workspace）
RUN mkdir -p /var/run/codex /workspace/.codex /root/.codex

# 5. 将 .codex 声明为 Volume（可选，提示 Docker 这是一个数据卷）
VOLUME ["/root/.codex", "/workspace"]

WORKDIR /workspace

# 默认指令 (实际会被 Go 网关启动容器时的 Cmd 覆盖)
CMD ["codex", "app-server","--listen","ws://0.0.0.0:4500"]
