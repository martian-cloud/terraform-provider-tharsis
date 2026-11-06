# Minimal GitLab stand-in for Tharsis acceptance tests: completes the OAuth code exchange
# and answers the user and project lookups a workspace VCS provider link needs.
#
# The link tests are opt-in. Run it where the Tharsis API can reach it, e.g. next to a local
# tharsis-all-in-one-docker container on a shared Docker network:
#
#   docker network create trn-vcs && docker network connect trn-vcs <tharsis-container>
#   docker run -d --name fake-vcs --network trn-vcs \
#     -v "$PWD/internal/provider/testdata/fake_vcs:/app:ro" python:3-alpine python /app/server.py
#   THARSIS_TEST_FAKE_VCS_URL=http://fake-vcs:8080 TF_ACC=1 go test ./internal/provider/ -run TestTRNReferenceLink
import json, time
from http.server import BaseHTTPRequestHandler, HTTPServer

class H(BaseHTTPRequestHandler):
    def _json(self, code, body):
        b = json.dumps(body).encode()
        self.send_response(code); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_POST(self):
        if self.path.startswith("/oauth/token"):
            return self._json(200, {"access_token": "fake-access-token", "refresh_token": "fake-refresh",
                                    "expires_in": 86400, "created_at": int(time.time())})
        self._json(404, {"message": "not found"})
    def do_GET(self):
        if self.path.startswith("/api/v4/user"):
            return self._json(200, {"id": 1, "username": "fake"})
        if self.path.startswith("/api/v4/projects/"):
            return self._json(200, {"default_branch": "main"})
        self._json(404, {"message": "not found"})
    def log_message(self, *a): print(self.command, self.path, flush=True)

HTTPServer(("0.0.0.0", 8080), H).serve_forever()
