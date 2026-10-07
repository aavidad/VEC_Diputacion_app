import { crearClienteEscenario } from './cliente.js?v=20261001-formacion-preparacion-v1';
import { pintarFormacion } from './vista.js?v=20261001-formacion-preparacion-v1';
import { cargarTextos, reintentarTextos } from '../../../comun/textos.js';
import { montarSelectorIdioma } from '../../../comun/idioma.js';

const raiz = document.querySelector('#espacio-trabajo');
const selector = document.querySelector('#formacion-idioma');
const avisoIdioma = document.querySelector('#formacion-idioma-estado');
let controlador = null; let textos = null; let carga = null; let modalidad = ''; let seleccion = null; let turno = 0; let turnoIdioma = 0; let activa = true;
let estadoCarga = { clave: '', error: false };
const urls = new Set();
const ubicacionIdioma = Object.freeze({
  get href() { return location.href; },
  assign(destino) { void cambiarIdiomaEnVista(destino); },
});
function fijarIdiomaEnURL(idioma, destino = location.href) {
  const actual = new URL(location.href); const url = new URL(destino, actual);
  if (url.origin !== actual.origin || url.pathname !== actual.pathname) throw new TypeError('destino');
  url.searchParams.set('lang', idioma);
  history.replaceState(history.state, '', url);
}
function mostrarAvisoIdioma(clave, reintentar) {
  avisoIdioma.replaceChildren(); avisoIdioma.hidden = !clave;
  if (!clave) return;
  const panel = document.createElement('div'); panel.className = 'siguiente-paso con-incidencia';
  const mensaje = document.createElement('p'); mensaje.setAttribute('role', 'alert'); mensaje.textContent = textos.traducir(clave);
  panel.append(mensaje);
  if (reintentar) {
    const boton = document.createElement('button'); boton.type = 'button'; boton.className = 'boton-secundario';
    boton.textContent = textos.traducir('reintentar_idioma'); boton.addEventListener('click', () => {
      boton.disabled = true;
      const url = new URL(location.href); url.searchParams.set('lang', reintentar);
      void cambiarIdiomaEnVista(url.href);
    });
    panel.append(boton);
  }
  avisoIdioma.append(panel);
}
const mensaje = (clave, error = false) => {
  estadoCarga = { clave, error };
  const p = document.createElement('p'); p.className = 'siguiente-paso'; p.textContent = textos.traducir(clave);
  p.setAttribute('role', error ? 'alert' : 'status'); raiz.replaceChildren(p);
  if (error && clave === 'error_escenario') {
    const b = document.createElement('button'); b.type = 'button'; b.className = 'boton-secundario';
    b.textContent = textos.traducir('reintentar'); b.addEventListener('click', cargar); raiz.append(b);
  }
};
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
function pintarIdioma(siguiente) {
  textos = siguiente; document.documentElement.lang = textos.idioma; document.title = textos.traducir('titulo');
  document.querySelectorAll('[data-texto]').forEach(n => { n.textContent = textos.traducir(n.dataset.texto); });
  document.querySelector('summary[data-texto]').setAttribute('aria-label', textos.traducir('ayuda'));
  if (carga) {
    const focoActual = raiz.contains(document.activeElement) ? document.activeElement.dataset.foco : null;
    pintar(focoActual);
  } else if (estadoCarga.clave) mensaje(estadoCarga.clave, estadoCarga.error);
}
async function cambiarIdiomaEnVista(destino) {
  if (!activa || !textos) return;
  let url, actualURL;
  try { url = new URL(destino, location.href); actualURL = new URL(location.href); }
  catch { selector.value = textos.idioma; mostrarAvisoIdioma('idioma_error'); return; }
  if (url.origin !== actualURL.origin || url.pathname !== actualURL.pathname) {
    selector.value = textos.idioma; mostrarAvisoIdioma('idioma_error'); return;
  }
  const solicitado = url.searchParams.get('lang');
  if (solicitado === textos.idioma && !textos.incidenciaCatalogo) { selector.value = textos.idioma; return; }
  const actual = ++turnoIdioma; selector.disabled = true;
  try {
    const siguiente = await (textos.incidenciaCatalogo
      ? reintentarTextos('formacion', { idioma: solicitado })
      : cargarTextos('formacion', { idioma: solicitado }));
    if (!activa || actual !== turnoIdioma) return;
    pintarIdioma(siguiente); fijarIdiomaEnURL(siguiente.idioma, url);
    mostrarAvisoIdioma(siguiente.incidenciaCatalogo ? 'idioma_respaldo' : '', siguiente.incidenciaCatalogo?.idioma);
  } catch {
    if (activa && actual === turnoIdioma) mostrarAvisoIdioma('idioma_error', solicitado);
  } finally {
    if (activa && actual === turnoIdioma) { selector.value = textos.idioma; selector.disabled = false; }
  }
}
async function cargar() {
  controlador?.abort(); controlador = new AbortController(); const actual = ++turno; carga = null; modalidad = ''; seleccion = null; mensaje('cargando');
  try {
    const resultado = await crearClienteEscenario().cargar({ signal: controlador.signal });
    if (actual !== turno) return;
    carga = resultado; estadoCarga = { clave: '', error: false }; pintar();
  } catch {
    if (actual !== turno || controlador.signal.aborted) return;
    carga = null; mensaje('error_escenario', true);
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
window.addEventListener('pagehide', () => { activa = false; ++turno; ++turnoIdioma; controlador?.abort(); foco.abort(); if (frame !== null) cancelAnimationFrame(frame); urls.forEach(url => URL.revokeObjectURL(url)); });

try {
  const inicial = await cargarTextos('formacion');
  pintarIdioma(inicial); fijarIdiomaEnURL(inicial.idioma);
  montarSelectorIdioma(selector, ubicacionIdioma, inicial.idioma);
  if (inicial.incidenciaCatalogo) mostrarAvisoIdioma('idioma_respaldo', inicial.incidenciaCatalogo.idioma);
  await cargar();
} catch {
  // Catálogo de error propio: la vista de datos espera textos legibles.
  try {
    const error = await cargarTextos('formacion-error');
    document.documentElement.lang = error.idioma; document.title = error.traducir('titulo'); fijarIdiomaEnURL(error.idioma);
    document.querySelector('h1').textContent = error.traducir('titulo');
    const p = document.createElement('p'); p.setAttribute('role', 'alert'); p.textContent = error.traducir('mensaje');
    const boton = document.createElement('button'); boton.type = 'button'; boton.className = 'boton-secundario';
    boton.textContent = error.traducir('reintentar'); boton.addEventListener('click', () => location.reload());
    raiz.replaceChildren(p, boton);
    document.querySelector('.selector-idioma').remove(); document.querySelector('.acciones-cabecera details').remove();
  } catch { raiz.setAttribute('role', 'alert'); }
}
