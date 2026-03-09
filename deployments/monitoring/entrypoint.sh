#!/bin/bash
set -e

echo "=== 云雾API花费监控容器已启动 ==="
echo "调度: 每小时05分查询上一小时花费"
echo "飞书webhook: ${FEISHU_WEBHOOK_URL:-(未配置)}"
echo ""

# 首次启动时执行一次 dry-run 验证
echo "=== 启动验证 ==="
python3 /app/yunwu_cost_monitor.py --dry-run --hours 1
echo ""
echo "=== 验证完成，进入定时循环 ==="

# 定时循环: 每小时05分执行
while true; do
    # 计算到下一个 XX:05 的秒数
    CURRENT_MIN=$(date +%M | sed 's/^0//')
    CURRENT_SEC=$(date +%S | sed 's/^0//')

    if [ "$CURRENT_MIN" -lt 5 ]; then
        WAIT_MIN=$((5 - CURRENT_MIN))
    else
        WAIT_MIN=$((65 - CURRENT_MIN))
    fi
    WAIT_SEC=$((WAIT_MIN * 60 - CURRENT_SEC))

    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 下次执行在 ${WAIT_SEC} 秒后"
    sleep "$WAIT_SEC"

    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 开始查询..."
    python3 /app/yunwu_cost_monitor.py --hours 1 2>&1 | tee -a /var/log/yunwu-cost-monitor.log
    echo ""
done
