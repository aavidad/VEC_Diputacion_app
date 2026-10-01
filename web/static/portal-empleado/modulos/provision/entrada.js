import { cargarTextos } from '../../../comun/textos.js';
import { INDICE_IDIOMAS } from '../../../comun/idioma.js';
import { crearClienteProvisionLocal } from './cliente-local.js?v=20261001-provision-v1';
import { montarModuloProvision } from './montaje.js?v=20261001-provision-v1';
const textos = await cargarTextos('provision'); const t = textos.traducir;
document.documentElement.lang = textos.idioma; document.title = t('titulo');
document.querySelectorAll('[data-texto]').forEach(n => { n.textContent = t(n.dataset.texto); });
const selector = document.querySelector('#provision-idioma');
INDICE_IDIOMAS.idiomas.forEach(i => { const opt = document.createElement('option'); opt.value = i.codigo; opt.textContent = i.nombre; opt.selected = i.codigo === textos.idioma; selector.append(opt); });
selector.addEventListener('change', () => { const url = new URL(location.href); url.searchParams.set('lang', selector.value); location.assign(url); });
const raiz = document.querySelector('#espacio-trabajo'); const cliente = crearClienteProvisionLocal(); let montaje = null; let controlador = null;
function mensaje(clave) { const p = document.createElement('p'); p.textContent = t(clave); p.setAttribute('role', 'status'); raiz.replaceChildren(p); }
async function cargar() {
  controlador?.abort(); controlador = new AbortController(); mensaje('carga_preparacion');
  try { const dto = await cliente.listar({ signal: controlador.signal }); if (!Array.isArray(dto.ejemplos) || !dto.ejemplos.length) { mensaje('estados.sin_puestos'); return; } const label = document.createElement('label'); label.className = 'campo';
    const nombre = document.createElement('span'); nombre.textContent = t('ejemplo_selector');
    const select = document.createElement('select'); select.id = 'provision-ejemplo';
    dto.ejemplos.forEach((ejemplo, i) => { const option = document.createElement('option'); option.value = String(i); option.textContent = ejemplo.proyeccion?.denominacion_clave ? t(ejemplo.proyeccion.denominacion_clave) : ejemplo.proyeccion?.denominacion ?? ejemplo.ejemplo_ref; select.append(option); });
    label.append(nombre, select); const espacio = document.createElement('div'); raiz.replaceChildren(label, espacio);
    const elegir = async () => { montaje?.desmontar(); const ejemplo = dto.ejemplos[Number(select.value)]; montaje = await montarModuloProvision({ raiz: espacio, cliente, preparacion: ejemplo, proyeccion: ejemplo.proyeccion, textos }); };
    select.addEventListener('change', elegir); await elegir(); }
  catch (error) { if (controlador.signal.aborted) return; mensaje(error?.codigo === 'denegado' ? 'estados.denegado' : 'error_preparacion'); const b = document.createElement('button'); b.className = 'boton-secundario'; b.type = 'button'; b.textContent = t('reintentar_preparacion'); b.addEventListener('click', cargar); raiz.append(b); }
}
window.addEventListener('pagehide', () => { controlador?.abort(); montaje?.desmontar(); });
await cargar();
