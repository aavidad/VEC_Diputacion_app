import { crearLectorBasesHTTP } from './cliente-http.js?v=20261016-f140-modalidad-v1';
import { validarReferencia, validarRevision, validarHuella } from './contrato-http.js?v=20261016-f140-modalidad-v1';
import { leerArchivo } from './modelo.js?v=20261016-f140-modalidad-v1';
import { pintarSalida, enfocarEstadoSiConfirmacionOculta, eleccionTrasConsulta, crearIntentoGuardado } from './vista.js?v=20261016-f140-modalidad-v2';
import { cargarTextos } from '../../../../comun/textos.js';
import { INDICE_IDIOMAS, leerRecursoJSON } from '../../../../comun/idioma.js';

const archivo = document.querySelector('#bases-archivo'); const estado = document.querySelector('#bases-estado');
const raiz = document.querySelector('#bases-resultados'); const descargar = document.querySelector('#bases-descargar');
const formulario = document.querySelector('#bases-consulta'); const consultar = document.querySelector('#bases-consultar');
const cancelar = document.querySelector('#bases-cancelar');
const lector = crearLectorBasesHTTP(); let peticion = null;
const cerrar = document.querySelector('#bases-cerrar'); const urls = new Set();
const panelGuardar = document.querySelector('#bases-guardar-panel');
const formularioGuardar = document.querySelector('#bases-guardar');
const modalidad = document.querySelector('#bases-modalidad');
const revisar = document.querySelector('#bases-revisar');
const confirmar = document.querySelector('#bases-confirmar');
const reintentarGuardado = document.querySelector('#bases-reintentar');
const resumenGuardar = document.querySelector('#bases-guardar-revision');
const avisoGuardar = document.querySelector('#bases-guardar-estado');
let turno = 0; let carga = null; let textos; let estadoActual = { clave: 'vacio', error: false, variables: {} };
let guardadoPendiente = null; let guardando = false; let revisionGuardado = false;
let deseoConflicto = null;
let estadoGuardado = { clave: '', error: false, variables: {} };
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
function mensajeGuardado(clave = '', error = false, variables = {}) {
  estadoGuardado = { clave, error, variables };
  avisoGuardar.textContent = clave ? textos.traducir(clave, variables) : '';
  avisoGuardar.setAttribute('role', error ? 'alert' : 'status');
}
function pintarGuardado() {
  const focoConfirmar = document.activeElement === confirmar;
  panelGuardar.hidden = !carga?.institucional;
  if (panelGuardar.hidden) return;
  const pendiente = Boolean(guardadoPendiente) || guardando;
  const bloqueado = pendiente || ['guardar_version_en_conflicto', 'guardar_no_encontrada'].includes(estadoGuardado.clave);
  modalidad.disabled = bloqueado;
  revisar.hidden = revisionGuardado || bloqueado;
  confirmar.hidden = !revisionGuardado || bloqueado;
  reintentarGuardado.hidden = !guardadoPendiente || guardando;
  resumenGuardar.hidden = !revisionGuardado || bloqueado;
  if (!resumenGuardar.hidden) resumenGuardar.textContent = textos.traducir('guardar_revision', {
    modalidad: textos.traducir(`modalidad_inscripcion.${modalidad.value}`),
  });
  consultar.disabled = pendiente;
  archivo.disabled = pendiente;
  cerrar.disabled = pendiente;
  enfocarEstadoSiConfirmacionOculta(document, confirmar, avisoGuardar, focoConfirmar);
}
function limpiar() {
  if (guardadoPendiente || guardando) return;
  ++turno; peticion?.abort(); peticion = null; carga = null; descargar.disabled = true; descargar.hidden = false; cancelar.disabled = true; cerrar.disabled = true; raiz.replaceChildren();
  panelGuardar.hidden = true; revisionGuardado = false; mensajeGuardado();
  urls.forEach(url => URL.revokeObjectURL(url)); urls.clear();
}
archivo.addEventListener('change', async () => {
  if (guardadoPendiente || guardando) { archivo.value = ''; return; }
  limpiar(); borrarErroresFormulario(); const actual = turno; const file = archivo.files?.[0];
  if (!file) { mensaje('vacio'); return; }
  cerrar.disabled = false; mensaje('cargando');
  try {
    const resultado = await leerArchivo(file); if (actual !== turno) return;
    pintarSalida({ raiz, dto: resultado.dto, textos }); carga = { ...resultado, institucional: false, nombre: file.name }; descargar.disabled = false;
    mensaje('cargado', false, { archivo: file.name });
  } catch (error) { if (actual === turno) mensaje(error.message === 'tamano' ? 'error_tamano' : 'error_formato', true); }
});
cerrar.addEventListener('click', () => { if (guardadoPendiente || guardando) return; limpiar(); archivo.value = ''; mensaje('vacio'); archivo.focus(); });
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
  e.preventDefault(); if (guardadoPendiente || guardando) return;
  limpiar(); archivo.value = ''; if (!validarFormulario()) return; const actual = turno;
  peticion = new AbortController(); cancelar.disabled = false; mensaje('consultando');
  try {
    const resultado = await lector.consultar(selectorConsulta(), { signal: peticion.signal });
    if (actual !== turno) return;
    carga = { ...resultado, institucional: true }; pintarSalida({ raiz, dto: carga.dto, textos, institucional: true });
    descargar.hidden = true; cerrar.disabled = false;
    const guardada = carga.dto.preparacion.material.contenido.modalidad_inscripcion || 'separada';
    const referencia = carga.dto.preparacion.estado.preparacion_ref;
    const eleccion = eleccionTrasConsulta(deseoConflicto, referencia, guardada);
    deseoConflicto = eleccion.deseo;
    modalidad.value = eleccion.modalidad;
    if (deseoConflicto) mensajeGuardado('guardar_comparacion', false, {
      guardada: textos.traducir(`modalidad_inscripcion.${guardada}`),
      deseada: textos.traducir(`modalidad_inscripcion.${modalidad.value}`),
    });
    pintarGuardado();
    mensaje('consulta_obtenida');
  } catch (error) { if (actual === turno) { limpiar(); mensaje(`consulta_errores.${error.codigo || 'servicio_no_disponible'}`, true); } }
  finally { if (actual === turno) { peticion = null; cancelar.disabled = true; } }
});
formulario.addEventListener('input', () => { if (guardadoPendiente || guardando) return; limpiar(); borrarErroresFormulario(); archivo.value = ''; mensaje('consulta_lista'); });
formulario.addEventListener('change', () => {
  if (guardadoPendiente || guardando) return;
  limpiar(); borrarErroresFormulario(); archivo.value = ''; mensaje('consulta_lista');
  const exacta = document.querySelector('#bases-modo').value === 'exacta'; document.querySelector('#bases-exacta').hidden = !exacta;
  for (const id of ['bases-revision', 'bases-huella']) document.getElementById(id).required = exacta;
});
cancelar.addEventListener('click', () => { if (guardadoPendiente || guardando) return; limpiar(); borrarErroresFormulario(); mensaje('consulta_cancelada'); consultar.focus(); });

modalidad.addEventListener('change', () => {
  if (guardadoPendiente || guardando) return;
  if (deseoConflicto) deseoConflicto.modalidad = modalidad.value;
  revisionGuardado = false;
  if (deseoConflicto) mensajeGuardado('guardar_comparacion', false, {
    guardada: textos.traducir(`modalidad_inscripcion.${carga.dto.preparacion.material.contenido.modalidad_inscripcion || 'separada'}`),
    deseada: textos.traducir(`modalidad_inscripcion.${modalidad.value}`),
  }); else mensajeGuardado();
  pintarGuardado();
});
revisar.addEventListener('click', () => {
  if (!carga?.institucional || guardadoPendiente || guardando) return;
  if (modalidad.value === carga.dto.preparacion.material.contenido.modalidad_inscripcion) {
    mensajeGuardado('guardar_sin_cambios'); return;
  }
  revisionGuardado = true; mensajeGuardado(); pintarGuardado(); confirmar.focus();
});
async function guardarModalidad() {
  if (!guardadoPendiente || guardando) return;
  guardando = true; mensajeGuardado('guardar_enviando'); pintarGuardado();
  try {
    const resultado = await lector.guardar(guardadoPendiente);
    carga = { ...resultado, institucional: true };
    guardadoPendiente = null; revisionGuardado = false;
    deseoConflicto = null;
    modalidad.value = carga.dto.preparacion.material.contenido.modalidad_inscripcion;
    pintarSalida({ raiz, dto: carga.dto, textos, institucional: true });
    mensajeGuardado(resultado.dto.estado === 'guardada' ? 'guardar_guardada' : 'guardar_recuperada', false,
      { recibo: resultado.dto.recibo.recibo_ref });
  } catch (error) {
    if (['version_en_conflicto', 'acceso_denegado', 'no_encontrada', 'selector_invalido'].includes(error?.codigo)) {
      if (error.codigo === 'version_en_conflicto') deseoConflicto = {
        referencia: carga.dto.preparacion.estado.preparacion_ref, modalidad: modalidad.value,
      };
      guardadoPendiente = null; revisionGuardado = false;
      mensajeGuardado(`guardar_${error.codigo}`, true);
    } else mensajeGuardado('guardar_pendiente', true);
  } finally { guardando = false; pintarGuardado(); }
}
formularioGuardar.addEventListener('submit', e => {
  e.preventDefault();
  if (!revisionGuardado || !carga?.institucional || guardadoPendiente || guardando) return;
  if (!['separada', 'conjunta'].includes(modalidad.value) || !globalThis.crypto?.randomUUID) {
    mensajeGuardado('guardar_selector_invalido', true); return;
  }
  guardadoPendiente = crearIntentoGuardado(carga, modalidad.value, () => crypto.randomUUID());
  void guardarModalidad();
});
reintentarGuardado.addEventListener('click', () => { if (guardadoPendiente && !guardando) void guardarModalidad(); });

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
  if (estadoGuardado.clave) mensajeGuardado(estadoGuardado.clave, estadoGuardado.error, estadoGuardado.variables);
  pintarGuardado();
  mostrarErroresFormulario();
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
window.addEventListener('pagehide', () => { limpiar(); foco.abort(); cancelAnimationFrame(frame); urls.forEach(url => URL.revokeObjectURL(url)); });
try {
  archivo.disabled = true; ayuda.disabled = true; consultar.disabled = true;
  const inicial = await cargarTextos('seleccion-bases-preparacion'); await idioma(inicial.idioma);
  const selector = document.querySelector('#bases-idioma');
  for (const i of INDICE_IDIOMAS.idiomas) { const opt = document.createElement('option'); opt.value = i.codigo; opt.textContent = i.nombre; opt.selected = i.codigo === textos.idioma; selector.append(opt); }
  selector.addEventListener('change', async () => { selector.disabled = true; try { await idioma(selector.value); } catch { mensaje('error_catalogo', true); } finally { selector.value = textos.idioma; selector.disabled = false; } });
  archivo.disabled = false; ayuda.disabled = false; consultar.disabled = false;
} catch {
  // Un catálogo ausente no habilita un visor sin límites comprensibles.
  archivo.disabled = true; ayuda.disabled = true; consultar.disabled = true; estado.setAttribute('role', 'alert');
  const respaldo = await cargarTextos('seleccion-bases-preparacion-error', textos ? { idioma: textos.idioma } : {})
    .then(t => ({ idioma: t.idioma, titulo: t.traducir('titulo'), mensaje: t.traducir('mensaje') }))
    .catch(() => leerRecursoJSON(new URL('./error-catalogo.json', import.meta.url))).catch(() => null);
  if (respaldo) { estado.textContent = respaldo.mensaje; document.documentElement.lang = respaldo.idioma; document.title = respaldo.titulo; document.querySelector('h1').textContent = respaldo.titulo; }
}
