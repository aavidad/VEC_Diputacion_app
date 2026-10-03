"""UI S2 con respuestas sintéticas; no acredita sesión, API nominal ni PostgreSQL."""
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
    scratch = Path(os.environ['VEC_S2_SCRATCH']); carpeta = Path(__file__).parent
    cli = json.loads((carpeta / 'testdata/preparacion-completa.json').read_bytes())
    material = cli['preparacion']['material_propuesto']; h = 'a' * 64
    dto = {'estado': 'obtenida', 'preparacion': {
        'ambito': {'organizacion_ref': 'org_' + 'a' * 16},
        'estado': {'preparacion_ref': 'preparacion:sintetica', 'revision': 1, 'huella_material_sha256': h},
        'material': {'contenido': material['contenido'], 'referencias': [{'campo': k, 'referencia': v} for k, v in material['referencias'].items()]}},
        'pendientes': cli['preparacion']['pendientes'],
        'recibo': {'recibo_ref': 'recibo:original', 'historia_ref': 'historia:sintetica', 'auditoria_ref': 'auditoria:original', 'evento_ref': 'evento:sintetico', 'huella_intencion_sha256': h, 'confirmada_en': '2026-10-01T12:00:00Z'},
        'acceso': {'decision_ref': 'decision:sintetica', 'consumo_huella_sha256': h, 'auditoria_ref': 'auditoria:consulta', 'recibo_ref': 'recibo:consulta', 'correlacion_ref': 'correlacion:sintetica', 'accedida_en': '2026-10-03T12:00:00Z'}}
    original = (json.dumps(dto, ensure_ascii=False, indent=2) + '\n').encode()
    recursos, entrada = cargar_recursos(RAIZ / 'web/static', 'seleccion-bases-preparacion')
    for nombre in ['cliente-http.js', 'contrato-http.js']:
        recursos['/portal-empleado/modulos/seleccion/preparacion-bases/' + nombre] = ((carpeta / nombre).read_bytes(), 'application/javascript')
    with ThreadingHTTPServer(('127.0.0.1', 0), handler_para(recursos)) as servidor:
        hilo = threading.Thread(target=servidor.serve_forever, daemon=True); hilo.start(); origen = f'http://127.0.0.1:{servidor.server_port}'
        try:
            with sync_playwright() as pw:
                navegador = pw.chromium.launch(executable_path='/opt/google/chrome/google-chrome', headless=True, args=['--no-sandbox', '--disable-dev-shm-usage', '--disable-background-networking'])
                contexto = navegador.new_context(accept_downloads=True, viewport={'width':1440,'height':900})
                pagina = contexto.new_page(); errores = []; peticiones = []; observadas = []; modo = {'valor':'ok'}
                pagina.on('pageerror', lambda e: errores.append(str(e))); pagina.on('request', lambda r: peticiones.append(r.url))
                def responder(route):
                    observadas.append(route.request.post_data_json)
                    if modo['valor'] == 'tardia': pagina.locator('#bases-referencia').fill('preparacion:otra')
                    if modo['valor'] in ['403','404','409','503']: route.fulfill(status=int(modo['valor']), content_type='application/json', body='{"error":"sintetico"}')
                    else: route.fulfill(status=200, content_type='application/json', body='{}' if modo['valor']=='malformada' else original)
                pagina.route('**/api/vec/seleccion/preparacion-bases/consultar', responder)
                assert pagina.goto(origen + entrada + '?lang=es').status == 200
                pagina.wait_for_function("() => !document.querySelector('#bases-consultar').disabled")
                ref = pagina.locator('#bases-referencia'); consultar = pagina.locator('#bases-consultar')
                consultar.click()
                assert pagina.evaluate("document.activeElement.id === 'bases-referencia'")
                assert ref.get_attribute('aria-invalid') == 'true'
                assert pagina.locator('#bases-referencia-error').is_visible()
                assert not observadas
                ref.fill('preparacion:sintetica'); consultar.click()
                pagina.wait_for_function("() => !document.querySelector('#bases-cerrar').disabled")
                assert observadas[-1] == {'modo':'actual','preparacion_ref':'preparacion:sintetica','revision':0,'huella_material_sha256':''}
                assert pagina.get_by_role('heading', name='Pendiente de resolver').count() == 1
                assert 'Sin aprobación' in pagina.locator('#bases-resultados').inner_text()
                pagina.get_by_text('Ver revisión, recibos y respuesta completa', exact=True).click()
                assert pagina.get_by_role('heading', name='Recibo de la preparación conservada').count() == 1
                assert pagina.get_by_role('heading', name='Acceso de esta consulta').count() == 1
                assert pagina.locator('#bases-descargar').is_disabled()
                for ancho in [1440,390]:
                    pagina.set_viewport_size({'width':ancho,'height':900}); pagina.evaluate("document.querySelector('#espacio-trabajo').scrollTop=0"); assert pagina.evaluate('document.documentElement.scrollWidth <= innerWidth'); pagina.screenshot(path=str(scratch/f'consulta-es-{ancho}.png'))
                pagina.locator('#bases-idioma').select_option('en'); pagina.wait_for_function("() => document.documentElement.lang === 'en'")
                assert pagina.get_by_role('heading',name='Outstanding checks').count()==1; assert pagina.locator('#bases-descargar').is_disabled(); pagina.screenshot(path=str(scratch/'consulta-en-390.png'))
                pagina.locator('#bases-modo').select_option('exacta'); assert pagina.locator('#bases-resultados').inner_text()==''; assert pagina.locator('#bases-descargar').is_disabled()
                peticiones_antes = len(observadas); consultar.click()
                assert len(observadas) == peticiones_antes
                assert pagina.evaluate("document.activeElement.id === 'bases-revision'")
                assert pagina.locator('#bases-revision').get_attribute('aria-invalid') == 'true'
                assert pagina.locator('#bases-huella').get_attribute('aria-invalid') == 'true'
                assert 'Enter a revision number' in pagina.locator('#bases-revision-error').inner_text()
                assert 'Enter the original' in pagina.locator('#bases-huella-error').inner_text()
                pagina.locator('#bases-revision').fill('1'); pagina.locator('#bases-huella').fill(h); consultar.click(); pagina.wait_for_function("() => !document.querySelector('#bases-cerrar').disabled")
                assert observadas[-1]['modo']=='exacta' and observadas[-1]['revision']==1 and observadas[-1]['huella_material_sha256']==h
                for valor in ['403','404','409','503','malformada']:
                    modo['valor']=valor; consultar.click(); pagina.wait_for_function("() => document.querySelector('#bases-estado').getAttribute('role') === 'alert'")
                    assert pagina.locator('#bases-resultados').inner_text()==''; assert pagina.locator('#bases-descargar').is_disabled(); assert pagina.locator('#bases-descargar').is_disabled()
                modo['valor']='tardia'; consultar.click(); pagina.wait_for_timeout(100)
                assert pagina.locator('#bases-resultados').inner_text()==''; assert pagina.locator('#bases-descargar').is_disabled()
                modo['valor']='ok'; pagina.locator('#bases-archivo').set_input_files(str(carpeta/'testdata/preparacion-completa.json')); pagina.wait_for_function("() => !document.querySelector('#bases-descargar').disabled")
                assert pagina.locator('#bases-descargar').is_visible(); pagina.locator('#bases-cerrar').click(); assert pagina.locator('#bases-resultados').inner_text()==''
                ref.focus(); pagina.keyboard.press('Tab'); assert pagina.evaluate("document.activeElement.id === 'bases-modo'")
                assert not errores, errores; assert all(u.startswith(origen+'/') for u in peticiones); assert contexto.cookies()==[]; assert pagina.evaluate('localStorage.length===0 && sessionStorage.length===0')
                print(json.dumps({'chrome':'sistema','UI_respuesta_sintetica':True,'actual_exacta':True,'descarga_consulta_ausente':True,'errores_retirada':5,'respuesta_tardia_descartada':True,'idiomas':['es','en'],'anchos':[1440,390],'JS_errores':len(errores),'cookies_storage':0,'API_nominal_PostgreSQL':'no_ejecutados'})); navegador.close()
        finally: servidor.shutdown(); hilo.join(5)

if __name__ == '__main__': main()
