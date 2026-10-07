import { nodo, tabla } from './dom.js?v=20261004-codexa-s6-notas-v1';

/** Renderiza exclusivamente la explicación recibida; no calcula admisión ni orden. */
export function mostrarResultado(raiz, resultado, configuracion, textos) {
  const d = raiz.ownerDocument; const t = textos.traducir;
  const puntos = valor => valor === null ? t('sin_puntuacion') : textos.numero(valor / 1000000, { maximumFractionDigits: 6 });
  const nombreFase = referencia => {
    const indice = configuracion.fases.findIndex(f => f.referencia === referencia);
    const fase = configuracion.fases[indice];
    return t(`fase.${fase.tipo}`, { numero: textos.numero(indice + 1) });
  };
  const avisos = nodo(d, 'div');
  avisos.append(nodo(d, 'p', t(`resultado.${resultado.estado}`)));
  if (resultado.causas.length) {
    const lista = nodo(d, 'ul'); resultado.causas.forEach(c => lista.append(nodo(d, 'li', t(`causa.${c}`)))); avisos.append(lista);
  }
  raiz.replaceChildren(avisos);
  if (!resultado.solicitudes.length) { raiz.append(nodo(d, 'p', t('sin_solicitudes'))); return; }
  const acceso = tabla(d, [t('solicitud'), t('acceso'), t('motivo')], t('acceso_titulo'));
  for (const s of resultado.solicitudes) {
    const fila = nodo(d, 'tr');
    const estadoAcceso = nodo(d, 'td');
    estadoAcceso.append(nodo(d, 'span', t(`acceso_estado.${s.acceso}`), `estado-chip ${s.acceso === 'cumple' ? 'exito' : s.acceso === 'no_cumple' ? 'peligro' : 'violeta'}`));
    fila.append(nodo(d, 'th', s.nombre), estadoAcceso);
    fila.firstChild.scope = 'row';
    const motivo = nodo(d, 'td'); motivo.className = 'envuelve';
    motivo.append(nodo(d, 'span', s.causas.filter(c => c.startsWith('requisito_')).map(c => t(`causa.${c}`)).join(' · ') || t('sin_incidencias_acceso')));
    fila.append(motivo); acceso.cuerpo.append(fila);
  }
  raiz.append(nodo(d, 'h3', t('acceso_titulo')), acceso.contenedor, nodo(d, 'h3', t('lista_titulo')));
  const lista = tabla(d, [t('solicitud'), t('orden'), t('puntos'), t('propuesta'), t('detalle')], t('lista_titulo'));
  for (const s of resultado.solicitudes) {
    const fila = nodo(d, 'tr');
    const nombre = nodo(d, 'th', s.nombre); nombre.scope = 'row';
    const orden = nodo(d, 'td', s.orden === null ? t('sin_orden') : textos.numero(s.orden), 'columna-numero');
    const total = nodo(d, 'td', puntos(s.total_micropuntos), 'columna-numero');
    const propuesta = nodo(d, 'td');
    propuesta.append(nodo(d, 'span', t(`propuesta_estado.${s.propuesta}`), `estado-chip ${s.propuesta === 'propuesta_provisional' ? 'info' : s.propuesta === 'pendiente' ? 'violeta' : 'neutro'}`));
    const detalle = nodo(d, 'td'); const desplegable = nodo(d, 'details');
    const resumen = nodo(d, 'summary', t('ver_detalle')); resumen.setAttribute('aria-label', t('detalle_de', { nombre: s.nombre }));
    desplegable.append(resumen, nodo(d, 'p', t(`solicitud_estado.${s.estado}`)));
    const requisitos = nodo(d, 'ul');
    for (const requisito of s.acceso_detalle) {
      const nombre = t(`requisito.${requisito.referencia}`);
      const estado = t(`acceso_estado.${requisito.estado}`);
      requisitos.append(nodo(d, 'li', t('detalle_requisito', { nombre, estado })));
    }
    desplegable.append(nodo(d, 'h4', t('requisitos_titulo')), requisitos, nodo(d, 'h4', t('fases_titulo')));
    const causas = nodo(d, 'ul'); s.causas.forEach(c => causas.append(nodo(d, 'li', t(`causa.${c}`))));
    desplegable.append(causas);
    const fases = nodo(d, 'ul');
    s.fases.forEach(f => {
      const item = nodo(d, 'li', t('detalle_fase', { fase: nombreFase(f.referencia), puntos: puntos(f.puntos_micropuntos), estado: t(`fase_estado.${f.estado}`) }));
      item.append(nodo(d, 'p', t(`origen.${f.origen}`)));
      if (f.reglas.length) {
        const reglas = nodo(d, 'ul');
        f.reglas.forEach((regla, indice) => reglas.append(nodo(d, 'li', t('detalle_regla', { numero: textos.numero(indice + 1), puntos: puntos(regla.puntos_micropuntos) }))));
        item.append(reglas);
      }
      fases.append(item);
    });
    desplegable.append(fases); detalle.append(desplegable);
    fila.append(nombre, orden, total, propuesta, detalle); lista.cuerpo.append(fila);
  }
  raiz.append(lista.contenedor);
}
