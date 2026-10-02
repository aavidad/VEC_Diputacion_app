import { leerArchivo } from './modelo.js?v=20261002-selectivos-bases-v1';
import { pintarSalida } from './vista.js?v=20261002-selectivos-bases-v1';
import { cargarTextos } from '../../../../comun/textos.js';
import { INDICE_IDIOMAS, IDIOMA_ACTUAL, leerRecursoJSON } from '../../../../comun/idioma.js';

const archivo = document.querySelector('#bases-archivo'); const estado = document.querySelector('#bases-estado');
const raiz = document.querySelector('#bases-resultados'); const descargar = document.querySelector('#bases-descargar');
const cerrar = document.querySelector('#bases-cerrar'); const urls = new Set();
let turno = 0; let carga = null; let textos; let estadoActual = { clave: 'vacio', error: false, variables: {} };
function mensaje(clave, error = false, variables = {}) {
  estadoActual = { clave, error, variables };
  estado.textContent = textos.traducir(clave, variables); estado.setAttribute('role', error ? 'alert' : 'status');
  archivo.setAttribute('aria-invalid', String(error && ['error_formato', 'error_tamano'].includes(clave)));
}
function limpiar() {
  ++turno; carga = null; descargar.disabled = true; cerrar.disabled = true; raiz.replaceChildren();
}
archivo.addEventListener('change', async () => {
  limpiar(); const actual = turno; const file = archivo.files?.[0];
  if (!file) { mensaje('vacio'); return; }
  cerrar.disabled = false; mensaje('cargando');
  try {
    const resultado = await leerArchivo(file); if (actual !== turno) return;
    pintarSalida({ raiz, dto: resultado.dto, textos }); carga = resultado; descargar.disabled = false;
    mensaje('cargado', false, { archivo: file.name });
  } catch (error) { if (actual === turno) mensaje(error.message === 'tamano' ? 'error_tamano' : 'error_formato', true); }
});
cerrar.addEventListener('click', () => { limpiar(); archivo.value = ''; mensaje('vacio'); archivo.focus(); });
descargar.addEventListener('click', () => {
  if (!carga) return;
  const url = URL.createObjectURL(new Blob([carga.bytes], { type: 'application/json' })); urls.add(url);
  const enlace = document.createElement('a'); enlace.href = url; enlace.download = textos.traducir('nombre_descarga');
  document.body.append(enlace); enlace.click(); enlace.remove(); mensaje('descarga_lista');
  setTimeout(() => { URL.revokeObjectURL(url); urls.delete(url); }, 1000);
});
const ayuda = document.querySelector('#bases-ayuda');
ayuda.addEventListener('click', () => {
  const contenido = document.querySelector('#bases-ayuda-contenido'); contenido.hidden = !contenido.hidden;
  ayuda.setAttribute('aria-expanded', String(!contenido.hidden)); if (!contenido.hidden) contenido.scrollIntoView({ block: 'nearest' });
});
// El cambio de idioma no pierde el archivo elegido: permanece sólo en memoria.
async function idioma(codigo) {
  const siguiente = await cargarTextos('seleccion-bases-preparacion', { idioma: codigo }); textos = siguiente;
  document.documentElement.lang = textos.idioma; document.title = textos.traducir('titulo');
  document.querySelectorAll('[data-texto]').forEach(n => { if (n !== estado) n.textContent = textos.traducir(n.dataset.texto); });
  mensaje(estadoActual.clave, estadoActual.error, estadoActual.variables);
  document.querySelectorAll('[data-aria]').forEach(n => { n.setAttribute('aria-label', textos.traducir(n.dataset.aria)); });
  if (carga) { pintarSalida({ raiz, dto: carga.dto, textos }); mensaje('cargado', false, { archivo: archivo.files[0].name }); }
}
const foco = new AbortController(); let frame;
document.addEventListener('focusin', e => {
  const n = e.target; if (!document.querySelector('#espacio-trabajo').contains(n)) return;
  cancelAnimationFrame(frame); frame = requestAnimationFrame(() => {
    if (!n.isConnected || n !== document.activeElement) return;
    const caja = n.getBoundingClientRect(); const area = document.querySelector('#espacio-trabajo').getBoundingClientRect();
    const superior = Math.max(area.top, document.querySelector('.cabecera-portal').getBoundingClientRect().bottom);
    if (caja.top < superior || caja.bottom > Math.min(innerHeight, area.bottom)) n.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' });
  });
}, { signal: foco.signal });
window.addEventListener('pagehide', () => { limpiar(); foco.abort(); cancelAnimationFrame(frame); urls.forEach(url => URL.revokeObjectURL(url)); });
try {
  archivo.disabled = true; ayuda.disabled = true;
  const inicial = await cargarTextos('seleccion-bases-preparacion'); await idioma(inicial.idioma);
  const selector = document.querySelector('#bases-idioma');
  for (const i of INDICE_IDIOMAS.idiomas) { const opt = document.createElement('option'); opt.value = i.codigo; opt.textContent = i.nombre; opt.selected = i.codigo === textos.idioma; selector.append(opt); }
  selector.addEventListener('change', async () => { selector.disabled = true; try { await idioma(selector.value); } catch { mensaje('error_catalogo', true); } finally { selector.value = textos.idioma; selector.disabled = false; } });
  archivo.disabled = false; ayuda.disabled = false;
} catch {
  // Un catálogo ausente no habilita un visor sin límites comprensibles.
  archivo.disabled = true; ayuda.disabled = true; estado.setAttribute('role', 'alert');
  const respaldo = await leerRecursoJSON(new URL(`../../../../textos/${IDIOMA_ACTUAL}/seleccion-bases-preparacion-error.json`, import.meta.url)).catch(() => null);
  if (respaldo) { estado.textContent = respaldo.mensaje; document.documentElement.lang = respaldo.idioma; document.title = respaldo.titulo; document.querySelector('h1').textContent = respaldo.titulo; }
}
