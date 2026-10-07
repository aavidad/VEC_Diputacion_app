import { leerArchivo } from './modelo.js?v=20261007-s5-cotejo-local-v1';
import { pintarSalida } from './vista.js?v=20261007-s5-cotejo-local-v1';
import { cargarTextos, urlCatalogo, crearTextos } from '../../../../comun/textos.js';
import { INDICE_IDIOMAS, leerRecursoJSON } from '../../../../comun/idioma.js';

const archivo = document.querySelector('#acta-archivo'), estado = document.querySelector('#acta-estado');
const raiz = document.querySelector('#acta-resultados'), descargar = document.querySelector('#acta-descargar');
const cerrar = document.querySelector('#acta-cerrar'), ayuda = document.querySelector('#acta-ayuda');
const selector = document.querySelector('#acta-idioma'), eventos = new AbortController(), urls = new Set();
let textos, carga = null, frame, turno = 0, activa = true;
let estadoActual = { clave: 'vacio', error: false };
function mensaje(clave, error = false) {
  estadoActual = { clave, error }; estado.textContent = textos.traducir(clave);
  estado.setAttribute('role', error ? 'alert' : 'status');
  archivo.setAttribute('aria-invalid', String(error && ['error_formato', 'error_tamano'].includes(clave)));
  if (error) estado.focus();
}
function retirar() {
  ++turno; carga = null; raiz.replaceChildren(); descargar.disabled = true; cerrar.disabled = true;
  urls.forEach(url => URL.revokeObjectURL(url)); urls.clear();
}
archivo.addEventListener('click', () => { archivo.value = ''; }, { signal: eventos.signal });
archivo.addEventListener('change', async () => {
  const seleccionado = archivo.files?.[0]; retirar(); const actual = turno;
  if (!seleccionado) { mensaje('vacio'); return; }
  cerrar.disabled = false; mensaje('cargando');
  try {
    const resultado = await leerArchivo(seleccionado); if (!activa || actual !== turno) return;
    pintarSalida({ raiz, dto: resultado.dto, textos }); carga = resultado; descargar.disabled = false;
    mensaje(Object.hasOwn(resultado.dto, 'cotejo_local') ? 'cargado_cotejo_declarado' : 'cargado');
  } catch (error) { if (activa && actual === turno) { retirar(); mensaje(error.message === 'tamano' ? 'error_tamano' : 'error_formato', true); } }
}, { signal: eventos.signal });
cerrar.addEventListener('click', () => { retirar(); archivo.value = ''; mensaje('vacio'); archivo.focus(); }, { signal: eventos.signal });
descargar.addEventListener('click', () => {
  if (!carga) return;
  const url = URL.createObjectURL(new Blob([carga.bytes], { type: 'application/json' })); urls.add(url);
  const enlace = document.createElement('a'); enlace.href = url; enlace.download = textos.traducir('nombre_descarga');
  document.body.append(enlace); enlace.click(); enlace.remove(); mensaje('descarga_lista');
  setTimeout(() => { URL.revokeObjectURL(url); urls.delete(url); }, 1000);
}, { signal: eventos.signal });
ayuda.addEventListener('click', () => {
  const contenido = document.querySelector('#acta-ayuda-contenido'); contenido.hidden = !contenido.hidden;
  ayuda.setAttribute('aria-expanded', String(!contenido.hidden)); if (!contenido.hidden) contenido.scrollIntoView({ block: 'nearest' });
}, { signal: eventos.signal });
function pintarIdioma(siguiente) {
  textos = siguiente; document.documentElement.lang = textos.idioma; document.title = textos.traducir('titulo');
  document.querySelectorAll('[data-texto]').forEach(n => { n.textContent = textos.traducir(n.dataset.texto); });
  document.querySelectorAll('[data-aria]').forEach(n => { n.setAttribute('aria-label', textos.traducir(n.dataset.aria)); });
  mensaje(estadoActual.clave, estadoActual.error);
  if (carga) pintarSalida({ raiz, dto: carga.dto, textos });
}
selector.addEventListener('change', async () => {
  selector.disabled = true;
  try { const siguiente = await cargarTextos('selectivos-acta-visor', { idioma: selector.value }); if (activa) pintarIdioma(siguiente); }
  catch { if (activa) { retirar(); mensaje('error_catalogo', true); } }
  finally { if (activa) { selector.value = textos.idioma; selector.disabled = false; } }
}, { signal: eventos.signal });
document.addEventListener('focusin', e => {
  const n = e.target; if (!document.querySelector('#espacio-trabajo').contains(n)) return;
  cancelAnimationFrame(frame); frame = requestAnimationFrame(() => {
    if (!n.isConnected || n !== document.activeElement) return;
    const caja = n.getBoundingClientRect(), area = document.querySelector('#espacio-trabajo').getBoundingClientRect();
    const superior = Math.max(area.top, document.querySelector('.cabecera-portal').getBoundingClientRect().bottom);
    if (caja.top < superior || caja.bottom > Math.min(innerHeight, area.bottom)) n.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' });
  });
}, { signal: eventos.signal });
window.addEventListener('pagehide', () => { activa = false; retirar(); eventos.abort(); cancelAnimationFrame(frame); });
window.addEventListener('pageshow', e => {
  if (!e.persisted) return;
  if (textos) {
    const destino = new URL(location.href); destino.searchParams.set('lang', textos.idioma);
    history.replaceState(history.state, '', destino);
  }
  location.reload();
});
try {
  const inicial = await cargarTextos('selectivos-acta-visor');
  if (activa) {
    pintarIdioma(inicial);
    for (const i of INDICE_IDIOMAS.idiomas) { const opt = document.createElement('option'); opt.value = i.codigo; opt.lang = i.codigo; opt.textContent = i.nombre; selector.append(opt); }
    selector.value = textos.idioma; archivo.disabled = false; ayuda.disabled = false; selector.disabled = false;
  }
} catch {
  for (const i of INDICE_IDIOMAS.idiomas) {
    try {
      const respaldo = await leerRecursoJSON(urlCatalogo(i.codigo, 'selectivos-acta-visor'));
      if (!activa) break;
      pintarIdioma(crearTextos({ modulo: 'selectivos-acta-visor', idioma: i.codigo, localizacion: i.localizacion, respaldo }));
      mensaje('error_catalogo', true); break;
    } catch { /* Sin catálogo legible, el visor permanece deshabilitado. */ }
  }
}
