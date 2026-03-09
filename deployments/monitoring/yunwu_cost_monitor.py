#!/usr/bin/env python3
"""
云雾API 花费监控脚本
每小时查询上一小时的API调用花费，汇总后发送到飞书群。

用法:
  python3 yunwu_cost_monitor.py                # 查询上一小时
  python3 yunwu_cost_monitor.py --hours 3      # 查询过去3小时
  python3 yunwu_cost_monitor.py --dry-run      # 仅打印不发送飞书
  python3 yunwu_cost_monitor.py --test-feishu  # 测试飞书webhook连通性

环境变量:
  YUNWU_USERNAME       云雾api登录用户名 (必需)
  YUNWU_PASSWORD       云雾api登录密码 (必需)
  YUNWU_USER_ID        云雾api用户ID (必需)
  YUNWU_BASE_URL       云雾api地址 (默认: https://yunwu.ai)
  FEISHU_WEBHOOK_URL   飞书群webhook URL (可选)
"""

import argparse
import json
import os
import sys
import time
from collections import defaultdict
from datetime import datetime, timedelta, timezone

try:
    import requests
except ImportError:
    print("请先安装 requests: pip3 install requests")
    sys.exit(1)

# ========== 配置（全部从环境变量读取）==========
YUNWU_BASE_URL = os.environ.get("YUNWU_BASE_URL", "https://yunwu.ai")
YUNWU_USERNAME = os.environ.get("YUNWU_USERNAME", "")
YUNWU_PASSWORD = os.environ.get("YUNWU_PASSWORD", "")
YUNWU_USER_ID = os.environ.get("YUNWU_USER_ID", "")
FEISHU_WEBHOOK_URL = os.environ.get("FEISHU_WEBHOOK_URL", "")

# 500000 quota = 1 USD (云雾api标准换算)
QUOTA_TO_USD = 500000

# 时区: 东八区
CST = timezone(timedelta(hours=8))


def check_env():
    """检查必需的环境变量"""
    missing = []
    if not YUNWU_USERNAME:
        missing.append("YUNWU_USERNAME")
    if not YUNWU_PASSWORD:
        missing.append("YUNWU_PASSWORD")
    if not YUNWU_USER_ID:
        missing.append("YUNWU_USER_ID")
    if missing:
        print(f"[ERROR] 缺少必需环境变量: {', '.join(missing)}")
        print("请在 .env 文件中配置或通过环境变量传入")
        sys.exit(1)


def login_yunwu():
    """登录云雾api，返回带session的requests.Session"""
    session = requests.Session()
    resp = session.post(
        f"{YUNWU_BASE_URL}/api/user/login",
        json={"username": YUNWU_USERNAME, "password": YUNWU_PASSWORD},
        timeout=15,
    )
    data = resp.json()
    if not data.get("success"):
        raise RuntimeError(f"登录失败: {data.get('message', resp.text)}")
    return session


def fetch_logs(session, start_ts, end_ts):
    """获取指定时间段的所有使用日志"""
    all_items = []
    page = 1
    page_size = 100

    while True:
        resp = session.get(
            f"{YUNWU_BASE_URL}/api/log/self",
            params={
                "p": page,
                "page_size": page_size,
                "type": 0,
                "start_timestamp": start_ts,
                "end_timestamp": end_ts,
            },
            headers={"new-api-user": YUNWU_USER_ID},
            timeout=15,
        )
        data = resp.json()
        if not data.get("success"):
            raise RuntimeError(f"获取日志失败: {data.get('message', resp.text)}")

        items = data.get("data", {}).get("items", [])
        total = data.get("data", {}).get("total", 0)
        all_items.extend(items)

        if len(all_items) >= total or len(items) == 0:
            break
        page += 1

    return all_items


def aggregate_costs(items):
    """按模型聚合花费数据"""
    models = defaultdict(lambda: {
        "calls": 0,
        "quota": 0,
        "prompt_tokens": 0,
        "completion_tokens": 0,
        "total_time": 0,
        "group": "",
        "errors": 0,
    })

    total_quota = 0
    total_calls = 0
    total_errors = 0

    for item in items:
        model = item.get("model_name", "unknown")
        quota = item.get("quota", 0)
        m = models[model]
        m["calls"] += 1
        m["quota"] += quota
        m["prompt_tokens"] += item.get("prompt_tokens", 0)
        m["completion_tokens"] += item.get("completion_tokens", 0)
        m["total_time"] += item.get("use_time", 0)
        if item.get("group"):
            m["group"] = item["group"]

        if quota == 0 and item.get("prompt_tokens", 0) == 0:
            m["errors"] += 1
            total_errors += 1

        total_quota += quota
        total_calls += 1

    return {
        "models": dict(models),
        "total_quota": total_quota,
        "total_cost_usd": total_quota / QUOTA_TO_USD,
        "total_calls": total_calls,
        "total_errors": total_errors,
    }


def format_report(agg, start_time, end_time):
    """格式化为飞书消息文本"""
    lines = []
    lines.append("⚡ 云雾API花费报告")
    lines.append("📅 {} ~ {}".format(start_time.strftime('%m-%d %H:%M'), end_time.strftime('%H:%M')))
    lines.append("💰 总花费: ⚡{:.4f} ({} 次调用)".format(agg['total_cost_usd'], agg['total_calls']))
    if agg["total_errors"] > 0:
        lines.append("❌ 失败请求: {} 次".format(agg['total_errors']))
    lines.append("")

    sorted_models = sorted(
        agg["models"].items(),
        key=lambda x: x[1]["quota"],
        reverse=True,
    )

    for model, stats in sorted_models:
        cost = stats["quota"] / QUOTA_TO_USD
        pct = (stats["quota"] / agg["total_quota"] * 100) if agg["total_quota"] > 0 else 0
        tokens = stats["prompt_tokens"] + stats["completion_tokens"]
        group_label = " [{}]".format(stats['group']) if stats["group"] else ""
        error_label = " (❌{})".format(stats['errors']) if stats["errors"] > 0 else ""

        lines.append(
            "  ■ {}{}: ⚡{:.4f} ({:.1f}%) | {}次{} | {:,} tokens".format(
                model, group_label, cost, pct, stats['calls'], error_label, tokens
            )
        )

    if agg["total_quota"] == 0 and agg["total_calls"] > 0:
        lines.append("")
        lines.append("⚠️ 所有请求quota为0，可能全部失败（额度不足？）")

    return "\n".join(lines)


def send_feishu(webhook_url, text):
    """发送消息到飞书群"""
    if not webhook_url:
        print("[WARN] 飞书webhook URL 未配置，跳过发送")
        return False

    payload = {
        "msg_type": "text",
        "content": {
            "text": text,
        },
    }

    resp = requests.post(webhook_url, json=payload, timeout=10)
    result = resp.json()
    if result.get("code") == 0 or result.get("StatusCode") == 0:
        print("[OK] 飞书消息发送成功")
        return True
    else:
        print("[ERROR] 飞书发送失败: {}".format(result))
        return False


def main():
    parser = argparse.ArgumentParser(description="云雾API花费监控")
    parser.add_argument("--hours", type=int, default=1, help="查询过去N小时 (默认1)")
    parser.add_argument("--dry-run", action="store_true", help="仅打印不发送飞书")
    parser.add_argument("--test-feishu", action="store_true", help="测试飞书webhook")
    parser.add_argument("--webhook", type=str, default="", help="飞书webhook URL (覆盖环境变量)")
    args = parser.parse_args()

    check_env()

    webhook = args.webhook or FEISHU_WEBHOOK_URL

    if args.test_feishu:
        if not webhook:
            print("请通过 --webhook 或 FEISHU_WEBHOOK_URL 环境变量指定webhook URL")
            sys.exit(1)
        send_feishu(webhook, "✅ 云雾API花费监控 - 飞书连通测试成功")
        return

    now = datetime.now(CST)
    end_time = now.replace(minute=0, second=0, microsecond=0)
    start_time = end_time - timedelta(hours=args.hours)

    start_ts = int(start_time.timestamp())
    end_ts = int(end_time.timestamp())

    print("查询时段: {} ~ {} CST".format(start_time.strftime('%Y-%m-%d %H:%M'), end_time.strftime('%H:%M')))

    print("登录云雾api...")
    session = login_yunwu()
    print("登录成功")

    print("获取使用日志...")
    items = fetch_logs(session, start_ts, end_ts)
    print("获取到 {} 条日志".format(len(items)))

    agg = aggregate_costs(items)
    report = format_report(agg, start_time, end_time)
    print("\n" + report)

    if not args.dry_run and webhook:
        send_feishu(webhook, report)
    elif not webhook:
        print("\n[INFO] 未配置飞书webhook，仅本地输出")


if __name__ == "__main__":
    main()
