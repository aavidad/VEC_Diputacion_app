import { crearEstado, cambiarPreferencias, peticionSimulacion, validarResultado, actualizarConfiguracion, VISTAS } from './modelo.js?v=20261001-provision-validacion-v2';
import { pintarProvision } from './vista.js?v=20261001-provision-validacion-v2';
import { cargarTextos } from '../../../comun/textos.js';
export async function montarModuloProvision({ raiz, cliente, preparacion, proyeccion = {}, textos, registrarDesmontar } = {}) {
  if (!raiz?.ownerDocument || !preparacion) throw new TypeError('provision.montaje');
  textos ??= await cargarTextos('provision');
  let estado = crearEstado(preparacion); let activa = true; let controlador = null; let turno = 0;
  const cancelar = () => { turno += 1; controlador?.abort(); controlador = null; };
  const pintar = (foco) => { if (!activa) return; pintarProvision({ raiz, estado, textos, proyeccion, acciones }); if (foco) (Array.from(raiz.querySelectorAll('[data-foco]')).find(n => n.dataset.foco === foco && !n.disabled) ?? Array.from(raiz.querySelectorAll('[data-foco]')).find(n => n.dataset.foco === foco.replace(/^abajo-/, 'arriba-').replace(/^arriba-/, foco.startsWith('arriba-') ? 'abajo-' : 'arriba-') && !n.disabled) ?? Array.from(raiz.querySelectorAll('[data-foco]')).find(n => n.dataset.foco === `vista-${estado.vista}`))?.focus(); };
  const acciones = {
    puedeSimular: typeof cliente?.simular === 'function',
    vista(vista, foco) { if (!VISTAS.includes(vista)) return; cancelar(); estado = { ...estado, vista, estado: estado.estado === 'cargando' ? 'pendiente' : estado.estado }; pintar(foco ?? `vista-${vista}`); },
    preferencia(ref, accion) { const siguiente = cambiarPreferencias(estado, ref, accion); if (siguiente === estado) return; cancelar(); estado = siguiente; const proy = proyeccion.puestos?.[ref]; const puesto = proy?.denominacion_clave ? textos.traducir(proy.denominacion_clave) : proy?.denominacion ?? ref; const clave = accion === 'seleccionar' ? 'preferencia_agregada' : accion === 'quitar' ? 'preferencia_quitada' : 'preferencia_movida'; estado.mensaje = textos.traducir(clave, { puesto, orden: textos.numero(estado.preferencias.indexOf(ref) + 1) }); pintar(`${accion}-${ref}`); },
    configuracion(cambio) {
      const clave = cambio.regla === undefined ? cambio.campo : `${cambio.campo}-${cambio.regla}`;
      let valido = false;
      try {
        const siguiente = actualizarConfiguracion(estado, cambio); cancelar(); estado = siguiente;
        estado.invalidos = { ...estado.invalidos }; estado.errores = { ...estado.errores };
        delete estado.invalidos[clave]; delete estado.errores[clave]; valido = true;
      } catch (error) {
        estado.invalidos = { ...estado.invalidos, [clave]: cambio.valor };
        estado.errores = { ...estado.errores, [clave]: error.message === 'provision.fecha' ? 'configuracion.fecha_invalida' : 'configuracion.invalida' };
      }
      const hayErrores = Object.keys(estado.invalidos).length > 0;
      estado.mensaje = textos.traducir(hayErrores ? 'configuracion.errores_pendientes' : 'configuracion.actualizada');
      const aviso = raiz.querySelector('[data-provision-aviso]');
      if (aviso) { aviso.textContent = estado.mensaje; aviso.setAttribute('role', hayErrores ? 'alert' : 'status'); }
      return { valido, mensaje: valido ? '' : textos.traducir(estado.errores[clave]) };
    },
    async simular() {
      if (!activa || !acciones.puedeSimular || !estado.preferencias.length || Object.keys(estado.invalidos ?? {}).length > 0) return;
      cancelar(); controlador = new AbortController(); const intento = turno; const revision = estado.revision;
      estado = { ...estado, estado: 'cargando', resultado: null, mensaje: '' }; pintar();
      try { const resultado = await cliente.simular(peticionSimulacion(estado), { signal: controlador.signal }); if (!activa || intento !== turno || revision !== estado.revision) return; estado = { ...estado, resultado: validarResultado(resultado, estado), estado: 'completo' }; }
      catch (error) { if (!activa || intento !== turno) return; estado = { ...estado, resultado: null, estado: error?.codigo === 'denegado' ? 'denegado' : error?.codigo === 'conflicto' ? 'conflicto' : error?.codigo === 'validacion' ? 'validacion' : 'error' }; }
      pintar('simular');
    },
  };
  const desmontar = () => { if (!activa) return; activa = false; cancelar(); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar); pintar();
  return { desmontar, obtenerEstado: () => structuredClone(estado) };
}
