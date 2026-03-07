#!/bin/bash
set -e

# ====== 配置（请修改为你自己的服务器信息）======
SERVER_IP="your-server-ip"
SERVER_USER="root"
SERVER_PASS="your-server-password"
DOMAIN="your-domain.com"
REMOTE_DIR="/opt/idea-arena"
SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"

# ====== 使用说明 ======
# 1. 复制此文件为 deploy.sh:  cp deploy.example.sh deploy.sh
# 2. 修改上方的服务器配置
# 3. 确保本地安装了 sshpass:  brew install hudochenkov/sshpass/sshpass (macOS)
# 4. 运行部署:  bash deploy.sh
#
# 注意: deploy.sh 已被 .gitignore 排除，不会被提交到 Git

echo ""
echo "=========================================="
echo "  Idea Arena 一键部署脚本"
echo "  目标: ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}"
echo "=========================================="
echo ""

# ====== 1. 本地打包 ======
echo ">>> [1/5] 打包项目..."
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARBALL="/tmp/idea-arena-deploy.tar.gz"

tar -czf "$TARBALL" \
    --exclude='node_modules' \
    --exclude='.next' \
    --exclude='.git' \
    --exclude='backend/data' \
    --exclude='backend/server' \
    --exclude='backend/configs/config.yaml' \
    --exclude='screenshots' \
    --exclude='ref_*.png' \
    -C "$SCRIPT_DIR" \
    backend frontend docker-compose.yaml deployments

echo "    打包完成: $(du -h $TARBALL | cut -f1)"

# ====== 2. 上传到服务器 ======
echo ""
echo ">>> [2/5] 上传到服务器..."
sshpass -p "$SERVER_PASS" ssh $SSH_OPTS ${SERVER_USER}@${SERVER_IP} "mkdir -p ${REMOTE_DIR}"
sshpass -p "$SERVER_PASS" scp $SSH_OPTS "$TARBALL" ${SERVER_USER}@${SERVER_IP}:${REMOTE_DIR}/deploy.tar.gz
echo "    上传完成"

# ====== 3. 远程部署 ======
echo ""
echo ">>> [3/5] 远程部署..."
sshpass -p "$SERVER_PASS" ssh $SSH_OPTS ${SERVER_USER}@${SERVER_IP} bash -s << 'DEPLOY_SCRIPT'
set -e
REMOTE_DIR="/opt/idea-arena"
DOMAIN="your-domain.com"
BT_VHOST_DIR="/www/server/panel/vhost/nginx"
BT_NGINX_BIN="/www/server/nginx/sbin/nginx"

cd ${REMOTE_DIR}

# 解压（保留已有数据目录和配置）
echo "    解压文件..."
tar -xzf deploy.tar.gz
rm -f deploy.tar.gz

# 创建数据和日志目录
mkdir -p data/sqlite data/logs/backend

# ====== 配置文件管理（不覆盖已有配置）======
echo "    检查配置文件..."
CONFIG_FILE="${REMOTE_DIR}/data/config.yaml"
OLD_CONFIG="${REMOTE_DIR}/backend/configs/config.yaml"
EXAMPLE_CONFIG="${REMOTE_DIR}/backend/configs/config.example.yaml"

if [ -f "$CONFIG_FILE" ]; then
    echo "    ✅ 使用已有配置: ${CONFIG_FILE}"
elif [ -f "$OLD_CONFIG" ]; then
    echo "    📦 迁移旧配置到新路径..."
    cp "$OLD_CONFIG" "$CONFIG_FILE"
    echo "    ✅ 已迁移: ${OLD_CONFIG} → ${CONFIG_FILE}"
elif [ -f "$EXAMPLE_CONFIG" ]; then
    echo "    📝 首次部署，从模板创建配置..."
    cp "$EXAMPLE_CONFIG" "$CONFIG_FILE"
    echo "    ⚠️  请编辑 ${CONFIG_FILE} 填入正确的 API Key 和 Webhook 地址"
else
    echo "    ❌ 未找到配置文件和模板，请手动创建 ${CONFIG_FILE}"
    exit 1
fi

# ====== 配置 Nginx（宝塔面板方式 + HTTPS）======
echo "    配置 Nginx..."
SSL_CERT="/etc/nginx/ssl/${DOMAIN}/fullchain.pem"
SSL_KEY="/etc/nginx/ssl/${DOMAIN}/privkey.pem"

if [ -f "$SSL_CERT" ] && [ -f "$SSL_KEY" ]; then
cat > ${BT_VHOST_DIR}/${DOMAIN}.conf << NGINX_VHOST
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};
    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl;
    http2 on;
    server_name ${DOMAIN} www.${DOMAIN};

    ssl_certificate ${SSL_CERT};
    ssl_certificate_key ${SSL_KEY};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;

    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }

    location /healthz {
        proxy_pass http://127.0.0.1:8000;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
NGINX_VHOST
echo "    ✅ Nginx HTTPS 配置完成"
else
echo "    ⚠️  未找到 SSL 证书，使用 HTTP 模式"
cat > ${BT_VHOST_DIR}/${DOMAIN}.conf << NGINX_VHOST
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};
    access_log /www/wwwlogs/${DOMAIN}.log;
    error_log /www/wwwlogs/${DOMAIN}.error.log;

    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }

    location /healthz {
        proxy_pass http://127.0.0.1:8000;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
NGINX_VHOST
echo "    ✅ Nginx HTTP 配置完成"
fi

${BT_NGINX_BIN} -t && ${BT_NGINX_BIN} -s reload
echo "    ✅ Nginx reload 完成"

# ====== 构建并启动 Docker 容器 ======
echo "    构建 Docker 镜像（首次较慢，请耐心等待）..."
cd ${REMOTE_DIR}

# 确保 swap 足够（低内存服务器）
if [ $(free -m | awk '/Swap/{print $2}') -lt 2048 ]; then
    echo "    增加 swap 空间到 2G..."
    if [ ! -f /swapfile_idea ]; then
        dd if=/dev/zero of=/swapfile_idea bs=1M count=2048
        chmod 600 /swapfile_idea
        mkswap /swapfile_idea
    fi
    swapon /swapfile_idea 2>/dev/null || true
fi

docker compose down --remove-orphans 2>/dev/null || true

# 逐个构建避免内存不足
echo "    [构建] 后端镜像..."
docker compose build backend
echo "    [构建] 前端镜像..."
docker compose build frontend

echo "    启动容器..."
docker compose up -d

echo "    等待服务启动..."
sleep 8

# 检查服务状态
docker compose ps
echo ""

# 检查后端健康
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8000/healthz 2>/dev/null || echo "000")
echo "    后端健康检查: HTTP ${HTTP_CODE}"

# 检查前端健康
HTTP_CODE_FE=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000 2>/dev/null || echo "000")
echo "    前端健康检查: HTTP ${HTTP_CODE_FE}"

echo ""
echo "=========================================="
echo "  ✅ 部署完成！"
echo "  HTTP:  http://${DOMAIN}"
echo ""
echo "  💡 SSL 证书可通过宝塔面板申请:"
echo "     宝塔面板 → 网站 → 添加站点 → SSL"
echo "     或手动: certbot --nginx -d ${DOMAIN}"
echo "=========================================="
DEPLOY_SCRIPT

# ====== 4. 本地验证 ======
echo ""
echo ">>> [4/5] 验证服务..."
sleep 3
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' http://${DOMAIN}/healthz 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    echo "    ✅ 服务正常 (HTTP $HTTP_CODE)"
else
    echo "    ⚠️  外网访问状态: HTTP $HTTP_CODE"
    echo "    提示: 请确认域名 ${DOMAIN} 已解析到 ${SERVER_IP}"
    echo "    提示: 请确认安全组已开放 80/443 端口"
fi

echo ""
echo ">>> [5/5] 部署完成！"
echo ""
echo ">>> 常用运维命令（SSH 到服务器后执行）:"
echo "    cd ${REMOTE_DIR}"
echo "    docker compose ps              # 查看状态"
echo "    docker compose logs -f backend # 查看后端日志"
echo "    docker compose logs -f frontend# 查看前端日志"
echo "    docker compose restart         # 重启服务"
echo "    docker compose down && docker compose up -d --build  # 重新构建"
echo ""
echo ">>> 数据和日志位置（服务器上）:"
echo "    SQLite 数据: ${REMOTE_DIR}/data/sqlite/"
echo "    后端日志:     ${REMOTE_DIR}/data/logs/backend/"
echo "    Nginx 日志:   /www/wwwlogs/${DOMAIN}.log"

# 清理临时文件
rm -f "$TARBALL"
