/** Preparación y coordinación de ensayos; no contiene algoritmos de adjudicación. */
export function prepararAdjudicacion(ejemplo) {
  if (!ejemplo?.ejemplo_ref || ejemplo.configuracion?.schema_version !== 'provision.adjudicacion.v1' || !Array.isArray(ejemplo.configuracion.desempates) || !Array.isArray(ejemplo.resumen?.vacantes) || !Array.isArray(ejemplo.resumen?.solicitudes)) throw new TypeError('provision.ensayo');
  return { ...structuredClone(ejemplo), resultado: null, estado: 'listo', invalidos: {}, revision: 0 };
}
export function cambiarAdjudicacion(estado, cambio) {
  const siguiente = structuredClone(estado); const c = siguiente.configuracion;
  if (cambio.campo) {
    if (!['politica_ref', 'politica_version', 'bases_ref', 'version'].includes(cambio.campo) || typeof cambio.valor !== 'string' || cambio.valor.trim() !== cambio.valor || !cambio.valor || cambio.valor.length > 160) throw new TypeError('provision.ensayo.configuracion');
    c[cambio.campo] = cambio.valor;
  } else {
    const i = cambio.indice; if (!Number.isInteger(i) || !c.desempates[i]) throw new TypeError('provision.ensayo.desempate');
    if (cambio.sentido) { if (!['mayor', 'menor'].includes(cambio.sentido)) throw new TypeError('provision.ensayo.desempate'); c.desempates[i].sentido = cambio.sentido; }
    else if (cambio.accion === 'arriba' && i > 0) [c.desempates[i - 1], c.desempates[i]] = [c.desempates[i], c.desempates[i - 1]];
    else if (cambio.accion === 'abajo' && i < c.desempates.length - 1) [c.desempates[i + 1], c.desempates[i]] = [c.desempates[i], c.desempates[i + 1]];
    else if (cambio.accion === 'quitar') c.desempates.splice(i, 1);
    else throw new TypeError('provision.ensayo.desempate');
  }
  siguiente.resultado = null; siguiente.estado = 'listo'; siguiente.revision += 1; return siguiente;
}
export function validarAdjudicacion(resultado, estado) {
  const c = estado.configuracion;
  if (resultado?.schema_version !== 'provision.adjudicacion.v1' || resultado.alcance !== 'simulacion_sintetica_sin_efectos'
    || !['propuesta_simulada', 'pendiente', 'no_soportado'].includes(resultado.estado)
    || resultado.proceso_ref !== c.proceso_ref || resultado.version !== c.version || resultado.politica_ref !== c.politica_ref || resultado.politica_version !== c.politica_version || resultado.metodo !== c.metodo
    || !Array.isArray(resultado.asignaciones) || !Array.isArray(resultado.incidencias)
    || (resultado.estado !== 'propuesta_simulada' && resultado.asignaciones.length)) throw new TypeError('provision.ensayo.resultado');
  const personas = new Set(); const vacantes = new Set();
  for (const a of resultado.asignaciones) {
    if (personas.has(a.persona_ref) || vacantes.has(a.vacante_ref)
      || !estado.resumen.solicitudes.some(s => s.persona_ref === a.persona_ref)
      || !estado.resumen.vacantes.some(v => v.vacante_ref === a.vacante_ref && v.puesto_ref === a.puesto_ref)) throw new TypeError('provision.ensayo.resultado');
    personas.add(a.persona_ref); vacantes.add(a.vacante_ref);
  }
  return structuredClone(resultado);
}
export function crearControladorAdjudicacion({ cliente, notificar }) {
  let estado = { estado: 'cargando' }; let ejemplos = []; let controlador; let turno = 0; let activa = true;
  const avisar = () => { if (activa) notificar(structuredClone(estado)); };
  const cancelar = () => { turno += 1; controlador?.abort(); };
  const error = e => { estado = { ...estado, estado: ['validacion', 'denegado', 'conflicto'].includes(e?.codigo) ? e.codigo : 'error', resultado: null }; avisar(); };
  const elegir = indice => { cancelar(); if (!ejemplos[indice]) return; estado = prepararAdjudicacion(ejemplos[indice]); estado.ejemplos = ejemplos.map(e => e.ejemplo_ref); estado.indice = indice; avisar(); };
  async function cargar() {
    cancelar(); controlador = new AbortController(); const intento = turno; estado = { estado: 'cargando' }; avisar();
    try { const dto = await cliente.listarAdjudicaciones({ signal: controlador.signal }); if (!activa || turno !== intento) return; ejemplos = dto.ejemplos; if (!Array.isArray(ejemplos)) throw new TypeError('provision.ensayo'); if (!ejemplos.length) { estado = { estado: 'vacio' }; avisar(); } else elegir(0); }
    catch (e) { if (activa && turno === intento) error(e); }
  }
  function cambiar(cambio) {
    const clave = cambio.campo ?? `desempate-${cambio.indice}`;
    try { cancelar(); estado = cambiarAdjudicacion(estado, cambio); delete estado.invalidos[clave]; }
    catch { estado.invalidos = { ...estado.invalidos, [clave]: cambio.valor }; }
    estado.resultado = null;
    estado.estado = Object.keys(estado.invalidos).length ? 'validacion' : 'listo';
    return { valido: !Object.hasOwn(estado.invalidos, clave), estado: structuredClone(estado) };
  }
  async function simular() {
    if (!activa || !estado.configuracion || Object.keys(estado.invalidos).length) return;
    cancelar(); controlador = new AbortController(); const intento = turno;
    estado = { ...estado, estado: 'calculando', resultado: null }; avisar();
    try { const resultado = await cliente.simularAdjudicacion({ ejemplo_ref: estado.ejemplo_ref, configuracion: structuredClone(estado.configuracion) }, { signal: controlador.signal }); if (!activa || turno !== intento) return; estado = { ...estado, resultado: validarAdjudicacion(resultado, estado), estado: 'resultado' }; avisar(); }
    catch (e) { if (activa && turno === intento) error(e); }
  }
  return { cargar, elegir, cambiar, simular, repintar: avisar, desmontar() { activa = false; cancelar(); }, obtenerEstado: () => structuredClone(estado) };
}
