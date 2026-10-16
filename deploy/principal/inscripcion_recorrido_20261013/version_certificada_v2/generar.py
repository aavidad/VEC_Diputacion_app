#!/usr/bin/env python3
"""Prepara publicaciones CC1/BC1 v2 desde preimágenes públicas exactas; sin conexión."""

import argparse
import copy
import hashlib
import json
from pathlib import Path


def canon(valor):
    return json.dumps(valor, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def sha(valor):
    if isinstance(valor, str):
        valor = valor.encode("utf-8")
    return hashlib.sha256(valor).hexdigest()


def sql_texto(valor):
    return "'" + valor.replace("'", "''") + "'"


def anterior(raiz, archivo, huella):
    texto = (raiz / archivo).read_text(encoding="utf-8").removesuffix("\n")
    if sha(texto) != huella or canon(json.loads(texto)) != texto:
        raise ValueError("preimagen canónica distinta: " + archivo)
    return texto, json.loads(texto)


def generar(raiz, salida, fuente=None):
    fuente = raiz / "fuente_v2.json" if fuente is None else Path(fuente)
    f = json.loads(fuente.read_text(encoding="utf-8"))
    if f["esquema"] != "vec.inscripcion.version_certificada_provisional.v2" or f["uso"] != "preparacion_operativa_sin_bases_aprobadas":
        raise ValueError("fuente v2 inesperada")
    p = f["politica"]
    c = f["convocatoria"]
    cat = f["categoria"]
    modalidad_presente = "modalidad_inscripcion" in c
    modalidad = c.get("modalidad_inscripcion")
    if modalidad_presente and (type(modalidad) is not str or modalidad not in ("separada", "conjunta")):
        raise ValueError("modalidad de inscripción no prevista")
    if (p["catalogo_id"], p["version_anterior"], p["version_nueva"]) != ("bolsa.politica.inscripcion", 1, 2):
        raise ValueError("secuencia CC1 incompatible")
    if (c["id"], c["secuencia_anterior"], c["secuencia_nueva"]) != ("proceso:bolsa:ensayo-auxiliar-2026", 1, 2):
        raise ValueError("secuencia BC1 incompatible")
    if (cat["catalogo_id"], cat["version"], cat["categoria_ref"]) != ("categorias_rpt", 1, "categoria:rpt:administrativo"):
        raise ValueError("categoría existente incompatible")
    if c["requisito"] != {"referencia": "identidad_certificada", "orden": 1,
                           "descripcion": "Identificación con certificado digital o DNIe admitido", "obligatorio": True}:
        raise ValueError("único requisito certificado incompatible")
    if (p["archivo_anterior"], c["archivo_anterior"], p["archivo_preimagenes"]) != (
            "politica_v1_canonica.json", "convocatoria_v1_canonica.json", "preimagenes_politica_v1.json"):
        raise ValueError("preimágenes fuera del director certificado")
    p1_texto, p1 = anterior(raiz, p["archivo_anterior"], p["huella_anterior_sha256"])
    c1_texto, c1 = anterior(raiz, c["archivo_anterior"], c["huella_anterior_sha256"])
    pre = json.loads((raiz / p["archivo_preimagenes"]).read_text(encoding="utf-8"))
    claves_v1 = {entrada["clave"] for entrada in p1["entradas"]}
    if (p1["id"], p1["version"], len(claves_v1)) != (p["catalogo_id"], 1, 8) or set(pre) != claves_v1:
        raise ValueError("política v1 incompleta")
    if any(valor != {"version": 1, "huella_sha256": p["huella_anterior_sha256"],
                     "revision": 1, "estado": "habilitada"} for valor in pre.values()):
        raise ValueError("preimágenes CC1 distintas")
    if (c1["id"], c1["secuencia"], c1["estado_gobierno"]) != (c["id"], 1, "publicada"):
        raise ValueError("convocatoria v1 distinta")
    if c1["contenido"]["categorias"] != [cat["categoria_ref"]] or len(c1["contenido"]["requisitos"]) != 2:
        raise ValueError("categoría o requisitos v1 distintos")
    if c1["contenido"]["catalogo_categorias"] != {"catalogo_id": cat["catalogo_id"],
            "catalogo_version": 1, "catalogo_huella_sha256": cat["huella_sha256"]}:
        raise ValueError("catálogo RPT distinto")

    id_sha = sha(c["id"])
    ref_v1 = f"cv1_{id_sha}_v1"
    ref_v2 = f"cv1_{id_sha}_v2"
    if p1["entradas"][5]["atributos"]["convocatoria_ref"] != ref_v1:
        raise ValueError("ámbito v1 distinto")
    instante = f["metadatos_operativos"]["instante"]
    p2 = copy.deepcopy(p1)
    p2.update({"version": 2, "revision": 2,
               "fuente_ref": "fuente:inscripcion-version-certificada-provisional-v2",
               "creado_por": f["metadatos_operativos"]["actor_ref"], "creado_en": instante,
               "publicado_por": f["metadatos_operativos"]["revision_ref"], "publicado_en": instante,
               "aprobacion_ref": "revision:operativa-inscripcion-v2",
               "motivo_creacion": f["metadatos_operativos"]["declaracion"],
               "motivo_publicacion": f["metadatos_operativos"]["declaracion"],
               "naturaleza": "datos_provisionales_sin_bases_aprobadas"})
    asociacion = [e for e in p2["entradas"] if e["clave"] == "asociacion.manual.acta"]
    if len(asociacion) != 1 or asociacion[0]["atributos"]["alcance_ref"] != ref_v1:
        raise ValueError("política de asociación v1 distinta")
    asociacion[0]["atributos"]["alcance_ref"] = ref_v2
    antiguo_ambito = [e for e in p2["entradas"] if e["clave"] == f"ambito.gestion.{id_sha}.v1"]
    if len(antiguo_ambito) != 1 or antiguo_ambito[0]["atributos"]["convocatoria_ref"] != ref_v1:
        raise ValueError("ámbito de gestión v1 distinto")
    nuevo_ambito = copy.deepcopy(antiguo_ambito[0])
    nuevo_ambito["clave"] = f"ambito.gestion.{id_sha}.v2"
    nuevo_ambito["orden"] = 9
    nuevo_ambito["vigente_desde"] = instante
    nuevo_ambito["atributos"]["convocatoria_ref"] = ref_v2
    p2["entradas"].append(nuevo_ambito)
    p2_texto = canon(p2)
    p2_sha = sha(p2_texto)

    c2 = copy.deepcopy(c1)
    c2["secuencia"] = 2
    c2["publicada_en"] = instante
    c2["predecesora"] = {"referencia": c["id"] + "#1", "huella_sha256": c["huella_anterior_sha256"]}
    c2["aprobacion_publicacion"] = {"convocatoria_ref": c["id"] + "#2",
                                      "naturaleza": "preparacion_operativa_sin_firma_legal"}
    c2["comprobacion_dependencias"] = {"convocatoria_ref": c["id"] + "#2",
                                        "naturaleza": "datos_provisionales_sin_bases_aprobadas"}
    contenido = c2["contenido"]
    contenido["identificador_publico"] = c["identificador_publico"]
    contenido["titulo"] = c["titulo"]
    contenido["resumen"] = c["resumen"]
    contenido["descripcion"] = c["descripcion"]
    if modalidad_presente:
        contenido["modalidad_inscripcion"] = modalidad
    contenido["plazos"] = [{"tipo": "inscripcion", "referencia": c["plazo_ref"],
                            "abre_en": c["plazo_abre_en"], "cierra_en": c["plazo_cierra_en"]}]
    contenido["requisitos"] = [c["requisito"]]
    c2["configuracion"]["catalogos"] = {"id": p["catalogo_id"], "version": 2,
                                            "huella_contenido_sha256": p2_sha}
    c2["configuracion"]["documentos"] = [{"rol": "bases", "publicacion_ref": c["bases_ref"]}]
    c2_texto = canon(c2)
    c2_sha = sha(c2_texto)
    pre_texto = canon(pre)

    sql = [r"\set ON_ERROR_STOP on", "BEGIN ISOLATION LEVEL SERIALIZABLE;",
           "SET LOCAL search_path=pg_catalog,pg_temp;", "SET LOCAL standard_conforming_strings=on;",
           "SET LOCAL TimeZone='UTC';",
           "SET LOCAL lock_timeout='5s';", "SET LOCAL statement_timeout='60s';",
           "SELECT pg_advisory_xact_lock(hashtextextended('vec:inscripcion:version-certificada:v2',0));",
           "SET LOCAL ROLE vec_catalogos_configurables_propietario;",
           "DO $politica$ BEGIN",
           " IF NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.publicacion"
           f" WHERE catalogo_id={sql_texto(p['catalogo_id'])} AND version=1"
           f" AND huella_sha256={sql_texto(p['huella_anterior_sha256'])}"
           f" AND documento_canonico={sql_texto(p1_texto)})"
           " THEN RAISE EXCEPTION 'v2: política v1 distinta'; END IF;",
           " IF NOT EXISTS (SELECT 1 FROM vec_catalogos_configurables.publicacion"
           f" WHERE catalogo_id={sql_texto(p['catalogo_id'])} AND version=2"
           f" AND huella_sha256={sql_texto(p2_sha)} AND documento_canonico={sql_texto(p2_texto)})"
           " AND (SELECT jsonb_object_agg(categoria_id,jsonb_build_object("
           "'version',version,'huella_sha256',huella_sha256,'revision',revision,'estado',estado))"
           " FROM vec_catalogos_configurables.categoria_control"
           f" WHERE catalogo_id={sql_texto(p['catalogo_id'])}) IS DISTINCT FROM {sql_texto(pre_texto)}::jsonb"
           " THEN RAISE EXCEPTION 'v2: control CC1 cambiado'; END IF;",
           " PERFORM vec_catalogos_configurables.publicar("
           f"{sql_texto(p['catalogo_id'])},2,{sql_texto(p2_sha)},{sql_texto(p2_texto)},"
           f"{sql_texto(pre_texto)}::jsonb,encode(sha256(convert_to({sql_texto(pre_texto)}::jsonb::text,'UTF8')),'hex'),"
           "'revision:operativa:a:inscripcion-v2','revision:operativa:b:inscripcion-v2',"
           f"{sql_texto(f['metadatos_operativos']['actor_ref'])},'decision:preparacion-operativa:inscripcion-v2',"
           f"'recibo:cc1:politica-inscripcion:v2',{sql_texto(f['metadatos_operativos']['motivo_ref'])});",
           "END $politica$;",
           "SET LOCAL ROLE vec_bolsa_convocatorias_propietario;",
           "DO $convocatoria$ DECLARE anterior record; nueva record; BEGIN",
           " SELECT estado,huella_version_sha256,version_canonica INTO anterior"
           " FROM vec_bolsa_convocatorias.version_convocatoria"
           f" WHERE convocatoria_id={sql_texto(c['id'])} AND secuencia=1;",
           f" IF anterior.estado IS DISTINCT FROM 'publicada' OR anterior.huella_version_sha256 IS DISTINCT FROM {sql_texto(c['huella_anterior_sha256'])}"
           f" OR anterior.version_canonica IS DISTINCT FROM convert_to({sql_texto(c1_texto)},'UTF8')"
           " THEN RAISE EXCEPTION 'v2: convocatoria v1 distinta'; END IF;",
           " IF EXISTS (SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria"
           f" WHERE convocatoria_id={sql_texto(c['id'])} AND secuencia>2)"
           " THEN RAISE EXCEPTION 'v2: ya existe una versión posterior'; END IF;",
           " SELECT estado,referencia,huella_version_sha256,version_canonica INTO nueva"
           " FROM vec_bolsa_convocatorias.version_convocatoria"
           f" WHERE convocatoria_id={sql_texto(c['id'])} AND secuencia=2;",
           " IF FOUND THEN",
           f"  IF nueva.estado IS DISTINCT FROM 'publicada' OR nueva.referencia IS DISTINCT FROM {sql_texto(c['id']+'#2')}"
           f" OR nueva.huella_version_sha256 IS DISTINCT FROM {sql_texto(c2_sha)}"
           f" OR nueva.version_canonica IS DISTINCT FROM convert_to({sql_texto(c2_texto)},'UTF8')"
           " THEN RAISE EXCEPTION 'v2: convocatoria v2 distinta'; END IF;",
           " ELSE",
           "  INSERT INTO vec_bolsa_convocatorias.version_convocatoria"
           " (convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)"
           f" VALUES ({sql_texto(c['id'])},2,{sql_texto(c['id']+'#2')},'publicada',"
           f"convert_to({sql_texto(c2_texto)},'UTF8'),{sql_texto(c2_sha)},{sql_texto(instante)});",
           " END IF; END $convocatoria$;", "COMMIT;"]
    salida.mkdir(parents=True, exist_ok=True)
    (salida / "cargar_version_certificada_v2.sql").write_text("\n".join(sql) + "\n", encoding="utf-8")
    manifiesto = {"esquema": f["esquema"], "politica_v1_sha256": p["huella_anterior_sha256"],
                  "politica_v2_sha256": p2_sha, "convocatoria_id": c["id"],
                  "convocatoria_v1_sha256": c["huella_anterior_sha256"],
                  "convocatoria_v2_sha256": c2_sha, "convocatoria_ref_v2": ref_v2,
                  "categoria_ref": cat["categoria_ref"], "requisitos_v2": [c["requisito"]],
                  "solicitudes_creadas": 0}
    if modalidad_presente:
        manifiesto["modalidad_inscripcion"] = modalidad
    (salida / "manifiesto.json").write_text(canon(manifiesto) + "\n", encoding="utf-8")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--salida", type=Path, required=True)
    parser.add_argument("--fuente", type=Path)
    args = parser.parse_args()
    generar(Path(__file__).resolve().parent, args.salida, args.fuente)
