#!/usr/bin/env python3
"""Arranque y comprobación TLS de la raíz pública existente."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import ssl
import sys
import urllib.error
import urllib.parse
import urllib.request

from aprovisionar import LOGIN, private_file

REQUIRED = {
    "VEC_HTTP_ADDR", "VEC_TLS_CERT_FILE", "VEC_TLS_KEY_FILE",
    "VEC_BOLSA_PUBLICA_DATABASE_URL", "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256",
    "VEC_BOLSA_CATEGORIES_CATALOG_ID", "VEC_BOLSA_CATEGORIES_CATALOG_VERSION",
    "VEC_BOLSA_CATEGORIES_CATALOG_SHA256",
    "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256",
}


def load(path):
    cfg = json.loads(private_file(path))
    if set(cfg) != {"binary", "binary_sha256", "web_sha256", "environment", "check"}:
        raise ValueError("config")
    binary = Path(cfg["binary"])
    if not binary.is_absolute() or binary.is_symlink() or not binary.is_file():
        raise ValueError("binary")
    if not re.fullmatch(r"[a-f0-9]{64}", cfg["binary_sha256"]):
        raise ValueError("binary checksum")
    with binary.open("rb") as stream:
        digest = hashlib.file_digest(stream, "sha256").hexdigest()
    if digest != cfg["binary_sha256"]:
        raise ValueError("binary checksum")
    web = binary.parent / "web"
    inventory = cfg["web_sha256"]
    if not isinstance(inventory, dict) or "produccion.manifest" not in inventory:
        raise ValueError("web inventory")
    for name, expected in inventory.items():
        if (name != "produccion.manifest" and not name.startswith("static/")) or ".." in name or "\\" in name:
            raise ValueError("web path")
        resource = web / name
        if resource.is_symlink() or hashlib.sha256(resource.read_bytes()).hexdigest() != expected:
            raise ValueError("web checksum")
    if set(inventory) != {"produccion.manifest", *(web / "produccion.manifest").read_text().splitlines()}:
        raise ValueError("web manifest")
    if set(inventory) != {str(p.relative_to(web)) for p in web.rglob("*") if p.is_file()}:
        raise ValueError("unexpected web resource")
    env = cfg["environment"]
    if set(env) != REQUIRED or any(not isinstance(v, str) or not v or "\x00" in v for v in env.values()):
        raise ValueError("environment")
    for key in REQUIRED:
        if key.endswith("SHA256") and (not re.fullmatch(r"[a-f0-9]{64}", env[key]) or env[key] == "0" * 64):
            raise ValueError("anchor")
    if not re.fullmatch(r"[1-9][0-9]*", env["VEC_BOLSA_CATEGORIES_CATALOG_VERSION"]):
        raise ValueError("catalog version")
    dsn = urllib.parse.urlsplit(env["VEC_BOLSA_PUBLICA_DATABASE_URL"])
    # parse_qs omite por defecto los valores vacíos: una segunda opción vacía
    # podría pasar esta lista positiva y llegar de otra forma al cliente SQL.
    query = urllib.parse.parse_qsl(dsn.query, keep_blank_values=True, strict_parsing=True)
    options = dict(query)
    if (dsn.scheme not in ("postgres", "postgresql") or dsn.username != LOGIN
            or not dsn.hostname or not dsn.password or dsn.fragment
            or not re.fullmatch(r"/vec_bolsa_publica[a-z0-9_]*", dsn.path)
            or len(query) != 2 or len(options) != 2 or any(not value for _, value in query)
            or set(options) != {"sslmode", "sslrootcert"}
            or options["sslmode"] != "verify-full"
            or not Path(options["sslrootcert"]).is_absolute()):
        raise ValueError("database TLS")
    private_file(env["VEC_TLS_KEY_FILE"])
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    context.load_cert_chain(env["VEC_TLS_CERT_FILE"], env["VEC_TLS_KEY_FILE"])
    env = dict(env, VEC_EXECUTION_PROFILE="produccion", VEC_AUTH_MODE="disabled")
    return cfg, env


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def check(cfg):
    options = cfg["check"]
    if set(options) != {"origin", "ca_file", "convocatoria", "bolsa"}:
        raise ValueError("check")
    origin = urllib.parse.urlsplit(options["origin"])
    if (origin.scheme != "https" or not origin.hostname or origin.username or origin.password
            or origin.path not in ("", "/") or origin.query or origin.fragment):
        raise ValueError("HTTP TLS")
    for key in ("convocatoria", "bolsa"):
        if not re.fullmatch(r"[a-zA-Z0-9:_-]{1,160}", options[key]):
            raise ValueError("public reference")
    context = ssl.create_default_context(cafile=options["ca_file"])
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect(),
                                        urllib.request.HTTPSHandler(context=context))
    routes = {
        "/bolsa/": 200,
        "/readyz": 200,
        "/api/publico/bolsa/convocatorias": 200,
        "/api/publico/bolsa/convocatorias/" + options["convocatoria"]: 200,
        "/api/publico/bolsa/bolsas": 200,
        "/api/publico/bolsa/bolsas/" + options["bolsa"] + "/lista": 200,
        "/api/vec/contratacion-temporal/solicitudes": 404,
        "/api/vec/bolsa/bolsas": 404,
        "/portal-empleado/": 404,
        "/admin/": 404,
        "/rrhh/": 404,
    }
    snapshot = {}
    for route, expected in routes.items():
        try:
            response = opener.open(options["origin"].rstrip("/") + route, timeout=10)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            body = response.read(4 * 1024 * 1024 + 1)
            print(json.dumps({"route": route, "status": response.code}, sort_keys=True))
            if (response.code != expected or len(body) > 4 * 1024 * 1024
                    or response.headers.get("Set-Cookie") is not None):
                raise ValueError("HTTP response")
            if expected == 200 and route.startswith("/api/"):
                json.loads(body)
                snapshot[route] = hashlib.sha256(body).hexdigest()
    print(json.dumps({"snapshot_sha256": hashlib.sha256(
        json.dumps(snapshot, sort_keys=True).encode()).hexdigest()}, sort_keys=True))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("start", "check"))
    parser.add_argument("--config", required=True)
    args = parser.parse_args()
    try:
        cfg, env = load(args.config)
        if args.action == "start":
            # El proceso sólo recibe transporte, lector público y anclas externas.
            os.chdir(Path(cfg["binary"]).parent)
            os.execve(cfg["binary"], [cfg["binary"]], env)
        check(cfg)
    except (OSError, ValueError, KeyError, TypeError, ssl.SSLError):
        print("runtime_publico_rechazado", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
