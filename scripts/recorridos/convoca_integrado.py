#!/usr/bin/env python3
"""convoca_readonly."""

from __future__ import annotations

import argparse
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from hashlib import sha256
import json
import os
from pathlib import Path
import re
import sys
import stat
from typing import Any
from urllib.parse import urlsplit

SCRIPTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPTS))
from revision_web.auditoria import auditar_dom_y_estado  # noqa: E402
from revision_web.navegador import guardar_captura  # noqa: E402

CAPACIDADES = ("publico.listado", "publico.detalle", "externo.mi_bolsa", "externo.preferencias")
API_PUBLICA = "/api/publico/bolsa/convocatorias"
API_BOLSA = "/api/vec/bolsa/mi-bolsa"
API_PREFERENCIAS = "/api/vec/usuarios/area-personal/mis-preferencias"
DEPENDENCIAS_AUXILIARES = ("/api/vec/usuarios/area-personal/mi-imagen", "/api/vec/usuarios/area-personal/mis-correos")
IDENTIFICADOR = re.compile(r"[a-z0-9][a-z0-9-]{2,79}\Z")
TAMANOS = ((1440, 900), (390, 844))

# Los validadores pertenecen al consumidor real. No se mantiene aquí otro DTO.
CONSULTAR_JS = r"""async ({ruta, capacidad}) => {
  let resultado = {http: 0, contrato_valido: false};
  const moduloMontado = async (ruta, exportacion) => {
    const recurso = performance.getEntriesByType('resource').map(r => r.name).reverse()
      .find(nombre => new URL(nombre).origin === location.origin && new URL(nombre).pathname === ruta);
    if (!recurso) throw new Error('modulo_no_montado');
    const modulo = await import(recurso);
    const version = new URL(recurso).searchParams.get('v');
    resultado.modulo = {ruta, version: version && /^[A-Za-z0-9_.-]{1,128}$/.test(version) ? version : null,
      exportacion, tipo: typeof modulo[exportacion]};
    if (typeof modulo[exportacion] !== 'function') throw new Error('exportacion_no_montada');
    return modulo[exportacion];
  };
  try {
    const validarMiBolsa = capacidad === 'externo.mi_bolsa'
      ? await moduloMontado('/area-personal/contrato.js', 'validarRespuestaMiBolsa') : null;
    const crearPreferencias = capacidad === 'externo.preferencias'
      ? await moduloMontado('/area-personal/cliente-http.js', 'crearClientePreferencias') : null;
    const respuesta = await fetch(ruta, {
      method: 'GET', credentials: capacidad.startsWith('publico.') ? 'omit' : 'same-origin',
      cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer',
      headers: {Accept: 'application/json'}, signal: AbortSignal.timeout(10000)
    });
    resultado.http = respuesta.status;
    if (!respuesta.headers.get('content-type')?.includes('application/json')) return resultado;
    const lector = respuesta.body.getReader();
    const partes = []; let total = 0;
    for (;;) {
      const {done, value} = await lector.read(); if (done) break;
      total += value.length;
      if (total > 524288) { await lector.cancel(); return {...resultado, fallo: 'tamano_respuesta'}; }
      partes.push(value);
    }
    const bytes = new Uint8Array(total); let offset = 0;
    for (const parte of partes) { bytes.set(parte, offset); offset += parte.length; }
    const datos = JSON.parse(new TextDecoder().decode(bytes));
    if (respuesta.status !== 200) {
      const codigo = datos?.error?.codigo;
      if (typeof codigo === 'string' && /^[a-z][a-z0-9_]{0,99}$/.test(codigo)) resultado.codigo = codigo;
      return resultado;
    }
    if (capacidad.startsWith('publico.')) {
      if (datos?.fuente?.demostracion !== false) return {...resultado, fallo: 'fuente_no_gobernada'};
      const contrato = globalThis.VECBolsaContratoV2;
      if (capacidad === 'publico.listado') {
        contrato.validarListado(datos);
        resultado.identificadores = datos.convocatorias.map(c => c.identificador_publico);
        resultado.total = datos.paginacion.total;
      } else {
        contrato.validarDetalle(datos);
        resultado.identificador = datos.convocatoria.identificador_publico;
        resultado.version = datos.convocatoria.version;
        resultado.huella_sha256 = datos.convocatoria.huella_sha256;
        resultado.requisitos = datos.requisitos.length;
        resultado.documentos = datos.documentos.length;
      }
    } else if (capacidad === 'externo.mi_bolsa') {
      const consulta = validarMiBolsa(datos);
      resultado.participaciones = consulta.participaciones.length;
    } else {
      const cliente = crearPreferencias({fetchImpl: async () =>
        new Response(JSON.stringify(datos), {status: 200, headers: {'Content-Type': 'application/json'}})});
      await cliente.cargar();
    }
    return {...resultado, contrato_valido: true};
  } catch { return {...resultado, contrato_valido: false, fallo: 'consulta_o_contrato'}; }
}"""


def url_local(valor: str) -> str:
    url = urlsplit(valor)
    if (url.scheme not in ("http", "https") or url.hostname not in ("127.0.0.1", "localhost", "::1")
        or url.username or url.password or url.query or url.fragment or url.path not in ("", "/")):
        raise ValueError("url_base_invalida")
    if not url.port:
        raise ValueError("puerto_requerido")
    return valor.rstrip("/")


def cargar_expectativas(ruta: Path) -> tuple[dict[str, Any], dict[str, list[int]]]:
    if ruta.stat().st_size > 16_384:
        raise ValueError("expectativas_tamano_excedido")
    datos = json.loads(ruta.read_text(encoding="utf-8"))
    if (not isinstance(datos, dict) or set(datos) - {"esquema", "datos", "capacidades", "dependencias_auxiliares"}
        or datos.get("esquema") != "vec.recorrido.convoca.v1" or datos.get("datos") != "sinteticos"
        or not isinstance(datos.get("capacidades"), dict)):
        raise ValueError("expectativas_esquema_invalido")
    if set(datos["capacidades"]) != set(CAPACIDADES):
        raise ValueError("expectativas_capacidades_invalidas")
    for expectativa in datos["capacidades"].values():
        if not isinstance(expectativa, dict) or set(expectativa) - {"estado", "http", "codigos_h6"}:
            raise ValueError("expectativa_invalida")
        if expectativa.get("estado") not in ("disponible", "cerrada"):
            raise ValueError("expectativa_estado_invalido")
        estados = expectativa.get("http", [])
        if expectativa["estado"] == "cerrada" and (not isinstance(estados, list) or not estados or any(type(s) is not int or s not in (401, 403, 404, 503) for s in estados)):
            raise ValueError("expectativa_http_invalido")
        if any(not isinstance(c, str) or not re.fullmatch(r"[a-z][a-z0-9_]{0,99}", c) for c in expectativa.get("codigos_h6", [])):
            raise ValueError("expectativa_codigo_h6_invalido")
    auxiliares = datos.get("dependencias_auxiliares", {})
    if (not isinstance(auxiliares, dict) or set(auxiliares) - set(DEPENDENCIAS_AUXILIARES)
        or any(not isinstance(estados, list) or not estados or any(type(s) is not int or s not in (401, 403, 404, 503) for s in estados) for estados in auxiliares.values())):
        raise ValueError("dependencias_auxiliares_invalidas")
    return datos["capacidades"], auxiliares


def clasificar(capacidad: str, observado: dict[str, Any], expectativa: dict[str, Any]) -> dict[str, Any]:
    disponible = observado.get("http") == 200 and observado.get("contrato_valido") is True
    cerrado = observado.get("http") in expectativa.get("http", []) and expectativa["estado"] == "cerrada"
    estado = "consulta_disponible" if disponible else "dependencia_cerrada" if cerrado else "fallo"
    coincide = (disponible and expectativa["estado"] == "disponible") or cerrado
    # No se atribuye H6 por el estado HTTP: requiere su código exacto configurado.
    return {"capacidad": capacidad, "expectativa": expectativa["estado"], "estado": estado,
            "coincide": coincide, "flujo_funcional_completado": False,
            "dependencia_h6_identificada": cerrado and observado.get("codigo") in expectativa.get("codigos_h6", []),
            "observado": observado}


def resumir_auditoria(page: Any, context: Any) -> dict[str, Any]:
    auditoria, hallazgos = auditar_dom_y_estado(page, context)
    almacen = auditoria.get("almacenamiento", {})
    return {"hallazgos": [h["codigo"] for h in hallazgos],
            "desbordamiento": auditoria.get("desbordamiento_horizontal", {}).get("existe"),
            "controles_sin_nombre": len(auditoria.get("controles_sin_nombre", [])),
            "estado_navegador": any(bool(almacen.get(c)) for c in ("local", "sesion", "indexeddb", "cache", "cookie_documento", "cookies_contexto"))}


def registrar_http(response: Any) -> dict[str, Any]:
    return {"ruta": urlsplit(response.url).path, "http": response.status,
            "metodo": getattr(getattr(response, "request", None), "method", "GET"),
            "set_cookie": "set-cookie" in response.all_headers()}


def registrar_fallo_red(request: Any) -> dict[str, str]:
    codigo = request.failure
    return {"ruta": urlsplit(request.url).path, "metodo": request.method,
            "codigo": codigo if isinstance(codigo, str) and re.fullmatch(r"net::ERR_[A-Z_]{1,64}", codigo) else "red_sin_respuesta"}


def registrar_consola(message: Any) -> dict[str, Any]:
    ubicacion = message.location
    registro = {"ruta": urlsplit(ubicacion.get("url", "")).path, "codigo": "consola_error"}
    # Se reconoce transitoriamente el diagnóstico de recurso de Chrome; no se guarda su texto.
    recurso = re.fullmatch(r"Failed to load resource: the server responded with a status of ([0-9]{3}) \([^\r\n)]{1,64}\)", message.text)
    if recurso and ubicacion.get("lineNumber", 0) == 0 and ubicacion.get("columnNumber", 0) == 0:
        registro.update(codigo="http_recurso", http=int(recurso.group(1)))
    return registro


def consola_inesperada(vista: dict[str, Any], cierres: dict[str, list[int]]) -> int:
    disponibles = Counter((r["ruta"], r["http"]) for r in vista.get("red", [])
                          if r.get("metodo") == "GET" and r["http"] in cierres.get(r["ruta"], []))
    registros = vista.get("consola_error", [])
    exentos = 0
    for registro in registros:
        llave = (registro.get("ruta"), registro.get("http"))
        if registro.get("codigo") == "http_recurso" and disponibles[llave] > 0:
            disponibles[llave] -= 1
            exentos += 1
    return max(vista.get("errores_consola", 0), len(registros)) - exentos


def validar_material_externo(base: str, certificado: Path | None, clave: Path | None) -> list[dict[str, str]]:
    if certificado is None and clave is None:
        return []
    if certificado is None or clave is None or urlsplit(base).scheme != "https":
        raise ValueError("material_externo_invalido")
    base = url_local(base)
    for ruta in (certificado, clave):
        if not ruta.is_absolute() or ".." in ruta.parts:
            raise ValueError("material_ruta_invalida")
        for componente in (ruta, *ruta.parents):
            if stat.S_ISLNK(componente.lstat().st_mode) or (componente.is_dir() and (componente / ".git").exists()):
                raise ValueError("material_ruta_invalida")
        datos = ruta.lstat()
        directorio = ruta.parent.lstat()
        if (not stat.S_ISREG(datos.st_mode) or datos.st_nlink != 1 or datos.st_uid != os.getuid() or stat.S_IMODE(datos.st_mode) != 0o600
            or directorio.st_uid != os.getuid() or stat.S_IMODE(directorio.st_mode) & 0o077):
            raise ValueError("material_permisos_invalidos")
    return [{"origin": base, "certPath": str(certificado), "keyPath": str(clave)}]


def crear_contexto(browser: Any, superficie: str, ancho: int, alto: int, args: argparse.Namespace) -> Any:
    opciones = {"viewport": {"width": ancho, "height": alto}, "locale": "es-ES", "timezone_id": "Europe/Madrid",
                "service_workers": "block", "reduced_motion": "reduce", "ignore_https_errors": False,
                "accept_downloads": args.preparacion_local and superficie == "publico"}
    if superficie == "externo" and args.material_externo:
        opciones["client_certificates"] = args.material_externo
    return browser.new_context(**opciones)


def preparar_local(page: Any, identificador: str, detalle: dict[str, Any], navegar: Any, capturar: Any) -> dict[str, Any]:
    navegar(f"/bolsa/preparacion/?convocatoria={identificador}&lang=es")
    page.locator("#paso-revision").wait_for(state="visible")
    page.locator("#lectura-confirmada").check()
    page.locator("#siguiente").click()
    page.locator("#paso-archivos").wait_for(state="visible")
    # El resumen debe conservar metadatos de este fichero, nunca su contenido.
    contenido = b"CONTENIDO_SINTETICO_NO_DEBE_APARECER_EN_EL_BORRADOR"
    nombre = "documento-sintetico.txt"
    page.locator("#archivos-locales").set_input_files({"name": nombre, "mimeType": "text/plain", "buffer": contenido})
    page.locator("#siguiente").click()
    page.locator("#revision-resumen").wait_for(state="visible")
    if not page.locator("#presentar").is_disabled() or not page.locator("#motivo-presentacion").inner_text().strip():
        raise RuntimeError("presentacion_no_cerrada")
    capturar("preparacion-resumen")
    with page.expect_download() as descarga:
        page.locator("#descargar").click()
    archivo = descarga.value
    if archivo.suggested_filename != f"borrador-solicitud-{identificador}.txt":
        raise RuntimeError("borrador_nombre_invalido")
    ruta = archivo.path()
    if ruta is None or Path(ruta).stat().st_size > 524_288:
        raise RuntimeError("borrador_tamano_invalido")
    bytes_descargados = Path(ruta).read_bytes()
    texto = bytes_descargados.decode("utf-8")
    if (any(valor not in texto for valor in (nombre, identificador, detalle["version"], detalle["huella_sha256"]))
        or contenido.decode() in texto):
        raise RuntimeError("borrador_contenido_invalido")
    page.locator("#anterior").click()
    page.locator("#paso-archivos").wait_for(state="visible")
    return {"estado": "borrador_local_descargado", "bytes": len(bytes_descargados),
            "sha256": sha256(bytes_descargados).hexdigest(), "archivos_seleccionados": 1,
            "presentacion_deshabilitada": True, "flujo_funcional_completado": False}


def recorrer(superficie: str, base: str, args: argparse.Namespace, expectativas: dict[str, Any]) -> dict[str, Any]:
    from playwright.sync_api import sync_playwright

    resultado: dict[str, Any] = {"superficie": superficie, "vistas": [], "capacidades": []}
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(executable_path=str(args.chrome), headless=True, timeout=args.timeout_ms)
        try:
            for ancho, alto in TAMANOS:
                context = crear_contexto(browser, superficie, ancho, alto, args)
                try:
                    red: list[dict[str, Any]] = []
                    bloqueos: list[dict[str, str]] = []
                    errores: list[str] = []
                    consola_error: list[dict[str, Any]] = []
                    fallos_red: list[dict[str, str]] = []

                    def limitar_red(route: Any) -> None:
                        request = route.request
                        url = urlsplit(request.url)
                        if request.method not in ("GET", "HEAD") or f"{url.scheme}://{url.netloc}" != base:
                            bloqueos.append({"metodo": request.method, "motivo": "efecto_o_destino_no_autorizado"})
                            route.abort()
                        else:
                            route.continue_()

                    def respuesta(response: Any) -> None:
                        red.append(registrar_http(response))

                    context.route("**/*", limitar_red)
                    page = context.new_page()
                    page.set_default_timeout(args.timeout_ms)
                    page.set_default_navigation_timeout(args.timeout_ms)
                    page.on("response", respuesta)
                    page.on("pageerror", lambda _: errores.append("javascript"))
                    page.on("console", lambda m: consola_error.append(registrar_consola(m)) if m.type == "error" else None)
                    page.on("requestfailed", lambda request: fallos_red.append(registrar_fallo_red(request)))

                    def navegar(ruta: str) -> None:
                        res = page.goto(base + ruta, wait_until="networkidle")
                        if res is None or res.status != 200:
                            raise RuntimeError("vista_http_invalido")

                    def consultar(capacidad: str, ruta: str) -> dict[str, Any]:
                        observado = page.evaluate(CONSULTAR_JS, {"ruta": ruta, "capacidad": capacidad})
                        comprobacion = clasificar(capacidad, observado, expectativas[capacidad])
                        comprobacion["ancho"] = ancho
                        resultado["capacidades"].append(comprobacion)
                        return observado

                    def capturar(nombre: str) -> None:
                        destino = args.salida / f"{superficie}-{nombre}-{ancho}.png"
                        guardar_captura(page, destino, pagina_completa=False)
                        resultado["vistas"].append({"vista": nombre, "ancho": ancho,
                                                    "captura": destino.name,
                                                    "auditoria": resumir_auditoria(page, context)})

                    if superficie == "publico":
                        navegar("/bolsa/")
                        listado = consultar("publico.listado", API_PUBLICA + "?pagina=1&tamano=12")
                        capturar("listado")
                        identificadores = listado.get("identificadores", [])
                        identificador = args.identificador_publico or next(iter(identificadores), "")
                        if identificador and identificador in identificadores:
                            enlace = page.locator(f'article[data-identificador="{identificador}"] .enlace-detalle')
                            if enlace.count() != 1:
                                raise RuntimeError("convocatoria_no_visible")
                            enlace.click()
                            detalle = consultar("publico.detalle", API_PUBLICA + "/" + identificador)
                            if detalle.get("contrato_valido"):
                                page.locator("#contenido-detalle").wait_for(state="visible")
                                if detalle.get("identificador") != identificador:
                                    raise RuntimeError("detalle_identificador_invalido")
                            capturar("detalle")
                            if args.preparacion_local and detalle.get("contrato_valido"):
                                resultado.setdefault("preparacion_local", []).append({"ancho": ancho,
                                    **preparar_local(page, identificador, detalle, navegar, capturar)})
                        else:
                            resultado["capacidades"].append({"capacidad": "publico.detalle", "ancho": ancho,
                                                            "estado": "no_recorrida", "coincide": False,
                                                            "flujo_funcional_completado": False})
                        if "preparacion_local" not in resultado:
                            resultado["preparacion_local"] = "no_recorrida"
                    else:
                        navegar("/area-personal/?vista=llamamientos")
                        consultar("externo.mi_bolsa", API_BOLSA)
                        consultar("externo.preferencias", API_PREFERENCIAS)
                        capturar("mi-bolsa")
                        for vista in ("perfil", "preferencias"):
                            navegar("/area-personal/?vista=" + vista)
                            capturar(vista)
                    resultado["vistas"].append({"ancho": ancho, "red": red, "bloqueos": bloqueos,
                                                "errores_javascript": errores.count("javascript"),
                                                "errores_consola": len(consola_error), "consola_error": consola_error,
                                                "fallos_red": fallos_red})
                finally:
                    context.close()
        finally:
            browser.close()
    return resultado


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--url-publico", required=True)
    parser.add_argument("--url-externo", required=True)
    parser.add_argument("--datos-sinteticos", action="store_true", required=True)
    parser.add_argument("--expectativas", type=Path, default=Path(__file__).with_name("convoca_expectativas.json"))
    parser.add_argument("--identificador-publico", default="")
    parser.add_argument("--preparacion-local", action="store_true")
    parser.add_argument("--chrome", type=Path, default=Path("/usr/bin/google-chrome"))
    parser.add_argument("--salida", type=Path, required=True)
    parser.add_argument("--timeout-ms", type=int, default=12_000)
    parser.add_argument("--tipo-ejecucion", choices=("aplicacion", "prueba_guion"), default="aplicacion")
    parser.add_argument("--certificado-externo", type=Path)
    parser.add_argument("--clave-externa", type=Path)
    args = parser.parse_args(argv)
    try:
        bases = {"publico": url_local(args.url_publico), "externo": url_local(args.url_externo)}
        if urlsplit(bases["publico"]).port == urlsplit(bases["externo"]).port:
            raise ValueError("puertos_procesos_no_distintos")
        if args.identificador_publico and not IDENTIFICADOR.fullmatch(args.identificador_publico):
            raise ValueError("identificador_publico_invalido")
        if not args.chrome.is_file() or not 500 <= args.timeout_ms <= 30_000:
            raise ValueError("chrome_o_timeout_invalido")
        expectativas, auxiliares = cargar_expectativas(args.expectativas)
        args.material_externo = validar_material_externo(bases["externo"], args.certificado_externo, args.clave_externa)
        # Directorio nuevo: nunca reemplaza evidencia de otra ejecución.
        args.salida.mkdir(parents=True, exist_ok=False, mode=0o700)
        with ThreadPoolExecutor(max_workers=2) as pool:
            trabajos = {s: pool.submit(recorrer, s, base, args, expectativas) for s, base in bases.items()}
            resultados = []
            for superficie, futuro in trabajos.items():
                try:
                    resultados.append(futuro.result())
                except Exception as error:
                    resultados.append({"superficie": superficie, "fallo": "recorrido_interrumpido", "tipo_error": type(error).__name__})
        comprobaciones = [c for r in resultados for c in r.get("capacidades", [])]
        vistas = [v for r in resultados for v in r.get("vistas", [])]
        fallo = any("fallo" in r for r in resultados) or len(comprobaciones) != 8 or any(not c["coincide"] for c in comprobaciones)
        if args.preparacion_local:
            publico = next((r for r in resultados if r["superficie"] == "publico"), {})
            preparaciones = publico.get("preparacion_local")
            fallo = fallo or not isinstance(preparaciones, list) or len(preparaciones) != len(TAMANOS)
        rutas_cerradas = {API_BOLSA: expectativas["externo.mi_bolsa"], API_PREFERENCIAS: expectativas["externo.preferencias"]}
        cierres = {ruta: expectativa["http"] for ruta, expectativa in rutas_cerradas.items() if expectativa["estado"] == "cerrada"}
        cierres.update(auxiliares)
        def respuesta_inesperada(res: dict[str, Any]) -> bool:
            esperado = rutas_cerradas.get(res["ruta"], {})
            cerrado_esperado = esperado.get("estado") == "cerrada" and res["http"] in esperado.get("http", [])
            cerrado_esperado = cerrado_esperado or res["http"] in auxiliares.get(res["ruta"], [])
            return res["set_cookie"] or (res["http"] >= 400 and not cerrado_esperado)
        fallo = fallo or any(v.get("auditoria", {}).get("hallazgos") or v.get("bloqueos") or v.get("errores_javascript")
                             or v.get("fallos_red") or consola_inesperada(v, cierres)
                             or any(respuesta_inesperada(res) for res in v.get("red", [])) for v in vistas)
        cerrado = any(c["estado"] == "dependencia_cerrada" for c in comprobaciones)
        codigo = 1 if fallo else 2 if cerrado else 0
        informe = {"esquema": "vec.recorrido.convoca.resultado.v1", "generado_en": datetime.now(timezone.utc).isoformat(),
                   "tipo_ejecucion": args.tipo_ejecucion, "codigo_salida": codigo,
                   "mtls_material_configurado": bool(args.material_externo), "mtls_acreditado": False,
                   "dependencias_auxiliares_cerradas": [res for v in vistas for res in v.get("red", [])
                                                        if res["http"] in auxiliares.get(res["ruta"], [])],
                   "consultas_conformes": not fallo and not cerrado,
                   "flujo_funcional_completado": False, "registro_acreditado": False,
                   "persistencia_acreditada": False, "resultados": resultados}
        (args.salida / "resultado.json").write_text(json.dumps(informe, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        print(json.dumps({k: informe[k] for k in ("codigo_salida", "tipo_ejecucion", "consultas_conformes", "flujo_funcional_completado")}, ensure_ascii=False))
        return codigo
    except (ValueError, OSError, json.JSONDecodeError):
        parser.error("configuracion_invalida")
    return 3


if __name__ == "__main__":
    raise SystemExit(main())
