import { crearClienteEscenario } from './cliente.js?v=20261001-formacion-preparacion-v1';
import { pintarFormacion } from './vista.js?v=20261001-formacion-preparacion-v1';
import { cargarTextos } from '../../../comun/textos.js';
import { montarSelectorIdioma } from '../../../comun/idioma.js';

const raiz = document.querySelector('#espacio-trabajo');
let controlador = null; let textos = null; let carga = null; let modalidad = ''; let seleccion = null; let turno = 0;
const urls = new Set();
const mensaje = (clave, error = false) => { const p = document.createElement('p'); p.className = 'siguiente-paso'; p.textContent = textos.traducir(clave); p.setAttribute('role', error ? 'alert' : 'status'); raiz.replaceChildren(p); };
const acciones = {
  filtrar(valor) { modalidad = valor; seleccion = null; pintar('modalidad'); },
  elegir(ref) { seleccion = ref; pintar('detalle'); },
  cerrar() { const ref = seleccion; seleccion = null; pintar(`edicion-${ref}`); },
  exportar() {
    if (!carga) return;
    const url = URL.createObjectURL(new Blob([carga.bytes], { type: 'application/json' })); urls.add(url);
    const a = document.createElement('a'); a.href = url; a.download = textos.traducir('nombre_exportacion', { version: carga.escenario.version }); a.hidden = true; document.body.append(a); a.click(); a.remove();
    setTimeout(() => { URL.revokeObjectURL(url); urls.delete(url); }, 1000);
  },
};
function pintar(foco) {
  pintarFormacion({ raiz, escenario: carga.escenario, textos, modalidad, seleccion, acciones });
  if (foco) Array.from(raiz.querySelectorAll('[data-foco]')).find(n => n.dataset.foco === foco)?.focus();
}
async function cargar() {
  controlador?.abort(); controlador = new AbortController(); const actual = ++turno; carga = null; modalidad = ''; seleccion = null; mensaje('cargando');
  try {
    const resultado = await crearClienteEscenario().cargar({ signal: controlador.signal });
    if (actual !== turno) return;
    carga = resultado; pintar();
  } catch {
    if (actual !== turno || controlador.signal.aborted) return;
    carga = null; mensaje('error_escenario', true);
    const b = document.createElement('button'); b.type = 'button'; b.className = 'boton-secundario'; b.textContent = textos.traducir('reintentar'); b.addEventListener('click', cargar); raiz.append(b);
  }
}

// La cabecera del portal permanece fija en móvil: conserva el foco a la vista.
const foco = new AbortController(); let frame = null;
document.addEventListener('focusin', evento => {
  const n = evento.target; if (!raiz.contains(n)) return;
  if (frame !== null) cancelAnimationFrame(frame);
  frame = requestAnimationFrame(() => {
    frame = null; if (!n.isConnected || document.activeElement !== n) return;
    const caja = n.getBoundingClientRect(); const cabecera = document.querySelector('.cabecera-portal').getBoundingClientRect();
    const area = raiz.getBoundingClientRect(); const superior = Math.max(0, cabecera.bottom, area.top); const inferior = Math.min(innerHeight, area.bottom);
    if (caja.top < superior || caja.bottom > inferior) n.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' });
  });
}, { signal: foco.signal });
window.addEventListener('pagehide', () => { ++turno; controlador?.abort(); foco.abort(); if (frame !== null) cancelAnimationFrame(frame); urls.forEach(url => URL.revokeObjectURL(url)); });

try {
  textos = await cargarTextos('formacion'); document.documentElement.lang = textos.idioma; document.title = textos.traducir('titulo');
  document.querySelectorAll('[data-texto]').forEach(n => { n.textContent = textos.traducir(n.dataset.texto); });
  document.querySelector('summary[data-texto]').setAttribute('aria-label', textos.traducir('ayuda'));
  montarSelectorIdioma(document.querySelector('#formacion-idioma'), location, textos.idioma);
  await cargar();
} catch {
  // Respaldo público de datos para el fallo total del catálogo, sin afirmar traducción.
  try {
    const r = await fetch('./error-catalogo.json?v=20261001-formacion-preparacion-v1', { credentials: 'omit', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer' });
    if (!r.ok) throw new Error('formacion.catalogo');
    const error = await r.json(); const p = document.createElement('p'); p.setAttribute('role', 'alert'); p.lang = error.idioma; p.textContent = error.mensaje; raiz.replaceChildren(p);
    document.querySelector('.selector-idioma').hidden = true; document.querySelector('.acciones-cabecera details').hidden = true;
  } catch { raiz.setAttribute('role', 'alert'); }
}
