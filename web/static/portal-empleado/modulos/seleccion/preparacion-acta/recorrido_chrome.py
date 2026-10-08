"""Recorrido sintético local; requiere sandbox sin red externa y Chrome del sistema."""
import base64
import hashlib
import importlib.util
import json
import os
import sys
import threading
from http.server import ThreadingHTTPServer
from pathlib import Path
from playwright.sync_api import sync_playwright

RAIZ = Path(__file__).resolve().parents[6]


def main():
    salida = Path(os.environ["VEC_S5_ACTA_SCRATCH"])
    servidor_fuente = Path(os.environ.get("VEC_S5_ACTA_SERVER", str(RAIZ / "scripts/servir_preparacion_rrhh.py")))
    spec = importlib.util.spec_from_file_location("servidor_vec", servidor_fuente)
    servicio = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(servicio)
    recursos, entrada = servicio.cargar_recursos(RAIZ / "web/static", "selectivos-acta-visor")
    assert not any("testdata" in r or "recorrido" in r or "escenario" in r for r in recursos)
    original = (Path(__file__).parent / "testdata/sesion-propuesta.json").read_bytes()
    dto = json.loads(original)
    with ThreadingHTTPServer(("127.0.0.1", 0), servicio.handler_para(recursos)) as servidor:
        hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        origen = f"http://127.0.0.1:{servidor.server_port}"
        try:
            with sync_playwright() as pw:
                focal_volver = '--volver' in sys.argv
                navegador = pw.chromium.launch(executable_path=os.environ.get("VEC_S5_ACTA_CHROME", "/usr/bin/google-chrome"), headless=True,
                    ignore_default_args=["--disable-back-forward-cache"] if focal_volver else None,
                    args=["--no-sandbox", "--disable-dev-shm-usage", "--disable-background-networking"])
                contexto = navegador.new_context(accept_downloads=True, viewport={"width": 1440, "height": 900})
                pagina = contexto.new_page()
                errores, peticiones, dialogos = [], [], []
                pagina.on("pageerror", lambda e: errores.append(str(e)))
                pagina.on("request", lambda r: peticiones.append((r.method, r.url)))
                pagina.on("dialog", lambda d: (dialogos.append(d.message), d.dismiss()))
                restauraciones = []
                if focal_volver:
                    pagina.expose_function("reportarRestauracion", lambda persisted, trusted: restauraciones.append({"persisted": bool(persisted), "trusted": bool(trusted)}))
                    pagina.add_init_script("window.__documentoPrueba = Math.random(); window.addEventListener('pageshow', e => window.reportarRestauracion(e.persisted, e.isTrusted));")
                assert pagina.goto(origen + entrada + "?lang=es&vista=acta#revision").status == 200
                archivo = pagina.locator("#acta-archivo")

                def abrir(datos, nombre="salida.json"):
                    archivo.set_input_files({"name": nombre, "mimeType": "application/json", "buffer": datos})
                    pagina.wait_for_function("() => !document.querySelector('#acta-descargar').disabled")

                if focal_volver:
                    abrir(original)
                    pagina.locator("#acta-idioma").select_option("en")
                    pagina.wait_for_function("() => document.documentElement.lang === 'en'")
                    documento_anterior = pagina.evaluate("window.__documentoPrueba")
                    assert pagina.goto(origen + "/textos/idiomas.json").status == 200
                    pagina.go_back(wait_until="domcontentloaded")
                    pagina.wait_for_function("anterior => window.__documentoPrueba !== anterior && document.querySelector('#acta-archivo') && !document.querySelector('#acta-archivo').disabled", arg=documento_anterior)
                    assert pagina.locator("#acta-resultados").inner_text() == ""
                    assert pagina.locator("#acta-descargar").is_disabled()
                    assert archivo.input_value() == ""
                    cache_observada = any(e["persisted"] and e["trusted"] for e in restauraciones)
                    if cache_observada:
                        assert pagina.locator("html").get_attribute("lang") == "en"
                        assert "lang=en" in pagina.url
                    abrir(original)
                    with pagina.expect_download() as espera:
                        pagina.locator("#acta-descargar").click()
                    espera.value.save_as(salida / "original-tras-volver.json")
                    assert (salida / "original-tras-volver.json").read_bytes() == original
                    # Ejercicio sintético separado: el listener debe sobrevivir al AbortController.
                    pagina.locator("#acta-idioma").select_option("en")
                    pagina.wait_for_function("() => document.documentElement.lang === 'en'")
                    documento_anterior = pagina.evaluate("window.__documentoPrueba")
                    pagina.evaluate("window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted:true})); window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted:true}));")
                    pagina.wait_for_function("anterior => window.__documentoPrueba !== anterior && document.querySelector('#acta-archivo') && !document.querySelector('#acta-archivo').disabled", arg=documento_anterior)
                    assert pagina.locator("html").get_attribute("lang") == "en" and "lang=en" in pagina.url
                    assert pagina.locator("#acta-resultados").inner_text() == "" and archivo.input_value() == ""
                    assert pagina.locator("#acta-descargar").is_disabled()
                    abrir(original)
                    with pagina.expect_download() as espera:
                        pagina.locator("#acta-descargar").click()
                    espera.value.save_as(salida / "original-tras-ciclo-sintetico.json")
                    assert (salida / "original-tras-ciclo-sintetico.json").read_bytes() == original
                    assert not errores and not dialogos, (errores, dialogos)
                    assert all(m == "GET" and u.startswith(origen + "/") for m, u in peticiones), peticiones
                    assert contexto.cookies() == []
                    assert pagina.evaluate("localStorage.length === 0 && sessionStorage.length === 0")
                    navegador.close()
                    print(json.dumps({"chrome_volver": "OK", "pageshow_persisted_observado": cache_observada,
                        "events_pageshow": restauraciones, "ciclo_persistido_sintetico": "OK", "descarga_identica": True, "original_sha256": hashlib.sha256(original).hexdigest()}))
                    return
                abrir(original)
                assert dto["preparacion"]["material_propuesto"]["orden_dia_propuesto"][0]["texto_propuesto"] in pagina.locator("#acta-resultados").inner_text()
                assert "Sin fecha propuesta" in pagina.locator("#acta-resultados").inner_text()
                assert "Referencia y huella aportadas · Sin cotejar" in pagina.locator("#acta-resultados").inner_text()
                assert pagina.locator("#acta-resultados details[open]").count() == 0
                assert pagina.evaluate("document.documentElement.scrollHeight <= innerHeight && document.documentElement.scrollWidth <= innerWidth")
                assert pagina.locator("#espacio-trabajo").evaluate("n => n.scrollHeight > n.clientHeight")
                pagina.screenshot(path=str(salida / "acta-1440.png"))
                pagina.locator("#espacio-trabajo").evaluate("n => { const panel = document.querySelector('#acta-resultados .rejilla-dos'); n.scrollTop += panel.getBoundingClientRect().top - n.getBoundingClientRect().top; }")
                pagina.screenshot(path=str(salida / "acta-propuestas-1440.png"))
                with pagina.expect_download() as espera:
                    pagina.locator("#acta-descargar").click()
                espera.value.save_as(salida / "original-descargado.json")
                assert (salida / "original-descargado.json").read_bytes() == original
                pagina.locator("#acta-idioma").select_option("en")
                pagina.wait_for_function("() => document.documentElement.lang === 'en'")
                assert "Proposed agreements" in pagina.locator("#acta-resultados").inner_text()
                assert pagina.locator("#acta-descargar").is_enabled()
                assert "lang=en&vista=acta#revision" in pagina.url
                assert pagina.reload().status == 200
                pagina.wait_for_function("() => document.documentElement.lang === 'en' && !document.querySelector('#acta-archivo').disabled")
                assert pagina.locator("#acta-resultados").inner_text() == ""
                abrir(original)
                assert "Proposed agreements" in pagina.locator("#acta-resultados").inner_text()
                pagina.locator("#espacio-trabajo").evaluate("n => n.scrollTop = 0")
                pagina.screenshot(path=str(salida / "acta-1440-en.png"))
                pagina.locator("#acta-idioma").select_option("es")
                pagina.wait_for_function("() => document.documentElement.lang === 'es'")
                for ancho, alto in [(390, 844), (720, 450), (320, 700)]:
                    pagina.set_viewport_size({"width": ancho, "height": alto})
                    assert pagina.evaluate("document.documentElement.scrollWidth <= innerWidth")
                    pagina.locator("#espacio-trabajo").evaluate("n => n.scrollTop = 0")
                    if ancho == 390:
                        pagina.screenshot(path=str(salida / "acta-390.png"))
                        pagina.get_by_role("heading", name="Orden del día propuesto").scroll_into_view_if_needed()
                        pagina.screenshot(path=str(salida / "acta-propuestas-390.png"))
                pagina.locator("#acta-ayuda").click()
                assert pagina.locator("#acta-ayuda-contenido").is_visible()
                pagina.locator("#acta-ayuda").click()
                assert pagina.locator("#acta-ayuda-contenido").is_hidden()
                for ancho in [1440, 390]:
                    pagina.set_viewport_size({"width": ancho, "height": 900})
                    archivo.focus()
                    for _ in range(10):
                        pagina.keyboard.press("Tab")
                        pagina.wait_for_timeout(40)
                        assert pagina.evaluate("""() => {
                          const n = document.activeElement, r = n.getBoundingClientRect();
                          if (n === document.body) return true;
                          const superior = document.querySelector('.cabecera-portal').getBoundingClientRect().bottom;
                          return r.left >= 0 && r.right <= innerWidth && r.top >= (n.closest('#espacio-trabajo') ? superior : 0) && r.bottom <= innerHeight;
                        }"""), pagina.evaluate("({html:document.activeElement.outerHTML,rect:document.activeElement.getBoundingClientRect().toJSON()})")
                vacio = json.loads(original)
                m = vacio["preparacion"]["material_propuesto"]
                m["orden_dia_propuesto"] = []
                m["acuerdos_propuestos"] = []
                for campo in ["orden_dia_propuesto", "acuerdos_propuestos"]:
                    vacio["preparacion"]["pendientes"].append({"campo": campo, "codigo": "material_ausente"})
                    vacio["mensajes"].append({"campo": campo, "codigo": "material_ausente", "mensaje": ""})
                abrir(json.dumps(vacio).encode())
                assert "No hay puntos propuestos" in pagina.locator("#acta-resultados").inner_text()
                payload = '<img src="https://externo.invalid/fuga" onerror="window.__xss=1">'
                malicioso = json.loads(original)
                malicioso["preparacion"]["material_propuesto"]["orden_dia_propuesto"][0]["texto_propuesto"] = payload
                abrir(json.dumps(malicioso).encode())
                assert payload in pagina.locator("#acta-resultados").inner_text()
                assert pagina.locator("#acta-resultados img, #acta-resultados a").count() == 0
                assert pagina.evaluate("window.__xss === undefined")
                for datos in [b'{', original.replace(b'"estado": "borrador_propuesto"', b'"estado": "firmada"'), b'x' * (4 * 1024 * 1024 + 1)]:
                    archivo.set_input_files({"name": "invalido.json", "mimeType": "application/json", "buffer": datos})
                    pagina.wait_for_function("() => document.querySelector('#acta-archivo').getAttribute('aria-invalid') === 'true'")
                    assert pagina.locator("#acta-resultados").inner_text() == ""
                    assert pagina.locator("#acta-descargar").is_disabled()
                    assert pagina.locator("#acta-estado").get_attribute("role") == "alert"
                    assert pagina.evaluate("document.activeElement.id") == "acta-estado"
                pagina.evaluate("""() => {
                  const original = File.prototype.arrayBuffer;
                  File.prototype.arrayBuffer = async function() {
                    if (this.name === 'lento.json') await new Promise(resolve => { window.__liberarLectura = resolve; });
                    return original.call(this);
                  };
                }""")
                archivo.set_input_files({"name": "lento.json", "mimeType": "application/json", "buffer": original})
                pagina.wait_for_function("() => typeof window.__liberarLectura === 'function'")
                abrir(json.dumps(vacio).encode())
                pagina.evaluate("window.__liberarLectura()")
                pagina.wait_for_timeout(100)
                assert "No hay puntos propuestos" in pagina.locator("#acta-resultados").inner_text()
                archivo.set_input_files({"name": "lento.json", "mimeType": "application/json", "buffer": original})
                pagina.wait_for_function("() => document.querySelector('#acta-estado').textContent.includes('Leyendo')")
                pagina.locator("#acta-cerrar").click()
                pagina.evaluate("window.__liberarLectura()")
                pagina.wait_for_timeout(100)
                assert pagina.locator("#acta-resultados").inner_text() == ""
                assert pagina.locator("#acta-descargar").is_disabled()
                assert pagina.evaluate("document.activeElement.id") == "acta-archivo"
                abrir(original)
                assert pagina.reload().status == 200
                pagina.wait_for_function("() => !document.querySelector('#acta-archivo').disabled")
                assert pagina.locator("#acta-resultados").inner_text() == ""
                assert pagina.locator("#acta-descargar").is_disabled()
                abrir(original)
                recursos.pop("/textos/en/selectivos-acta-visor.json")
                pagina.locator("#acta-idioma").select_option("en")
                pagina.wait_for_function("() => document.documentElement.lang === 'es' && document.querySelector('#acta-estado').textContent.includes('idioma elegido')")
                assert "lang=es&vista=acta#revision" in pagina.url
                assert pagina.locator("#acta-archivo").is_enabled()
                assert pagina.locator("#acta-descargar").is_enabled()
                assert "Orden del día propuesto" in pagina.locator("#acta-resultados").inner_text()
                recursos.pop("/textos/es/selectivos-acta-visor.json")
                pagina.locator("#acta-idioma").select_option("en")
                pagina.wait_for_function("() => document.querySelector('#acta-estado').textContent.includes('No se ha podido cambiar el idioma')")
                assert pagina.locator("html").get_attribute("lang") == "es" and "lang=es&vista=acta#revision" in pagina.url
                assert pagina.locator("#acta-resultados").inner_text() != ""
                assert pagina.locator("#acta-descargar").is_enabled()
                assert not errores and not dialogos, (errores, dialogos)
                assert all(m == "GET" and u.startswith(origen + "/") for m, u in peticiones), peticiones
                assert contexto.cookies() == []
                assert pagina.evaluate("localStorage.length === 0 && sessionStorage.length === 0")
                assert pagina.evaluate("indexedDB.databases()") == []
                navegador.close()
                resultado = {"chrome": "OK", "ancho": [1440, 390, 720, 320], "errores_js": errores, "solicitudes_externas": 0,
                    "http_negocio": 0, "original_sha256": hashlib.sha256(original).hexdigest(), "descarga_identica": True,
                    "idioma_url_y_refresco": "OK", "respaldo_y_error_sin_perder_archivo": "OK"}
                if os.environ.get("VEC_S5_ACTA_EXPORT") == "1":
                    resultado["capturas"] = {n: base64.b64encode((salida / n).read_bytes()).decode()
                        for n in ("acta-1440.png", "acta-1440-en.png", "acta-390.png", "acta-propuestas-1440.png", "acta-propuestas-390.png")}
                print(json.dumps(resultado))
        finally:
            servidor.shutdown()
            hilo.join(timeout=2)


if __name__ == "__main__":
    main()
