#!/usr/bin/env python3
"""Siembra expedientes de ejemplo en Peticiones de personal temporal (solo Python estándar).

Usa exclusivamente la API HTTP de VEC con los certificados mTLS de desarrollo:
RRHH (alta, análisis, cobertura, asignación, informe, subsanación, llamamiento)
e Intervención (fiscalización). Nunca escribe en tablas: historia, auditoría,
recibos y outbox los genera la propia aplicación.

Es idempotente: cada operación lleva una clave derivada de un espacio fijo y del
código del caso, y los datos de cada caso son fijos (fechas absolutas). Al
relanzarlo, el alta se repite con la misma clave (la API devuelve el mismo
expediente), el guion lee los hitos del expediente y solo ejecuta lo que falta.
Los pasos del llamamiento se repiten con su misma clave y la API devuelve el
recibo original.

Los plazos de fase los calcula el servidor desde el día en que el expediente
entra en la fase: lo sembrado hoy queda «en plazo» (cinco o diez días hábiles).
No hay forma legítima de fechar hacia atrás una operación.

Modos:  --plan (sin escribir), --ejecutar --confirmar-entorno-sintetico (escribe),
--resumen (solo lee el cuadro).
Se ejecuta dentro del contenedor de la aplicación:
  podman exec -i APP python3 - --ejecutar --confirmar-entorno-sintetico \
    --casos-b64 "$(base64 -w0 casos.json)" < sembrar_ejemplo_ct.py
Termina con SEMBRADO-OK (0), SEMBRADO-PARCIAL (1) o SEMBRADO-FALLO (2).
"""

from __future__ import annotations

import argparse
import base64
import collections
import datetime as dt
import hashlib
import http.client
import ipaddress
import json
import os
import ssl
import sys
import time
import urllib.parse
import uuid

API = "/api/vec/contratacion-temporal"
# Secuencia de operaciones de cada objetivo (los hitos que deja en el expediente).
ACCION = {
    "analisis": "contratacion_temporal.analisis.registrar",
    "cobertura": "contratacion_temporal.cobertura.decidir",
    "asignacion": "contratacion_temporal.unidad.asignar",
    "informe": "contratacion_temporal.informe_juridico.generar",
    "favorable": "contratacion_temporal.fiscalizacion.registrar",
    "desfavorable": "contratacion_temporal.fiscalizacion.registrar",
    "subsanacion": "contratacion_temporal.subsanacion_reparos.registrar",
}
BASE_FISCALIZADA = ["analisis", "cobertura", "asignacion", "informe"]
PLAN = {
    "solicitud": [],
    "analisis": ["analisis"],
    "cobertura": ["analisis", "cobertura"],
    "asignacion": ["analisis", "cobertura", "asignacion"],
    "informe": BASE_FISCALIZADA,
    "fiscalizacion": BASE_FISCALIZADA + ["favorable"],
    "llamamiento": BASE_FISCALIZADA + ["favorable"],
    "respuesta": BASE_FISCALIZADA + ["favorable"],
    "reparo": BASE_FISCALIZADA + ["desfavorable"],
    "subsanacion": BASE_FISCALIZADA + ["desfavorable", "subsanacion"],
}
LLAMAMIENTO = {"llamamiento": 2, "respuesta": 3}  # selección, comunicación, respuesta
UNIDAD, RESPONSABLE = "unidad:desarrollo:rrhh", "persona:responsable-sintetica-001"


class ErrorAPI(Exception):
    def __init__(self, paso: str, estado: int, cuerpo: object):
        codigo = cuerpo.get("error", {}).get("codigo", "") if isinstance(cuerpo, dict) else ""
        super().__init__(f"{paso}: HTTP {estado} {codigo}".strip())
        self.estado, self.codigo = estado, codigo


def validar_base(base: str, ejecutar: bool) -> None:
    """Acepta una raíz HTTPS; la escritura exige una dirección de loopback."""
    try:
        u = urllib.parse.urlsplit(base)
        puerto = u.port
    except ValueError as e:
        raise ValueError("--base no es una URL válida") from e
    if (u.scheme != "https" or not u.hostname or not u.netloc or
            u.username is not None or u.password is not None or
            u.path not in ("", "/") or u.query or u.fragment or puerto == 0):
        raise ValueError("--base debe ser una raíz HTTPS sin credenciales, ruta ni parámetros")
    if ejecutar:
        try:
            local = ipaddress.ip_address(u.hostname).is_loopback
        except ValueError:
            local = u.hostname == "localhost"
        if not local:
            raise ValueError("--ejecutar solo admite una dirección local de loopback")


def exigir_catalogo_de_ejemplo(catalogos: dict) -> None:
    preparacion = catalogos.get("preparacion_vias")
    if not isinstance(preparacion, dict) or preparacion.get("es_ejemplo") is not True:
        raise RuntimeError("el catálogo de vías no acredita datos de ejemplo; se cancela la escritura")


class Cliente:
    """Una conexión mTLS por petición; JSON compacto (algunas rutas exigen su forma canónica)."""

    def __init__(self, base: str, material: str, nombre: str):
        crt, key, ca = (f"{material}/mtls/{nombre}.crt", f"{material}/mtls/{nombre}.key", f"{material}/ca/ca.crt")
        if not (os.path.isfile(crt) and os.path.isfile(key)):
            raise SystemExit(f"SEMBRADO-FALLO: falta el certificado {nombre} en {material}/mtls")
        if not os.path.isfile(ca):
            raise SystemExit(f"SEMBRADO-FALLO: falta la CA en {material}/ca")
        ctx = ssl.create_default_context(cafile=ca)
        ctx.load_cert_chain(crt, key)
        u = urllib.parse.urlsplit(base)
        self.host, self.puerto, self.ctx = u.hostname, u.port or 443, ctx

    def pedir(self, metodo: str, ruta: str, cuerpo: dict | None = None) -> tuple[int, object]:
        datos = None if cuerpo is None else json.dumps(cuerpo, ensure_ascii=False, separators=(",", ":")).encode()
        cab = {"Accept": "application/json"}
        if datos is not None:
            cab["Content-Type"] = "application/json"
        con = http.client.HTTPSConnection(self.host, self.puerto, context=self.ctx, timeout=60)
        try:
            con.request(metodo, ruta, body=datos, headers=cab)
            r = con.getresponse()
            crudo = r.read()
        finally:
            con.close()
        try:
            return r.status, json.loads(crudo) if crudo else {}
        except ValueError:
            return r.status, {}

    def exigir(self, paso: str, metodo: str, ruta: str, cuerpo: dict | None = None) -> dict:
        estado, v = self.pedir(metodo, ruta, cuerpo)
        if not 200 <= estado < 300 or not isinstance(v, dict) or "data" not in v:
            raise ErrorAPI(paso, estado, v)
        return v["data"]


def clave(espacio: str, codigo: str, operacion: str) -> str:
    """UUID v4 determinista: la misma operación del mismo caso repite siempre la misma clave."""
    d = bytearray(hashlib.sha256(f"{espacio}:{codigo}:{operacion}".encode()).digest()[:16])
    d[6], d[8] = (d[6] & 0x0F) | 0x40, (d[8] & 0x3F) | 0x80
    return str(uuid.UUID(bytes=bytes(d)))


def instante(dia: str) -> str:
    return f"{dia}T00:00:00Z"


class Sembrador:
    def __init__(self, rrhh: Cliente, intervencion: Cliente, espacio: str):
        self.rrhh, self.intervencion, self.espacio = rrhh, intervencion, espacio
        self.catalogos = rrhh.exigir("catálogos del alta", "GET", f"{API}/catalogos-alta")
        self.configuracion = rrhh.exigir("configuración del análisis", "GET", f"{API}/configuracion-analisis")

    # ---------------------------------------------------------------- preparación
    def centro(self, codigo: str) -> dict:
        centros = [c for c in self.catalogos.get("centros") or []
                   if c.get("referencia", "").rsplit(":", 1)[-1] == codigo]
        if len(centros) != 1 or len(centros[0].get("contactos") or []) != 1:
            raise RuntimeError(f"centro {codigo}: se necesita una coincidencia exacta con un contacto")
        return centros[0]

    def categoria(self, nombre: str) -> tuple[str, str]:
        cats = [c for c in self.catalogos.get("categorias") or []
                if c.get("referencia", "").rsplit(":", 1)[-1] == nombre]
        if len(cats) != 1 or len(cats[0].get("grupos_subgrupos") or []) != 1:
            raise RuntimeError(f"categoría {nombre}: se necesita una coincidencia exacta con un grupo")
        c = cats[0]
        return c["referencia"], c["grupos_subgrupos"][0]["clave"]

    def plan(self, caso: dict) -> dict:
        centro = self.centro(caso["centro"])
        categoria, grupo = self.categoria(caso["categoria"])
        motivos = [m["clave"] for m in self.catalogos.get("motivos") or []]
        modalidades = [m["clave"] for m in self.configuracion.get("modalidades") or []]
        if "sustitucion" not in motivos:
            raise RuntimeError("el catálogo del alta no ofrece el motivo sustitución")
        if caso["modalidad"] not in modalidades:
            raise RuntimeError(f"modalidad {caso['modalidad']}: no figura en la configuración")
        return {
            "centro": centro, "categoria": categoria, "grupo": grupo,
            "motivo": "sustitucion", "modalidad": caso["modalidad"],
            "periodo": {"inicio": instante(caso["inicio"]), "fin": instante(caso["fin"])},
        }

    # ---------------------------------------------------------------- operaciones
    def alta(self, caso: dict, p: dict) -> dict:
        return self.rrhh.exigir("alta", "POST", f"{API}/solicitudes", {
            "clave_idempotencia": clave(self.espacio, caso["codigo"], "alta"),
            "solicitud": {
                "centro_ref": p["centro"]["referencia"], "contacto_ref": p["centro"]["contactos"][0]["referencia"],
                "categoria_ref": p["categoria"], "grupo_subgrupo": p["grupo"], "motivo_clave": p["motivo"],
                "detalle": caso["detalle"], "periodo": p["periodo"], "rc": {"existe": False},
                "documentos_adjuntos": [], "observaciones": "",
            }})

    def detalle(self, expediente: str) -> dict:
        return self.rrhh.exigir("detalle", "POST", f"{API}/expedientes/consultas",
                                {"expediente_ref": expediente, "version_observada": 0})

    def operar(self, op: str, caso: dict, p: dict, exp: str, version: int, repeticion: int) -> int:
        k = clave(self.espacio, caso["codigo"], f"{op}:{repeticion}")
        base = {"expediente_ref": exp, "version_esperada": version, "clave_idempotencia": k}
        if op == "analisis":
            rc = self.configuracion["entradas_rc"][0]
            analisis = {"modalidad_clave": p["modalidad"], "categoria_ref": p["categoria"], "grupo_subgrupo": p["grupo"],
                        "causa_clave": self.configuracion["causas"][0]["clave"], "periodo": p["periodo"],
                        "porcentaje_jornada": int(caso["jornada"]) * 100,
                        "entrada_rc": {"referencia": rc["referencia"], "huella_sha256": rc["huella_sha256"]},
                        "observaciones": "Necesidad y crédito comprobados con el servicio."}
            if caso.get("urgencia") and self.configuracion.get("urgencia_disponible"):
                analisis["urgencia_motivo"] = caso["urgencia"]
            r = self.rrhh.exigir("análisis", "POST", f"{API}/analisis/registros", {
                "expediente_ref": exp, "version_esperada": version, "clave_idempotencia": k,
                "artefacto_ref": self.configuracion["artefacto_ref"], "analisis": analisis})
        elif op == "cobertura":
            propuesta = self.rrhh.exigir("propuesta de cobertura", "POST", f"{API}/cobertura/propuesta",
                                         {"expediente_ref": exp, "version_esperada": version})
            if propuesta.get("estado") != "viable" or not propuesta.get("via_recomendada"):
                raise RuntimeError("la propuesta de cobertura no ofrece una vía viable")
            r = self.rrhh.exigir("decisión de cobertura", "POST", f"{API}/cobertura/decisiones", {
                "expediente_ref": exp, "version_esperada": version, "clave_idempotencia": k,
                "identidad_semantica": propuesta["identidad_semantica"], "via_elegida": propuesta["via_recomendada"],
                "motivo_clave": ""})
        elif op == "asignacion":
            r = self.rrhh.exigir("asignación", "POST", f"{API}/asignaciones",
                                 {**base, "unidad_ref": UNIDAD, "responsable_ref": RESPONSABLE})
        elif op == "informe":
            r = self.rrhh.exigir("informe jurídico", "POST", f"{API}/informes-juridicos/preparaciones", base)
        elif op in ("favorable", "desfavorable"):
            r = self.intervencion.exigir("fiscalización", "POST", f"{API}/fiscalizaciones/resultados", {
                **base, "resultado": op, "observaciones": caso.get("reparo", "") if op == "desfavorable" else ""})
        elif op == "subsanacion":
            r = self.rrhh.exigir("subsanación", "POST", f"{API}/subsanacion-reparos",
                                 {**base, "observaciones": caso["subsanacion"]})
        else:
            raise RuntimeError(f"operación desconocida {op}")
        return int(r["version_resultante"])

    def llamamiento(self, caso: dict, exp: str, version: int, pasos: int) -> list[str]:
        """Selección de candidato en la bolsa, comunicación y, si toca, la respuesta recibida."""
        k = lambda op: clave(self.espacio, caso["codigo"], op)  # noqa: E731
        hechos = []
        sel = self.rrhh.exigir("selección del llamamiento", "POST", f"{API}/llamamientos/seleccion",
                               {"expediente_ref": exp, "version_esperada": version, "clave_idempotencia": k("seleccion")})
        hechos.append("selección")
        com = self.rrhh.exigir("comunicación del llamamiento", "POST", f"{API}/llamamientos/comunicaciones", {
            "clave_idempotencia": k("comunicacion"), "organizacion_ref": sel["organizacion_ref"], "expediente_ref": exp,
            "llamamiento_ref": sel["llamamiento_ref"], "version_esperada": sel["version_llamamiento"],
            "prueba_entrega_ref": sel["recibo_ref"]})
        hechos.append("comunicación")
        if pasos < 3:
            return hechos
        # La respuesta llega un segundo después de registrar la comunicación: mismo
        # instante en cada repetición (la comunicación devuelve su fecha original).
        registrada = dt.datetime.fromisoformat(com["registrada_en"].replace("Z", "+00:00"))
        recibida = (registrada + dt.timedelta(seconds=1)).replace(microsecond=0)
        espera = (recibida - dt.datetime.now(dt.timezone.utc)).total_seconds()
        if espera > 0:
            time.sleep(espera + 0.5)
        self.rrhh.exigir("respuesta recibida", "POST", f"{API}/llamamientos/respuestas/registro", {
            "clave_idempotencia": k("respuesta"), "organizacion_ref": sel["organizacion_ref"], "expediente_ref": exp,
            "llamamiento_ref": sel["llamamiento_ref"], "comunicacion_ref": com["comunicacion_ref"],
            "version_comunicacion_esperada": com["version_resultante"], "respuesta": "aceptacion",
            "correo_ref": "correo:respuesta:" + k("correo"),
            "correo_sha256": hashlib.sha256(f"{self.espacio}:{caso['codigo']}:correo".encode()).hexdigest(),
            "recibida_en": recibida.strftime("%Y-%m-%dT%H:%M:%SZ")})
        hechos.append("respuesta de aceptación recibida")
        return hechos

    # ---------------------------------------------------------------- caso completo
    def sembrar(self, caso: dict) -> dict:
        p = self.plan(caso)
        salida = {"codigo": caso["codigo"], "objetivo": caso["objetivo"], "hechos": []}
        a = self.alta(caso, p)
        exp = a["expediente_ref"]
        salida["numero"] = a.get("numero_visible", "")
        d = self.detalle(exp)
        hitos = [h.get("accion_clave") for h in d.get("hitos") or [] if h.get("accion_clave") != "alta"]
        version = int(d["resumen"]["version"])
        esperadas = [ACCION[o] for o in PLAN[caso["objetivo"]]]
        if hitos != esperadas[:len(hitos)]:
            raise RuntimeError("el expediente tiene otra historia (se tocó a mano): no se continúa")
        salida["ya_estaba"] = len(hitos)
        for i, op in enumerate(PLAN[caso["objetivo"]][len(hitos):], start=len(hitos)):
            repeticion = PLAN[caso["objetivo"]][:i].count(op)
            version = self.operar(op, caso, p, exp, version, repeticion)
            salida["hechos"].append(op)
        if caso["objetivo"] in LLAMAMIENTO:
            pasos = self.llamamiento(caso, exp, version, LLAMAMIENTO[caso["objetivo"]])
            salida["hechos"].append("llamamiento al día (" + ", ".join(pasos) + ")")
        salida["expediente_ref"] = exp
        return salida


# -------------------------------------------------------------------- lectura del cuadro
def cuadro(rrhh: Cliente) -> tuple[list[dict], str]:
    expedientes, cursor, generada = [], "", ""
    for _ in range(100):
        d = rrhh.exigir("cuadro", "POST", f"{API}/cuadro/consultas", {
            "filtros": {"texto": "", "estado_clave": "", "fase_clave": ""}, "paginacion": {"limite": 100, "cursor": cursor}})
        expedientes += d.get("expedientes") or []
        generada = d.get("generada_en", generada)
        if not d.get("hay_mas"):
            break
        cursor = d.get("cursor_siguiente", "")
    return expedientes, generada


FASE_RRHH = {"solicitud": "1 Solicitud", "analisis": "2 Análisis RRHH", "asignacion": "3 Gestión de bolsa",
             "asignacion_unidad": "3 Gestión de bolsa", "informe_juridico": "3 Gestión de bolsa",
             "fiscalizacion": "4 Fiscalización", "subsanacion_unidad": "4 Fiscalización",
             "llamamiento": "5 Obtención del candidato", "nombramiento": "6 Nombramiento",
             "incorporacion": "7 Incorporación", "seguimiento": "8 Seguimiento", "cierre": "8 Seguimiento"}


def tramo(e: dict, hoy: str) -> str:
    p = e.get("plazo_fase") or {}
    if not p or p.get("estado") == "no_calculado":
        return "sin plazo"
    if p["estado"] == "vencido":
        return "vencido"
    if p["estado"] == "vence_hoy":
        return "vence hoy"
    dias = (dt.date.fromisoformat(p["ultimo_dia"]) - dt.date.fromisoformat(hoy)).days
    return "vence en 7 días" if dias <= 6 else "vence más adelante"


def resumen(rrhh: Cliente, propios: set[str]) -> None:
    expedientes, generada = cuadro(rrhh)
    hoy = generada[:10] or dt.date.today().isoformat()
    tabla = collections.Counter()
    for e in expedientes:
        if e.get("estado_clave") in ("completado", "cerrado", "cancelado"):
            continue
        tabla[(FASE_RRHH.get(e.get("fase_clave"), e.get("fase_clave")), tramo(e, hoy))] += 1
    propio = f" ({len(propios)} de este sembrado)" if propios else ""
    print(f"Cuadro de RRHH a {hoy}: {len(expedientes)} expedientes{propio}")
    for (fase, t), n in sorted(tabla.items()):
        print(f"  {fase:<28} {t:<20} {n}")


def main() -> int:
    a = argparse.ArgumentParser(description="Siembra expedientes de ejemplo por la API de VEC.")
    a.add_argument("--base", default="https://localhost:18443")
    a.add_argument("--material", default="/vec-material")
    a.add_argument("--rrhh", default="cliente")
    a.add_argument("--intervencion", default="intervencion")
    a.add_argument("--casos", help="fichero JSON de casos")
    a.add_argument("--casos-b64", help="el mismo JSON en base64 (para ejecutar dentro del contenedor)")
    a.add_argument("--confirmar-entorno-sintetico", action="store_true",
                   help="confirma que el destino y sus datos son de desarrollo sintético")
    modo = a.add_mutually_exclusive_group(required=True)
    modo.add_argument("--plan", action="store_true", help="muestra qué haría, sin escribir")
    modo.add_argument("--ejecutar", action="store_true", help="crea o completa los expedientes")
    modo.add_argument("--resumen", action="store_true", help="solo lee el cuadro de RRHH")
    o = a.parse_args()
    try:
        validar_base(o.base, o.ejecutar)
    except ValueError as e:
        a.error(str(e))
    if o.ejecutar and not o.confirmar_entorno_sintetico:
        a.error("--ejecutar exige --confirmar-entorno-sintetico")
    if o.confirmar_entorno_sintetico and not o.ejecutar:
        a.error("--confirmar-entorno-sintetico solo se usa con --ejecutar")
    if not o.resumen and bool(o.casos) == bool(o.casos_b64):
        a.error("indique --casos o --casos-b64")
    rrhh = Cliente(o.base, o.material, o.rrhh)
    if o.resumen:
        resumen(rrhh, set())
        return 0
    datos = json.loads(base64.b64decode(o.casos_b64) if o.casos_b64 else open(o.casos, encoding="utf-8").read())
    if datos.get("esquema") != "vec.ct.sembrado-ejemplo.v1":
        a.error("fichero de casos con otro esquema")
    casos = datos["casos"]
    s = Sembrador(rrhh, Cliente(o.base, o.material, o.intervencion), datos["espacio_claves"])
    try:
        if o.ejecutar:
            exigir_catalogo_de_ejemplo(s.catalogos)
        for c in casos:
            PLAN[c["objetivo"]]
        planes = [s.plan(c) for c in casos]
    except (RuntimeError, KeyError, TypeError, ValueError) as e:
        print(f"SEMBRADO-FALLO: no se ha escrito nada: {e}")
        return 2
    if o.plan:
        for c, p in zip(casos, planes):
            print(f"{c['codigo']}  {c['objetivo']:<13} {p['centro'].get('etiqueta', '')[:34]:<34} {p['categoria']:<40} "
                  f"{' → '.join(PLAN[c['objetivo']]) or 'alta'}")
        return 0
    fallos, propios = 0, set()
    for c in casos:
        try:
            r = s.sembrar(c)
            propios.add(r["expediente_ref"])
            hecho = ", ".join(r["hechos"]) or "nada nuevo"
            print(f"OK    {c['codigo']} {r['numero']:<16} {c['objetivo']:<13} ya tenía {r['ya_estaba']} pasos; ahora: {hecho}")
        except (ErrorAPI, RuntimeError, KeyError, ValueError) as e:
            fallos += 1
            print(f"FALLO {c['codigo']} {c['objetivo']:<13} {e}")
    resumen(rrhh, propios)
    if fallos == 0:
        print("SEMBRADO-OK")
        return 0
    print(f"SEMBRADO-PARCIAL ({fallos} de {len(casos)} casos sin terminar; relanzar continúa donde se quedó)")
    return 1 if fallos < len(casos) else 2


if __name__ == "__main__":
    sys.exit(main())
