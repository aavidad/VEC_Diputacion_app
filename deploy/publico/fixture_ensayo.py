#!/usr/bin/env python3
"""Reutiliza material sintético existente; escribe sólo en el scratch del ensayo."""
import json
import hashlib
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import sys


def prepare(root, scratch):
    text = (root / "deploy/postgresql/bolsa_publica/probar_integracion.sh").read_text()
    matches = re.findall(r"\$proyeccion\$\n(.*?)\n\$proyeccion\$::jsonb", text, re.S)
    if len(matches) != 1:
        raise ValueError("fixture")
    projection = json.loads(matches[0])
    categories = projection["categorias"]
    snapshots = [{
        "huella_gobernada_sha256": s["huella_gobernada_sha256"],
        "huella_proyeccion_sha256": s["huella_proyeccion_publica_sha256"],
        "catalogo": {"esquema": "vec.bolsa.catalogo-categorias-publico.v1",
                     "catalogo_id": s["catalogo_id"], "version": s["version"],
                     "categorias": s["categorias"]},
    } for s in categories["snapshots"]]
    manifest = {
        "esquema": "vec.bolsa.manifiesto-publico.canonico.v2",
        "fuente": projection["fuente"], "catalogos": projection["catalogos"],
        "categorias": {"actual": categories["actual"], "snapshots": snapshots},
        "convocatorias": [{"identificador_publico": c["identificador_publico"],
                           "huella_completa_sha256": c["huella_publica_sha256"],
                           "huella_resumen_sha256": c["huella_resumen_publico_sha256"]}
                          for c in projection["convocatorias"]],
    }
    bolsa = {"generado_en": projection["fuente"]["actualizada_en"], "bolsas": [{
        "bolsa_ref": "bolsa:auxiliares:2026", "categoria": "Auxiliar administrativo",
        "categoria_clave": "auxiliar-administrativo", "grupos": ["C2"],
        "tipo_lista": "definitiva", "vigente_desde": "2026-07-22T10:00:00Z",
        "vigente_hasta": None, "total": 1,
        "posiciones": [{"orden": 1, "documento_enmascarado": "***1234**", "estado_clave": "disponible"}],
    }]}
    for name, data in (("projection.json", projection), ("manifest.json", manifest), ("bolsas.json", bolsa)):
        path = scratch / name
        path.write_text(json.dumps(data, ensure_ascii=False))
        path.chmod(0o600)


def write_private(path, text):
    path.write_text(text)
    path.chmod(0o600)


def configure(scratch, port, system, binary):
    for name in ("reader.pass", "publisher.pass"):
        write_private(scratch / name, secrets.token_urlsafe(36))
    write_private(scratch / "service", "[publico_dba]\n"
                  "host=localhost\nport=5432\nuser=postgres\n"
                  "dbname=vec_bolsa_publica_ensayo\nsslmode=verify-full\nrequire_auth=scram-sha-256\n"
                  "sslrootcert=" + str(scratch / "ca.crt") + "\n")
    write_private(scratch / "pgpass", "localhost:5432:vec_bolsa_publica_ensayo:postgres:"
                  + (scratch / "admin.pass").read_text().strip() + "\n")
    write_private(scratch / "install.json", json.dumps({
        "service_file": str(scratch / "service"), "pass_file": str(scratch / "pgpass"),
        "service": "publico_dba", "database": "vec_bolsa_publica_ensayo",
        "system_identifier": system, "reader_password_file": str(scratch / "reader.pass"),
        "publisher_password_file": str(scratch / "publisher.pass"),
    }))
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        http_port = sock.getsockname()[1]
    with Path(binary).open("rb") as stream:
        digest = hashlib.file_digest(stream, "sha256").hexdigest()
    env = {
        "VEC_HTTP_ADDR": "127.0.0.1:" + str(http_port),
        "VEC_TLS_CERT_FILE": str(scratch / "server.crt"),
        "VEC_TLS_KEY_FILE": str(scratch / "server.key"),
        "VEC_BOLSA_PUBLICA_DATABASE_URL": "postgres://vec_publico_login:"
          + (scratch / "reader.pass").read_text().strip()
          + "@127.0.0.1:" + str(port) + "/vec_bolsa_publica_ensayo?sslmode=verify-full&sslrootcert="
          + str(scratch / "ca.crt"),
        "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256": "pending",
        "VEC_BOLSA_CATEGORIES_CATALOG_ID": "categorias-profesionales",
        "VEC_BOLSA_CATEGORIES_CATALOG_VERSION": "2",
        "VEC_BOLSA_CATEGORIES_CATALOG_SHA256": "b" * 64,
        "VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256": "b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593",
    }
    write_private(scratch / "runtime.json", json.dumps({
        "binary": binary, "binary_sha256": digest, "environment": env,
        "web_sha256": json.loads((Path(binary).parent / "artefacto.json").read_text())["web_sha256"],
        "check": {"origin": "https://127.0.0.1:" + str(http_port),
                  "ca_file": str(scratch / "ca.crt"), "convocatoria": "auxiliares-2026",
                  "bolsa": "bolsa:auxiliares:2026"},
    }))


def publish(scratch, port, binary):
    dsn = "postgres://vec_bolsa_publica_publicador_login:"
    dsn += (scratch / "publisher.pass").read_text().strip()
    dsn += "@127.0.0.1:" + str(port) + "/vec_bolsa_publica_ensayo?sslmode=verify-full&sslrootcert="
    dsn += str(scratch / "ca.crt")
    result = subprocess.run([binary, "publicar-proyeccion-publica",
        "--proyeccion-v2", str(scratch / "projection.json"),
        "--manifiesto-v2", str(scratch / "manifest.json"),
        "--bolsas-v1", str(scratch / "bolsas.json")],
        env={"VEC_EXECUTION_PROFILE": "produccion", "VEC_BOLSA_PUBLICA_DATABASE_URL": dsn},
        capture_output=True, text=True, timeout=120)
    match = re.fullmatch(r"publicacion_publica ancla=([a-f0-9]{64})\n", result.stdout)
    if result.returncode or not match:
        raise ValueError("publication rejected")
    config_path = scratch / "runtime.json"
    config = json.loads(config_path.read_text())
    config["environment"]["VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256"] = match[1]
    write_private(config_path, json.dumps(config))
    print(result.stdout, end="")


if __name__ == "__main__":
    action, scratch = sys.argv[1], Path(sys.argv[2])
    if action == "fixture":
        prepare(Path(__file__).resolve().parents[2], scratch)
    elif action == "configure":
        configure(scratch, int(sys.argv[3]), sys.argv[4], sys.argv[5])
    elif action == "publish":
        publish(scratch, int(sys.argv[3]), sys.argv[4])
    elif action == "diagnostic":
        text = (scratch / "process.log").read_text()[-2000:]
        for name in ("reader.pass", "publisher.pass", "admin.pass"):
            text = text.replace((scratch / name).read_text().strip(), "[redacted]")
        print(text.replace(str(scratch), "[scratch]"), file=sys.stderr)
    elif action == "assert_no_secrets":
        text = (scratch / "postgresql.log").read_text()
        if any((scratch / name).read_text().strip() in text
               for name in ("reader.pass", "publisher.pass", "admin.pass")):
            raise ValueError("database_log_contains_secret")
    else:
        raise ValueError("action")
