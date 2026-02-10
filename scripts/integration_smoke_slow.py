#!/usr/bin/env python3
import json
import os
import sys
import time
import uuid
import threading
import tempfile
import subprocess
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib import request, error, parse

API_BASE = "http://localhost:8080/api/v1"
PROM_STUB_PORT = 18081
WEBHOOK_STUB_PORT = 18082
PROM_STUB_URL = f"http://host.docker.internal:{PROM_STUB_PORT}"
WEBHOOK_URL = f"http://host.docker.internal:{WEBHOOK_STUB_PORT}/hook"

class _State:
    firing = True
    webhook_hits = []

class PromStubHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/api/v1/query"):
            self._handle_query()
            return
        self.send_response(404)
        self.end_headers()

    def _handle_query(self):
        now = time.time()
        if _State.firing:
            result = [{
                "metric": {"__name__": "up", "instance": "stub"},
                "value": [now, "1"],
            }]
        else:
            result = []
        body = {
            "status": "success",
            "data": {
                "resultType": "vector",
                "result": result,
            },
        }
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(body).encode("utf-8"))

    def log_message(self, fmt, *args):
        return

class WebhookStubHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length > 0 else b""
        _State.webhook_hits.append({"path": self.path, "body": raw.decode("utf-8")})
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b"{\"ok\":true}")

    def log_message(self, fmt, *args):
        return


def start_server(port, handler):
    server = HTTPServer(("0.0.0.0", port), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server


def http(method, path, data=None, headers=None, timeout=15):
    url = API_BASE + path
    body = None
    if data is not None:
        body = json.dumps(data).encode("utf-8")
    req = request.Request(url, data=body, method=method)
    req.add_header("Content-Type", "application/json")
    if headers:
        for k, v in headers.items():
            req.add_header(k, v)
    try:
        with request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw)
    except error.HTTPError as e:
        raw = e.read().decode("utf-8")
        raise RuntimeError(f"HTTP {e.code}: {raw}")


def wait_for(predicate, timeout_s, step_s=5, label=""):
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        if predicate():
            return True
        time.sleep(step_s)
    raise RuntimeError(f"timeout waiting for {label}")


def capture_ws_messages(expected_count, timeout_s):
    tmp = tempfile.NamedTemporaryFile(delete=False)
    tmp_path = tmp.name
    tmp.close()
    script = f"""
const fs = require('fs');
let WebSocketCtor = global.WebSocket;
if (!WebSocketCtor) {{
  try {{ WebSocketCtor = require('ws'); }} catch (e) {{
    fs.writeFileSync('{tmp_path}', 'NO_WEBSOCKET');
    process.exit(1);
  }}
}}
const ws = new WebSocketCtor('ws://localhost:8080/api/v1/ws');
let msgs = [];
ws.onmessage = (evt) => {{
  msgs.push(evt.data.toString());
  if (msgs.length >= {expected_count}) {{
    fs.writeFileSync('{tmp_path}', msgs.join('\\n'));
    process.exit(0);
  }}
}};
setTimeout(() => {{
  fs.writeFileSync('{tmp_path}', msgs.join('\\n'));
  process.exit(0);
}}, {int(timeout_s*1000)});
"""
    proc = subprocess.Popen(["node", "-e", script])
    return proc, tmp_path


def main():
    prom = start_server(PROM_STUB_PORT, PromStubHandler)
    webhook = start_server(WEBHOOK_STUB_PORT, WebhookStubHandler)

    created = {"rule_id": None, "channel_id": None, "template_id": None, "sla_config_id": None, "user_id": None, "escalation_id": None}
    results = []

    def check(name, fn):
        try:
            fn()
            results.append((name, "ok", None))
        except Exception as e:
            results.append((name, "fail", str(e)))

    login = http("POST", "/auth/login", {"username": "admin", "password": "admin123"})
    token = login["data"]["token"]
    admin_headers = {"Authorization": f"Bearer {token}"}

    groups = http("GET", "/business-groups", headers=admin_headers)["data"]["data"]
    if not groups:
        raise RuntimeError("no business group found")
    group_id = groups[0]["id"]

    def create_test_user():
        username = f"it_user_{uuid.uuid4().hex[:6]}"
        password = "pass1234"
        create = http("POST", "/users", {
            "username": username,
            "password": password,
            "email": "it@example.com",
            "role": "user",
            "status": 1,
        }, headers=admin_headers)
        created["user_id"] = create["data"]["id"]
        # login as user for escalation pending
        user_login = http("POST", "/auth/login", {"username": username, "password": password})
        return user_login["data"]["token"], username

    def create_sla_config():
        resp = http("POST", "/sla/configs", {
            "name": f"IT-SLA-{uuid.uuid4().hex[:6]}",
            "severity": "warning",
            "response_time_mins": 1,
            "resolution_time_mins": 1,
            "priority": 999,
        }, headers=admin_headers)
        created["sla_config_id"] = resp["data"]["id"]

    def create_template():
        resp = http("POST", "/templates", {
            "name": f"IT-Template-{uuid.uuid4().hex[:6]}",
            "description": "integration template",
            "content": "rule={{ruleName}} severity={{severity}}",
            "variables": {"ruleName": "Rule Name", "severity": "Severity"},
            "type": "markdown",
            "status": 1,
        }, headers=admin_headers)
        created["template_id"] = resp["data"]["id"]

    def create_channel():
        resp = http("POST", "/channels", {
            "name": f"IT-Channel-{uuid.uuid4().hex[:6]}",
            "type": "webhook",
            "description": "integration test webhook",
            "config": {"url": WEBHOOK_URL},
            "group_id": group_id,
            "status": 1,
        }, headers=admin_headers)
        created["channel_id"] = resp["data"]["id"]

    def create_rule_and_bind():
        resp = http("POST", "/alert-rules", {
            "name": f"IT-Rule-{uuid.uuid4().hex[:6]}",
            "description": "integration test rule",
            "expression": "up",
            "evaluation_interval_seconds": 60,
            "for_duration": 1,
            "severity": "warning",
            "labels": {"env": "it"},
            "annotations": {"summary": "integration test"},
            "group_id": group_id,
            "template_id": created["template_id"],
            "data_source_type": "prometheus",
            "data_source_url": PROM_STUB_URL,
            "status": 1,
            "effective_start_time": "00:00",
            "effective_end_time": "23:59",
            "exclusion_windows": [],
        }, headers=admin_headers)
        created["rule_id"] = resp["data"]["id"]
        http("POST", f"/alert-rules/{created['rule_id']}/bindings", {"channel_ids": [created["channel_id"]]}, headers=admin_headers)

    alert_history_entry = {}

    def wait_for_firing():
        def has_firing():
            data = http("GET", "/alert-history", headers=admin_headers)["data"]["data"]
            for row in data or []:
                if row.get("rule_id") == created["rule_id"] and row.get("status") == "firing":
                    alert_history_entry.update(row)
                    return True
            return False
        wait_for(has_firing, 140, label="firing alert history")

    def wait_for_webhook(min_count):
        wait_for(lambda: len(_State.webhook_hits) >= min_count, 140, label="webhook delivery")

    def wait_for_resolved():
        def has_resolved():
            data = http("GET", "/alert-history", headers=admin_headers)["data"]["data"]
            for row in data or []:
                if row.get("rule_id") == created["rule_id"] and row.get("status") == "resolved":
                    return True
            return False
        wait_for(has_resolved, 140, label="resolved alert history")

    def sla_breach_check():
        http("POST", "/sla/breaches/check", headers=admin_headers)
        data = http("GET", "/sla/breaches", headers=admin_headers)["data"]["data"]
        if not data:
            raise RuntimeError("expected SLA breach record")

    def ws_check(proc, path):
        proc.wait(timeout=130)
        with open(path, "r", encoding="utf-8") as f:
            content = f.read().strip()
        os.unlink(path)
        if content == "NO_WEBSOCKET" or content == "":
            raise RuntimeError("no websocket messages captured")
        if "\"type\":\"alert\"" not in content:
            raise RuntimeError("missing alert websocket message")
        if "\"type\":\"ticket\"" not in content:
            raise RuntimeError("missing ticket websocket message")

    def oncall_flow():
        sched = http("POST", "/oncall/schedules", {
            "name": f"IT-Schedule-{uuid.uuid4().hex[:6]}",
            "description": "integration schedule",
            "timezone": "UTC",
            "rotation_type": "weekly",
            "rotation_start": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        }, headers=admin_headers)
        sched_id = sched["data"]["id"]
        http("POST", f"/oncall/schedules/{sched_id}/members", {
            "user_id": login["data"]["user"]["id"],
            "username": "admin",
            "priority": 1,
        }, headers=admin_headers)
        http("GET", f"/oncall/schedules/{sched_id}/members", headers=admin_headers)
        http("POST", f"/oncall/schedules/{sched_id}/generate-rotations", {
            "end_time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time()+86400))
        }, headers=admin_headers)
        http("GET", f"/oncall/schedules/{sched_id}/assignments", headers=admin_headers)
        http("GET", "/oncall/current", headers=admin_headers)
        http("GET", "/oncall/who", headers=admin_headers)
        http("GET", "/oncall/report", headers=admin_headers)
        http("POST", f"/oncall/schedules/{sched_id}/generate", {
            "start_time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "end_time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time()+7200)),
            "shift_duration": 60,
            "timezone": "UTC",
        }, headers=admin_headers)
        http("GET", f"/oncall/schedules/{sched_id}/coverage", headers=admin_headers)
        http("GET", f"/oncall/schedules/{sched_id}/suggest", headers=admin_headers)
        http("GET", f"/oncall/schedules/{sched_id}/validate", headers=admin_headers)
        http("DELETE", f"/oncall/schedules/{sched_id}", headers=admin_headers)

    def correlation_flow():
        alert_id = alert_history_entry.get("id")
        fingerprint = alert_history_entry.get("fingerprint")
        http("GET", f"/correlation/analyze/{alert_id}?window_minutes=30", headers=admin_headers)
        http("GET", f"/correlation/patterns?hours=1&min_occurrences=1", headers=admin_headers)
        http("GET", f"/correlation/groups?hours=1&threshold=0.5", headers=admin_headers)
        if fingerprint:
            safe_fp = parse.quote(fingerprint, safe="")
            http("GET", f"/correlation/timeline/{safe_fp}?hours=1", headers=admin_headers)
        http("GET", f"/correlation/flapping?rule_id={created['rule_id']}&hours=1&threshold=1", headers=admin_headers)
        http("GET", f"/correlation/predict/{created['rule_id']}?hours=1", headers=admin_headers)

    def escalation_flow(user_token, username):
        alert_id = alert_history_entry.get("id")
        user_headers = {"Authorization": f"Bearer {user_token}"}
        # create escalation
        esc = http("POST", "/escalations", {
            "alert_id": alert_id,
            "to_user_id": created["user_id"],
            "to_username": username,
            "reason": "integration test",
        }, headers=admin_headers)
        esc_id = esc["data"]["id"]
        # pending for user
        http("GET", "/escalations/pending", headers=user_headers)
        http("POST", f"/escalations/{esc_id}/accept", headers=user_headers)
        http("POST", f"/escalations/{esc_id}/resolve", headers=admin_headers)
        # reject path
        esc2 = http("POST", "/escalations", {
            "alert_id": alert_id,
            "to_user_id": created["user_id"],
            "to_username": username,
            "reason": "integration reject",
        }, headers=admin_headers)
        esc2_id = esc2["data"]["id"]
        http("POST", f"/escalations/{esc2_id}/reject", headers=user_headers)
        http("GET", f"/escalations/alert/{alert_id}", headers=admin_headers)
        http("GET", "/escalations", headers=admin_headers)
        http("GET", "/escalations/stats", headers=admin_headers)

    def create_ticket_for_ws():
        http("POST", "/tickets", {
            "title": f"IT-Ticket-{uuid.uuid4().hex[:6]}",
            "description": "ws ticket",
            "priority": "low",
            "assignee_name": "admin",
        }, headers=admin_headers)

    def cleanup(user_token=None):
        if created["rule_id"]:
            try:
                http("DELETE", f"/alert-rules/{created['rule_id']}", headers=admin_headers)
            except Exception:
                pass
        if created["channel_id"]:
            try:
                http("DELETE", f"/channels/{created['channel_id']}", headers=admin_headers)
            except Exception:
                pass
        if created["template_id"]:
            try:
                http("DELETE", f"/templates/{created['template_id']}", headers=admin_headers)
            except Exception:
                pass
        if created["sla_config_id"]:
            try:
                http("DELETE", f"/sla/configs/{created['sla_config_id']}", headers=admin_headers)
            except Exception:
                pass
        if created["user_id"]:
            try:
                http("DELETE", f"/users/{created['user_id']}", headers=admin_headers)
            except Exception:
                pass

    try:
        ws_proc, ws_path = capture_ws_messages(3, 160)
        user_token, username = create_test_user()
        check("create_sla_config", create_sla_config)
        check("create_template", create_template)
        check("create_channel", create_channel)
        check("create_rule_and_bind", create_rule_and_bind)
        check("alert_firing_history", wait_for_firing)
        check("webhook_firing_delivery", lambda: wait_for_webhook(1))

        # create ticket to validate WS ticket notifications
        check("ticket_ws_emit", create_ticket_for_ws)

        _State.firing = False
        check("alert_resolved_history", wait_for_resolved)
        check("webhook_resolved_delivery", lambda: wait_for_webhook(2))

        # wait for SLA deadlines (1 minute) to pass
        time.sleep(70)
        check("sla_breach_check", sla_breach_check)
        check("websocket_notifications", lambda: ws_check(ws_proc, ws_path))

        check("oncall_flow", oncall_flow)
        check("correlation_flow", correlation_flow)
        check("escalation_flow", lambda: escalation_flow(user_token, username))
        # SLA report + alert SLA read
        if alert_history_entry.get("id"):
            http("GET", f"/sla/alerts/{alert_history_entry['id']}", headers=admin_headers)
        http("GET", "/sla/report", headers=admin_headers)
    finally:
        cleanup()
        prom.shutdown()
        webhook.shutdown()

    print("Summary:")
    ok = True
    for name, status, err in results:
        if status != "ok":
            ok = False
        line = f"- {name}: {status}"
        if err:
            line += f" ({err})"
        print(line)

    if not ok:
        sys.exit(1)

if __name__ == "__main__":
    main()
