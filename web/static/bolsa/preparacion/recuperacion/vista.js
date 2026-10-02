import { cargarTextos } from '../../../comun/textos.js';
import { INDICE_IDIOMAS, resolverIdiomaNavegacion } from '../../../comun/idioma.js';
import { leerArchivoLocal } from './material.js?v=20261002-codexa-s3-v1';

export function rutaElegirConvocatoria(idioma) {
  return `/bolsa/?${new URLSearchParams({ lang: idioma })}`;
}

export async function montarRecuperacion(d = document) {
  const idioma = resolverIdiomaNavegacion();
  const textos = await cargarTextos('seleccion-recuperacion', { idioma }); const t = textos.traducir;
  const configuracion = await cargarTextos('convoca-preparacion', { idioma });
  const limites = configuracion.seccion('limites');
  d.documentElement.lang = textos.idioma; d.title = t('titulo');
  d.querySelectorAll('[data-texto]').forEach(n => { n.textContent = t(n.dataset.texto); });
  d.querySelectorAll('[data-aria]').forEach(n => { n.setAttribute('aria-label', t(n.dataset.aria)); });
  const id = valor => d.getElementById(valor);
  id('elegir-convocatoria').href = rutaElegirConvocatoria(textos.idioma);
  const selector = id('idioma');
  for (const item of INDICE_IDIOMAS.idiomas) {
    const option = d.createElement('option'); option.value = item.codigo; option.textContent = item.nombre;
    option.selected = item.codigo === textos.idioma; selector.append(option);
  }
  selector.addEventListener('change', () => { const url = new URL(location.href); url.searchParams.set('lang', selector.value); location.assign(url); });
  let material = null, turno = 0, activo = true, descarga;
  function revocar() { if (descarga) URL.revokeObjectURL(descarga); descarga = null; }
  const nodo = (tag, texto, clase) => { const n = d.createElement(tag); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n; };
  function apartado(raiz, titulo, elementos, construir) {
    const panel = nodo('section', undefined, 'panel-listado'); panel.append(nodo('h3', t(titulo)));
    const lista = nodo('ul', undefined, 'lista-detalle');
    if (!elementos.length) lista.append(nodo('li', t('sin_datos')));
    else elementos.forEach(e => lista.append(construir(e)));
    panel.append(lista); raiz.append(panel);
  }
  function pintar() {
    id('guardar').disabled = !material; id('cerrar').disabled = !material; id('estado-archivo').hidden = !material; id('datos').replaceChildren();
    if (!material) { id('datos').append(nodo('p', t('datos_vacios'))); return; }
    const r = material.resumen;
    id('datos').append(nodo('h3', r.convocatoria.titulo), nodo('p', t('segun_archivo')));
    const datos = nodo('dl');
    for (const [clave, valor] of [['version', r.convocatoria.version], ['generada', textos.fecha(r.generada_en, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Europe/Madrid' })], ['lectura', t(r.lectura_confirmada ? 'lectura_si' : 'lectura_no')]]) datos.append(nodo('dt', t(clave)), nodo('dd', valor));
    id('datos').append(datos);
    const detalles = nodo('details'); detalles.append(nodo('summary', t('datos_tecnicos')), nodo('p', r.convocatoria.identificador_publico), nodo('p', r.convocatoria.huella_sha256)); id('datos').append(detalles);
    const secciones = nodo('div', undefined, 'recuperacion-secciones');
    apartado(secciones, 'requisitos', r.requisitos, req => {
      const li = nodo('li'); li.append(nodo('strong', req.titulo), nodo('p', req.descripcion), nodo('span', t('cumplimiento_pendiente'), 'etiqueta'), nodo('p', t(req.obligatorio ? 'obligatorio' : 'opcional'))); return li;
    });
    apartado(secciones, 'plazos', r.plazos, p => {
      const li = nodo('li'); li.append(nodo('strong', p.titulo), nodo('p', t('desde_hasta', { desde: textos.fecha(p.abre_en, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Europe/Madrid' }), hasta: textos.fecha(p.cierra_en, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Europe/Madrid' }) })), nodo('p', p.etiqueta_situacion)); if (p.descripcion) li.append(nodo('p', p.descripcion)); return li;
    });
    apartado(secciones, 'documentos', r.documentos_publicos, doc => { const li = nodo('li'); li.append(nodo('strong', doc.titulo), nodo('p', doc.url), nodo('small', t('ruta_no_verificada'))); return li; });
    apartado(secciones, 'archivos', r.archivos_locales, a => nodo('li', t('archivo_metadatos', { nombre: a.nombre, tamano: textos.numero(a.tamano) })));
    id('datos').append(secciones);
  }
  id('elegir').addEventListener('click', () => id('archivo').click());
  id('archivo').addEventListener('change', async event => {
    const archivo = event.target.files?.[0]; const actual = ++turno;
    material = null; revocar(); pintar(); id('error').hidden = true;
    if (!archivo) { id('estado').textContent = t('vacio'); return; }
    id('estado').textContent = t('cargando');
    try {
      const recibido = await leerArchivoLocal(archivo, limites);
      if (!activo || actual !== turno) return;
      material = recibido; pintar(); id('estado').textContent = t('recuperado'); id('datos').focus();
    } catch {
      if (!activo || actual !== turno) return;
      id('estado').textContent = t('vacio'); id('error').textContent = t('error_archivo'); id('error').hidden = false;
    } finally { if (activo && actual === turno) event.target.value = ''; }
  });
  id('guardar').addEventListener('click', () => {
    if (!material) return; revocar(); descarga = URL.createObjectURL(new Blob([material.copiarOriginal()], { type: 'application/json' }));
    const a = d.createElement('a'); a.href = descarga; a.download = t('nombre_material'); a.click(); id('estado').textContent = t('guardado');
  });
  function cerrar() { turno += 1; material = null; revocar(); pintar(); id('estado').textContent = t('vacio'); id('error').hidden = true; }
  id('cerrar').addEventListener('click', () => { cerrar(); id('elegir').focus(); });
  globalThis.addEventListener('pagehide', () => { activo = false; cerrar(); }, { once: true });
  globalThis.addEventListener('pageshow', event => { if (event.persisted) location.reload(); });
  return { cerrar };
}
if (globalThis.document?.getElementById('archivo')) await montarRecuperacion();
