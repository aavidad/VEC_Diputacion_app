"""Recorrido local: ejecutar en sandbox sin red externa y con Chrome del sistema."""
import hashlib
import json
import os
import sys
import threading
from http.server import ThreadingHTTPServer
from pathlib import Path
from playwright.sync_api import sync_playwright

RAIZ = Path(__file__).resolve().parents[6]
sys.path.insert(0, str(RAIZ))
from scripts.servir_preparacion_rrhh import cargar_recursos, handler_para


def main():
    salida = Path(os.environ["VEC_S2_SCRATCH"])
    completa = Path(__file__).parent / "testdata/preparacion-completa.json"
    incompleta = Path(__file__).parent / "testdata/preparacion-incompleta.json"
    original = completa.read_bytes()
    recursos, entrada = cargar_recursos(RAIZ / "web/static", "seleccion-bases-preparacion")
    assert not any("testdata" in r or "escenario.json" in r or "recorrido" in r for r in recursos)
    with ThreadingHTTPServer(("127.0.0.1", 0), handler_para(recursos)) as servidor:
        hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        origen = f"http://127.0.0.1:{servidor.server_port}"
        try:
            with sync_playwright() as pw:
                navegador = pw.chromium.launch(executable_path=os.environ.get("VEC_S2_CHROME", "/usr/bin/google-chrome"), headless=True, args=["--no-sandbox", "--disable-dev-shm-usage", "--disable-background-networking"])
                contexto = navegador.new_context(accept_downloads=True, viewport={"width": 1440, "height": 900})
                pagina = contexto.new_page()
                errores, peticiones, dialogos = [], [], []
                pagina.on("pageerror", lambda e: errores.append(str(e)))
                pagina.on("request", lambda r: peticiones.append(r.url))
                pagina.on("dialog", lambda d: (dialogos.append(d.message), d.dismiss()))
                if "--indice-404" in sys.argv or "--catalogo-404" in sys.argv:
                    indice_fallido = "--indice-404" in sys.argv
                    recursos.pop("/textos/idiomas.json" if indice_fallido else "/textos/es/seleccion-bases-preparacion.json")
                    idioma_respaldo = "es" if indice_fallido else "en"
                    assert pagina.goto(origen + entrada + "?lang=" + idioma_respaldo).status == 200
                    pagina.wait_for_function("() => document.querySelector('#bases-estado').textContent !== ''")
                    assert ("No se han podido cargar los textos" if indice_fallido else "could not be loaded") in pagina.locator("#bases-estado").inner_text()
                    assert pagina.locator("#bases-archivo").is_disabled()
                    assert pagina.locator("#bases-descargar").is_disabled()
                    titulo_respaldo = "Preparación de bases" if indice_fallido else "Preparing selection rules"
                    assert pagina.title() == titulo_respaldo
                    assert pagina.get_by_role("heading", name=titulo_respaldo).count() == 1
                    assert pagina.locator("html").get_attribute("lang") == idioma_respaldo
                    assert not errores and not dialogos, (errores, dialogos)
                    assert all(u.startswith(origen + "/") for u in peticiones)
                    navegador.close()
                    print(json.dumps({"chrome_fallback_404": "indice" if indice_fallido else "catalogo", "visor_deshabilitado": True, "aviso_desde_catalogo": True}))
                    return
                assert pagina.goto(origen + entrada + "?lang=es").status == 200
                archivo = pagina.locator("#bases-archivo")
                archivo.set_input_files(str(completa))
                pagina.locator("#bases-descargar").wait_for(state="visible")
                pagina.wait_for_function("() => !document.querySelector('#bases-descargar').disabled")
                assert pagina.locator("#bases-resultados").get_by_role("heading", name="Pendiente de resolver").count() == 1, pagina.locator("#bases-resultados").inner_text()
                assert "Propuesta sintética de bases" in pagina.locator("#bases-resultados").inner_text()
                assert pagina.evaluate("document.documentElement.scrollHeight <= innerHeight && document.documentElement.scrollWidth <= innerWidth")
                with pagina.expect_download() as espera:
                    pagina.locator("#bases-descargar").click()
                descarga = espera.value
                descarga.save_as(salida / "original-descargado.json")
                assert (salida / "original-descargado.json").read_bytes() == original
                pagina.screenshot(path=str(salida / "bases-1440.png"))
                # Un fallo al cambiar de idioma conserva material e idioma anterior.
                ruta_catalogo = "/textos/es/seleccion-bases-preparacion.json"
                catalogo = recursos.pop(ruta_catalogo)
                pagina.locator("#bases-idioma").select_option("en")
                pagina.wait_for_function("() => !document.querySelector('#bases-idioma').disabled")
                assert "No se han podido cargar los textos" in pagina.locator("#bases-estado").inner_text()
                assert pagina.locator("#bases-descargar").is_enabled()
                assert pagina.locator("#bases-idioma").input_value() == "es"
                assert archivo.get_attribute("aria-invalid") == "false"
                recursos[ruta_catalogo] = catalogo
                pagina.locator("#bases-idioma").select_option("en")
                pagina.wait_for_function("() => document.documentElement.lang === 'en'")
                assert pagina.get_by_role("heading", name="Outstanding checks").count() == 1
                assert pagina.locator("#bases-descargar").is_enabled()
                pagina.screenshot(path=str(salida / "bases-1440-en.png"))
                pagina.locator("#bases-idioma").select_option("es")
                pagina.wait_for_function("() => document.documentElement.lang === 'es'")
                pagina.set_viewport_size({"width": 390, "height": 844})
                assert pagina.evaluate("document.documentElement.scrollWidth <= innerWidth")
                pagina.screenshot(path=str(salida / "bases-390.png"))
                pagina.locator("#bases-ayuda").click()
                assert pagina.locator("#bases-ayuda-contenido").is_visible()
                pagina.locator("#bases-ayuda").click()
                assert pagina.locator("#bases-ayuda-contenido").is_hidden()
                # Ancho equivalente de trabajo al ampliar una ventana 1440 al 200 %.
                pagina.set_viewport_size({"width": 720, "height": 450})
                assert pagina.evaluate("document.documentElement.scrollWidth <= innerWidth")
                # Foco visible incluso con cabecera fija y desplazamiento interno.
                for ancho in [1440, 390]:
                    pagina.set_viewport_size({"width": ancho, "height": 900})
                    archivo.focus()
                    for _ in range(9):
                        pagina.keyboard.press("Tab")
                        pagina.wait_for_timeout(40)
                        assert pagina.evaluate("""() => {
                          const n = document.activeElement; const r = n.getBoundingClientRect();
                          if (n === document.body) return true;
                          const superior = document.querySelector('.cabecera-portal').getBoundingClientRect().bottom;
                          return r.left >= 0 && r.right <= innerWidth && r.top >= (n.closest('.cabecera-portal') || n.closest('.portal-lateral') ? 0 : superior) && r.bottom <= innerHeight;
                        }"""), pagina.evaluate("document.activeElement.outerHTML")
                archivo.set_input_files(str(incompleta))
                pagina.wait_for_function("() => !document.querySelector('#bases-descargar').disabled")
                assert "No se han aportado elementos" in pagina.locator("#bases-resultados").inner_text()
                # Cadenas aportadas sólo como texto; las rutas no originan solicitudes.
                malicioso = json.loads(original)
                payload = '<img src="https://externo.invalid/fuga" onerror="window.__xss=1">'
                malicioso["preparacion"]["material_propuesto"]["contenido"]["titulo"] = payload
                malicioso["preparacion"]["material_propuesto"]["contenido"]["documentos"][0]["url"] = "javascript:window.__xss=2"
                archivo.set_input_files({"name": "prueba-xss.json", "mimeType": "application/json", "buffer": json.dumps(malicioso).encode()})
                pagina.wait_for_function("() => !document.querySelector('#bases-descargar').disabled")
                assert payload in pagina.locator("#bases-resultados").inner_text()
                assert pagina.locator("#bases-resultados img, #bases-resultados a").count() == 0
                assert pagina.evaluate("window.__xss === undefined")
                for contenido in [b'{', original.replace(b'"estado": "pendiente"', b'"estado": "aprobado"')]:
                    archivo.set_input_files({"name": "invalido.json", "mimeType": "application/json", "buffer": contenido})
                    pagina.wait_for_function("() => document.querySelector('#bases-archivo').getAttribute('aria-invalid') === 'true'")
                    assert pagina.locator("#bases-resultados").inner_text() == ""
                    assert pagina.locator("#bases-descargar").is_disabled()
                    assert "Genere de nuevo" in pagina.locator("#bases-estado").inner_text()
                pagina.locator("#bases-idioma").select_option("en")
                pagina.wait_for_function("() => document.documentElement.lang === 'en'")
                assert "Cannot open the file" in pagina.locator("#bases-estado").inner_text()
                pagina.locator("#bases-cerrar").click()
                assert pagina.locator("#bases-archivo").input_value() == ""
                assert pagina.locator("#bases-descargar").is_disabled()
                # Ausencia del catálogo principal: respaldo de datos en el idioma pedido.
                catalogo = recursos.pop(ruta_catalogo)
                assert pagina.goto(origen + entrada + "?lang=en").status == 200
                pagina.wait_for_function("() => document.querySelector('#bases-estado').textContent.includes('could not be loaded')")
                assert pagina.locator("#bases-archivo").is_disabled()
                assert pagina.get_by_role("heading", name="Preparing selection rules").count() == 1
                recursos[ruta_catalogo] = catalogo
                assert not errores and not dialogos, (errores, dialogos)
                assert all(u.startswith(origen + "/") for u in peticiones), peticiones
                assert contexto.cookies() == []
                assert pagina.evaluate("localStorage.length === 0 && sessionStorage.length === 0")
                navegador.close()
                print(json.dumps({"chrome": "OK", "ancho": [1440, 390], "errores_js": errores, "solicitudes_externas": 0, "original_sha256": hashlib.sha256(original).hexdigest(), "descarga_identica": True}))
        finally:
            servidor.shutdown()
            hilo.join(timeout=2)


if __name__ == "__main__":
    main()
