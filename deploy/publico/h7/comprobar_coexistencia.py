#!/usr/bin/env python3
"""Compare simultaneous, read-only public HTTPS and internal mTLS probes."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import http.client
import json
import re
import socket
import ssl
import sys
import threading

from preparar_proxy import (dns_name, exact_keys, loopback, port, private_json,
                            read_file, tls_context)

MAX_BODY = 4 * 1024 * 1024
TIMEOUT = 10
PRIVATE_ROUTES = (
    "/api/vec/contratacion-temporal/solicitudes", "/api/vec/bolsa/bolsas",
    "/api/vec/session", "/api/vec/bolsa/mi-bolsa", "/portal-empleado/",
    "/area-personal/", "/admin/", "/rrhh/",
)
PUBLIC_REQUIRED = {"/", "/bolsa/", "/readyz", "/api/publico/bolsa/convocatorias",
                   "/api/publico/bolsa/bolsas"}
INTERNAL_REQUIRED = {"/livez", "/readyz", "/portal-empleado/"}
ENDPOINT_KEYS = {"address", "port", "server_name", "ca_file", "checks"}
CONFIG_KEYS = {"fuente_commit", "binario_file", "binario_sha256", "evidencia_h6_file",
               "evidencia_h6_sha256", "publico", "interno", "rondas"}


def digest(value, length=64):
    if (not isinstance(value, str) or not re.fullmatch(r"[a-f0-9]{" + str(length) + r"}", value)
            or value == "0" * length):
        raise ValueError("digest")
    return value


def path(value):
    if (not isinstance(value, str) or len(value) > 512
            or not re.fullmatch(r"/[a-zA-Z0-9/:_.-]*", value) or "//" in value
            or any(part in (".", "..") for part in value.split("/"))):
        raise ValueError("route")
    return value


def validate_endpoint(endpoint, *, internal):
    keys = ENDPOINT_KEYS | ({"client_certificate_file", "client_key_file"} if internal else set())
    exact_keys(endpoint, keys)
    loopback(endpoint["address"])
    port(endpoint["port"])
    dns_name(endpoint["server_name"])
    checks = endpoint["checks"]
    if not isinstance(checks, list) or not 1 <= len(checks) <= 32:
        raise ValueError("checks")
    paths = set()
    for check in checks:
        exact_keys(check, {"path", "sha256"})
        route = path(check["path"])
        if route in paths:
            raise ValueError("duplicate_route")
        if internal:
            admitted = route in INTERNAL_REQUIRED or route.startswith(("/portal-empleado/", "/api/vec/"))
        else:
            admitted = route in PUBLIC_REQUIRED or route.startswith("/api/publico/bolsa/")
        if not admitted:
            raise ValueError("route_scope")
        paths.add(route)
        digest(check["sha256"])
    if not (INTERNAL_REQUIRED if internal else PUBLIC_REQUIRED) <= paths:
        raise ValueError("required_route")
    if not internal and (not any(p.startswith("/api/publico/bolsa/convocatorias/") for p in paths)
                         or not any(p.startswith("/api/publico/bolsa/bolsas/") and p.endswith("/lista")
                                    for p in paths)):
        raise ValueError("required_detail")
    tls_context(endpoint["ca_file"], endpoint.get("client_certificate_file"),
                endpoint.get("client_key_file"))


def load_config(value):
    cfg = private_json(value)
    exact_keys(cfg, CONFIG_KEYS)
    digest(cfg["fuente_commit"], 40)
    for file_key, hash_key, limit in (("binario_file", "binario_sha256", 256 * 1024 * 1024),
                                      ("evidencia_h6_file", "evidencia_h6_sha256", 1024 * 1024)):
        expected = digest(cfg[hash_key])
        actual = hashlib.sha256(read_file(cfg[file_key], private=file_key == "evidencia_h6_file",
                                          limit=limit)).hexdigest()
        if actual != expected:
            raise ValueError("provenance")
    if type(cfg["rondas"]) is not int or not 2 <= cfg["rondas"] <= 5:
        raise ValueError("rounds")
    validate_endpoint(cfg["publico"], internal=False)
    validate_endpoint(cfg["interno"], internal=True)
    if (loopback(cfg["publico"]["address"]), cfg["publico"]["port"]) == (
            loopback(cfg["interno"]["address"]), cfg["interno"]["port"]):
        raise ValueError("shared_listener")
    return cfg


class LoopbackHTTPS(http.client.HTTPSConnection):
    def __init__(self, endpoint, context):
        super().__init__(endpoint["server_name"], endpoint["port"], timeout=TIMEOUT, context=context)
        self.address = loopback(endpoint["address"])

    def connect(self):
        raw = socket.create_connection((self.address, self.port), timeout=self.timeout)
        try:
            self.sock = self._context.wrap_socket(raw, server_hostname=self.host)
        except BaseException:
            raw.close()
            raise


def fetch(endpoint, route, *, headers=None, client=True):
    context = tls_context(endpoint["ca_file"],
                          endpoint.get("client_certificate_file") if client else None,
                          endpoint.get("client_key_file") if client else None)
    connection = LoopbackHTTPS(endpoint, context)
    try:
        # A literal loopback connection, a verified SNI name and an explicit Host.
        connection.request("GET", path(route), headers={"Host": endpoint["server_name"],
                           "Accept-Encoding": "identity", **(headers or {})})
        response = connection.getresponse()
        if response.getheader("Set-Cookie") is not None or response.getheader("Location") is not None:
            raise ValueError("cookie_or_redirect")
        if 300 <= response.status < 400:
            raise ValueError("redirect")
        if response.getheader("Trailer") is not None:
            raise ValueError("trailer")
        body = response.read(MAX_BODY + 1)
        if len(body) > MAX_BODY:
            raise ValueError("body_size")
        return response.status, hashlib.sha256(body).hexdigest()
    finally:
        connection.close()


def probe(endpoint, *, internal, barrier):
    barrier.wait(timeout=TIMEOUT)
    snapshot = {}
    for check in endpoint["checks"]:
        status, body_hash = fetch(endpoint, check["path"])
        if status != 200 or body_hash != check["sha256"]:
            raise ValueError("response_mismatch")
        snapshot[check["path"]] = body_hash
    if not internal:
        for route in PRIVATE_ROUTES:
            if fetch(endpoint, route)[0] != 404:
                raise ValueError("private_route")
        for header in ("Cookie", "Authorization", "Proxy-Authorization", "X-Vec-Actor",
                       "X-Auth-User", "X-Remote-User", "Forwarded", "X-Forwarded-For"):
            if fetch(endpoint, "/", headers={header: "h7-synthetic-probe"})[0] != 400:
                raise ValueError("credential_laundering")
    return snapshot


def anonymous_internal_denied(endpoint):
    try:
        status, _ = fetch(endpoint, "/portal-empleado/", client=False)
    except ConnectionResetError:
        # TLS 1.3 may finish client-side before the server rejects the missing
        # certificate. The authenticated probe is checked again immediately after.
        return
    except ssl.SSLError as error:
        if error.reason in ("TLSV13_ALERT_CERTIFICATE_REQUIRED", "SSLV3_ALERT_HANDSHAKE_FAILURE"):
            return
        raise
    else:
        if status not in (401, 403):
            raise ValueError("anonymous_internal_access")


def check(cfg):
    baseline = None
    with ThreadPoolExecutor(max_workers=2) as pool:
        for _ in range(cfg["rondas"]):
            barrier = threading.Barrier(2)
            public = pool.submit(probe, cfg["publico"], internal=False, barrier=barrier)
            internal = pool.submit(probe, cfg["interno"], internal=True, barrier=barrier)
            current = {"publico": public.result(), "interno": internal.result()}
            if baseline is not None and current != baseline:
                raise ValueError("continuity")
            baseline = current
        anonymous_internal_denied(cfg["interno"])
        # Recheck after the denial probe; no restart or process manipulation.
        status, body_hash = fetch(cfg["interno"], "/portal-empleado/")
        if (status, body_hash) != (200, baseline["interno"]["/portal-empleado/"]):
            raise ValueError("continuity")
    return {"result": "checks_passed", "rounds": cfg["rondas"],
            "snapshot_sha256": hashlib.sha256(json.dumps(baseline, sort_keys=True).encode()).hexdigest(),
            "fuente_commit": cfg["fuente_commit"], "binario_sha256": cfg["binario_sha256"],
            "evidencia_h6_sha256": cfg["evidencia_h6_sha256"],
            "h6_ready_accredited": False, "restart_tested": False, "writes": False}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True)
    args = parser.parse_args()
    try:
        print(json.dumps(check(load_config(args.config)), sort_keys=True))
    except (OSError, ValueError, TypeError, KeyError, ssl.SSLError,
            http.client.HTTPException, threading.BrokenBarrierError):
        print("coexistence_check_rejected", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
