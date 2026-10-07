/** Preparación efímera: no interpreta reglas ni calcula puntuaciones. */
export const VISTAS = Object.freeze(['convocatoria', 'puestos', 'personal', 'valoracion', 'adjudicacion', 'ciclo', 'tramitacion']);
export function crearEstado({ ejemplo_ref, proceso, preferencias = [] }) {
  if (!ejemplo_ref || !proceso?.configuracion || !Array.isArray(proceso.puestos)) throw new TypeError('provision.preparacion');
  const copia = structuredClone(proceso);
  const refs = new Set(copia.puestos.map(p => p.referencia));
  const ordenadas = [...preferencias].sort((a, b) => a.orden - b.orden).map(p => p.puesto_ref);
  if (new Set(ordenadas).size !== ordenadas.length || ordenadas.some(ref => !refs.has(ref))) throw new TypeError('provision.preferencias');
  return { ejemplo_ref, proceso: copia, preferencias: ordenadas, vista: 'convocatoria', resultado: null, estado: 'pendiente', mensaje: '', revision: 0 };
}
export function cambiarPreferencias(estado, referencia, accion) {
  if (!estado.proceso.puestos.some(p => p.referencia === referencia)) return estado;
  const preferencias = [...estado.preferencias];
  const indice = preferencias.indexOf(referencia);
  if (accion === 'seleccionar' && indice < 0) preferencias.push(referencia);
  else if (accion === 'quitar' && indice >= 0) preferencias.splice(indice, 1);
  else if (accion === 'arriba' && indice > 0) [preferencias[indice - 1], preferencias[indice]] = [preferencias[indice], preferencias[indice - 1]];
  else if (accion === 'abajo' && indice >= 0 && indice < preferencias.length - 1) [preferencias[indice + 1], preferencias[indice]] = [preferencias[indice], preferencias[indice + 1]];
  else return estado;
  return { ...estado, preferencias, resultado: null, estado: 'pendiente', revision: estado.revision + 1 };
}
export function peticionSimulacion(estado) {
  return { ejemplo_ref: estado.ejemplo_ref, configuracion: structuredClone(estado.proceso.configuracion), preferencias: estado.preferencias.map((puesto_ref, i) => ({ orden: i + 1, puesto_ref })) };
}
export function validarResultado(resultado, estado) {
  if (!resultado || resultado.alcance !== 'simulacion' || resultado.estado !== 'borrador'
    || resultado.proceso?.referencia !== estado.proceso.referencia
    || resultado.proceso?.version !== estado.proceso.version
    || resultado.solicitud?.version_reglas !== estado.proceso.configuracion.version
    || !Array.isArray(resultado.valoraciones) || resultado.valoraciones.length !== estado.preferencias.length) throw new TypeError('provision.resultado');
  const vistos = new Set();
  for (const v of resultado.valoraciones) {
    const r = v.resultado;
    if (vistos.has(v.puesto_ref) || estado.preferencias[v.orden - 1] !== v.puesto_ref
      || !['cumple', 'no_cumple', 'pendiente'].includes(v.requisitos_estado)
      || !r || r.puesto_ref !== v.puesto_ref || !Array.isArray(r.desglose)
      || (r.completo === true ? !/^\d+$/.test(r.total) : r.total !== null)
      || r.desglose.some(d => !['calculado', 'pendiente_dato'].includes(d.estado) || !/^\d+$/.test(d.resultado))) throw new TypeError('provision.resultado');
    vistos.add(v.puesto_ref);
  }
  return structuredClone(resultado);
}
/** Solo formatea micro-puntos recibidos; BigInt evita perder precisión. */
export function formatearPuntos(valor, localizacion) {
  if (typeof valor !== 'string' || !/^\d+$/.test(valor)) throw new TypeError('provision.puntos');
  const entero = BigInt(valor);
  const parte = new Intl.NumberFormat(localizacion).format(entero / 1000000n);
  const decimales = (entero % 1000000n).toString().padStart(6, '0').replace(/0+$/, '');
  const separador = new Intl.NumberFormat(localizacion).formatToParts(1.1).find(p => p.type === 'decimal')?.value;
  return decimales ? `${parte}${separador}${decimales}` : parte;
}
/** Conversión de entrada decimal a la unidad del contrato, sin cálculo de baremo. */
export function puntosDesdeDecimal(valor) {
  const m = /^(\d{1,12})(?:[.,](\d{1,6}))?$/.exec(String(valor));
  if (!m) throw new TypeError('provision.decimal');
  return (BigInt(m[1]) * 1000000n + BigInt((m[2] ?? '').padEnd(6, '0'))).toString();
}
function fechaCivilValida(valor) {
  if (typeof valor !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00.000Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor;
}
export function actualizarConfiguracion(estado, { campo, valor, regla }) {
  const proceso = structuredClone(estado.proceso);
  if (regla !== undefined) {
    if (!['coeficiente', 'maximo'].includes(campo) || !proceso.configuracion.reglas[regla]) throw new TypeError('provision.configuracion');
    proceso.configuracion.reglas[regla][campo] = puntosDesdeDecimal(valor);
  } else {
    if (!['fecha_corte', 'ventana_desde'].includes(campo)) throw new TypeError('provision.fecha');
    const fechas = {
      ventana_desde: estado.invalidos?.ventana_desde ?? proceso.configuracion.ventana_desde,
      fecha_corte: estado.invalidos?.fecha_corte ?? proceso.configuracion.fecha_corte,
      [campo]: valor,
    };
    if (!fechaCivilValida(fechas.ventana_desde) || !fechaCivilValida(fechas.fecha_corte)
      || fechas.ventana_desde >= fechas.fecha_corte) throw new TypeError('provision.fecha');
    Object.assign(proceso.configuracion, fechas);
  }
  return { ...estado, proceso, resultado: null, estado: 'pendiente', revision: estado.revision + 1 };
}
