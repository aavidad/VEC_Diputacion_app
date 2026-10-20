import { crearLectorBasesHTTP } from './cliente-http.js?v=VEC_WEB_VERSION';
import { crearEscritorBasesHTTP, prepararGuardado } from './guardar.js?v=VEC_WEB_VERSION';
import { validarReferencia, validarRevision, validarHuella } from './contrato-http.js?v=VEC_WEB_VERSION';
import { leerArchivo } from './modelo.js?v=VEC_WEB_VERSION';
import { pintarSalida } from './vista.js?v=VEC_WEB_VERSION';
import { cargarTextos } from '../../../../comun/textos.js';
import { INDICE_IDIOMAS, leerRecursoJSON } from '../../../../comun/idioma.js';

const archivo = document.querySelector('#bases-archivo'); const estado = document.querySelector('#bases-estado');
const raiz = document.querySelector('#bases-resultados'); const descargar = document.querySelector('#bases-descargar');
const formulario = document.querySelector('#bases-consulta'); const consultar = document.querySelector('#bases-consultar');
const cancelar = document.querySelector('#bases-cancelar');
const lector = crearLectorBasesHTTP(); let peticion = null;
const escritor = crearEscritorBasesHTTP(); const edicion = document.querySelector('#bases-edicion');
const titulo = document.querySelector('#bases-titulo'), modalidad = document.querySelector('#bases-modalidad');
const guardar = document.querySelector('#bases-guardar'), reintentar = document.querySelector('#bases-reintentar');
const confirmacion = document.querySelector('#bases-confirmacion'), avisoEdicion = document.querySelector('#bases-edicion-error');
let baseEdicion = null, pendienteGuardado = null, peticionGuardar = null;
let avisoGuardado = { clave: '', error: false, variables: {} };
function estadoEdicion(clave, error = false, variables = {}) {
  avisoGuardado = { clave, error, variables };
  confirmacion.textContent = clave ? textos.traducir(clave, variables) : '';
  confirmacion.setAttribute('role', error ? 'alert' : 'status');
}
function habilitarEdicion() {
  const ocupada = Boolean(peticionGuardar || pendienteGuardado || peticion);
  titulo.disabled = modalidad.disabled = ocupada; guardar.disabled = ocupada || !textos;
  reintentar.hidden = !pendienteGuardado; reintentar.disabled = !pendienteGuardado || Boolean(peticionGuardar);
  if (textos) { consultar.disabled = Boolean(pendienteGuardado || peticionGuardar); archivo.disabled = Boolean(pendienteGuardado || peticionGuardar); }
}
async function enviarGuardado() {
  const solicitud = pendienteGuardado;
  peticionGuardar = new AbortController(); habilitarEdicion(); estadoEdicion('guardando');
  try {
    const dto = await escritor.guardar(solicitud, { signal: peticionGuardar.signal });
    if (pendienteGuardado !== solicitud) return;
    baseEdicion = dto; pendienteGuardado = null;
    document.querySelector('#bases-referencia').value = dto.preparacion.estado.preparacion_ref;
    pintarSalida({ raiz, dto, textos, institucional: true }); carga = { dto, institucional: true };
    descargar.hidden = true; cerrar.disabled = false;
    estadoEdicion('guardado', false, { referencia: dto.preparacion.estado.preparacion_ref, recibo: dto.recibo.recibo_ref });
  } catch (error) {
    if (pendienteGuardado === solicitud) estadoEdicion(`guardado_errores.${error.codigo in textos.mensajes.guardado_errores ? error.codigo : 'incierto'}`, true);
  } finally { peticionGuardar = null; habilitarEdicion(); }
}
edicion.addEventListener('submit', e => {
  e.preventDefault(); if (pendienteGuardado || peticionGuardar || !textos) return;
  avisoEdicion.hidden = true;
  if (!titulo.value.trim() || titulo.value.length > 180 || !['separada', 'conjunta'].includes(modalidad.value)) {
    avisoEdicion.textContent = textos.traducir('edicion_validacion'); avisoEdicion.hidden = false;
    (!titulo.value.trim() ? titulo : modalidad).focus(); return;
  }
  try { pendienteGuardado = prepararGuardado(baseEdicion, titulo.value, modalidad.value); enviarGuardado(); }
  catch { estadoEdicion('guardado_errores.solicitud_invalida', true); }
});
reintentar.addEventListener('click', () => { if (pendienteGuardado && !peticionGuardar) enviarGuardado(); });
edicion.addEventListener('input', () => { if (!pendienteGuardado) { avisoEdicion.hidden = true; estadoEdicion(''); } });
const cerrar = document.querySelector('#bases-cerrar'); const urls = new Set();
let turno = 0; let carga = null; let textos; let estadoActual = { clave: 'vacio', error: false, variables: {} };
let erroresFormulario = [];
function mostrarErroresFormulario() {
  for (const id of ['bases-referencia', 'bases-revision', 'bases-huella']) {
    const entrada = document.getElementById(id), aviso = document.getElementById(`${id}-error`);
    const error = erroresFormulario.find(item => item.id === id);
    entrada.setAttribute('aria-invalid', String(Boolean(error)));
    aviso.hidden = !error;
    aviso.textContent = error ? textos.traducir(error.clave) : '';
  }
}
function borrarErroresFormulario() { erroresFormulario = []; mostrarErroresFormulario(); }
function validarFormulario() {
  const referencia = document.getElementById('bases-referencia');
  const revision = document.getElementById('bases-revision');
  const huella = document.getElementById('bases-huella');
  erroresFormulario = [];
  if (!validarReferencia(referencia.value)) erroresFormulario.push({ id: referencia.id, clave: 'consulta_validacion.referencia' });
  if (document.getElementById('bases-modo').value === 'exacta') {
    if (!validarRevision(Number(revision.value)) || revision.value.trim() === '') erroresFormulario.push({ id: revision.id, clave: 'consulta_validacion.revision' });
    if (!validarHuella(huella.value)) erroresFormulario.push({ id: huella.id, clave: 'consulta_validacion.huella' });
  }
  mostrarErroresFormulario();
  if (!erroresFormulario.length) return true;
  mensaje('consulta_errores.selector_invalido', true);
  document.getElementById(erroresFormulario[0].id).focus();
  return false;
}
function mensaje(clave, error = false, variables = {}) {
  estadoActual = { clave, error, variables };
  estado.textContent = textos.traducir(clave, variables); estado.setAttribute('role', error ? 'alert' : 'status');
  archivo.setAttribute('aria-invalid', String(error && ['error_formato', 'error_tamano'].includes(clave)));
}
function limpiar() {
  ++turno; peticion?.abort(); peticion = null; carga = null; descargar.disabled = true; descargar.hidden = false; cancelar.disabled = true; cerrar.disabled = true; raiz.replaceChildren();
  urls.forEach(url => URL.revokeObjectURL(url)); urls.clear();
  if (textos) habilitarEdicion();
}
archivo.addEventListener('change', async () => {
  limpiar(); baseEdicion = null; borrarErroresFormulario(); const actual = turno; const file = archivo.files?.[0];
  if (!file) { mensaje('vacio'); return; }
  cerrar.disabled = false; mensaje('cargando');
  try {
    const resultado = await leerArchivo(file); if (actual !== turno) return;
    pintarSalida({ raiz, dto: resultado.dto, textos }); carga = { ...resultado, institucional: false, nombre: file.name }; descargar.disabled = false;
    mensaje('cargado', false, { archivo: file.name });
  } catch (error) { if (actual === turno) mensaje(error.message === 'tamano' ? 'error_tamano' : 'error_formato', true); }
});
cerrar.addEventListener('click', () => { limpiar(); archivo.value = ''; mensaje('vacio'); archivo.focus(); });
function descargarOriginal() {
  if (!carga || carga.institucional) return;
  const url = URL.createObjectURL(new Blob([carga.bytes], { type: 'application/json' })); urls.add(url);
  const enlace = document.createElement('a'); enlace.href = url; enlace.download = textos.traducir('nombre_descarga');
  document.body.append(enlace); enlace.click(); enlace.remove(); mensaje('descarga_lista');
  setTimeout(() => { URL.revokeObjectURL(url); urls.delete(url); }, 1000);
}
descargar.addEventListener('click', descargarOriginal);

function selectorConsulta() {
  const modo = document.querySelector('#bases-modo').value;
  return { modo, preparacion_ref: document.querySelector('#bases-referencia').value,
    revision: modo === 'actual' ? 0 : Number(document.querySelector('#bases-revision').value),
    huella_material_sha256: modo === 'actual' ? '' : document.querySelector('#bases-huella').value };
}
formulario.addEventListener('submit', async e => {
  e.preventDefault(); limpiar(); baseEdicion = null; archivo.value = ''; if (!validarFormulario()) return; const actual = turno;
  peticion = new AbortController(); cancelar.disabled = false; mensaje('consultando');
  habilitarEdicion();
  try {
    const resultado = await lector.consultar(selectorConsulta(), { signal: peticion.signal });
    if (actual !== turno) return;
    carga = { ...resultado, institucional: true }; pintarSalida({ raiz, dto: carga.dto, textos, institucional: true });
    baseEdicion = carga.dto; titulo.value = carga.dto.preparacion.material.contenido.titulo;
    modalidad.value = carga.dto.preparacion.material.contenido.modalidad_inscripcion || '';
    estadoEdicion('edicion_cargada');
    descargar.hidden = true; cerrar.disabled = false;
    mensaje('consulta_obtenida');
  } catch (error) { if (actual === turno) { limpiar(); mensaje(`consulta_errores.${error.codigo || 'servicio_no_disponible'}`, true); } }
  finally { if (actual === turno) { peticion = null; cancelar.disabled = true; habilitarEdicion(); } }
});
formulario.addEventListener('input', () => { limpiar(); baseEdicion = null; estadoEdicion(''); borrarErroresFormulario(); archivo.value = ''; mensaje('consulta_lista'); });
formulario.addEventListener('change', () => {
  limpiar(); baseEdicion = null; estadoEdicion(''); borrarErroresFormulario(); archivo.value = ''; mensaje('consulta_lista');
  const exacta = document.querySelector('#bases-modo').value === 'exacta'; document.querySelector('#bases-exacta').hidden = !exacta;
  for (const id of ['bases-revision', 'bases-huella']) document.getElementById(id).required = exacta;
});
cancelar.addEventListener('click', () => { limpiar(); borrarErroresFormulario(); mensaje('consulta_cancelada'); consultar.focus(); });

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
  mostrarErroresFormulario();
  estadoEdicion(avisoGuardado.clave, avisoGuardado.error, avisoGuardado.variables);
  document.querySelectorAll('[data-aria]').forEach(n => { n.setAttribute('aria-label', textos.traducir(n.dataset.aria)); });
  if (carga) pintarSalida({ raiz, dto: carga.dto, textos, institucional: carga.institucional });
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
window.addEventListener('pagehide', () => { limpiar(); peticionGuardar?.abort(); foco.abort(); cancelAnimationFrame(frame); urls.forEach(url => URL.revokeObjectURL(url)); });
try {
  archivo.disabled = true; ayuda.disabled = true; consultar.disabled = true;
  const inicial = await cargarTextos('seleccion-bases-preparacion'); await idioma(inicial.idioma);
  const selector = document.querySelector('#bases-idioma');
  for (const i of INDICE_IDIOMAS.idiomas) { const opt = document.createElement('option'); opt.value = i.codigo; opt.textContent = i.nombre; opt.selected = i.codigo === textos.idioma; selector.append(opt); }
  selector.addEventListener('change', async () => { selector.disabled = true; try { await idioma(selector.value); } catch { mensaje('error_catalogo', true); } finally { selector.value = textos.idioma; selector.disabled = false; } });
  archivo.disabled = false; ayuda.disabled = false; consultar.disabled = false; habilitarEdicion();
} catch {
  // Un catálogo ausente no habilita un visor sin límites comprensibles.
  archivo.disabled = true; ayuda.disabled = true; consultar.disabled = true; estado.setAttribute('role', 'alert');
  const respaldo = await cargarTextos('seleccion-bases-preparacion-error', textos ? { idioma: textos.idioma } : {})
    .then(t => ({ idioma: t.idioma, titulo: t.traducir('titulo'), mensaje: t.traducir('mensaje') }))
    .catch(() => leerRecursoJSON(new URL('./error-catalogo.json', import.meta.url))).catch(() => null);
  if (respaldo) { estado.textContent = respaldo.mensaje; document.documentElement.lang = respaldo.idioma; document.title = respaldo.titulo; document.querySelector('h1').textContent = respaldo.titulo; }
}
