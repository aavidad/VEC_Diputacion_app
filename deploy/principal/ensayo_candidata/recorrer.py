#!/usr/bin/env python3
"""Recorrido HZ39: pantallas reales por perfil, evidencia privada y reanudación."""

from __future__ import annotations
import argparse
import hashlib
import json
import os
import re
import stat
import subprocess
import sys
import time
from pathlib import Path
from urllib.parse import urlsplit

REPO = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO / "scripts"))
from recorridos.convoca_integrado import (
    registrar_http,
    registrar_fallo_red,
    registrar_consola,
    resumir_auditoria,
    url_local,
)
from recorridos.intervencion.recorrer import (
    guardar_privado,
    abrir_directorio_privado,
    responder_sin_redireccion,
)
from revision_web.navegador import guardar_captura

ACTORES = {
    "aspirante": "candidato.p12",
    "centro": "centro-solicitante.p12",
    "ratificador": "centro-ratificador.p12",
    "rrhh": "cliente.p12",
    "intervencion": "intervencion.p12",
}
TIPOS = {"goto", "click", "fill", "select", "check", "text", "externo", "upload"}
MAX_JSON = 256 * 1024


class EntradaInvalida(ValueError):
    pass


class PendienteExterno(RuntimeError):
    pass


def leer_json(ruta: Path, *, privado: bool) -> object:
    if not ruta.is_file() or ruta.is_symlink() or ruta.stat().st_size > MAX_JSON:
        raise EntradaInvalida("fichero_json_invalido")
    if privado and (
        ruta.stat().st_uid != os.getuid() or stat.S_IMODE(ruta.stat().st_mode) != 0o600
    ):
        raise EntradaInvalida("fichero_json_no_privado")
    return json.loads(ruta.read_text(encoding="utf-8"))


def catalogo(ruta: Path) -> list[dict]:
    pasos = leer_json(ruta, privado=False)
    if not isinstance(pasos, list) or not pasos or len(pasos) > 100:
        raise EntradaInvalida("catalogo_invalido")
    ids = [p.get("id") for p in pasos]
    if any(
        not isinstance(i, str) or not re.fullmatch(r"[a-z0-9_-]{1,32}", i) for i in ids
    ) or len(ids) != len(set(ids)):
        raise EntradaInvalida("pasos_repetidos_o_invalidos")
    for p in pasos:
        if p.get("actor") not in {*ACTORES, "firmantes", "jefatura_firmante"} or any(
            d not in ids for d in p.get("depende_de", [])
        ):
            raise EntradaInvalida("actor_o_dependencia_invalida")
        if any(a.get("tipo") not in TIPOS for a in p.get("acciones", [])):
            raise EntradaInvalida("accion_no_admitida")
        for r in p.get("http", []) + p.get("lecturas", []):
            if not r.get("ruta", "").startswith("/") or r["ruta"].startswith("//"):
                raise EntradaInvalida("ruta_fuera_del_origen")
    return pasos


def resolver(valor: str, datos: dict) -> str:
    def cambiar(m):
        if m[1] not in datos:
            raise EntradaInvalida("dato_ausente:" + m[1])
        return str(datos[m[1]])

    return re.sub(r"\$([a-zA-Z_][a-zA-Z0-9_]*)", cambiar, valor)


def ruta_local(valor: str) -> str:
    if not valor.startswith("/") or valor.startswith("//") or urlsplit(valor).netloc:
        raise EntradaInvalida("ruta_fuera_del_origen")
    return valor


def identidades(carpeta: Path, claves: dict, origen: str) -> dict:
    reales = {}
    huellas = set()
    for actor, nombre in ACTORES.items():
        ruta = carpeta / nombre
        if not ruta.is_file() or ruta.is_symlink() or ruta.stat().st_size > (1 << 20):
            reales[actor] = None
            continue
        if (
            ruta.resolve().is_relative_to(REPO)
            or stat.S_IMODE(ruta.stat().st_mode) & 0o077
        ):
            raise EntradaInvalida("certificado_no_privado:" + actor)
        passphrase = claves.get(actor)
        if isinstance(passphrase, str) and any(
            c in passphrase for c in ("\n", "\r", "\0")
        ):
            raise EntradaInvalida("clave_p12_invalida:" + actor)
        if not isinstance(passphrase, str):
            reales[actor] = None
            continue
        # Validar únicamente el certificado público del P12, sin extraer
        # la clave ni escribir la contraseña en argumentos o temporales.
        pem = subprocess.run(
            [
                "openssl",
                "pkcs12",
                "-in",
                str(ruta),
                "-passin",
                "stdin",
                "-clcerts",
                "-nokeys",
            ],
            input=(passphrase + "\n").encode(),
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            timeout=10,
        )
        if pem.returncode:
            raise EntradaInvalida("p12_no_legible:" + actor)
        der = subprocess.run(
            ["openssl", "x509", "-outform", "DER"],
            input=pem.stdout,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            timeout=10,
        )
        if der.returncode:
            raise EntradaInvalida("certificado_no_legible:" + actor)
        digest = hashlib.sha256(der.stdout).hexdigest()
        if digest in huellas:
            raise EntradaInvalida("certificado_repetido:" + actor)
        huellas.add(digest)
        reales[actor] = {
            "origin": origen,
            "pfxPath": str(ruta),
            "passphrase": passphrase,
        }
    return reales


def escribir(salida: Path, nombre: str, valor: dict) -> None:
    guardar_privado(
        salida / nombre,
        (json.dumps(valor, ensure_ascii=False, indent=2) + "\n").encode(),
    )


def campo(datos: object, camino: str) -> object:
    for c in camino.split("."):
        if not isinstance(datos, dict) or c not in datos:
            raise EntradaInvalida("referencia_respuesta_ausente")
        datos = datos[c]
    return datos


def accion(page, a: dict, datos: dict, origen: str, externo=None) -> None:
    tipo = a["tipo"]
    if tipo == "externo":
        if externo is None or not externo():
            raise PendienteExterno("firma_y_plan_ADMIN_a_cargo_de_H")
        return
    if tipo == "goto":
        page.goto(
            origen + ruta_local(resolver(a["valor"], datos)),
            wait_until="domcontentloaded",
        )
        return
    elemento = page.locator(resolver(a["selector"], datos))
    if tipo == "check" and a.get("todos"):
        for i in range(elemento.count()):
            elemento.nth(i).check()
        return
    if elemento.count() != 1:
        raise EntradaInvalida("selector_no_unico")
    if tipo == "click":
        elemento.click()
    elif tipo == "fill":
        if not a.get("si_editable") or elemento.is_editable():
            elemento.fill(resolver(a["valor"], datos))
    elif tipo == "select":
        elemento.select_option(value=resolver(a["valor"], datos))
    elif tipo == "check":
        elemento.check()
    elif tipo == "text":
        from playwright.sync_api import expect

        expect(elemento).to_contain_text(resolver(a["valor"], datos))
    elif tipo == "upload":
        archivo = Path(resolver(a["valor"], datos)).expanduser().resolve(strict=True)
        if archivo.suffix.lower() not in {
            ".pdf",
            ".docx",
            ".doc",
            ".png",
            ".jpg",
            ".txt",
        } or archivo.stat().st_size > (16 << 20):
            raise EntradaInvalida("archivo_aportado_invalido")
        elemento.set_input_files(str(archivo))


def recibos(datos: object) -> list[dict]:
    resultado = []
    if isinstance(datos, dict):
        if isinstance(datos.get("recibo_ref"), str):
            resultado.append(
                {
                    k: v
                    for k, v in datos.items()
                    if k
                    in {
                        "recibo_ref",
                        "expediente_ref",
                        "peticion_ref",
                        "auditoria_ref",
                        "version",
                        "version_resultante",
                        "fecha",
                        "registrada_en",
                        "fase",
                        "estado",
                    }
                    and isinstance(v, (str, int, bool))
                }
            )
        for v in datos.values():
            resultado.extend(recibos(v))
    elif isinstance(datos, list):
        for v in datos[:100]:
            resultado.extend(recibos(v))
    return resultado[:100]


def comprobar_http(registro: dict, esperado: list[dict], datos: dict) -> None:
    for e in esperado:
        ruta = urlsplit(resolver(e["ruta"], datos)).path
        observadas = [
            r
            for r in registro["http"]
            if r["ruta"] == ruta and r["metodo"] == e["metodo"]
        ]
        if not observadas or not any(r["http"] in e["codigos"] for r in observadas):
            raise EntradaInvalida("http_esperado_ausente:" + e["ruta"])


class Ejecucion:
    def __init__(self, salida: Path, datos: dict):
        self.salida, self.datos = salida, datos
        self.actual = None
        self.pendientes = {}
        self.contador = 0
        self.videos = []
        self.inicios = {}
        self.respuestas = {}
        self.solo_lectura = False
        self.lecturas_permitidas = set()
        self.efectos_actuales = set()

    def red(self, response, actor: str):
        if self.actual is None or actor != self.actual.get("actor_navegador"):
            return
        r = registrar_http(response)
        tiempo = response.request.timing
        inicio = self.inicios.pop(response.request, None)
        r["tiempo_ms"] = round(
            (time.monotonic() - inicio) * 1000
            if inicio is not None
            else max(0, tiempo.get("responseStart", 0)),
            3,
        )
        self.actual["http"].append(r)
        if (r["metodo"], r["ruta"]) in self.efectos_actuales:
            if r["http"] >= 500:
                self.actual["resultado_incierto"] = True
            elif 200 <= r["http"] < 300:
                self.actual["efecto_confirmado_http"] = True
        if r["http"] >= 400:
            try:
                codigo = response.json().get("error", {}).get("codigo", "")
                r["codigo_error"] = (
                    codigo
                    if isinstance(codigo, str)
                    and re.fullmatch(r"[a-z0-9_:-]{1,128}", codigo)
                    else "error_sin_codigo"
                )
            except Exception:
                r["codigo_error"] = "respuesta_no_json"
        if "/api/" in r["ruta"] and response.ok:
            try:
                cuerpo = response.json()
            except Exception:
                return
            self.respuestas[(r["metodo"], r["ruta"])] = cuerpo
            self.actual["recibos"].extend(recibos(cuerpo))
            for x in self.datos.get("extraer", []):
                if x["ruta"] == r["ruta"] and x["metodo"] == r["metodo"]:
                    v = campo(cuerpo, x["campo"])
                    if not isinstance(v, (str, int)) or len(str(v)) > 256:
                        raise EntradaInvalida("referencia_respuesta_invalida")
                    self.datos["valores"][x["variable"]] = v

    def filtrar(self, route, origen: str):
        req = route.request
        self.inicios[req] = time.monotonic()
        if (
            self.solo_lectura
            and req.method not in {"GET", "HEAD"}
            and (req.method, urlsplit(req.url).path) not in self.lecturas_permitidas
        ):
            if self.actual is not None:
                self.actual["red_fallida"].append(
                    {
                        "metodo": req.method,
                        "ruta": urlsplit(req.url).path,
                        "codigo": "efecto_bloqueado_en_recuperacion",
                    }
                )
            route.abort()
            return
        if (
            self.actual is not None
            and req.method not in {"GET", "HEAD"}
            and "/api/" in urlsplit(req.url).path
        ):
            if (req.method, urlsplit(req.url).path) in self.efectos_actuales:
                self.actual["efecto_iniciado"] = True
            referencia = {
                "metodo": req.method,
                "ruta": urlsplit(req.url).path,
                "actor": self.actual["actor"],
                "paso": self.actual["id"],
            }
            try:
                body = req.post_data_json
                if isinstance(body, dict):
                    referencia.update(
                        {
                            k: body[k]
                            for k in ("clave_idempotencia", "version_esperada")
                            if k in body
                        }
                    )
                    referencia["solicitud_sha256"] = hashlib.sha256(
                        req.post_data.encode()
                    ).hexdigest()
            except Exception:
                referencia["solicitud_sha256"] = None
            self.contador += 1
            escribir(self.salida, f"intencion-{self.contador:04}.json", referencia)
        responder_sin_redireccion(route, origen)

    def firmas_completas(self, paso: str) -> bool:
        cuerpo = self.respuestas.get(
            ("POST", "/api/vec/contratacion-temporal/firmas-documento/consultas-v2"), {}
        )
        estado = cuerpo.get("data", {})
        if estado.get("verificacion_disponible") is not True:
            return False
        documentos = estado.get("documentos", [])
        requeridos = (
            {"informe_definitivo": 2}
            if paso == "07b"
            else {"informe_definitivo": 2, "resolucion": 3}
        )
        for nombre, cantidad in requeridos.items():
            elegidos = [d for d in documentos if d.get("documento") == nombre]
            if len(elegidos) != 1 or elegidos[0].get("completo") is not True:
                return False
            pasos = elegidos[0].get("pasos", [])
            if len(pasos) != cantidad or any(
                x.get("estado") != "firmado"
                or not x.get("recibo_ref")
                or not x.get("registrada_en")
                for x in pasos
            ):
                return False
            if len({x["recibo_ref"] for x in pasos}) != cantidad:
                return False
        return True

    def descargar(self, download):
        if self.actual is None:
            return
        carpeta = self.salida / "documentos"
        carpeta.mkdir(mode=0o700, exist_ok=True)
        self.contador += 1
        extension = Path(download.suggested_filename).suffix.lower()
        destino = (
            carpeta
            / f"documento-{self.contador:04}{extension if extension in {'.pdf', '.docx', '.doc', '.zip'} else '.bin'}"
        )
        download.save_as(str(destino))
        os.chmod(destino, 0o600)
        if destino.stat().st_size > (16 << 20):
            raise EntradaInvalida("descarga_demasiado_grande")
        self.actual.setdefault("documentos", []).append(
            {
                "archivo": str(destino.relative_to(self.salida)),
                "bytes": destino.stat().st_size,
                "sha256": hashlib.sha256(destino.read_bytes()).hexdigest(),
            }
        )


def ejecutar(
    args, pasos: list[dict], datos: dict, ident: dict, anterior: dict | None
) -> dict:
    from playwright.sync_api import sync_playwright

    salida = args.salida.expanduser().absolute()
    if salida.exists():
        raise EntradaInvalida("salida_ya_existe")
    salida.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    # Reutiliza el mismo control privado que los recorridos de Intervención.
    descriptor = abrir_directorio_privado(salida)
    os.close(descriptor)
    salida.mkdir(mode=0o700)
    e = Ejecucion(salida, datos)
    e.solo_lectura = args.modo == "recuperar"
    e.lecturas_permitidas = {
        (x["metodo"], urlsplit(resolver(x["ruta"], datos["valores"])).path)
        for p in pasos
        for x in p.get("lecturas", [])
        if x.get("solo_lectura") and x.get("metodo")
    }
    resultado = {
        "candidata": datos.get("candidata"),
        "modo": args.modo,
        "pasos": [],
        "completo": False,
        "valores": dict(datos["valores"]),
    }
    previos = {p["id"]: p for p in (anterior or {}).get("pasos", [])}
    with sync_playwright() as pw:
        browser = pw.chromium.launch(
            executable_path="/usr/bin/google-chrome", headless=not args.visible
        )
        contextos = {}
        try:
            for p in pasos:
                if args.pasos and p["id"] not in args.pasos.split(","):
                    continue
                if (
                    args.modo == "continuar"
                    and previos.get(p["id"], {}).get("estado") == "OK"
                ):
                    resultado["pasos"].append(previos[p["id"]])
                    continue
                pendientes = [
                    d
                    for d in p.get("depende_de", [])
                    if not any(
                        x["id"] == d and x["estado"] == "OK" for x in resultado["pasos"]
                    )
                ]
                actor = (
                    "rrhh"
                    if p["actor"] in {"firmantes", "jefatura_firmante"}
                    else p["actor"]
                )
                registro = {
                    "id": p["id"],
                    "actor": p["actor"],
                    "actor_navegador": actor,
                    "dueno": p["dueno"],
                    "estado": "PENDIENTE",
                    "http": [],
                    "recibos": [],
                    "errores_js": 0,
                    "red_fallida": [],
                }
                resultado["pasos"].append(registro)
                if pendientes and args.modo != "recuperar":
                    registro.update(
                        causa="dependencia_pendiente", depende_de=pendientes
                    )
                    continue
                if not ident.get(actor):
                    registro["causa"] = "certificado_o_clave_no_disponible"
                    continue
                anterior_paso = previos.get(p["id"], {})
                if (
                    args.modo == "continuar"
                    and anterior_paso.get("efecto_iniciado")
                    and anterior_paso.get("estado") != "OK"
                ):
                    registro["causa"] = "recuperar_efecto_anterior_antes_de_continuar"
                    escribir(salida, f"paso-{p['id']}.json", registro)
                    continue
                e.actual = registro
                e.efectos_actuales = set()
                inicio = time.monotonic()
                try:
                    e.efectos_actuales = {
                        (
                            x["metodo"],
                            urlsplit(resolver(x["ruta"], datos["valores"])).path,
                        )
                        for x in p.get("http", [])
                        if x["metodo"] not in {"GET", "HEAD"}
                        and not x.get("solo_lectura")
                    }
                    if actor not in contextos:
                        opciones = {
                            "client_certificates": [ident[actor]],
                            "ignore_https_errors": False,
                            "service_workers": "block",
                            "locale": "es-ES",
                            "timezone_id": "Europe/Madrid",
                            "viewport": {"width": 1440, "height": 900},
                            "accept_downloads": True,
                        }
                        if args.video:
                            video = salida / "video"
                            video.mkdir(mode=0o700, exist_ok=True)
                            opciones["record_video_dir"] = str(video)
                        c = browser.new_context(**opciones)
                        c.route(
                            "**/*", lambda route, o=args.base_url: e.filtrar(route, o)
                        )
                        c.on("response", lambda response, a=actor: e.red(response, a))
                        page = c.new_page()
                        page.set_default_timeout(args.timeout_ms)
                        page.on(
                            "pageerror",
                            lambda _error: (
                                e.actual.update(errores_js=e.actual["errores_js"] + 1)
                                if e.actual
                                else None
                            ),
                        )
                        page.on(
                            "requestfailed",
                            lambda req: (
                                e.actual["red_fallida"].append(registrar_fallo_red(req))
                                if e.actual
                                else None
                            ),
                        )
                        page.on(
                            "console",
                            lambda msg, a=actor: (
                                e.actual.setdefault("consola_error", []).append(
                                    registrar_consola(msg)
                                )
                                if msg.type == "error"
                                and e.actual
                                and e.actual.get("actor_navegador") == a
                                else None
                            ),
                        )
                        page.on("download", e.descargar)
                        contextos[actor] = (c, page)
                    c, page = contextos[actor]
                    page.bring_to_front()
                    if args.modo == "recuperar":
                        page.goto(
                            args.base_url
                            + ruta_local(resolver(p["ruta"], datos["valores"])),
                            wait_until="domcontentloaded",
                        )
                        acciones = p.get("acciones_recuperacion", [])
                        if acciones:
                            for a in acciones:
                                accion(page, a, datos["valores"], args.base_url)
                        else:
                            # Reutiliza sólo la apertura de la ficha ya definida en
                            # el tramo; jamás los botones que registran un efecto.
                            for a in p.get("acciones", []):
                                if a.get(
                                    "tipo"
                                ) == "click" and "data-ct-exp-abrir" in a.get(
                                    "selector", ""
                                ):
                                    accion(page, a, datos["valores"], args.base_url)
                            for a in p.get("lecturas", []):
                                if a.get(
                                    "tipo"
                                ) == "ui" and "data-ct-fiscalizacion-acceso" in a.get(
                                    "selector", ""
                                ):
                                    page.locator(
                                        resolver(a["selector"], datos["valores"])
                                    ).click()
                        esperado = [x for x in p.get("lecturas", []) if "metodo" in x]
                    else:
                        for a in p.get("acciones", []):
                            accion(
                                page,
                                a,
                                datos["valores"],
                                args.base_url,
                                lambda: e.firmas_completas(p["id"]),
                            )
                        esperado = p.get("http", [])
                    listo = p.get(
                        "esperar_recuperacion"
                        if args.modo == "recuperar"
                        else "esperar",
                        {},
                    )
                    if listo.get("selector"):
                        page.locator(
                            resolver(listo["selector"], datos["valores"])
                        ).wait_for(state="visible")
                        if listo.get("texto"):
                            from playwright.sync_api import expect

                            expect(
                                page.locator(
                                    resolver(listo["selector"], datos["valores"])
                                )
                            ).to_contain_text(
                                resolver(listo["texto"], datos["valores"])
                            )
                    comprobar_http(registro, esperado, datos["valores"])
                    registro["navegador"] = resumir_auditoria(page, c)
                    if (
                        any(r["set_cookie"] for r in registro["http"])
                        or registro["red_fallida"]
                        or registro.get("consola_error")
                        or any(r["http"] >= 400 for r in registro["http"])
                        or registro["errores_js"]
                        or registro["navegador"].get("estado_navegador")
                        or registro["navegador"].get("desbordamiento")
                    ):
                        raise EntradaInvalida("estado_navegador_no_valido")
                    if p["id"] in {"07b", "15"} and not e.firmas_completas(p["id"]):
                        raise PendienteExterno("firmas_no_completas")
                    if args.modo == "recuperar":
                        originales = {
                            r["recibo_ref"]
                            for r in previos.get(p["id"], {}).get("recibos", [])
                        }
                        observados = {r["recibo_ref"] for r in registro["recibos"]}
                        if originales and not originales.issubset(observados):
                            raise EntradaInvalida("recibo_original_no_recuperado")
                    if args.modo == "recuperar" and p["id"] == "14":
                        if {
                            x["sha256"]
                            for x in previos.get(p["id"], {}).get("documentos", [])
                        } != {x["sha256"] for x in registro.get("documentos", [])}:
                            raise EntradaInvalida("documentos_no_identicos")
                    registro["estado"] = "OK"
                except PendienteExterno as exc:
                    registro["causa"] = str(exc)
                except Exception as exc:
                    registro["estado"] = "FALLO"
                    registro["causa"] = (
                        str(exc)
                        if isinstance(exc, EntradaInvalida)
                        else type(exc).__name__
                    )
                finally:
                    registro["tiempo_ms"] = round((time.monotonic() - inicio) * 1000, 3)
                    if actor in contextos:
                        c, page = contextos[actor]
                        for ancho, alto in ((1440, 900), (390, 844)):
                            page.set_viewport_size({"width": ancho, "height": alto})
                            try:
                                imagen = salida / f"{p['id']}-{ancho}.png"
                                guardar_captura(page, imagen, pagina_completa=False)
                                os.chmod(imagen, 0o600)
                            except Exception as exc:
                                registro.setdefault("capturas_error", []).append(
                                    type(exc).__name__
                                )
                        page.set_viewport_size({"width": 1440, "height": 900})
                    resultado["valores"] = dict(datos["valores"])
                    escribir(salida, f"paso-{p['id']}.json", registro)
                    print(
                        json.dumps(
                            {
                                k: registro.get(k)
                                for k in ("id", "actor", "estado", "causa", "tiempo_ms")
                            },
                            ensure_ascii=False,
                        ),
                        flush=True,
                    )
        finally:
            for c, _page in contextos.values():
                c.close()
            browser.close()
    resultado["completo"] = len(resultado["pasos"]) == len(pasos) and all(
        p["estado"] == "OK" for p in resultado["pasos"]
    )
    escribir(salida, "resultado.json", resultado)
    return resultado


def main() -> int:
    os.umask(0o077)
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--base-url", required=True)
    p.add_argument("--certificados", type=Path, required=True)
    p.add_argument("--claves", type=Path, required=True)
    p.add_argument("--datos", type=Path, required=True)
    p.add_argument("--salida", type=Path, required=True)
    p.add_argument(
        "--catalogo", type=Path, default=Path(__file__).with_name("pasos_hz39.json")
    )
    p.add_argument(
        "--modo",
        choices=["comprobar", "ejecutar", "continuar", "recuperar"],
        default="comprobar",
    )
    p.add_argument("--anterior", type=Path)
    p.add_argument("--reinicio-acreditado", action="store_true")
    p.add_argument("--pasos")
    p.add_argument("--visible", action="store_true")
    p.add_argument("--video", action="store_true")
    p.add_argument("--timeout-ms", type=int, default=15000)
    a = p.parse_args()
    try:
        a.base_url = url_local(a.base_url)
        if urlsplit(a.base_url).scheme != "https":
            raise EntradaInvalida("https_requerido")
        pasos = catalogo(a.catalogo)
        datos = leer_json(a.datos, privado=True)
        claves = leer_json(a.claves, privado=True)
        if (
            not isinstance(datos, dict)
            or not isinstance(datos.get("valores"), dict)
            or not isinstance(claves, dict)
        ):
            raise EntradaInvalida("datos_invalidos")
        ident = identidades(
            a.certificados.expanduser().resolve(strict=True), claves, a.base_url
        )
        anterior = leer_json(a.anterior, privado=True) if a.anterior else None
        if a.modo in {"continuar", "recuperar"} and not anterior:
            raise EntradaInvalida("evidencia_anterior_requerida")
        if a.modo == "recuperar" and not a.reinicio_acreditado:
            raise EntradaInvalida("reinicio_no_acreditado")
        if anterior:
            if anterior.get("candidata") != datos.get("candidata"):
                raise EntradaInvalida("candidata_distinta")
            datos["valores"].update(anterior.get("valores", {}))
        if a.modo == "comprobar":
            print(
                json.dumps(
                    {
                        "estado": "PREPARADO",
                        "ejecutado": False,
                        "pasos": len(pasos),
                        "actores_disponibles": [k for k, v in ident.items() if v],
                        "bloqueos_contrato": [
                            x["id"] for x in pasos if x.get("bloqueo_contrato")
                        ],
                    }
                )
            )
            return 0
        return 0 if ejecutar(a, pasos, datos, ident, anterior)["completo"] else 1
    except Exception as exc:
        print(
            json.dumps(
                {
                    "estado": "NO_EJECUTADO",
                    "causa": str(exc)
                    if isinstance(exc, EntradaInvalida)
                    else type(exc).__name__,
                }
            )
        )
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
