# Codex 沙箱基础镜像：只负责系统工具、文档解析、字体和 Python 环境。
# 可运行的 Codex 镜像请使用同目录的 Dockerfile.codex 构建。
# 使用基于 Debian 的 Node.js 官方轻量镜像
FROM node:26.7-bookworm-slim

# 设置环境变量 ENV
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 1. 安装 Python、文档解析、格式转换和基础工具链
RUN sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list.d/debian.sources

# 大体积依赖下载时多给几次重试，避免字体和 pandoc 因网络抖动中断。
RUN printf '%s\n' \
    'Acquire::Retries "10";' \
    'Acquire::http::Timeout "300";' \
    'Acquire::https::Timeout "300";' \
    > /etc/apt/apt.conf.d/99-network

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
    poppler-utils \
    ghostscript \
    antiword \
    catdoc \
    libreoffice-writer \
    libreoffice-calc \
    libreoffice-impress \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-chi-sim \
    tesseract-ocr-chi-tra \
    && rm -rf /var/lib/apt/lists/*

# 字体和 pandoc 体积较大，单独一层安装；apt 会按目标架构自动选择 deb 包。
RUN apt-get update && apt-get install -y --no-install-recommends \
        fonts-noto-cjk \
        pandoc \
    && fc-cache -f \
    && rm -rf /var/lib/apt/lists/*

# 2. 预装文档分析依赖。系统 Python 受 PEP 668 管理，统一放进独立 venv。
# NumPy 2.4+ 的 x86_64 wheel 要求 x86-64-v2，老 CPU 会在导入时直接报错；固定在 2.3.5。
# PyMuPDF 2.x 会引入 onnxruntime，1.x 足够覆盖 PDF 解析和渲染，兼容性也更好。
RUN python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install --no-cache-dir --upgrade pip \
    && /opt/venv/bin/pip install --no-cache-dir -i https://pypi.tuna.tsinghua.edu.cn/simple \
        openpyxl xlrd xlsxwriter \
        python-docx python-pptx odfpy \
        pypdf PyPDF2 pdfplumber PyMuPDF==1.28.2 \
        pandas numpy==2.3.5 matplotlib seaborn pillow \
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
