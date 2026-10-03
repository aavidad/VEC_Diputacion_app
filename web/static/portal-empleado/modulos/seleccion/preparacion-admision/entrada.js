import { leerArchivo } from './modelo.js?v=20261003-s4-visor-v3';
import { crearCargaLocal } from './controlador.js?v=20261003-s4-visor-v3';
import { pintarSalida } from './vista.js?v=20261003-s4-visor-v3';
import { cargarTextos, urlCatalogo, crearTextos } from '../../../../comun/textos.js';
import { INDICE_IDIOMAS, leerRecursoJSON } from '../../../../comun/idioma.js';

const archivo = document.querySelector('#admision-archivo'), estado = document.querySelector('#admision-estado');
const raiz = document.querySelector('#admision-resultados'), descargar = document.querySelector('#admision-descargar');
const cerrar = document.querySelector('#admision-cerrar'), ayuda = document.querySelector('#admision-ayuda');
const selector = document.querySelector('#admision-idioma');
const eventos = new AbortController(); const urls = new Set(); let textos, carga = null, frame;
let estadoActual = { clave: 'vacio', error: false };
function mensaje(clave, error = false) {
  estadoActual = { clave, error }; estado.textContent = textos.traducir(clave);
  estado.setAttribute('role', error ? 'alert' : 'status');
  archivo.setAttribute('aria-invalid', String(error && ['error_formato', 'error_tamano'].includes(clave)));
  if (error) estado.focus();
}
function retirar() {
  carga = null; raiz.replaceChildren(); descargar.disabled = true; cerrar.disabled = true;
  urls.forEach(url => URL.revokeObjectURL(url)); urls.clear();
}
const controlador = crearCargaLocal({ leer: leerArchivo, retirar,
  mostrar(resultado) { pintarSalida({ raiz, dto: resultado.dto, textos }); carga = resultado; descargar.disabled = false; cerrar.disabled = false; },
  anunciar: mensaje,
});
// Permite elegir de nuevo el mismo archivo después de corregirlo o de un error.
archivo.addEventListener('click', () => { archivo.value = ''; }, { signal: eventos.signal });
archivo.addEventListener('change', () => {
  const lectura = controlador.abrir(archivo.files?.[0]); cerrar.disabled = !archivo.files?.length;
  void lectura.finally(() => { cerrar.disabled = !archivo.value && !carga; });
}, { signal: eventos.signal });
cerrar.addEventListener('click', () => { controlador.cerrar(); archivo.value = ''; mensaje('vacio'); archivo.focus(); }, { signal: eventos.signal });
descargar.addEventListener('click', () => {
  if (!carga) return;
  const url = URL.createObjectURL(new Blob([carga.bytes], { type: 'application/json' })); urls.add(url);
  const enlace = document.createElement('a'); enlace.href = url; enlace.download = textos.traducir('nombre_descarga');
  document.body.append(enlace); enlace.click(); enlace.remove(); mensaje('descarga_lista');
  setTimeout(() => { URL.revokeObjectURL(url); urls.delete(url); }, 1000);
}, { signal: eventos.signal });
ayuda.addEventListener('click', () => {
  const contenido = document.querySelector('#admision-ayuda-contenido'); contenido.hidden = !contenido.hidden;
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
  try { pintarIdioma(await cargarTextos('selectivos-admision-visor', { idioma: selector.value })); }
  catch { controlador.cerrar(); mensaje('error_catalogo', true); }
  finally { selector.value = textos.idioma; selector.disabled = false; }
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
window.addEventListener('pagehide', () => { controlador.desmontar(); eventos.abort(); cancelAnimationFrame(frame); });
window.addEventListener('pageshow', e => {
  if (!e.persisted) return;
  if (textos) {
    const destino = new URL(location.href); destino.searchParams.set('lang', textos.idioma);
    history.replaceState(history.state, '', destino);
  }
  location.reload();
});
try {
  pintarIdioma(await cargarTextos('selectivos-admision-visor'));
  for (const i of INDICE_IDIOMAS.idiomas) { const opt = document.createElement('option'); opt.value = i.codigo; opt.lang = i.codigo; opt.textContent = i.nombre; selector.append(opt); }
  selector.value = textos.idioma; archivo.disabled = false; ayuda.disabled = false; selector.disabled = false;
} catch {
  // Sólo datos de catálogo como respaldo; sin textos ni idioma embebidos.
  for (const i of INDICE_IDIOMAS.idiomas) {
    try {
      const respaldo = await leerRecursoJSON(urlCatalogo(i.codigo, 'selectivos-admision-visor'));
      pintarIdioma(crearTextos({ modulo: 'selectivos-admision-visor', idioma: i.codigo, localizacion: i.localizacion, respaldo }));
      mensaje('error_catalogo', true); break;
    } catch { /* El visor sigue deshabilitado si no hay un catálogo disponible. */ }
  }
}
