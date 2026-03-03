"""
Uptime measurement script for MozCloud services.

For each service:
  1. Queries request volume to auto-determine the volume tier and rate window
  2. Queries service uptime using the appropriate rate window
  3. Outputs per-service results and SLO compliance summary as JSON
"""

import json
import sys
from datetime import datetime, timezone

import google.auth
import google.auth.transport.requests
import requests
import argparse

V2_PROJECT_ID = "moz-fx-metric-scope-v2-prod"
MGMT_PROJECT_ID = "moz-fx-metric-scope-mgmt-prod"

# Ordered high → low. First tier whose min_rps is met wins.
VOLUME_TIERS = [
    {
        "name": "very-high",
        "min_rps": 10000,
        "rate_window": "5m",
        "error_threshold": 0.01,
    },
    {"name": "high", "min_rps": 100, "rate_window": "10m", "error_threshold": 0.01},
    {"name": "medium", "min_rps": 10, "rate_window": "20m", "error_threshold": 0.01},
    {"name": "low", "min_rps": 1, "rate_window": "20m", "error_threshold": 0.05},
    {"name": "very-low", "min_rps": 0, "rate_window": "30m", "error_threshold": 0.10},
]

SLO_THRESHOLDS = [99.9, 99.0, 98.0, 95.0]

APP_CODES = {
    "grafana": {},
    "monitor": {},
    "vpn": {},
    "autograph": {},
    "relay": {},
    "fxa": {},
    "remote-settings": {},
    "merino": {},
    "sync": {},
    "autopush": {},
    "experimenter": {},
}

TRAFFIC_QUERY = """
avg_over_time(
  sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
    backend_target_name=~".*-${APP_TOKEN}.*",
    monitored_resource="https_lb_rule",
    backend_target_type="BACKEND_SERVICE",
    backend_type="NETWORK_ENDPOINT_GROUP",
    cache_result="DISABLED"
  }[10m]))[30d:10m]
)
"""

UPTIME_QUERY = """
100 *
avg_over_time(
  (
    (
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        backend_target_name=~".*-${APP_TOKEN}.*",
        monitored_resource="https_lb_rule",
        backend_target_type="BACKEND_SERVICE",
        backend_type="NETWORK_ENDPOINT_GROUP",
        cache_result="DISABLED",
        response_code_class="500"
      }[${RATE_WINDOW}]))
      /
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        backend_target_name=~".*-${APP_TOKEN}.*",
        monitored_resource="https_lb_rule",
        backend_target_type="BACKEND_SERVICE",
        backend_type="NETWORK_ENDPOINT_GROUP",
        cache_result="DISABLED"
      }[${RATE_WINDOW}]))
    ) < bool ${THRESHOLD}
  )[30d:${RATE_WINDOW}]
)
"""


def determine_tier(rps: float) -> dict:
    for tier in VOLUME_TIERS:
        if rps >= tier["min_rps"]:
            return tier
    return VOLUME_TIERS[-1]


def get_authentication_token() -> str:
    try:
        print("Authenticating with Google Cloud...")
        creds, _ = google.auth.default(
            scopes=["https://www.googleapis.com/auth/monitoring.read"]
        )
        auth_req = google.auth.transport.requests.Request()
        creds.refresh(auth_req)
        print("Authentication successful.\n")
        return creds.token
    except google.auth.exceptions.DefaultCredentialsError:
        print("\n--- AUTHENTICATION ERROR ---", file=sys.stderr)
        print(
            "Please run 'gcloud auth application-default login' and try again.",
            file=sys.stderr,
        )
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error during authentication: {e}", file=sys.stderr)
        sys.exit(1)


def query_promql(
    project_id: str,
    promql_query: str,
    query_time: datetime,
    access_token: str,
    debug: bool = False,
) -> float | None:
    url = f"https://monitoring.googleapis.com/v1/projects/{project_id}/location/global/prometheus/api/v1/query"
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/x-www-form-urlencoded",
    }
    time_str = query_time.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    result = None
    try:
        response = requests.post(
            url, headers=headers, data={"query": promql_query, "time": time_str}
        )
        response.raise_for_status()
        result = response.json()

        if debug:
            print(f"    API response: {result}")

        if result.get("status") == "success" and result.get("data"):
            data_result = result["data"].get("result")
            result_type = result["data"].get("resultType")
            if data_result and result_type == "scalar":
                return float(data_result[1])
            elif data_result and result_type == "vector":
                return float(data_result[0]["value"][1])

    except requests.exceptions.RequestException as e:
        print(f"  HTTP error: {e}", file=sys.stderr)
        if e.response:
            print(f"  Response body: {e.response.text}", file=sys.stderr)
    except (KeyError, IndexError, ValueError) as e:
        print(f"  Parse error: {e}", file=sys.stderr)

    return None


def build_slo_summary(services: list[dict]) -> dict:
    measured = [s for s in services if s["uptime_pct"] is not None]
    summary = {
        "total_services": len(services),
        "measured_services": len(measured),
    }
    if measured:
        summary["mean_uptime_pct"] = round(
            sum(s["uptime_pct"] for s in measured) / len(measured), 4
        )

    slo_compliance = {}
    for threshold in SLO_THRESHOLDS:
        meeting = [s["app"] for s in measured if s["uptime_pct"] >= threshold]
        slo_compliance[str(threshold)] = {
            "services_meeting": meeting,
            "count": len(meeting),
            "pct_of_measured": round(len(meeting) / len(measured) * 100, 1)
            if measured
            else None,
        }
    summary["slo_compliance"] = slo_compliance
    return summary


def print_service_group(services: list[dict], summary: dict, label: str) -> None:
    if not services:
        return
    measured = [s for s in services if s["uptime_pct"] is not None]
    col = "{:<20} {:>8} {:>10} {:>10} {:>10}"
    print(f"\n{label}")
    print(col.format("App", "Tier", "Req/sec", "Threshold", "Uptime %"))
    print("-" * 62)
    for s in sorted(services, key=lambda x: x["uptime_pct"] or -1, reverse=True):
        uptime_str = (
            f"{s['uptime_pct']:.2f}%" if s["uptime_pct"] is not None else "no data"
        )
        rps_str = (
            f"{s['requests_per_sec']:.3f}"
            if s["requests_per_sec"] is not None
            else "no data"
        )
        threshold_str = f"<{s['error_threshold_pct']:.0f}% err"
        print(
            col.format(s["app"], s["volume_tier"], rps_str, threshold_str, uptime_str)
        )
    if measured:
        mean = sum(s["uptime_pct"] for s in measured) / len(measured)
        print("-" * 62)
        print(col.format("Mean", "", "", "", f"{mean:.2f}%"))


def print_summary(services: list[dict], summary: dict) -> None:
    print("\n" + "=" * 60)
    print("SERVICE UPTIME REPORT")
    print("=" * 60)

    high = [s for s in services if s["volume_tier"] in ["high", "very-high"]]
    other = [s for s in services if s["volume_tier"] != "high"]

    print_service_group(high, summary, "High Volume")
    print_service_group(other, summary, "Medium / Low Volume")

    print("\nSLO Compliance (% of measured services)")
    print("-" * 40)
    for threshold, data in summary["slo_compliance"].items():
        bar = "█" * data["count"] + "░" * (summary["measured_services"] - data["count"])
        print(
            f"  ≥{threshold:>5}%  {bar}  {data['count']}/{summary['measured_services']} ({data['pct_of_measured']}%)"
        )
        if data["services_meeting"]:
            print(f"           {', '.join(data['services_meeting'])}")
    print()


def main():
    parser = argparse.ArgumentParser(
        description="Measure service uptime and SLO compliance for MozCloud apps."
    )
    parser.add_argument(
        "--app",
        type=str,
        help="Filter to a single app (e.g. 'fxa'). If omitted, all apps are measured.",
    )
    parser.add_argument(
        "--output",
        type=str,
        default="uptime_report.json",
        help="Path for JSON output (default: uptime_report.json)",
    )
    parser.add_argument(
        "--debug",
        action="store_true",
        help="Print full API responses",
    )
    args = parser.parse_args()

    query_time = datetime.now(timezone.utc)
    access_token = get_authentication_token()

    apps = (
        {args.app: APP_CODES[args.app]}
        if args.app and args.app in APP_CODES
        else APP_CODES
    )
    if args.app and args.app not in APP_CODES:
        print(f"Unknown app '{args.app}'. Available: {', '.join(APP_CODES)}")
        sys.exit(1)

    services = []
    for app_key, app_meta in apps.items():
        app_token = f"{app_key}-prod"
        project_id = app_meta.get("metric_scope", V2_PROJECT_ID)
        print(f"[{app_key}] Querying traffic volume...")

        rps = query_promql(
            project_id,
            TRAFFIC_QUERY.replace("${APP_TOKEN}", app_token),
            query_time,
            access_token,
            args.debug,
        )

        if rps is None:
            print(f"No RPS found for {app_key}")
            continue

        tier = determine_tier(rps) if rps is not None else VOLUME_TIERS[-1]
        rate_window = tier["rate_window"]
        error_threshold = tier["error_threshold"]
        print(
            f"[{app_key}] {rps:.4f} req/sec → {tier['name']} tier ({rate_window} window, threshold: {error_threshold:.0%})"
        )

        print(f"[{app_key}] Querying uptime...")
        uptime = query_promql(
            project_id,
            UPTIME_QUERY.replace("${APP_TOKEN}", app_token)
            .replace("${RATE_WINDOW}", rate_window)
            .replace("${THRESHOLD}", str(error_threshold)),
            query_time,
            access_token,
            args.debug,
        )
        print(
            f"[{app_key}] Uptime: {f'{uptime:.2f}%' if uptime is not None else 'no data'}"
        )

        services.append(
            {
                "app": app_key,
                "app_token": app_token,
                "requests_per_sec": round(rps, 4) if rps is not None else None,
                "volume_tier": tier["name"],
                "rate_window": rate_window,
                "error_threshold_pct": error_threshold * 100,
                "uptime_pct": round(uptime, 4) if uptime is not None else None,
            }
        )

    summary = build_slo_summary(services)
    print_summary(services, summary)

    report = {
        "generated_at": query_time.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "lookback_days": 30,
        "error_thresholds_pct": {
            t["name"]: t["error_threshold"] * 100 for t in VOLUME_TIERS
        },
        "services": services,
        "summary": summary,
    }

    with open(args.output, "w") as f:
        json.dump(report, f, indent=2)
    print(f"Report written to {args.output}")


if __name__ == "__main__":
    main()
