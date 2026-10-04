"""Recorrido sintético local; requiere sandbox sin red externa y Chrome del sistema."""
import base64
import hashlib
import importlib.util
import json
import os
import threading
from http.server import ThreadingHTTPServer
from pathlib import Path
from playwright.sync_api import sync_playwright


RAIZ = Path(__file__).resolve().parents[6]


def main():
    salida = Path(os.environ["VEC_S4_SCRATCH"])
    servidor_fuente = Path(os.environ.get("VEC_S4_SERVER", str(RAIZ / "scripts/servir_preparacion_rrhh.py")))
    spec = importlib.util.spec_from_file_location("servidor_vec", servidor_fuente)
    servicio = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(servicio)
    recursos, entrada = servicio.cargar_recursos(RAIZ / "web/static", "selectivos-admision-visor")
    assert not any("testdata" in r or "recorrido" in r or "escenario" in r for r in recursos)
    original = (Path(__file__).parent / "testdata/admision-preparada.json").read_bytes()
    dto = json.loads(original)
    with ThreadingHTTPServer(("127.0.0.1", 0), servicio.handler_para(recursos)) as servidor:
        hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
        hilo.start()
        origen = f"http://127.0.0.1:{servidor.server_port}"
        try:
            with sync_playwright() as pw:
                navegador = pw.chromium.launch(executable_path=os.environ.get("VEC_S4_CHROME", "/usr/bin/google-chrome"), headless=True,
                    ignore_default_args=["--disable-back-forward-cache"],
                    args=["--no-sandbox", "--disable-dev-shm-usage", "--disable-background-networking"])
                contexto = navegador.new_context(accept_downloads=True, viewport={"width": 1440, "height": 900})
                pagina = contexto.new_page()
                errores, peticiones, dialogos = [], [], []
                pagina.on("pageerror", lambda e: errores.append(str(e)))
                pagina.on("request", lambda r: peticiones.append((r.method, r.url)))
                pagina.on("dialog", lambda d: (dialogos.append(d.message), d.dismiss()))
                assert pagina.goto(origen + entrada + "?lang=es").status == 200
                archivo = pagina.locator("#admision-archivo")

                def abrir(datos, nombre="salida.json"):
                    archivo.set_input_files({"name": nombre, "mimeType": "application/json", "buffer": datos})
                    pagina.wait_for_function("() => !document.querySelector('#admision-descargar').disabled")

                abrir(original)
                assert dto["requisitos"][0]["requisito"]["titulo_propuesto"] in pagina.locator("#admision-resultados").inner_text()
                assert "Resumen local vinculado · Sin presentar" in pagina.locator("#admision-resultados").inner_text()
                assert pagina.locator("#admision-resultados details[open]").count() == 0
                assert pagina.evaluate("document.documentElement.scrollHeight <= innerHeight && document.documentElement.scrollWidth <= innerWidth")
                assert pagina.locator("#espacio-trabajo").evaluate("n => n.scrollHeight > n.clientHeight")
                pagina.screenshot(path=str(salida / "admision-1440.png"))
                pagina.locator("#espacio-trabajo").evaluate("n => { const panel = document.querySelector('#admision-resultados .panel:nth-child(2)'); n.scrollTop += panel.getBoundingClientRect().top - n.getBoundingClientRect().top; }")
                pagina.screenshot(path=str(salida / "admision-checklist-1440.png"))
                with pagina.expect_download() as espera:
                    pagina.locator("#admision-descargar").click()
                espera.value.save_as(salida / "original-descargado.json")
                assert (salida / "original-descargado.json").read_bytes() == original
                pagina.locator("#admision-idioma").select_option("en")
                pagina.wait_for_function("() => document.documentElement.lang === 'en'")
                assert "Why it remains pending" in pagina.locator("#admision-resultados").inner_text()
                assert pagina.locator("#admision-descargar").is_enabled()
                pagina.locator("#espacio-trabajo").evaluate("n => n.scrollTop = 0")
                pagina.screenshot(path=str(salida / "admision-1440-en.png"))
                pagina.locator("#admision-idioma").select_option("es")
                pagina.wait_for_function("() => document.documentElement.lang === 'es'")
                for ancho, alto in [(390, 844), (720, 450), (320, 700)]:
                    pagina.set_viewport_size({"width": ancho, "height": alto})
                    assert pagina.evaluate("document.documentElement.scrollWidth <= innerWidth")
                    pagina.locator("#espacio-trabajo").evaluate("n => n.scrollTop = 0")
                    if ancho == 390:
                        pagina.screenshot(path=str(salida / "admision-390.png"))
                        pagina.get_by_role("heading", name="Requisitos propuestos: motivos y siguiente paso").scroll_into_view_if_needed()
                        assert pagina.locator(".admision-tabla th[scope='row']").first.evaluate("n => n.clientWidth >= 250")
                        pagina.screenshot(path=str(salida / "admision-checklist-390.png"))
                pagina.locator("#admision-ayuda").click()
                assert pagina.locator("#admision-ayuda-contenido").is_visible()
                pagina.locator("#admision-ayuda").click()
                assert pagina.locator("#admision-ayuda-contenido").is_hidden()
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
                        }"""), pagina.evaluate("({html:document.activeElement.outerHTML, rect:document.activeElement.getBoundingClientRect().toJSON(), viewport:[innerWidth,innerHeight], scroll:[scrollX,scrollY]})")
                vacio = json.loads(original)
                vacio["requisitos"] = []
                del vacio["solicitud_contexto"]
                abrir(json.dumps(vacio).encode())
                assert "No hay requisitos propuestos" in pagina.locator("#admision-resultados").inner_text()
                assert "Sin solicitud vinculada" in pagina.locator("#admision-resultados").inner_text()
                payload = '<img src="https://externo.invalid/fuga" onerror="window.__xss=1">'
                malicioso = json.loads(original)
                malicioso["requisitos"][0]["requisito"]["titulo_propuesto"] = payload
                abrir(json.dumps(malicioso).encode())
                assert payload in pagina.locator("#admision-resultados").inner_text()
                assert pagina.locator("#admision-resultados img, #admision-resultados a").count() == 0
                assert pagina.evaluate("window.__xss === undefined")
                for datos in [b'{', original.replace(b'"admision_oficial": false', b'"admision_oficial": true'), b'x' * (4 * 1024 * 1024 + 1)]:
                    archivo.set_input_files({"name": "invalido.json", "mimeType": "application/json", "buffer": datos})
                    pagina.wait_for_function("() => document.querySelector('#admision-archivo').getAttribute('aria-invalid') === 'true'")
                    assert pagina.locator("#admision-resultados").inner_text() == ""
                    assert pagina.locator("#admision-descargar").is_disabled()
                    assert pagina.locator("#admision-estado").get_attribute("role") == "alert"
                    assert pagina.evaluate("document.activeElement.id") == "admision-estado"
                # Una lectura lenta no puede desplazar una reimportación ni volver tras cerrar.
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
                assert "No hay requisitos propuestos" in pagina.locator("#admision-resultados").inner_text()
                archivo.set_input_files({"name": "lento.json", "mimeType": "application/json", "buffer": original})
                pagina.wait_for_function("() => document.querySelector('#admision-estado').textContent.includes('Leyendo')")
                pagina.locator("#admision-cerrar").click()
                pagina.evaluate("window.__liberarLectura()")
                pagina.wait_for_timeout(100)
                assert pagina.locator("#admision-resultados").inner_text() == ""
                assert pagina.locator("#admision-descargar").is_disabled()
                assert pagina.evaluate("document.activeElement.id") == "admision-archivo"
                abrir(original)
                assert pagina.reload().status == 200
                pagina.wait_for_function("() => !document.querySelector('#admision-archivo').disabled")
                assert pagina.locator("#admision-resultados").inner_text() == ""
                assert pagina.locator("#admision-descargar").is_disabled()
                # Volver desde otra página debe permitir importar y descargar otra vez.
                abrir(original)
                assert pagina.goto(origen + entrada + "index.html?lang=es").status == 200
                pagina.wait_for_function("() => !document.querySelector('#admision-archivo').disabled")
                pagina.go_back(wait_until="domcontentloaded")
                pagina.wait_for_function("() => !document.querySelector('#admision-archivo').disabled")
                assert pagina.locator("#admision-resultados").inner_text() == ""
                abrir(original)
                with pagina.expect_download() as retorno:
                    pagina.locator("#admision-descargar").click()
                retorno.value.save_as(salida / "original-retorno.json")
                assert (salida / "original-retorno.json").read_bytes() == original
                # Falta del catálogo principal: mensaje de datos, sin habilitar importación.
                recursos.pop("/textos/es/selectivos-admision-visor.json")
                assert pagina.goto(origen + entrada + "?lang=en").status == 200
                pagina.wait_for_function("() => document.querySelector('#admision-estado').textContent.includes('could not be loaded')")
                assert pagina.locator("#admision-archivo").is_disabled()
                assert pagina.locator("#admision-descargar").is_disabled()
                assert not errores and not dialogos, (errores, dialogos)
                assert all(m == "GET" and u.startswith(origen + "/") for m, u in peticiones), peticiones
                assert contexto.cookies() == []
                assert pagina.evaluate("localStorage.length === 0 && sessionStorage.length === 0")
                assert pagina.evaluate("indexedDB.databases()") == []
                navegador.close()
                resultado = {"chrome": "OK", "ancho": [1440, 390, 720, 320], "errores_js": errores, "solicitudes_externas": 0,
                    "http_negocio": 0, "original_sha256": hashlib.sha256(original).hexdigest(), "descarga_identica": True}
                if os.environ.get("VEC_S4_EXPORT") == "1":
                    resultado["capturas"] = {n: base64.b64encode((salida / n).read_bytes()).decode()
                        for n in ("admision-1440.png", "admision-1440-en.png", "admision-390.png", "admision-checklist-1440.png", "admision-checklist-390.png")}
                print(json.dumps(resultado))
        finally:
            servidor.shutdown()
            hilo.join(timeout=2)


if __name__ == "__main__":
    main()
