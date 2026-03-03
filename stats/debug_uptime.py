"""
Diagnostic script for debugging Service Uptime query results.

Runs several queries for a given app to help understand:
  - Whether low traffic volume is causing noise
  - What the actual error rate looks like (without the bool threshold)
  - How different rate windows and thresholds affect the uptime %
"""
import sys
from datetime import datetime, timezone

import google.auth
import google.auth.transport.requests
import requests
import argparse

V2_PROJECT_ID = "moz-fx-metric-scope-v2-prod"
MGMT_PROJECT_ID = "moz-fx-metric-scope-mgmt-prod"

VOLUME_TIERS = {
    "high":   "10m",
    "medium": "20m",
    "low":    "30m",
}

APP_CODES = {
    "grafana":         {"metric_scope": V2_PROJECT_ID, "volume": "low"},
    "monitor":         {"metric_scope": V2_PROJECT_ID, "volume": "medium"},
    "fxa":             {"metric_scope": V2_PROJECT_ID, "volume": "high"},
    "remote-settings": {"metric_scope": V2_PROJECT_ID, "volume": "medium"},
    "merino":          {"metric_scope": V2_PROJECT_ID, "volume": "high"},
    "sync":            {"metric_scope": V2_PROJECT_ID, "volume": "high"},
    "autopush":        {"metric_scope": V2_PROJECT_ID, "volume": "medium"},
    "experimenter":    {"metric_scope": V2_PROJECT_ID, "volume": "low"},
}

BASE_FILTERS = """
    backend_target_name=~".*-${APP_NAME}.*",
    monitored_resource="https_lb_rule",
    backend_target_type="BACKEND_SERVICE",
    backend_type="NETWORK_ENDPOINT_GROUP",
    cache_result="DISABLED"
"""

DIAGNOSTIC_QUERIES = [
    {
        "name": "Mean requests/sec",
        "description": "Average traffic rate — low values confirm a low-volume service",
        "query": """
avg_over_time(
  sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
    """ + BASE_FILTERS + """
  }[10m]))[30d:10m]
)
""",
    },
    {
        "name": "Raw mean error rate % (10m window)",
        "description": "Actual mean 500 error rate % with no threshold — shows if errors are genuine",
        "query": """
100 *
avg_over_time(
  (
    sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
      """ + BASE_FILTERS + """,
      response_code_class="500"
    }[10m]))
    /
    sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
      """ + BASE_FILTERS + """
    }[10m]))
  )[30d:10m]
)
""",
    },
    {
        "name": "Uptime % (${RATE_WINDOW} window, 1% threshold)  [current]",
        "description": "Current production query using this app's volume tier window",
        "query": """
100 *
avg_over_time(
  (
    (
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        """ + BASE_FILTERS + """,
        response_code_class="500"
      }[${RATE_WINDOW}]))
      /
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        """ + BASE_FILTERS + """
      }[${RATE_WINDOW}]))
    ) < bool 0.01
  )[30d:${RATE_WINDOW}]
)
""",
    },
    {
        "name": "Uptime % (30m window, 1% threshold)",
        "description": "Wider rate window — smooths out noise for low-volume services",
        "query": """
100 *
avg_over_time(
  (
    (
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        """ + BASE_FILTERS + """,
        response_code_class="500"
      }[30m]))
      /
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        """ + BASE_FILTERS + """
      }[30m]))
    ) < bool 0.01
  )[30d:30m]
)
""",
    },
    {
        "name": "Uptime % (30m window, 5% threshold)",
        "description": "Wider window + more lenient threshold — typical for lower-traffic SLOs",
        "query": """
100 *
avg_over_time(
  (
    (
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        """ + BASE_FILTERS + """,
        response_code_class="500"
      }[30m]))
      /
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        """ + BASE_FILTERS + """
      }[30m]))
    ) < bool 0.05
  )[30d:30m]
)
""",
    },
]


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
    data = {"query": promql_query, "time": time_str}

    result = None
    try:
        response = requests.post(url, headers=headers, data=data)
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
        print(f"    HTTP error: {e}")
        if e.response:
            print(f"    Response body: {e.response.text}")
    except (KeyError, IndexError, ValueError) as e:
        print(f"    Parse error: {e} — response: {result}")

    return None


def main():
    parser = argparse.ArgumentParser(
        description="Diagnose Service Uptime query results for a given app."
    )
    parser.add_argument(
        "--app",
        type=str,
        required=True,
        help="App to diagnose (e.g. 'fxa', 'merino', 'autopush')",
    )
    parser.add_argument(
        "--debug",
        action="store_true",
        help="Print full API responses",
    )
    args = parser.parse_args()

    app_key = args.app
    if app_key not in APP_CODES:
        print(f"Unknown app '{app_key}'. Available: {', '.join(APP_CODES)}")
        sys.exit(1)

    app_meta = APP_CODES[app_key]
    app_token = f"{app_key}-prod"
    project_id = app_meta["metric_scope"]
    query_time = datetime.now(timezone.utc)

    rate_window = VOLUME_TIERS[app_meta.get("volume", "medium")]
    access_token = get_authentication_token()

    print(f"App:          {app_key}")
    print(f"Token:        {app_token}")
    print(f"Project:      {project_id}")
    print(f"Volume tier:  {app_meta.get('volume', 'medium')} ({rate_window} window)")
    print(f"Query time:   {query_time.isoformat()}")
    print(f"Lookback:     30 days")
    print("=" * 60)

    for q in DIAGNOSTIC_QUERIES:
        promql = q["query"].replace("${APP_NAME}", app_token).replace("${RATE_WINDOW}", rate_window)
        name = q["name"].replace("${RATE_WINDOW}", rate_window)
        print(f"\n{name}")
        print(f"  {q['description']}")
        if args.debug:
            print(f"  Query:\n{promql}")
        value = query_promql(project_id, promql, query_time, access_token, args.debug)
        if value is None:
            print("  Result: no data")
        else:
            print(f"  Result: {value:.4f}")

    print("\n" + "=" * 60)
    print("Interpretation guide:")
    print("  Mean requests/sec < 0.1  → very low volume, noise likely")
    print("  Raw error rate high      → genuine errors, not noise")
    print("  Large gap between 10m and 30m uptime → noise is the problem")
    print("  Large gap between 1% and 5% threshold → borderline error rate")


if __name__ == "__main__":
    main()
