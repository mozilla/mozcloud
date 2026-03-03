from pprint import pprint
import sys
from datetime import datetime, timezone

import google.auth
import google.auth.transport.requests
import pandas as pd
import requests
from google.cloud import bigquery
import argparse

# --- Configuration ---
# IMPORTANT: Replace with your Google Cloud Project ID.
V2_PROJECT_ID = "moz-fx-metric-scope-v2-prod"
V1_PROJECT_ID = "moz-fx-metric-scope-v1-prod"
MGMT_PROJECT_ID = "moz-fx-metric-scope-mgmt-prod"
BQ_PROJECT_ID = "moz-fx-data-shared-prod"

# --- BigQuery Configuration ---
# IMPORTANT: Replace with your BigQuery Dataset and Table IDs.
BIGQUERY_DATASET_ID = "analysis"
BIGQUERY_TABLE_ID = "wstuckey-mzcld-stats"

VOLUME_TIERS = {
    "high":   "10m",
    "medium": "20m",
    "low":    "30m",
}

APP_CODES = {
    "grafana":         {"metric_scope": V2_PROJECT_ID, "apps": ["grafana"], "volume": "low"},
    # "argocd":        {"metric_scope": MGMT_PROJECT_ID},
    "monitor":         {"metric_scope": V2_PROJECT_ID, "volume": "medium"},
    "fxa":             {"metric_scope": V2_PROJECT_ID, "volume": "high"},
    "remote-settings": {"metric_scope": V2_PROJECT_ID, "volume": "medium"},
    # "mdn":           {"metric_scope": V2_PROJECT_ID},
    "merino":          {"metric_scope": V2_PROJECT_ID, "volume": "high"},
    "sync":            {"metric_scope": V2_PROJECT_ID, "volume": "high"},
    "autopush":        {"metric_scope": V2_PROJECT_ID, "volume": "medium"},
    "experimenter":    {"metric_scope": V2_PROJECT_ID, "volume": "low"},
}
APP_GENERAL_QUERIES = [
    # {
    #     "name": "Mean Requests Per Second",
    #     "query": 'sum(rate(loadbalancing_googleapis_com:https_request_count{monitored_resource="https_lb_rule", target_proxy_name=~".*-${APP_NAME}.*"}[90d]))',
    # },
    # {
    #     "name": "Mean p95 Request Latency (ms)",
    #     "query": 'histogram_quantile(0.95, sum by(le) (increase(loadbalancing_googleapis_com:https_total_latencies_bucket{monitored_resource="https_lb_rule", target_proxy_name=~".*-${APP_NAME}.*"}[90d])))',
    # },
    # {
    #     "name": "Mean Error Rate (Ratio)",
    #     "query": 'sum(increase(loadbalancing_googleapis_com:https_request_count{target_proxy_name=~".*-${APP_NAME}.*", response_code_class!="300", response_code_class!="200", response_code_class!="100"}[90d])) / sum(increase(loadbalancing_googleapis_com:https_request_count{target_proxy_name=~".*-${APP_NAME}.*"}[90d]))',
    # },
    {
        "name": "Service Uptime",
        "query": """
100 *
avg_over_time(
  (
    (
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        backend_target_name=~".*-${APP_NAME}.*",
        monitored_resource="https_lb_rule",
        backend_target_type="BACKEND_SERVICE",
        backend_type="NETWORK_ENDPOINT_GROUP",
        cache_result="DISABLED",
        response_code_class="500"
      }[${RATE_WINDOW}]))
      /
      sum(rate(loadbalancing_googleapis_com:https_backend_request_count{
        backend_target_name=~".*-${APP_NAME}.*",
        monitored_resource="https_lb_rule",
        backend_target_type="BACKEND_SERVICE",
        backend_type="NETWORK_ENDPOINT_GROUP",
        cache_result="DISABLED"
      }[${RATE_WINDOW}]))
    ) < bool 0.01
  )[30d:${RATE_WINDOW}]
)
""",
    }
]
GRAFANA_QUERIES = [
    # {
    #     "app": "grafana",
    #     "name": "Mean Monthly Active Users",
    #     "query": "max(avg_over_time(grafana_stat_active_users{}[90d]))",
    # },
    # {
    #     "app": "grafana",
    #     "name": "Mean Monthly Total Users",
    #     "query": "max(avg_over_time(grafana_stat_total_users{}[90d]))",
    # },
    # {
    #     "app": "grafana",
    #     "name": "Mean Total Dashboards",
    #     "query": "max(avg_over_time(grafana_stat_totals_dashboard[90d]))",
    # },
    # {
    #     "app": "argocd",
    #     "name": "Total Successful ArgoCD Syncs",
    #     "query": "sum(increase(argocd_app_sync_total{namespace=\"argocd-webservices\", phase=\"Succeeded\"}[90d]) <= 200)",
    # },
    # {
    #     "app": "argocd",
    #     "name": "Total Failed ArgoCD Syncs",
    #     "query": "sum(increase(argocd_app_sync_total{namespace=\"argocd-webservices\", phase=\"Failed\"}[90d]) <= 200)",
    # },
    # {
    #     "app": "argocd",
    #     "name": "Mean ArgoCD Syncs Per Day",
    #     "query": "sum(increase(argocd_app_sync_total{namespace=\"argocd-webservices\"}[90d]) <= 200) / 90",
    # },
    # {
    #     "app": "argocd",
    #     "name": "Mean Count of Healthy ArgoCD Apps in Prod",
    #     "query": """sum(
    #             avg_over_time(argocd_app_info{
    #                 namespace=~\"argocd-(web|data)services\",
    #                 dest_namespace=~\".*-prod\",
    #                 health_status=\"Healthy\"
    #             }[90d])
    #         )
    #     """,
    # },
    # {
    #     "app": "argocd",
    #     "name": "Proportion of Degraded ArgoCD Apps in Prod",
    #     "query": """sum(
    #         avg_over_time(argocd_app_info{
    #             namespace=~"argocd-(web|data)services",
    #             dest_namespace=~".*-prod",
    #             health_status="Degraded"
    #         }[90d])
    #     ) / sum(
    #         avg_over_time(argocd_app_info{
    #             namespace=~"argocd-(web|data)services",
    #             dest_namespace=~".*-prod",
    #             health_status="Healthy"
    #         }[90d])
    #     )
    #     """,
    # },
    # {
    #     "app": "argocd",
    #     "name": "Proportion of Failed Syncs",
    #     "query": """sum(
    #         increase(argocd_app_sync_total{namespace="argocd-webservices", phase="Failed"}[90d]) <= 200
    #     ) / sum(
    #         increase(argocd_app_sync_total{namespace="argocd-webservices"}[90d]) <= 200
    #     )"""
    # }
]
# --- End of Configuration ---


def get_quarter_end_times(cq_string: None | str = None) -> tuple[datetime, datetime]:
    """Calculates the end times for the current and previous quarters."""
    if cq_string:
        cq = pd.Period(cq_string, freq="Q")
        current_q_end = cq.to_timestamp(how="end")
    else:
        current_q_end = datetime.now(timezone.utc) + pd.tseries.offsets.QuarterEnd(0)

    previous_q_end = current_q_end - pd.tseries.offsets.QuarterEnd(1)

    # Normalize to timezone-aware UTC datetimes
    if isinstance(current_q_end, pd.Timestamp):
        current_q_end = current_q_end.to_pydatetime()
    if current_q_end.tzinfo is None:
        current_q_end = current_q_end.replace(tzinfo=timezone.utc)

    if isinstance(previous_q_end, pd.Timestamp):
        previous_q_end = previous_q_end.to_pydatetime()
    if previous_q_end.tzinfo is None:
        previous_q_end = previous_q_end.replace(tzinfo=timezone.utc)

    return current_q_end, previous_q_end


def query_promql_value(
    project_id: str,
    promql_query: str,
    query_time: datetime,
    access_token: str,
    debug: bool = False,
) -> float | None:
    """
    Executes a PromQL query using the GCP Monitoring REST API.

    Args:
        project_id: The GCP project ID.
        promql_query: The PromQL query string to execute.
        query_time: The timestamp at which to evaluate the query.
        access_token: The OAuth2 access token for authorization.
        debug: If True, outputs full query string and response details.

    Returns:
        The value from the query as a float, or None if no data.
    """
    if debug:
        print(f"  Executing PromQL: {promql_query} at {query_time.isoformat()}")
    else:
        print(f"  Executing PromQL: {promql_query[:80]}... at {query_time.isoformat()}")
    # print(f"  Executing PromQL: {promql_query} at {query_time.isoformat()}")

    # The API is global, so the location is 'global'.
    url = f"https://monitoring.googleapis.com/v1/projects/{project_id}/location/global/prometheus/api/v1/query"

    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/x-www-form-urlencoded",
    }

    # The 'time' parameter can be an RFC3339 string or a Unix timestamp.
    # Ensure RFC3339 UTC with trailing 'Z' and without offset duplication.
    time_utc = query_time.astimezone(timezone.utc)
    time_str = time_utc.strftime("%Y-%m-%dT%H:%M:%SZ")
    data = {
        "query": promql_query,
        "time": time_str,
    }

    result = None
    try:
        response = requests.post(url, headers=headers, data=data)
        response.raise_for_status()  # Raise an exception for bad status codes (4xx or 5xx)
        result = response.json()

        if debug:
            print(f"  Full API Response: {result}")

        if result.get("status") == "success" and result.get("data"):
            data_result = result["data"].get("result")
            if data_result and result["data"].get("resultType") == "scalar":
                # For scalar results, it's a list with [timestamp, value].
                # The value is a string and needs to be cast to float.
                value_str = data_result[1]
                return float(value_str)
            elif data_result and result["data"].get("resultType") == "vector":
                # For vector results, take the value from the first item in the vector.
                value_str = data_result[0]["value"][1]
                return float(value_str)

    except requests.exceptions.RequestException as e:
        print(f"  An HTTP error occurred: {e}")
        if e.response:
            print(f"  Response body: {e.response.text}")
    except (KeyError, IndexError, ValueError) as e:
        print(f"  Error parsing the API response: {e}")
        print(f"  Response JSON: {result if result is not None else 'N/A'}")

    print("  No data found or result was not a scalar/vector.")
    print("  query: %s" % promql_query)
    return None


def get_authentication_token() -> str:
    """
    Retrieves the OAuth2 access token for Google Cloud APIs.

    Returns:
        The access token as a string.
    """
    # --- Get Authentication Token ---
    try:
        print("Authenticating with Google Cloud...")
        creds, _ = google.auth.default(
            scopes=["https://www.googleapis.com/auth/monitoring.read"]
        )
        auth_req = google.auth.transport.requests.Request()
        creds.refresh(auth_req)
        access_token = creds.token
        print("Authentication successful.")
        return access_token
    except google.auth.exceptions.DefaultCredentialsError:
        print("\n--- AUTHENTICATION ERROR ---", file=sys.stderr)
        print("Could not find default credentials.", file=sys.stderr)
        print(
            "Please run 'gcloud auth application-default login' in your terminal and try again.",
            file=sys.stderr,
        )
        sys.exit(1)
    except Exception as e:
        print(
            f"An unexpected error occurred during initialization: {e}", file=sys.stderr
        )
        sys.exit(1)


def main():
    """Main function to orchestrate the querying and writing to BigQuery."""
    parser = argparse.ArgumentParser(
        description="Query Prometheus metrics and write quarter-over-quarter stats to BigQuery."
    )
    parser.add_argument(
        "--current-quarter",
        type=str,
        help="Override the current quarter with quarter notation: Q2 2025",
    )
    parser.add_argument("--now", action="store_true")
    parser.add_argument(
        "--app",
        type=str,
        help="Filter results by specific app name (e.g., 'amo', 'fxa', 'mdn'). If not provided, all apps will be processed.",
    )
    parser.add_argument(
        "--debug",
        action="store_true",
        help="Enable debug mode to output full query strings and API responses.",
    )
    args = parser.parse_args()

    now = args.now

    current_q_end, prev_q_end = get_quarter_end_times(args.current_quarter or None)
    access_token = get_authentication_token()
    bigquery_client = bigquery.Client(project=BQ_PROJECT_ID)

    print(f"Evaluating current quarter as of: {current_q_end.isoformat()}")
    print(f"Evaluating previous quarter as of: {prev_q_end.isoformat()}")
    print("-" * 30)
    # return

    results = []
    queries = GRAFANA_QUERIES + APP_GENERAL_QUERIES
    for app_key, app_meta in APP_CODES.items():
        if args.app and app_key != args.app:
            print(f"Skipping app {app_key} as it does not match the filter: {args.app}")
            continue

        # Token used inside PromQL replacement (may differ from logical app key)
        app_token = (
            app_meta["app_name"] if "app_name" in app_meta else f"{app_key}-prod"
        )

        apps = app_meta.get("apps") or [""]
        print(f"Debug: app_token: ${app_token}")

        for app in apps:
            app_token = f"{app_token}-{app}"
            rate_window = VOLUME_TIERS[app_meta.get("volume", "medium")]

        for query in queries:
                if query.get("app") is None:
                    promql_query = query["query"].replace("${APP_NAME}", app_token)
                    promql_query = promql_query.replace("${RATE_WINDOW}", rate_window)
                    print(f"Processing generic query for {app_key}: {query['name']} (rate window: {rate_window})")
                elif query["app"] != app_key:
                    print(f"Skipping query for {query['app']} in {app_key} context.")
                    continue
                else:
                    promql_query = query["query"]

                metric_name = query["name"]
                project_id = (
                    app_meta["metric_scope"]
                    if "metric_scope" in app_meta
                    else V2_PROJECT_ID
                )

                promql_query = promql_query.replace("${PROJECT_ID}", project_id)
                print(
                    f"Processing metric: {metric_name} for app: {app_key} (target: {app_token}) in project: {project_id}"
                )

                if not now:
                    current_val = query_promql_value(
                        project_id,
                        promql_query,
                        current_q_end,
                        access_token,
                        args.debug,
                    )

                    if app_meta.get("metric_scope_pq"):
                        project_id = app_meta["metric_scope_pq"]
                    prev_val = query_promql_value(
                        project_id, promql_query, prev_q_end, access_token, args.debug
                    )

                    qoq_change = None
                    if (
                        current_val is not None
                        and prev_val is not None
                        and prev_val != 0
                    ):
                        qoq_change = (current_val - prev_val) / prev_val
                else:
                    current_val = query_promql_value(
                        project_id,
                        promql_query,
                        datetime.now(timezone.utc),
                        access_token,
                        args.debug,
                    )
                    prev_val = None
                    qoq_change = None

                results.append(
                    {
                        "app": app_key,
                        "metric_name": metric_name,
                        "current_quarter_avg": current_val,
                        "previous_quarter_avg": prev_val,
                        "qoq_change": qoq_change,
                        "last_updated": datetime.now(timezone.utc).isoformat(),
                        "current_quarter": f"Q{pd.Timestamp(current_q_end).quarter}",
                        "previous_quarter": f"Q{pd.Timestamp(prev_q_end).quarter}",
                    }
                )
                print("-" * 30)

    if not results:
        print("No results to write to BigQuery.")
        return

    if args.debug:
        pprint(results)

    # --- Prepare DataFrame and Write to BigQuery ---
    df = pd.DataFrame(results)
    df["current_quarter_avg"] = df["current_quarter_avg"].astype("float64")
    df["previous_quarter_avg"] = df["previous_quarter_avg"].astype("float64")
    df["qoq_change"] = df["qoq_change"].astype("float64")
    df["last_updated"] = pd.to_datetime(df["last_updated"])

    # Load DataFrame to staging table
    staging_table_id = BIGQUERY_TABLE_ID + "_staging"
    staging_table_ref = bigquery_client.dataset(BIGQUERY_DATASET_ID).table(
        staging_table_id
    )
    job_config = bigquery.LoadJobConfig(
        write_disposition=bigquery.WriteDisposition.WRITE_TRUNCATE
    )
    try:
        print(
            f"Writing data to BigQuery staging table: {BQ_PROJECT_ID}.{BIGQUERY_DATASET_ID}.{staging_table_id}"
        )
        job = bigquery_client.load_table_from_dataframe(
            df, staging_table_ref, job_config=job_config
        )
        job.result()
        print(f"Successfully loaded {job.output_rows} rows into staging table.")
    except Exception as e:
        print(f"An error occurred while writing to staging table: {e}")
        return

    # Merge from staging to main table
    merge_sql = f"""
    MERGE `{BQ_PROJECT_ID}.{BIGQUERY_DATASET_ID}.{BIGQUERY_TABLE_ID}` T
    USING `{BQ_PROJECT_ID}.{BIGQUERY_DATASET_ID}.{staging_table_id}` S
    ON T.app = S.app AND T.metric_name = S.metric_name AND T.current_quarter = S.current_quarter
    WHEN MATCHED THEN
      UPDATE SET
        current_quarter_avg = S.current_quarter_avg,
        previous_quarter_avg = S.previous_quarter_avg,
        qoq_change = S.qoq_change,
        previous_quarter = S.previous_quarter,
        last_updated = S.last_updated
    WHEN NOT MATCHED THEN
      INSERT (app, metric_name, current_quarter, previous_quarter, current_quarter_avg, previous_quarter_avg, qoq_change, last_updated)
      VALUES (S.app, S.metric_name, S.current_quarter, S.previous_quarter, S.current_quarter_avg, S.previous_quarter_avg, S.qoq_change, S.last_updated)
    """
    try:
        print(
            f"Merging data from staging to main table: {BQ_PROJECT_ID}.{BIGQUERY_DATASET_ID}.{BIGQUERY_TABLE_ID}"
        )
        merge_job = bigquery_client.query(merge_sql)
        merge_job.result()
        print("Successfully merged data into main table.")
    except Exception as e:
        print(f"An error occurred during merge: {e}")
        return

    try:
        bigquery_client.delete_table(staging_table_ref, not_found_ok=True)
        print(f"Staging table {staging_table_id} deleted.")
    except Exception as e:
        print(f"Could not delete staging table: {e}")


if __name__ == "__main__":
    main()
