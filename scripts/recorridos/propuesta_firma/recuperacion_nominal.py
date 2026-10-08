"""Integridad del transporte nominal; la autoridad Go valida el canon histórico.

El texto del canon sólo se usa para comprobar su huella. Nunca sale de esta
función ni se incorpora al informe privado o público.
"""
import hashlib
import re


RUTA = "/api/vec/contratacion-temporal/firmas-documento/recuperaciones-v2"
MAX_RESPUESTA = 8 << 20
CAMPOS_BASE = {"esquema", "expediente_ref", "version_expediente", "documento",
               "historia_revision", "historia_sha256", "firmas", "recuperacion",
               "campos_no_disponibles", "firma_eficaz"}
CAMPOS_NOMINALES = {"firma_ref", "material_root_sha256", "canon_nominal",
                    "canon_nominal_sha256", "canon_nominal_ref"}
CAMPOS_RESUMEN = CAMPOS_NOMINALES - {"canon_nominal"}
SHA = re.compile(r"[0-9a-f]{64}\Z")
REF_CANON = re.compile(r"evidencia:competencia-firmante-ct:[0-9a-f]{64}\Z")


def validar(datos, informe, recibos, validar_tecnica, corte):
    """Comprueba bytes y enlace con recibos; devuelve exclusivamente metadatos."""
    def fallo():
        raise corte("recuperacion_nominal", "respuesta_recuperacion_nominal_invalida")

    if not isinstance(datos, dict) or set(datos) != CAMPOS_BASE | {"recuperaciones"} \
            or datos["esquema"] != "vec.contratacion-temporal.recuperacion-firmas-r5.v2" \
            or datos["recuperacion"] != "recuperada" \
            or datos["campos_no_disponibles"] != [] or datos["firma_eficaz"] is not False:
        fallo()
    # Se reutilizan las comprobaciones de recibos, historia y revisión del
    # transporte técnico. Esta proyección interna no concede autoridad.
    base = {k: datos[k] for k in CAMPOS_BASE}
    base.update(esquema="vec.contratacion-temporal.consulta-firmas-r5.v2",
                recuperacion="parcial",
                campos_no_disponibles=["material_root_sha256", "canon_nominal"])
    tecnica = validar_tecnica(base, informe, recibos)
    esperadas = {f["firma_ref"] for f in tecnica["firmas"] if f["revision_pdf"] is not None}
    filas = datos["recuperaciones"]
    if not isinstance(filas, list) or len(filas) != len(esperadas):
        fallo()
    originales = {r["firma_ref"]: r for r in recibos}
    vistos = set()
    resumen = []
    for fila in filas:
        if not isinstance(fila, dict) or set(fila) != CAMPOS_NOMINALES:
            fallo()
        ref = fila["firma_ref"]
        if not isinstance(ref, str) or ref not in esperadas or ref in vistos:
            fallo()
        if any(not isinstance(fila[k], str) or not SHA.fullmatch(fila[k])
               for k in ("material_root_sha256", "canon_nominal_sha256")) \
                or not isinstance(fila["canon_nominal_ref"], str) \
                or not REF_CANON.fullmatch(fila["canon_nominal_ref"]):
            fallo()
        canon = fila["canon_nominal"]
        if not isinstance(canon, str):
            fallo()
        try:
            exactos = canon.encode("utf-8", errors="strict")
        except UnicodeError:
            fallo()
        if not 512 <= len(exactos) <= 32768 \
                or hashlib.sha256(exactos).hexdigest() != fila["canon_nominal_sha256"]:
            fallo()
        if ref in originales and fila["material_root_sha256"] != originales[ref]["material_root_sha256"]:
            fallo()
        vistos.add(ref)
        resumen.append({k: fila[k] for k in CAMPOS_RESUMEN})
    if vistos != esperadas or not set(originales).issubset(vistos):
        fallo()
    resumen.sort(key=lambda fila: fila["firma_ref"])
    return resumen


def comparar(baseline, actual, corte):
    """No adopta la primera evidencia nominal después de un reinicio."""
    if not isinstance(baseline, list) or not baseline \
            or any(not isinstance(f, dict) or set(f) != CAMPOS_RESUMEN for f in baseline) \
            or baseline != actual:
        raise corte("recuperacion_nominal", "baseline_recuperacion_nominal_ausente_o_distinta")
