import { crearClienteEnsayosLocal } from './ensayos-cliente.js?v=20261001-provision-ciclo-v5';
import { cargarTextos } from '../../../comun/textos.js';
import { INDICE_IDIOMAS } from '../../../comun/idioma.js';
import { crearClienteProvisionLocal } from './cliente-local.js?v=20261001-provision-ciclo-v5';
import { montarModuloProvision } from './montaje.js?v=20261001-provision-ciclo-v5';
const textos = await cargarTextos('provision'); const t = textos.traducir;
document.documentElement.lang = textos.idioma; document.title = t('titulo');
document.querySelectorAll('[data-texto]').forEach(n => { n.textContent = t(n.dataset.texto); });
const selector = document.querySelector('#provision-idioma');
INDICE_IDIOMAS.idiomas.forEach(i => { const opt = document.createElement('option'); opt.value = i.codigo; opt.textContent = i.nombre; opt.selected = i.codigo === textos.idioma; selector.append(opt); });
selector.addEventListener('change', () => { const url = new URL(location.href); url.searchParams.set('lang', selector.value); location.assign(url); });
const raiz = document.querySelector('#espacio-trabajo'); const cliente = crearClienteProvisionLocal(); const clienteEnsayos = crearClienteEnsayosLocal(); let montaje = null; let controlador = null;
function mensaje(clave) { const p = document.createElement('p'); p.textContent = t(clave); p.setAttribute('role', 'status'); raiz.replaceChildren(p); }

// El encabezado móvil es fijo; conserva el foco visible al recorrer con teclado.
const focoActivo = new AbortController(); let frameFoco = null;
function mantenerFocoVisible(evento) {
  const elemento = evento.target;
  if (!raiz.contains(elemento)) return;
  if (frameFoco !== null) cancelAnimationFrame(frameFoco);
  frameFoco = requestAnimationFrame(() => {
    frameFoco = null;
    if (focoActivo.signal.aborted || document.activeElement !== elemento || !elemento.isConnected) return;
    const caja = elemento.getBoundingClientRect(); const espacio = raiz.getBoundingClientRect();
    const cabecera = document.querySelector('.cabecera-portal').getBoundingClientRect();
    const limiteSuperior = Math.max(0, cabecera.bottom, espacio.top);
    const limiteInferior = Math.min(window.innerHeight, espacio.bottom);
    if (caja.top < limiteSuperior || caja.bottom > limiteInferior) elemento.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' });
  });
}
document.addEventListener('focusin', mantenerFocoVisible, { signal: focoActivo.signal });
window.addEventListener('pagehide', () => { focoActivo.abort(); if (frameFoco !== null) cancelAnimationFrame(frameFoco); });

async function cargar() {
  controlador?.abort(); controlador = new AbortController(); mensaje('carga_preparacion');
  try { const dto = await cliente.listar({ signal: controlador.signal }); if (!Array.isArray(dto.ejemplos) || !dto.ejemplos.length) { mensaje('estados.sin_puestos'); return; } const label = document.createElement('label'); label.className = 'campo';
    const nombre = document.createElement('span'); nombre.textContent = t('ejemplo_selector');
    const select = document.createElement('select'); select.id = 'provision-ejemplo';
    dto.ejemplos.forEach((ejemplo, i) => { const option = document.createElement('option'); option.value = String(i); option.textContent = ejemplo.proyeccion?.denominacion_clave ? t(ejemplo.proyeccion.denominacion_clave) : ejemplo.proyeccion?.denominacion ?? ejemplo.ejemplo_ref; select.append(option); });
    label.append(nombre, select); const espacio = document.createElement('div'); raiz.replaceChildren(label, espacio);
    const elegir = async () => { montaje?.desmontar(); const ejemplo = dto.ejemplos[Number(select.value)]; montaje = await montarModuloProvision({ raiz: espacio, cliente, clienteEnsayos, preparacion: ejemplo, proyeccion: ejemplo.proyeccion, textos }); };
    select.addEventListener('change', elegir); await elegir(); }
  catch (error) { if (controlador.signal.aborted) return; mensaje(error?.codigo === 'denegado' ? 'estados.denegado' : 'error_preparacion'); const b = document.createElement('button'); b.className = 'boton-secundario'; b.type = 'button'; b.textContent = t('reintentar_preparacion'); b.addEventListener('click', cargar); raiz.append(b); }
}
window.addEventListener('pagehide', () => { controlador?.abort(); montaje?.desmontar(); });
await cargar();
