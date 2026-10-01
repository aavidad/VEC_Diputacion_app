import { cargarTextos } from '../../../comun/textos.js';
import { INDICE_IDIOMAS } from '../../../comun/idioma.js';
import { montarSeleccion } from './montaje.js?v=20261001-seleccion-v1';

const textos = await cargarTextos('seleccion');
const t = textos.traducir;
document.documentElement.lang = textos.idioma;
document.title = t('titulo');
document.querySelectorAll('[data-texto]').forEach(elemento => { elemento.textContent = t(elemento.dataset.texto); });
document.querySelectorAll('.migas a, .navegacion-modulos a').forEach(enlace => {
  const url = new URL(enlace.href); url.searchParams.set('lang', textos.idioma); enlace.href = url;
});
document.querySelector('.navegacion-modulos').setAttribute('aria-label', t('navegacion'));
const selector = document.getElementById('seleccion-idioma');
INDICE_IDIOMAS.idiomas.forEach(idioma => {
  const opcion = document.createElement('option');
  opcion.value = idioma.codigo; opcion.textContent = idioma.nombre;
  opcion.selected = idioma.codigo === textos.idioma; selector.append(opcion);
});
selector.addEventListener('change', () => {
  const url = new URL(location.href); url.searchParams.set('lang', selector.value); location.assign(url);
});
const menu = document.getElementById('seleccion-menu');
menu.addEventListener('click', () => {
  const abierto = menu.getAttribute('aria-expanded') !== 'true';
  document.body.dataset.menuAbierto = String(abierto);
  menu.setAttribute('aria-expanded', String(abierto));
  if (abierto) document.getElementById('seleccion-cerrar-menu').focus();
});
document.getElementById('seleccion-cerrar-menu').addEventListener('click', () => {
  document.body.dataset.menuAbierto = 'false'; menu.setAttribute('aria-expanded', 'false'); menu.focus();
});
document.addEventListener('keydown', evento => {
  if (evento.key === 'Escape' && menu.getAttribute('aria-expanded') === 'true') {
    document.body.dataset.menuAbierto = 'false'; menu.setAttribute('aria-expanded', 'false'); menu.focus();
  }
});
const ayuda = document.getElementById('seleccion-ayuda');
const panelAyuda = document.getElementById('seleccion-panel-ayuda');
ayuda.addEventListener('click', () => {
  panelAyuda.hidden = !panelAyuda.hidden;
  ayuda.setAttribute('aria-expanded', String(!panelAyuda.hidden));
});
const montaje = montarSeleccion({ raiz: document.getElementById('seleccion-raiz'), textos });
window.addEventListener('pagehide', () => montaje.desmontar(), { once: true });
