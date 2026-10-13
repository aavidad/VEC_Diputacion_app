#!/usr/bin/env python3
"""Prepara SQL de carga sintética CC1; no abre conexiones."""

import hashlib
import argparse
import base64
import json
import pathlib


def literal(valor: str) -> str:
    return "'" + valor.replace("'", "''") + "'"


def canonico(valor: object) -> str:
    return json.dumps(valor, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def huella(valor: str) -> str:
    return hashlib.sha256(valor.encode("utf-8")).hexdigest()


def opaco(prefijo: str, semilla: str) -> str:
    bytes_ref = hashlib.sha256(semilla.encode("utf-8")).digest()[:16]
    return prefijo + base64.urlsafe_b64encode(bytes_ref).decode("ascii").rstrip("=")


def registro_solicitud(sql: list[str], fuente: dict, convocatoria: dict, version: dict,
                       version_sha: str, catalogo_sha: str, politica_sha: str,
                       unidad_ref: str, ambito_ref: str, motivos_version: int,
                       motivos_sha: str, motivo_rechazo_ref: str) -> dict:
    nombre = fuente["nombre_operativo"]
    semilla = "cc1:inscripcion:20261013:" + nombre
    persona = opaco("per_", semilla + ":persona")
    contexto = opaco("vca_", semilla + ":contexto")
    solicitud = "solicitud_inscripcion_" + huella(persona + "\x1f" + fuente["convocatoria_ref"] +
                                               "\x1f" + convocatoria["categoria_ref"])
    declaracion = "declaracion_inscripcion_" + huella(semilla + ":declaracion")
    plazo = convocatoria["plazo"]
    requisitos = [{"referencia": r["referencia"], "codigo": r["referencia"],
                   "descripcion": r["descripcion"], "obligatorio": r["obligatorio"],
                   "estado": "pendiente", "motivo_codigo": "requisito.pendiente"}
                  for r in convocatoria["requisitos"]]
    req_json = canonico(requisitos)
    # BC9 vincula la huella a los requisitos publicados, no a su proyección pendiente.
    requisito_sha_sql = ("encode(sha256(convert_to(" +
                         literal(canonico(convocatoria["requisitos"])) +
                         "::jsonb::text,'UTF8')),'hex')")
    versiones = 1 if fuente["estado"] == "pendiente" else 2
    sql += [
        "DO $sol$ DECLARE previo record; BEGIN",
        " SELECT convocatoria_ref,categoria_ref,persona_ref,unidad_ref,ambito_ref,version_sha256"
        " INTO previo FROM vec_bolsa_llamamientos.solicitud_inscripcion"
        f" WHERE solicitud_ref={literal(solicitud)};",
        " IF FOUND THEN",
        f"  IF previo.convocatoria_ref IS DISTINCT FROM {literal(fuente['convocatoria_ref'])}"
        f" OR previo.categoria_ref IS DISTINCT FROM {literal(convocatoria['categoria_ref'])}"
        f" OR previo.persona_ref IS DISTINCT FROM {literal(persona)}"
        f" OR previo.unidad_ref IS DISTINCT FROM {literal(unidad_ref)}"
        f" OR previo.ambito_ref IS DISTINCT FROM {literal(ambito_ref)}"
        f" OR previo.version_sha256 IS DISTINCT FROM {literal(version_sha)}"
        " THEN RAISE EXCEPTION 'B96: solicitud existente distinta'; END IF;",
        "  IF (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_version"
        f" WHERE solicitud_ref={literal(solicitud)})<>{versiones}"
        " OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_historia"
        f" WHERE solicitud_ref={literal(solicitud)})<>{versiones}"
        " OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_outbox"
        f" WHERE solicitud_ref={literal(solicitud)})<>{versiones}"
        " OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_recibo"
        f" WHERE solicitud_ref={literal(solicitud)})<>{versiones}"
        " THEN RAISE EXCEPTION 'B96: historia sintética incompleta'; END IF;",
        "  IF (SELECT estado FROM vec_bolsa_llamamientos.solicitud_inscripcion_version"
        f" WHERE solicitud_ref={literal(solicitud)} ORDER BY version DESC LIMIT 1)"
        f" IS DISTINCT FROM {literal(fuente['estado'])}"
        " THEN RAISE EXCEPTION 'B96: estado existente distinto'; END IF;",
        " ELSE",
        "  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion(",
        "   solicitud_ref,persona_ref,contexto_actor_ref,contexto_actor_version,"
        "convocatoria_ref,convocatoria_id,secuencia,version_sha256,identificador_publico,"
        "categoria_ref,bases_ref,catalogo_ref,catalogo_version,catalogo_sha256,"
        "politica_catalogo_ref,politica_catalogo_version,politica_catalogo_sha256,"
        "unidad_ref,ambito_ref,ambito_fuente_ref,ambito_fuente_version,ambito_fuente_sha256,"
        "formulario_ref,formulario_version,formulario_sha256,plazo_ref,plazo_abre_en,plazo_cierra_en,"
        "requisitos,requisitos_sha256,declaraciones,declaracion_ref,clave_sha256,material_sha256,canal,presentada_en) VALUES (",
        f"   {literal(solicitud)},{literal(persona)},{literal(contexto)},1,"
        f"{literal(fuente['convocatoria_ref'])},{literal(convocatoria['id'])},1,{literal(version_sha)},"
        f"{literal(convocatoria['identificador_publico'])},{literal(convocatoria['categoria_ref'])},"
        f"{literal(convocatoria['bases_ref'])},'categorias_rpt',1,{literal(catalogo_sha)},"
        f"'bolsa.politica.inscripcion',1,{literal(politica_sha)},"
        f"{literal(unidad_ref)},{literal(ambito_ref)},"
        f"{literal('ambito.gestion.' + huella(convocatoria['id']) + '.v1')},1,{literal(politica_sha)},"
        f"{literal(version['configuracion']['flujo_solicitud']['id'])},1,"
        f"{literal(version['configuracion']['flujo_solicitud']['huella_contenido_sha256'])},"
        f"{literal(plazo['referencia'])},{literal(plazo['abre_en'])},{literal(plazo['cierra_en'])},"
        f"{literal(req_json)}::jsonb,{requisito_sha_sql},'[]'::jsonb,{literal(declaracion)},"
        f"{literal(huella(semilla + ':clave:1'))},{literal(huella(semilla + ':material:1'))},"
        f"'externa_personal',{literal(fuente['presentada_en'])});",
    ]
    for numero in range(1, versiones + 1):
        estado = "pendiente" if numero == 1 else fuente["estado"]
        actor = persona if numero == 1 else opaco("per_", "cc1:rrhh:revisor")
        perfil = opaco("prf_", semilla + (":perfil-persona" if numero == 1 else ":perfil-rrhh"))
        cuenta = opaco("cta_", semilla + (":cuenta-persona" if numero == 1 else ":cuenta-rrhh"))
        instante = fuente["presentada_en"] if numero == 1 else fuente["decidida_en"]
        evaluacion = requisitos if numero == 1 else [
            {**r, "estado": "cumple" if estado == "admitida_a_convocatoria" else "no_cumple",
             "motivo_codigo": "requisito.cumple" if estado == "admitida_a_convocatoria"
             else motivo_rechazo_ref} for r in requisitos]
        historia = "historia_inscripcion_" + huella(semilla + f":historia:{numero}")
        evento = "evento_inscripcion_" + huella(semilla + f":evento:{numero}")
        recibo = "recibo_inscripcion_" + huella(semilla + f":recibo:{numero}")
        auditoria = "auditoria:cc1:carga:" + huella(semilla + f":auditoria:{numero}")
        decision = "decision:cc1:carga:" + huella(semilla + f":decision:{numero}")
        motivo_sql = literal(motivo_rechazo_ref) if estado == "rechazada" else "NULL"
        motivo_cat_sql = "'bolsa.inscripcion.motivos'" if estado == "rechazada" else "NULL"
        motivo_version_sql = str(motivos_version) if estado == "rechazada" else "NULL"
        motivo_sha_sql = literal(motivos_sha) if estado == "rechazada" else "NULL"
        sql += [
            "  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_version("
            "solicitud_ref,version,estado,actor_ref,perfil_ref,cuenta_ref,motivo_ref,"
            "motivo_catalogo_ref,motivo_catalogo_version,motivo_catalogo_sha256,"
            "evaluacion,decision_ref,consumo_huella_sha256,auditoria_ref,aplicada_en) VALUES ("
            f"{literal(solicitud)},{numero},{literal(estado)},{literal(actor)},{literal(perfil)},"
            f"{literal(cuenta)},{motivo_sql},{motivo_cat_sql},{motivo_version_sql},{motivo_sha_sql},"
            f"{literal(canonico(evaluacion))}::jsonb,{literal(decision)},"
            f"{literal(huella(semilla + f':consumo:{numero}'))},{literal(auditoria)},{literal(instante)});",
            "  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_historia("
            "historia_ref,solicitud_ref,version,accion,actor_ref,estado,unidad_ref,ambito_ref,"
            "ambito_fuente_ref,ambito_fuente_version,ambito_fuente_sha256,registrada_en) VALUES ("
            f"{literal(historia)},{literal(solicitud)},{numero},"
            f"{literal('presentar' if numero == 1 else 'revisar')},{literal(actor)},{literal(estado)},"
            + (f"{literal(unidad_ref)},{literal(ambito_ref)},"
               f"{literal('ambito.gestion.' + huella(convocatoria['id']) + '.v1')},1,"
               f"{literal(politica_sha)}," if numero == 1 else "NULL,NULL,NULL,NULL,NULL,")
            + f"{literal(instante)});",
            "  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_outbox("
            "evento_ref,historia_ref,solicitud_ref,version,esquema,evento,creada_en) VALUES ("
            f"{literal(evento)},{literal(historia)},{literal(solicitud)},{numero},"
            f"{literal('vec.bolsa.inscripcion.presentada.v1' if numero == 1 else 'vec.bolsa.inscripcion.revisada.v1')},"
            f"jsonb_build_object('solicitud_ref',{literal(solicitud)},'estado',{literal(estado)}),"
            f"{literal(instante)});",
            "  INSERT INTO vec_bolsa_llamamientos.solicitud_inscripcion_recibo("
            "recibo_ref,solicitud_ref,version,persona_ref,canal,clave_sha256,material_sha256,"
            "historia_ref,evento_ref,auditoria_ref,emitida_en) VALUES ("
            f"{literal(recibo)},{literal(solicitud)},{numero},{literal(persona)},'externa_personal',"
            f"{literal(huella(semilla + f':clave:{numero}'))},"
            f"{literal(huella(semilla + f':material:{numero}'))},"
            f"{literal(historia)},{literal(evento)},{literal(auditoria)},{literal(instante)});",
        ]
    sql.append(" END IF; END $sol$;")
    return {"solicitud_ref": solicitud, "convocatoria_ref": fuente["convocatoria_ref"],
            "estado": fuente["estado"], "version": versiones}


def preparar(fuente: pathlib.Path, salida: pathlib.Path, unidad_ref: str,
             ambito_ref: str) -> None:
    datos = json.loads(fuente.read_text(encoding="utf-8"))
    unidad_ref = unidad_ref or datos["ambito_actual"]["unidad_ref"]
    ambito_ref = ambito_ref or datos["ambito_actual"]["ambito_ref"]
    if datos.get("esquema") != "vec.inscripcion.recorrido.cc1.v1" or datos.get("uso") != "ensayo_sintetico":
        raise ValueError("fuente CC1 inesperada")
    catalogos = datos["catalogos"]
    convocatorias = datos["convocatorias"]
    categorias_existentes = datos["catalogo_categorias_existente"]
    if len(catalogos) != 3 or len(convocatorias) != 2:
        raise ValueError("se esperan tres catálogos nuevos y dos convocatorias")
    if [c["id"] for c in catalogos] != [
        "bolsa.politica.inscripcion", "bolsa.conjunto.gestion.inscripcion",
        "bolsa.inscripcion.motivos",
    ]:
        raise ValueError("orden o identidad de catálogos incompatibles")
    if (categorias_existentes["id"] != "categorias_rpt"
            or categorias_existentes["version"] != 1
            or len(categorias_existentes["huella_sha256"]) != 64
            or categorias_existentes["categorias"] != [
                "categoria:rpt:administrativo", "categoria:rpt:operario"]):
        raise ValueError("referencia de categorías RPT incompatible")
    import re
    patron = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,127}$")
    if not patron.fullmatch(unidad_ref) or not patron.fullmatch(ambito_ref):
        raise ValueError("unidad o ámbito incompatibles")
    if [e["clave"] for e in catalogos[2]["entradas"]] != ["documentacion.incompleta"]:
        raise ValueError("motivo CC1 de rechazo incompatible")
    for catalogo in catalogos:
        for entrada in catalogo["entradas"]:
            atributos = entrada.get("atributos", {})
            for clave, valor in (("unidad_ref", unidad_ref), ("ambito_ref", ambito_ref)):
                if atributos.get(clave) == "__" + clave.upper() + "__":
                    atributos[clave] = valor
    if not any(e["clave"] == "inscripcion.gestion.rrhh.conjunto"
               for e in catalogos[1]["entradas"]):
        raise ValueError("conjunto RRHH ausente")
    asociacion = [e for e in catalogos[0]["entradas"]
                  if e["clave"] == "asociacion.manual.acta"]
    if len(asociacion) != 1 or asociacion[0].get("atributos") != {
        "valor": "true", "canal": "interna_corporativa",
        "motivo_ref": "asociacion.manual.acta.cc1",
        "alcance_ref": convocatorias[0]["convocatoria_ref"],
    }:
        raise ValueError("política de asociación CC11 incompatible")

    salida.mkdir(parents=True, exist_ok=True)
    sql = [
        r"\set ON_ERROR_STOP on",
        "BEGIN ISOLATION LEVEL SERIALIZABLE;",
        "SET LOCAL search_path=pg_catalog,pg_temp;",
        "SET LOCAL standard_conforming_strings=on;",
        "SET LOCAL TimeZone='UTC';",
        "SET LOCAL lock_timeout='5s';",
        "SET LOCAL statement_timeout='60s';",
        "SELECT pg_advisory_xact_lock(hashtextextended('vec:inscripcion:cc1:ensayo:20261013',0));",
        "SET LOCAL ROLE vec_catalogos_configurables_propietario;",
        "DO $rpt$ BEGIN",
        " IF NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.publicacion p"
        " WHERE p.catalogo_id='categorias_rpt' AND p.version=1"
        f" AND p.huella_sha256={literal(categorias_existentes['huella_sha256'])})"
        " THEN RAISE EXCEPTION 'CC1: categorías RPT ausentes o distintas'; END IF;",
        " IF (SELECT count(*) FROM vec_catalogos_configurables.categoria_control c"
        " WHERE c.catalogo_id='categorias_rpt' AND c.version=1"
        f" AND c.huella_sha256={literal(categorias_existentes['huella_sha256'])}"
        " AND c.estado='habilitada'"
        f" AND c.categoria_id IN ({','.join(literal(x) for x in categorias_existentes['categorias'])}))<>2"
        " THEN RAISE EXCEPTION 'CC1: categorías RPT no habilitadas'; END IF;",
        "END $rpt$;",
    ]
    huellas = {categorias_existentes["id"]: categorias_existentes["huella_sha256"]}
    for catalogo in catalogos:
        identificador = catalogo["id"]
        documento = {
            **catalogo,
            "revision": 1,
            "modulo_id": "bolsa",
            "nombre": "Inscripción en Bolsa",
            "fuente_ref": "fuente:cc1-ensayo",
            "motivo_creacion": "Datos para inscripción en Bolsa",
            "estado": "publicado",
            "creado_por": "actor:cc1-ensayo",
            "creado_en": "2026-10-13T00:00:00Z",
            "publicado_por": "actor:cc1-ensayo-revision",
            "publicado_en": "2026-10-13T00:00:00Z",
            "aprobacion_ref": "aprobacion:cc1-ensayo",
            "motivo_publicacion": "Carga autorizada para el recorrido de inscripción",
        }
        texto = canonico(documento)
        digest = huella(texto)
        huellas[identificador] = digest
        recibo = f"recibo:cc1:ensayo:{identificador}:v1"
        sql.extend([
            "DO $cc1$ DECLARE previo record; BEGIN",
            " SELECT huella_sha256,documento_canonico INTO previo FROM vec_catalogos_configurables.publicacion"
            f" WHERE catalogo_id={literal(identificador)} AND version=1;",
            " IF FOUND THEN",
            f"  IF previo.huella_sha256 IS DISTINCT FROM {literal(digest)} OR previo.documento_canonico IS DISTINCT FROM {literal(texto)}"
            " THEN RAISE EXCEPTION 'CC1: versión existente distinta'; END IF;",
            " ELSE",
            "  PERFORM vec_catalogos_configurables.publicar("
            f"{literal(identificador)},1,{literal(digest)},{literal(texto)},"
            "'{}'::jsonb,encode(sha256(convert_to('{}','UTF8')),'hex'),"
            f"{literal('aprobacion:cc1:a:' + identificador)},{literal('aprobacion:cc1:b:' + identificador)},"
            f"{literal('actor:cc1-ensayo')},{literal('decision:cc1:ensayo:' + identificador)},"
            f"{literal(recibo)},'motivos_bolsa:1:motivo_demo');",
            " END IF; END $cc1$;",
        ])

    sql.append("SET LOCAL ROLE vec_bolsa_convocatorias_propietario;")
    if len({c["id"] for c in convocatorias}) != 2:
        raise ValueError("identificadores de convocatoria repetidos")
    manifiesto = []
    versiones_por_id = {}
    for convocatoria in convocatorias:
        identificador = convocatoria["id"]
        referencia = identificador + "#1"
        publico = "cv1_" + huella(identificador) + "_v1"
        if convocatoria["convocatoria_ref"] != publico:
            raise ValueError("referencia pública discordante")
        categoria = convocatoria["categoria_ref"]
        if categoria not in categorias_existentes["categorias"]:
            raise ValueError("categoría sin entrada CC1")
        plazo = convocatoria["plazo"]
        if plazo["abre_en"] >= plazo["cierra_en"]:
            raise ValueError("plazo invertido")
        contenido = {
            "identificador_publico": convocatoria["identificador_publico"],
            "tipo": "bolsa",
            "titulo": convocatoria["titulo"],
            "resumen": "Presentación de solicitudes de la bolsa de empleo",
            "descripcion": "Bases y requisitos de la convocatoria de bolsa de empleo.",
            "catalogo_categorias": {
                "catalogo_id": categorias_existentes["id"],
                "catalogo_version": 1,
                "catalogo_huella_sha256": huellas[categorias_existentes["id"]],
            },
            "categorias": [categoria],
            "plazos": [{"tipo": "inscripcion", **plazo}],
            "requisitos": convocatoria["requisitos"],
            "documentos": [],
            "ayuda": [],
        }
        version = {
            "id": identificador,
            "secuencia": 1,
            "estado_gobierno": "publicada",
            "publicada_en": "2026-10-13T00:00:00Z",
            "aprobacion_publicacion": {"convocatoria_ref": referencia,
                                       "naturaleza": "ensayo_sintetico"},
            "comprobacion_dependencias": {"convocatoria_ref": referencia,
                                           "naturaleza": "ensayo_sintetico"},
            "contenido": contenido,
            "configuracion": {
                "catalogos": {"id": catalogos[0]["id"], "version": 1,
                              "huella_contenido_sha256": huellas[catalogos[0]["id"]]},
                "flujo_solicitud": {"id": "flujo:inscripcion-ensayo", "version": 1,
                                    "huella_contenido_sha256": huella("flujo:inscripcion-ensayo:v1")},
                "documentos": [{"rol": "bases", "publicacion_ref": convocatoria["bases_ref"]}],
            },
        }
        texto = canonico(version)
        digest = huella(texto)
        versiones_por_id[identificador] = (convocatoria, version, digest)
        manifiesto.append({"convocatoria_ref": publico, "categoria_ref": categoria,
                           "estado": "abierta" if plazo["cierra_en"] > "2026-10-13T00:00:00Z" else "vencida",
                           "version_sha256": digest})
        sql.extend([
            "DO $bc1$ DECLARE previo record; BEGIN",
            " SELECT estado,huella_version_sha256,version_canonica INTO previo"
            " FROM vec_bolsa_convocatorias.version_convocatoria"
            f" WHERE convocatoria_id={literal(identificador)} AND secuencia=1;",
            " IF FOUND THEN",
            f"  IF previo.estado IS DISTINCT FROM 'publicada' OR previo.huella_version_sha256 IS DISTINCT FROM {literal(digest)}"
            f" OR previo.version_canonica IS DISTINCT FROM convert_to({literal(texto)},'UTF8')"
            " THEN RAISE EXCEPTION 'BC1: versión existente distinta'; END IF;",
            " ELSE",
            "  IF EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria"
            f" WHERE convocatoria_id={literal(identificador)})"
            " THEN RAISE EXCEPTION 'BC1: identificador ya usado'; END IF;",
            "  INSERT INTO vec_bolsa_convocatorias.version_convocatoria"
            " (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)"
            f" VALUES ({literal(identificador)},1,{literal(referencia)},'publicada',"
            f"convert_to({literal(texto)},'UTF8'),{literal(digest)},'2026-10-13T00:00:00Z');",
            " END IF; END $bc1$;",
        ])
    sql.append("SET LOCAL ROLE vec_bolsa_llamamientos_propietario;")
    solicitudes = datos["solicitudes"]
    if len(solicitudes) != 3 or {s["estado"] for s in solicitudes} != {
        "pendiente", "admitida_a_convocatoria", "rechazada"
    }:
        raise ValueError("se esperan tres estados de solicitud")
    for solicitud in solicitudes:
        convocatoria, version, version_sha = versiones_por_id[solicitud["convocatoria_id"]]
        solicitud["convocatoria_ref"] = "cv1_" + huella(convocatoria["id"]) + "_v1"
        if not (convocatoria["plazo"]["abre_en"] <= solicitud["presentada_en"] <
                convocatoria["plazo"]["cierra_en"]):
            raise ValueError("solicitud fuera del plazo publicado")
        manifiesto.append(registro_solicitud(
            sql, solicitud, convocatoria, version, version_sha,
            huellas[categorias_existentes["id"]], huellas[catalogos[0]["id"]],
            unidad_ref, ambito_ref, 1, huellas[catalogos[2]["id"]],
            "documentacion.incompleta"))
    sql.append("COMMIT;")
    (salida / "cargar_cc1_y_convocatorias.sql").write_text("\n".join(sql) + "\n", encoding="utf-8")
    (salida / "manifiesto.json").write_text(canonico(manifiesto) + "\n", encoding="utf-8")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Prepara carga CC1 sin abrir conexiones")
    parser.add_argument("fuente", type=pathlib.Path)
    parser.add_argument("salida", type=pathlib.Path)
    parser.add_argument("--unidad-ref")
    parser.add_argument("--ambito-ref")
    argumentos = parser.parse_args()
    preparar(argumentos.fuente, argumentos.salida, argumentos.unidad_ref,
             argumentos.ambito_ref)
