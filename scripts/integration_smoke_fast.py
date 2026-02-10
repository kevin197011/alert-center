#!/usr/bin/env python3
import json
import time
import uuid
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib import request, error

API_BASE = "http://localhost:8080/api/v1"
WEBHOOK_PORT = 18082
TELEGRAM_PORT = 18083
LARK_PORT = 18084

WEBHOOK_URL = f"http://host.docker.internal:{WEBHOOK_PORT}/hook"
TELEGRAM_API_BASE = f"http://host.docker.internal:{TELEGRAM_PORT}"
LARK_WEBHOOK_URL = f"http://host.docker.internal:{LARK_PORT}/lark"

class SimpleOkHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b"{\"ok\":true}")
    def log_message(self, fmt, *args):
        return

class LarkOkHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b"{\"code\":0,\"msg\":\"success\"}")
    def log_message(self, fmt, *args):
        return


def start_server(port, handler):
    server = HTTPServer(("0.0.0.0", port), handler)
    t = threading.Thread(target=server.serve_forever, daemon=True)
    t.start()
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


def main():
    webhook = start_server(WEBHOOK_PORT, SimpleOkHandler)
    telegram = start_server(TELEGRAM_PORT, SimpleOkHandler)
    lark = start_server(LARK_PORT, LarkOkHandler)

    results = []

    def check(name, fn):
        try:
            fn()
            results.append((name, "ok", None))
        except Exception as e:
            results.append((name, "fail", str(e)))

    login = http("POST", "/auth/login", {"username": "admin", "password": "admin123"})
    token = login["data"]["token"]
    headers = {"Authorization": f"Bearer {token}"}

    groups = http("GET", "/business-groups", headers=headers)["data"]["data"]
    if not groups:
        raise RuntimeError("no business group found")
    group_id = groups[0]["id"]

    def templates_crud():
        name = f"IT-Template-{uuid.uuid4().hex[:6]}"
        create = http("POST", "/templates", {
            "name": name,
            "description": "integration template",
            "content": "hello {{ruleName}}",
            "variables": {"ruleName": "Rule Name"},
            "type": "markdown",
            "status": 1,
        }, headers=headers)
        tid = create["data"]["id"]
        http("GET", f"/templates/{tid}", headers=headers)
        http("PUT", f"/templates/{tid}", {"description": "updated"}, headers=headers)
        http("DELETE", f"/templates/{tid}", headers=headers)

    def channels_webhook():
        name = f"IT-Webhook-{uuid.uuid4().hex[:6]}"
        create = http("POST", "/channels", {
            "name": name,
            "type": "webhook",
            "description": "integration webhook",
            "config": {"url": WEBHOOK_URL},
            "group_id": group_id,
            "status": 1,
        }, headers=headers)
        cid = create["data"]["id"]
        http("GET", f"/channels/{cid}", headers=headers)
        http("PUT", f"/channels/{cid}", {"description": "updated"}, headers=headers)
        http("POST", "/channels/test-config", {"type": "webhook", "config": {"url": WEBHOOK_URL}}, headers=headers)
        http("DELETE", f"/channels/{cid}", headers=headers)

    def channels_lark():
        name = f"IT-Lark-{uuid.uuid4().hex[:6]}"
        create = http("POST", "/channels", {
            "name": name,
            "type": "lark",
            "description": "integration lark",
            "config": {"webhook_url": LARK_WEBHOOK_URL},
            "group_id": group_id,
            "status": 1,
        }, headers=headers)
        cid = create["data"]["id"]
        http("POST", "/channels/test-config", {"type": "lark", "config": {"webhook_url": LARK_WEBHOOK_URL}}, headers=headers)
        http("DELETE", f"/channels/{cid}", headers=headers)

    def channels_telegram():
        name = f"IT-TG-{uuid.uuid4().hex[:6]}"
        create = http("POST", "/channels", {
            "name": name,
            "type": "telegram",
            "description": "integration telegram",
            "config": {
                "bot_token": "TESTTOKEN",
                "chat_id": "12345",
                "api_base": TELEGRAM_API_BASE,
            },
            "group_id": group_id,
            "status": 1,
        }, headers=headers)
        cid = create["data"]["id"]
        http("POST", "/channels/test-config", {
            "type": "telegram",
            "config": {"bot_token": "TESTTOKEN", "chat_id": "12345", "api_base": TELEGRAM_API_BASE},
        }, headers=headers)
        http("DELETE", f"/channels/{cid}", headers=headers)

    def data_sources_crud():
        name = f"IT-DS-{uuid.uuid4().hex[:6]}"
        create = http("POST", "/data-sources", {
            "name": name,
            "type": "prometheus",
            "description": "integration ds",
            "endpoint": "http://localhost:9090",
            "config": {},
            "status": 1,
        }, headers=headers)
        did = create["data"]["id"]
        http("GET", f"/data-sources/{did}", headers=headers)
        http("PUT", f"/data-sources/{did}", {"description": "updated"}, headers=headers)
        try:
            http("POST", f"/data-sources/{did}/health-check", headers=headers)
        except Exception:
            pass
        http("DELETE", f"/data-sources/{did}", headers=headers)

    def silences_crud():
        now = int(time.time())
        create = http("POST", "/silences", {
            "name": f"IT-Silence-{uuid.uuid4().hex[:6]}",
            "description": "integration silence",
            "matchers": [{"env": "it"}],
            "start_time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(now)),
            "end_time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(now + 3600)),
        }, headers=headers)
        sid = create["data"]["id"]
        http("PUT", f"/silences/{sid}", {"description": "updated"}, headers=headers)
        http("POST", "/silences/check", {"labels": {"env": "it"}}, headers=headers)
        http("DELETE", f"/silences/{sid}", headers=headers)

    def tickets_crud():
        create = http("POST", "/tickets", {
            "title": f"IT-Ticket-{uuid.uuid4().hex[:6]}",
            "description": "integration ticket",
            "priority": "low",
            "assignee_name": "admin",
        }, headers=headers)
        tid = create["data"]["id"]
        http("GET", f"/tickets/{tid}", headers=headers)
        http("PUT", f"/tickets/{tid}", {"status": "in_progress"}, headers=headers)
        http("POST", f"/tickets/{tid}/resolve", headers=headers)
        http("POST", f"/tickets/{tid}/close", headers=headers)
        http("DELETE", f"/tickets/{tid}", headers=headers)
        http("GET", "/tickets/stats", headers=headers)

    def users_crud():
        username = f"it_user_{uuid.uuid4().hex[:6]}"
        password = "pass1234"
        create = http("POST", "/users", {
            "username": username,
            "password": password,
            "email": "it@example.com",
            "phone": "",
            "role": "user",
            "status": 1,
        }, headers=headers)
        uid = create["data"]["id"]
        http("GET", f"/users/{uid}", headers=headers)
        http("GET", "/users", headers=headers)
        http("PUT", f"/users/{uid}", {"email": "it2@example.com"}, headers=headers)
        http("POST", f"/users/{uid}/password", {"old_password": password, "new_password": "pass5678"}, headers=headers)
        http("DELETE", f"/users/{uid}", headers=headers)

    def batch_import_export():
        name = f"IT-Rule-{uuid.uuid4().hex[:6]}"
        rule = {
            "name": name,
            "description": "batch import",
            "expression": "1",
            "evaluation_interval_seconds": 60,
            "for_duration": 60,
            "severity": "warning",
            "labels": {"env": "it"},
            "annotations": {"summary": "batch"},
            "group_id": group_id,
            "data_source_type": "prometheus",
            "data_source_url": "http://localhost:9090",
            "status": 0,
        }
        http("POST", "/batch/import/rules", {"rules": [rule]}, headers=headers)
        http("GET", "/batch/export/rules", headers=headers)
        http("GET", "/batch/export/channels", headers=headers)
        silence = {
            "name": f"IT-Silence-{uuid.uuid4().hex[:6]}",
            "description": "batch",
            "matchers": [{"env": "it"}],
            "start_time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "end_time": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time()+3600)),
        }
        http("POST", "/batch/import/silences", {"silences": [silence]}, headers=headers)
        http("GET", "/batch/export/silences", headers=headers)

        # cleanup rule by name if present
        data = http("GET", "/alert-rules", headers=headers)["data"]["data"] or []
        for r in data:
            if r.get("name") == name:
                http("DELETE", f"/alert-rules/{r['id']}", headers=headers)

    def misc_reads():
        http("GET", "/statistics", headers=headers)
        http("GET", "/dashboard", headers=headers)
        http("GET", "/audit-logs", headers=headers)
        http("GET", "/audit-logs/export", headers=headers)

    check("templates_crud", templates_crud)
    check("channels_webhook", channels_webhook)
    check("channels_lark", channels_lark)
    check("channels_telegram", channels_telegram)
    check("data_sources_crud", data_sources_crud)
    check("silences_crud", silences_crud)
    check("tickets_crud", tickets_crud)
    check("users_crud", users_crud)
    check("batch_import_export", batch_import_export)
    check("misc_reads", misc_reads)

    webhook.shutdown()
    telegram.shutdown()
    lark.shutdown()

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
        raise SystemExit(1)

if __name__ == "__main__":
    main()
