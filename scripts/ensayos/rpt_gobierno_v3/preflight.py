#!/usr/bin/env python3
"""Preflight de solo lectura para el ensayo sintético RPT/V3.

No instala SQL ni emite decisiones. La ausencia de composición ADMIN es una
puerta de parada, no una invitación a simular una identidad HIGH.
"""

from __future__ import annotations

import hashlib
import json
import pathlib
import re
import subprocess
import sys


PREVIAS = "e528c7eaa9a90f1014593521a355746629d74de9"
SQL_GOBIERNO = "e56f8a302f7d271842479f009f97cee7b45a50d1"
CANDIDATA = "8e97e24ab255fcc0824c9c4586d5731e411d1616"
SQL = (
    "deploy/postgresql/catalogos_configurables/migraciones/000004_gobierno_categorias_rpt.up.sql",
    "deploy/postgresql/autorizacion_atestada_v3/migraciones/000134_gobierno_categorias_rpt.up.sql",
)
PLAN = "scripts/recorridos/sql_main_h6.txt"


def git(root: pathlib.Path, *args: str) -> bytes:
    return subprocess.check_output(("git", *args), cwd=root, stderr=subprocess.DEVNULL)


def main() -> int:
    root = pathlib.Path(__file__).resolve().parents[3]
    findings: list[str] = []
    try:
        head = git(root, "rev-parse", "HEAD").decode().strip()
        for ancestor in (CANDIDATA, PREVIAS, SQL_GOBIERNO):
            subprocess.run(("git", "merge-base", "--is-ancestor", ancestor, "HEAD"),
                           cwd=root, check=True, stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL)
        plan = git(root, "show", f"{PREVIAS}:{PLAN}").decode()
        entries = [line for line in plan.splitlines() if line and not line.startswith("#")]
        if len(entries) != 41 or any(len(line.split("\t")) != 3 for line in entries):
            findings.append("El plan de 41 SQL previas no tiene la forma fijada")
        for path in SQL:
            pinned = git(root, "show", f"{SQL_GOBIERNO}:{path}")
            current = (root / path).read_bytes()
            if current != pinned:
                findings.append(f"La SQL cambió respecto de {SQL_GOBIERNO}: {path}")
        source = root / "internal/vec/adapters/httpapi/catalogos_rpt_gobierno.go"
        if "#211 debe crear" not in source.read_text():
            findings.append("El contrato de composición ADMIN cambió: revisar manualmente")
        callers = []
        for base in (root / "cmd", root / "internal"):
            for path in base.rglob("*.go"):
                if path.name.endswith("_test.go") or path == source:
                    continue
                if re.search(r"\bNuevasRutasGobiernoCategoriaRPT\s*\(", path.read_text()):
                    callers.append(str(path.relative_to(root)))
        if not callers:
            findings.append("#211: ningún ensamblaje llama NuevasRutasGobiernoCategoriaRPT")
        result = {"candidata": head, "previas": PREVIAS, "sql_gobierno": SQL_GOBIERNO,
                  "sql_sha256": {p: hashlib.sha256((root / p).read_bytes()).hexdigest() for p in SQL},
                  "plan_previo_sql": len(entries), "llamadores_admin": callers,
                  "estado": "bloqueado" if findings else "preflight_fuente_superado",
                  "hallazgos": findings,
                  "limite": "Sin lectura PostgreSQL, sin decisión V3/COSE y sin recorrido HTTP"}
    except (OSError, subprocess.CalledProcessError, UnicodeError) as exc:
        result = {"estado": "bloqueado", "hallazgos": [f"Fuente no verificable: {type(exc).__name__}"],
                  "limite": "Sin lectura PostgreSQL, sin decisión V3/COSE y sin recorrido HTTP"}
    print(json.dumps(result, ensure_ascii=False, sort_keys=True, indent=2))
    return 0 if result["estado"] == "preflight_fuente_superado" else 2


if __name__ == "__main__":
    sys.exit(main())
