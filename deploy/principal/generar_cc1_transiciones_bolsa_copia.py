#!/usr/bin/env python3
"""Prepara la publicación CC1 ficticia de límites S6; no conecta a PostgreSQL."""

import argparse
from pathlib import Path
import re

from inscripcion_recorrido_20261013.catalogos.generar_carga_cc1 import canonico, huella, literal

PARES = (
    "disponible>pendiente_incorporacion", "disponible>renuncia", "disponible>excluido",
    "no_disponible>excluido", "trabajando>disponible",
    "trabajando>excluido", "trabajando>disponible_desde",
    "pendiente_incorporacion>disponible", "pendiente_incorporacion>trabajando",
    "pendiente_incorporacion>renuncia", "pendiente_incorporacion>excluido",
    "renuncia>excluido",
    "disponible_desde>disponible", "disponible_desde>excluido",
)
APROBACIONES = ("dato:copia:s6:aprobacion-sintetica-a", "dato:copia:s6:aprobacion-sintetica-b")
RECIBO = "dato:copia:s6:sin-recibo-gobernado"

def preparar(salida: Path, sistema: str, puerto: int, unidad: str, ambito: str) -> tuple[str, str]:
    patron = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$")
    if not re.fullmatch(r"[0-9]{15,20}", sistema) or not 1 <= puerto <= 65535 or not patron.fullmatch(unidad) or not patron.fullmatch(ambito):
        raise ValueError("destino o ámbito de copia incompatibles")
    entrada = {"clave": "bolsa.transiciones_situacion.limites", "etiqueta": "Cambios de situación",
               "unidad_ref": unidad, "ambito_ref": ambito, "transiciones": list(PARES)}
    documento = {"id": "bolsa.transiciones_situacion", "version": 1,
                 "estado": "publicado", "entradas": [entrada]}
    canon = canonico(documento)
    digest = huella(canon)
    if (unidad, ambito) == ("unidad:desarrollo:rrhh", "ambito:desarrollo:bolsa") and digest != "ae3545d1b5ec2d0069cac5c29ba3f6aa23f79308e5b2ed54258ecf0d39f21a44":
        raise ValueError("huella del catálogo B32 distinta")
    previo = ("v_previa.huella_sha256 IS DISTINCT FROM " + literal(digest) +
              " OR v_previa.documento_canonico IS DISTINCT FROM " + literal(canon) +
              " OR v_previa.preimagenes_control IS DISTINCT FROM '{}'::jsonb" +
              " OR v_previa.preimagenes_huella_sha256 IS DISTINCT FROM " + literal(huella("{}")) +
              " OR v_previa.aprobacion_a_ref IS DISTINCT FROM " + literal(APROBACIONES[0]) +
              " OR v_previa.aprobacion_b_ref IS DISTINCT FROM " + literal(APROBACIONES[1]) +
              " OR v_previa.actor_ref IS DISTINCT FROM 'dato:copia:s6:sin-acto'" +
              " OR v_previa.decision_ref IS DISTINCT FROM 'dato:copia:s6:sin-decision'" +
              " OR v_previa.recibo_ref IS DISTINCT FROM " + literal(RECIBO))
    guardia = ("(pg_control_system()).system_identifier::text IS DISTINCT FROM " + literal(sistema) +
               " OR current_setting('port') IS DISTINCT FROM " + literal(str(puerto)) +
               " OR current_database() IS DISTINCT FROM 'postgres'")
    sql = "\n".join([
        r"\set ON_ERROR_STOP on", "-- Dato ficticio de copia; dos aprobaciones declaradas, sin acto administrativo.",
        "BEGIN ISOLATION LEVEL SERIALIZABLE;", "SET LOCAL search_path=pg_catalog,pg_temp;",
        "SET LOCAL timezone='UTC';", "SET LOCAL lock_timeout='5s';", "SET LOCAL statement_timeout='30s';",
        "DO $cc1$ DECLARE v_previa record; BEGIN",
        " IF " + guardia + " OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999",
        " OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN",
        "  RAISE EXCEPTION 'S6 CC1: copia incompatible' USING ERRCODE='55000'; END IF;",
        " PERFORM pg_advisory_xact_lock(hashtextextended('vec_catalogos_configurables:bolsa.transiciones_situacion',0));",
        " SELECT * INTO v_previa FROM vec_catalogos_configurables.publicacion WHERE catalogo_id='bolsa.transiciones_situacion' AND version=1;",
        " IF FOUND THEN IF " + previo + " THEN RAISE EXCEPTION 'S6 CC1: publicación divergente' USING ERRCODE='55000'; END IF;",
        " ELSE PERFORM vec_catalogos_configurables.publicar('bolsa.transiciones_situacion',1," + literal(digest) + "," + literal(canon) + ",",
        " '{}'::jsonb," + literal(huella("{}")) + "," + literal(APROBACIONES[0]) + "," + literal(APROBACIONES[1]) + ",",
        " 'dato:copia:s6:sin-acto','dato:copia:s6:sin-decision'," + literal(RECIBO) + ",'datos_copia:1:publicacion'); END IF;",
        " IF " + guardia + " OR NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.publicacion p",
        " JOIN vec_catalogos_configurables.entrada_publicada e USING(catalogo_id,version,huella_sha256)",
        " JOIN vec_catalogos_configurables.categoria_control c ON c.categoria_id=e.categoria_id",
        " WHERE p.catalogo_id='bolsa.transiciones_situacion' AND p.version=1 AND p.huella_sha256=" + literal(digest),
        " AND e.categoria_id='bolsa.transiciones_situacion.limites' AND e.definicion=" + literal(canonico(entrada)) + "::jsonb",
        " AND c.catalogo_id=p.catalogo_id AND c.version=1 AND c.huella_sha256=p.huella_sha256 AND c.estado='habilitada') THEN",
        "  RAISE EXCEPTION 'S6 CC1: publicación incompleta' USING ERRCODE='55000'; END IF;",
        "END $cc1$;", "COMMIT;", "",
    ])
    if salida.is_symlink():
        raise ValueError("salida enlazada")
    salida.mkdir(parents=True, exist_ok=True, mode=0o700)
    if salida.stat().st_mode & 0o077:
        raise ValueError("salida accesible a otros usuarios")
    archivos = {"carga_cc1_transiciones_bolsa.sql": sql,
                "manifiesto_cc1_transiciones_bolsa.json": canonico({
                    "uso": "dato_ficticio_copia", "responsable": "Codex-S",
                    "retirada": "productor gobernado del catálogo normativo",
                    "catalogo_id": documento["id"], "version": 1, "catalogo_sha256": digest,
                    "sql_sha256": huella(sql), "aprobaciones_declaradas": list(APROBACIONES),
                    "unidad_ref": unidad, "ambito_ref": ambito, "sistema_id": sistema, "puerto": puerto}) + "\n"}
    for nombre, contenido in archivos.items():
        destino = salida / nombre
        if destino.is_symlink() or destino.exists() and destino.read_text(encoding="utf-8") != contenido:
            raise ValueError("artefacto existente divergente: " + nombre)
        destino.write_text(contenido, encoding="utf-8")
        destino.chmod(0o600)
    return digest, huella(sql)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Genera dato CC1 S6 de copia, sin conexión")
    parser.add_argument("salida", type=Path)
    parser.add_argument("--sistema-id", required=True)
    parser.add_argument("--puerto", required=True, type=int)
    parser.add_argument("--unidad-ref", default="unidad:desarrollo:rrhh")
    parser.add_argument("--ambito-ref", default="ambito:desarrollo:bolsa")
    a = parser.parse_args()
    print("catálogo_sha256=%s sql_sha256=%s" % preparar(a.salida, a.sistema_id, a.puerto,
                                                   a.unidad_ref, a.ambito_ref))
