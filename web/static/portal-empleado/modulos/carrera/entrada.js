import { crearClienteCarrera, crearBorrador, leerErrorCatalogo } from './cliente.js?v=20261001-carrera-preparacion-v1';
import { pintarCarrera, pintarEstado } from './vista.js?v=20261001-carrera-preparacion-v1';
async function iniciar() {
const { cargarTextos } = await import('../../../comun/textos.js');
const { INDICE_IDIOMAS } = await import('../../../comun/idioma.js');
const textos = await cargarTextos('carrera'); const t = textos.traducir;
document.documentElement.lang = textos.idioma; document.title = t('titulo');
document.querySelectorAll('[data-texto]').forEach(n => { n.textContent = t(n.dataset.texto); });
document.querySelectorAll('[data-texto-aria]').forEach(n => { n.setAttribute('aria-label', t(n.dataset.textoAria)); });
const selector = document.querySelector('#carrera-idioma');
INDICE_IDIOMAS.idiomas.forEach(i => { const o = document.createElement('option'); o.value = i.codigo; o.textContent = i.nombre; o.selected = i.codigo === textos.idioma; selector.append(o); });
selector.addEventListener('change', () => { const url = new URL(location.href); url.searchParams.set('lang', selector.value); location.assign(url); });
const raiz = document.querySelector('#espacio-trabajo'); const cliente = crearClienteCarrera(); let controlador;
const ayuda = document.querySelector('#carrera-ayuda'); ayuda.disabled = true;
ayuda.addEventListener('click', () => { const contenido = document.querySelector('#carrera-ayuda-contenido'); if (!contenido) return; contenido.hidden = !contenido.hidden; ayuda.setAttribute('aria-expanded', String(!contenido.hidden)); if (!contenido.hidden) contenido.scrollIntoView({ block: 'nearest' }); });
function descargar(dto) {
  const url = URL.createObjectURL(new Blob([crearBorrador(dto)], { type: 'application/json;charset=utf-8' })); const enlace = document.createElement('a'); enlace.href = url; enlace.download = 'carrera-borrador.json'; document.body.append(enlace); enlace.click(); enlace.remove(); setTimeout(() => URL.revokeObjectURL(url), 1000);
  document.querySelector('#carrera-descarga-estado').textContent = t('exportar.preparado');
}
async function cargar() {
  controlador?.abort(); const actual = new AbortController(); controlador = actual; ayuda.disabled = true; ayuda.setAttribute('aria-expanded', 'false'); pintarEstado({ raiz, textos, clave: 'carga' });
  try { const dto = await cliente.listar({ signal: actual.signal }); if (actual.signal.aborted) return; pintarCarrera({ raiz, dto, textos, acciones: { descargar: () => descargar(dto) } }); ayuda.disabled = false; }
  catch { if (!actual.signal.aborted) pintarEstado({ raiz, textos, clave: 'error', reintentar: cargar }); }
}
const foco = new AbortController(); document.addEventListener('focusin', e => { if (!raiz.contains(e.target)) return; const caja = e.target.getBoundingClientRect(); const cabecera = document.querySelector('.cabecera-portal').getBoundingClientRect(); if (caja.top < cabecera.bottom || caja.bottom > innerHeight) e.target.scrollIntoView({ block: 'center', inline: 'nearest' }); }, { signal: foco.signal });
window.addEventListener('pagehide', () => { controlador?.abort(); foco.abort(); });
await cargar();

}
try { await iniciar(); } catch {
  const raiz = document.querySelector('#espacio-trabajo');
  document.querySelector('.acciones-cabecera').hidden = true;
  try {
    const datos = await leerErrorCatalogo();
    document.documentElement.lang = datos.idioma; document.title = datos.titulo;
    document.querySelector('h1').textContent = datos.titulo;
    const p = document.createElement('p'); p.setAttribute('role', 'alert'); p.lang = datos.idioma; p.textContent = datos.mensaje; raiz.replaceChildren(p);
  } catch { raiz.setAttribute('role', 'alert'); }
}
