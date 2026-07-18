/**
 * Regla única de cálculo visual para el adaptador de presentación.
 *
 * Los conceptos vinculados a un mérito solo se incluyen si la persona lo ha
 * seleccionado. Los conceptos obtenidos de oficio por la convocatoria se
 * incluyen siempre. El servidor productivo sustituirá esta proyección por el
 * motor de reglas versionado, pero mantendrá el mismo contrato de resultado.
 */

function referenciasUnicas(valor) {
  const lista = Array.isArray(valor) ? valor : [];
  return [...new Set(lista.filter((item) => typeof item === "string" && item.trim() !== ""))];
}

function criterioPendiente(merito) {
  const puntos = Number(merito.puntos_estimados || 0);
  return {
    id: `criterio-${merito.id}`,
    merito_id: merito.id,
    nombre: merito.titulo,
    detalle: `${merito.tipo} · pendiente de regla aplicable`,
    estado: merito.estado,
    puntos,
    maximo: Math.max(1, puntos),
    de_oficio: false,
  };
}

export function calcularAutobaremo(datos, meritosIds) {
  const seleccionados = new Set(referenciasUnicas(meritosIds));
  const baremo = Array.isArray(datos?.baremo) ? datos.baremo : [];
  const meritos = Array.isArray(datos?.meritos) ? datos.meritos : [];
  const criterios = baremo.filter((criterio) => criterio.de_oficio === true
    || (typeof criterio.merito_id === "string" && seleccionados.has(criterio.merito_id)));
  const vinculados = new Set(criterios.map((criterio) => criterio.merito_id).filter(Boolean));
  meritos.filter((merito) => seleccionados.has(merito.id) && !vinculados.has(merito.id))
    .forEach((merito) => criterios.push(criterioPendiente(merito)));
  return Object.freeze({
    criterios: Object.freeze(criterios.map((criterio) => Object.freeze({ ...criterio }))),
    meritos_ids: Object.freeze([...seleccionados]),
    total: criterios.reduce((suma, criterio) => suma + Number(criterio.puntos || 0), 0),
    maximo: criterios.reduce((suma, criterio) => suma + Number(criterio.maximo || 0), 0),
  });
}
