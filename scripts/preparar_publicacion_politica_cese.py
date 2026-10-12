#!/usr/bin/env python3
"""Prepara SQL revisable para publicar una nueva política B45 desde datos JSON."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from pathlib import Path


RAIZ = Path(__file__).resolve().parents[1]
CATALOGO = RAIZ / "data/catalogos/bolsa/politica_cese_ejemplo_v2.json"
CLAVE = re.compile(r"^[a-z][a-z0-9._-]{1,79}(\|[a-z][a-z0-9._-]{1,79})?$")
REFERENCIA = re.compile(r"^[A-Za-z0-9:._/-]{1,512}$")
POLITICA = {"version", "catalogo_ref", "mapeo", "meses_general", "meses_acumulacion", "computo", "estado"}
COMPUTO_B45 = "fecha_cese_meses_calendario_ajuste_fin_mes"


def _sin_duplicados(pares: list[tuple[str, object]]) -> dict[str, object]:
    resultado = {}
    for clave, valor in pares:
        if clave in resultado:
            raise ValueError(f"clave JSON duplicada: {clave}")
        resultado[clave] = valor
    return resultado


def _validar_politica(politica: object, etiqueta: str) -> dict[str, object]:
    if not isinstance(politica, dict) or set(politica) != POLITICA:
        raise ValueError(f"{etiqueta}: claves esperadas {sorted(POLITICA)}, observadas {sorted(politica) if isinstance(politica, dict) else type(politica).__name__}")
    version = politica["version"]
    referencia = politica["catalogo_ref"]
    mapeo = politica["mapeo"]
    if type(version) is not int or version < 1 or not isinstance(referencia, str) or not REFERENCIA.fullmatch(referencia):
        raise ValueError(f"{etiqueta}: versión/referencia inválidas: {version!r}/{referencia!r}")
    if not isinstance(mapeo, dict) or not 1 <= len(mapeo) <= 100 or any(
        not isinstance(k, str) or not CLAVE.fullmatch(k)
        or not isinstance(v, str) or v not in {"general", "acumulacion_tareas"}
        for k, v in mapeo.items()
    ):
        raise ValueError(f"{etiqueta}: mapeo inválido; entradas observadas {len(mapeo) if isinstance(mapeo, dict) else type(mapeo).__name__}")
    for clave in ("meses_general", "meses_acumulacion"):
        valor = politica[clave]
        if type(valor) is not int or not 0 <= valor <= 120:
            raise ValueError(f"{etiqueta}: {clave} inválido: {valor!r}")
    if politica["computo"] != COMPUTO_B45 or politica["estado"] != "ejemplo_sintetico":
        raise ValueError(f"{etiqueta}: cómputo/estado incompatibles: {politica['computo']!r}/{politica['estado']!r}")
    return politica


def leer_corte(ruta: Path) -> tuple[dict[str, object], dict[str, object], str]:
    bruto = ruta.read_bytes()
    if len(bruto) > 32768:
        raise ValueError(f"catálogo excesivo: límite 32768, observado {len(bruto)}")
    datos = json.loads(bruto, object_pairs_hook=_sin_duplicados)
    if not isinstance(datos, dict) or set(datos) != {"esquema", "preimagen", "nueva"} or \
            datos["esquema"] != "vec.bolsa.politica-cese.publicacion.v1":
        raise ValueError("esquema/claves del corte no coinciden")
    pre = _validar_politica(datos["preimagen"], "preimagen")
    nueva = _validar_politica(datos["nueva"], "nueva")
    prefijo = pre["catalogo_ref"].removesuffix(f":v{pre['version']}")
    if (prefijo == pre["catalogo_ref"] or nueva["version"] != pre["version"] + 1
            or nueva["catalogo_ref"] != f"{prefijo}:v{nueva['version']}"):
        raise ValueError(f"versión/referencia: esperado {pre['version'] + 1}/nueva, observado {nueva['version']}/{nueva['catalogo_ref']}")
    if any(nueva[k] != pre[k] for k in ("meses_general", "meses_acumulacion", "computo", "estado")):
        raise ValueError("meses/cómputo/estado deben conservar exactamente la preimagen")
    anteriores = pre["mapeo"]
    posteriores = nueva["mapeo"]
    if len(posteriores) != len(anteriores) + 1 or any(posteriores.get(k) != v for k, v in anteriores.items()):
        raise ValueError(f"mapeo: esperado una clave añadida sin reescrituras; observadas {len(posteriores) - len(anteriores)}")
    return pre, nueva, hashlib.sha256(bruto).hexdigest()


def _q(valor: str) -> str:
    return "'" + valor.replace("'", "''") + "'"


def _mapa(politica: dict[str, object]) -> str:
    return _q(json.dumps(politica["mapeo"], ensure_ascii=False, sort_keys=True, separators=(",", ":"))) + "::jsonb"


def _sha_sql(mapa: str, politica: dict[str, object]) -> str:
    return "encode(sha256(convert_to(jsonb_build_object(" + \
        "'mapeo'," + mapa + \
        ",'meses_general'," + str(politica["meses_general"]) + \
        ",'meses_acumulacion'," + str(politica["meses_acumulacion"]) + \
        ",'computo'," + _q(politica["computo"]) + \
        ",'estado'," + _q(politica["estado"]) + ")::text,'UTF8')),'hex')"


def preparar_sql(pre: dict[str, object], nueva: dict[str, object], huella_archivo: str) -> str:
    mapa_pre, mapa_nuevo = _mapa(pre), _mapa(nueva)
    sha_pre, sha_nuevo = _sha_sql(mapa_pre, pre), _sha_sql(mapa_nuevo, nueva)
    return f"""\\set ON_ERROR_STOP on
-- Fuente única: data/catalogos/bolsa/politica_cese_ejemplo_v2.json
-- SHA256 del catálogo fuente: {huella_archivo}
-- Publicación controlada B45: solo dirección, en sesión DBA autorizada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '20s';
SELECT pg_advisory_xact_lock(hashtextextended('bolsa:politica-cese',0));
DO $pre$
DECLARE v vec_bolsa_llamamientos.politica_cese_bolsa;
BEGIN
 SELECT * INTO v FROM vec_bolsa_llamamientos.politica_cese_bolsa ORDER BY version DESC LIMIT 1;
 IF v.version={pre['version']} AND v.catalogo_ref={_q(pre['catalogo_ref'])}
    AND v.catalogo_sha256={sha_pre} AND v.mapeo={mapa_pre}
    AND v.meses_general={pre['meses_general']} AND v.meses_acumulacion={pre['meses_acumulacion']}
    AND v.computo={_q(pre['computo'])} AND v.estado={_q(pre['estado'])} THEN
  RETURN;
 END IF;
 IF v.version={nueva['version']} AND v.catalogo_ref={_q(nueva['catalogo_ref'])}
    AND v.catalogo_sha256={sha_nuevo} AND v.mapeo={mapa_nuevo}
    AND v.meses_general={nueva['meses_general']} AND v.meses_acumulacion={nueva['meses_acumulacion']}
    AND v.computo={_q(nueva['computo'])} AND v.estado={_q(nueva['estado'])} THEN
  RETURN; -- replay de una publicación ya confirmada.
 END IF;
 RAISE EXCEPTION 'politica_cese_version/catalogo_ref esperado %/% o %/%, observado %/%',
   {pre['version']},{_q(pre['catalogo_ref'])},{nueva['version']},{_q(nueva['catalogo_ref'])},
   coalesce(v.version::text,'ausente'),coalesce(v.catalogo_ref,'ausente') USING ERRCODE='55000';
END $pre$;
SELECT version,reutilizada,catalogo_sha256
FROM vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(
 {_q(nueva['catalogo_ref'])},{mapa_nuevo},{nueva['meses_general']},{nueva['meses_acumulacion']},{_q(nueva['estado'])});
DO $post$
DECLARE v1 vec_bolsa_llamamientos.politica_cese_bolsa;
        v2 vec_bolsa_llamamientos.politica_cese_bolsa;
BEGIN
 SELECT * INTO v1 FROM vec_bolsa_llamamientos.politica_cese_bolsa WHERE version={pre['version']};
 SELECT * INTO v2 FROM vec_bolsa_llamamientos.politica_cese_bolsa WHERE version={nueva['version']};
 IF v1.catalogo_ref IS DISTINCT FROM {_q(pre['catalogo_ref'])} OR v1.mapeo IS DISTINCT FROM {mapa_pre}
    OR v1.catalogo_sha256 IS DISTINCT FROM {sha_pre} THEN
  RAISE EXCEPTION 'politica_cese_v1 esperado %/%, observado %/%',
   {_q(pre['catalogo_ref'])},{sha_pre},coalesce(v1.catalogo_ref,'ausente'),coalesce(v1.catalogo_sha256,'ausente') USING ERRCODE='55000';
 END IF;
 IF v2.catalogo_ref IS DISTINCT FROM {_q(nueva['catalogo_ref'])} OR v2.mapeo IS DISTINCT FROM {mapa_nuevo}
    OR v2.meses_general IS DISTINCT FROM {nueva['meses_general']}
    OR v2.meses_acumulacion IS DISTINCT FROM {nueva['meses_acumulacion']}
    OR v2.computo IS DISTINCT FROM {_q(nueva['computo'])} OR v2.estado IS DISTINCT FROM {_q(nueva['estado'])}
    OR v2.catalogo_sha256 IS DISTINCT FROM {sha_nuevo} THEN
  RAISE EXCEPTION 'politica_cese_v2 esperado %/%, observado %/%',
   {_q(nueva['catalogo_ref'])},{sha_nuevo},coalesce(v2.catalogo_ref,'ausente'),coalesce(v2.catalogo_sha256,'ausente') USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
"""


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--catalogo", type=Path, default=CATALOGO)
    parser.add_argument("--salida", type=Path, help="archivo SQL a escribir; por defecto salida estándar")
    args = parser.parse_args(argv)
    pre, nueva, huella = leer_corte(args.catalogo)
    sql = preparar_sql(pre, nueva, huella)
    if args.salida:
        args.salida.write_text(sql, encoding="utf-8")
    else:
        sys.stdout.write(sql)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (ValueError, OSError, json.JSONDecodeError) as exc:
        print(f"No se pudo preparar la publicación: {exc}", file=sys.stderr)
        raise SystemExit(2) from exc
