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

Sin --ejecutar muestra un plan. Con --ejecutar pide teclear el destino en una TTY.
El kit lo ejecuta dentro del contenedor verificado de clon o principal, con
--destino, --puerto-interno, --huella-servidor-sha256 y --casos. El kit verifica
también el marcador CLON-OK del mismo paquete antes de la principal.
Termina con SEMBRADO-OK (0), SEMBRADO-PARCIAL (1) o SEMBRADO-FALLO (2).
"""

from __future__ import annotations

import argparse
import collections
import datetime as dt
import hashlib
import hmac
import http.client
import json
import os
import re
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
HOST_INTERNO = "localhost"
MATERIAL = "/vec-material"


class ErrorAPI(Exception):
    def __init__(self, paso: str, estado: int, cuerpo: object):
        codigo = cuerpo.get("error", {}).get("codigo", "") if isinstance(cuerpo, dict) else ""
        super().__init__(f"{paso}: HTTP {estado} {codigo}".strip())
        self.estado, self.codigo = estado, codigo


class ErrorDestino(Exception):
    """Fallo de identidad TLS: no se puede continuar con otro caso."""


def validar_huella(huella: str) -> str:
    if re.fullmatch(r"[0-9a-fA-F]{64}", huella) is None:
        raise ValueError("la huella SHA256 del servidor debe tener 64 dígitos hexadecimales")
    return huella.lower()


class Cliente:
    """Una conexión mTLS por petición; JSON compacto (algunas rutas exigen su forma canónica)."""

    def __init__(self, puerto: int, nombre: str, huella_servidor: str):
        material = MATERIAL
        crt, key, ca = (f"{material}/mtls/{nombre}.crt", f"{material}/mtls/{nombre}.key", f"{material}/ca/ca.crt")
        if not (os.path.isfile(crt) and os.path.isfile(key)):
            raise ErrorDestino(f"falta el certificado {nombre} en {material}/mtls")
        if not os.path.isfile(ca):
            raise ErrorDestino(f"falta la CA en {material}/ca")
        try:
            ctx = ssl.create_default_context(cafile=ca)
            ctx.load_cert_chain(crt, key)
        except (OSError, ssl.SSLError) as e:
            raise ErrorDestino("no se pudo cargar el material mTLS verificado") from e
        self.puerto, self.ctx = puerto, ctx
        self.huella_servidor = validar_huella(huella_servidor)

    def pedir(self, metodo: str, ruta: str, cuerpo: dict | None = None) -> tuple[int, object]:
        datos = None if cuerpo is None else json.dumps(cuerpo, ensure_ascii=False, separators=(",", ":")).encode()
        cab = {"Accept": "application/json"}
        if datos is not None:
            cab["Content-Type"] = "application/json"
        con = http.client.HTTPSConnection(HOST_INTERNO, self.puerto, context=self.ctx, timeout=60)
        try:
            try:
                con.connect()
            except (OSError, ssl.SSLError) as e:
                raise ErrorDestino("no se pudo establecer el canal TLS verificado") from e
            certificado = con.sock.getpeercert(binary_form=True)
            if not certificado or not hmac.compare_digest(hashlib.sha256(certificado).hexdigest(),
                                                           self.huella_servidor):
                raise ErrorDestino("la huella TLS del servidor no coincide con el destino autorizado")
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
        # El POST confirmado y su replay omiten registrada_en. La consulta autorizada
        # conserva la fecha y la versión de la comunicación original.
        comunicacion = self.consultar_comunicacion(exp, sel, com)
        registrada = dt.datetime.fromisoformat(comunicacion["registrada_en"].replace("Z", "+00:00"))
        if registrada.utcoffset() != dt.timedelta(0):
            raise RuntimeError("la fecha de comunicación no está en UTC")
        recibida = (registrada + dt.timedelta(seconds=1)).replace(microsecond=0)
        espera = (recibida - dt.datetime.now(dt.timezone.utc)).total_seconds()
        if espera > 0:
            time.sleep(espera + 0.5)
        self.rrhh.exigir("respuesta recibida", "POST", f"{API}/llamamientos/respuestas/registro", {
            "clave_idempotencia": k("respuesta"), "organizacion_ref": sel["organizacion_ref"], "expediente_ref": exp,
            "llamamiento_ref": sel["llamamiento_ref"], "comunicacion_ref": com["comunicacion_ref"],
            "version_comunicacion_esperada": comunicacion["version"], "respuesta": "aceptacion",
            "correo_ref": "correo:respuesta:" + k("correo"),
            "correo_sha256": hashlib.sha256(f"{self.espacio}:{caso['codigo']}:correo".encode()).hexdigest(),
            "recibida_en": recibida.strftime("%Y-%m-%dT%H:%M:%SZ")})
        hechos.append("respuesta de aceptación recibida")
        return hechos

    def consultar_comunicacion(self, exp: str, sel: dict, com: dict) -> dict:
        cursor = ""
        for _ in range(100):
            parametros = {"expediente_ref": exp, "limite": 20}
            if cursor:
                parametros["cursor"] = cursor
            ruta = f"{API}/expedientes/comunicaciones?{urllib.parse.urlencode(parametros)}"
            pagina = self.rrhh.exigir("consulta de comunicación", "GET", ruta)
            for fila in pagina.get("comunicaciones") or []:
                if fila.get("comunicacion_ref") != com["comunicacion_ref"]:
                    continue
                if (fila.get("expediente_ref") != exp or
                        fila.get("organizacion_ref") != sel["organizacion_ref"] or
                        fila.get("llamamiento_ref") != sel["llamamiento_ref"] or
                        fila.get("recibo_comunicacion_ref") != com["recibo_ref"] or
                        fila.get("version") != 2 or
                        fila.get("estado") != "registrada_localmente" or
                        fila.get("estado_respuesta") not in ("sin_respuesta", "registrada") or
                        not isinstance(fila.get("registrada_en"), str)):
                    raise RuntimeError("la comunicación consultada no coincide con el recibo")
                return fila
            siguiente = pagina.get("siguiente_cursor") or ""
            if not siguiente:
                break
            if siguiente == cursor:
                raise RuntimeError("la consulta de comunicaciones repitió el cursor")
            cursor = siguiente
        raise RuntimeError("la comunicación no aparece en la consulta autorizada")

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


def confirmar_destino(destino: str) -> bool:
    if not sys.stdin.isatty():
        raise ErrorDestino("--ejecutar exige una terminal interactiva")
    try:
        return input(f"Para escribir en {destino}, teclee {destino}: ") == destino
    except EOFError:
        return False


def main() -> int:
    a = argparse.ArgumentParser(description="Siembra expedientes de ejemplo por la API de VEC.")
    a.add_argument("--destino", choices=("clon", "principal"), required=True)
    a.add_argument("--puerto-interno", type=int, required=True)
    a.add_argument("--huella-servidor-sha256", required=True)
    a.add_argument("--casos", required=True, help="fichero JSON de casos dentro del contenedor")
    a.add_argument("--ejecutar", action="store_true", help="crea o completa expedientes tras confirmar el destino")
    o = a.parse_args()
    try:
        if not 1 <= o.puerto_interno <= 65535:
            raise ValueError("el puerto interno debe estar entre 1 y 65535")
        huella = validar_huella(o.huella_servidor_sha256)
    except ValueError as e:
        a.error(str(e))
    if o.ejecutar and not sys.stdin.isatty():
        a.error("--ejecutar exige una terminal interactiva")
    try:
        with open(o.casos, encoding="utf-8") as fichero:
            datos = json.load(fichero)
    except (OSError, UnicodeError, ValueError) as e:
        raise ErrorDestino("no se pudo leer el fichero de casos") from e
    if not isinstance(datos, dict) or datos.get("esquema") != "vec.ct.sembrado-ejemplo.v1":
        a.error("fichero de casos con otro esquema")
    casos = datos["casos"]
    rrhh = Cliente(o.puerto_interno, "cliente", huella)
    s = Sembrador(rrhh, Cliente(o.puerto_interno, "intervencion", huella), datos["espacio_claves"])
    try:
        for c in casos:
            PLAN[c["objetivo"]]
        planes = [s.plan(c) for c in casos]
    except (RuntimeError, KeyError, TypeError, ValueError) as e:
        print(f"SEMBRADO-FALLO: no se ha escrito nada: {e}")
        return 2
    print(f"Destino: {o.destino}. Casos: {len(casos)}")
    for c, p in zip(casos, planes):
        print(f"{c['codigo']}  {c['objetivo']:<13} {p['centro'].get('etiqueta', '')[:34]:<34} {p['categoria']:<40} "
              f"{' → '.join(PLAN[c['objetivo']]) or 'alta'}")
    if not o.ejecutar:
        print("SEMBRADO-PLAN: no se ha escrito nada")
        return 0
    if not confirmar_destino(o.destino):
        print("SEMBRADO-FALLO: el destino no se confirmó; no se ha escrito nada")
        return 2
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
    try:
        sys.exit(main())
    except ErrorDestino as e:
        print(f"SEMBRADO-FALLO: {e}", file=sys.stderr)
        sys.exit(2)
