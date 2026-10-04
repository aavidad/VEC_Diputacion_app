import { nodo, panel, tabla } from '../dom.js?v=20261004-codexa-s6-notas-v1';

export function textosRevision(textos) {
  return Object.freeze({ ...textos, traducir: (clave, valores) => textos.traducir(`revision_notas.${clave}`, valores) });
}

export function pintarSalida({ raiz, dto, textos }) {
  const d = raiz.ownerDocument, t = textos.traducir;
  const puntos = valor => valor === null ? t('pendiente') : textos.numero(valor / 1000000, { maximumFractionDigits: 6 });
  const orden = valor => valor === null ? t('sin_orden') : textos.numero(valor);
  const cambios = panel(d, t('cambios'));
  if (!dto.cambios.length) cambios.cuerpo.append(nodo(d, 'p', t('sin_cambios')));
  else {
    const tab = tabla(d, ['solicitud', 'fase', 'anterior', 'propuesta'].map(clave => t(clave)), t('cambios'));
    tab.contenedor.children[0].className += ' tabla-apilable';
    for (const cambio of dto.cambios) {
      const solicitud = dto.antecedente.solicitudes.find(s => s.referencia === cambio.solicitud_ref);
      const indice = dto.configuracion.fases.findIndex(f => f.referencia === cambio.fase_ref);
      const fila = nodo(d, 'tr'), th = nodo(d, 'th', solicitud.nombre); th.scope = 'row';
      fila.append(th, ...[t('nota', { numero: textos.numero(indice + 1) }), puntos(cambio.anterior_micropuntos), puntos(cambio.propuesta_micropuntos)].map((valor, i) => {
        const celda = nodo(d, 'td', valor); celda.dataset.etiqueta = t(['fase', 'anterior', 'propuesta'][i]); return celda;
      }));
      tab.cuerpo.append(fila);
    }
    cambios.cuerpo.append(tab.contenedor);
  }
  const efecto = panel(d, t('comparacion'));
  const tab = tabla(d, ['solicitud', 'estado_anterior', 'estado_propuesta', 'total_anterior', 'total_propuesta', 'orden_anterior', 'orden_propuesta'].map(clave => t(clave)), t('comparacion'));
  tab.contenedor.children[0].className += ' tabla-apilable';
  for (const anterior of dto.antecedente.solicitudes) {
    const propuesta = dto.propuesta.solicitudes.find(s => s.referencia === anterior.referencia);
    const fila = nodo(d, 'tr'), th = nodo(d, 'th', anterior.nombre); th.scope = 'row';
    fila.append(th, ...[t(`estado.${anterior.estado}`), t(`estado.${propuesta.estado}`), puntos(anterior.total_micropuntos), puntos(propuesta.total_micropuntos), orden(anterior.orden), orden(propuesta.orden)].map((v, i) => {
      const celda = nodo(d, 'td', v); celda.dataset.etiqueta = t(['estado_anterior', 'estado_propuesta', 'total_anterior', 'total_propuesta', 'orden_anterior', 'orden_propuesta'][i]); return celda;
    }));
    tab.cuerpo.append(fila);
  }
  efecto.cuerpo.append(tab.contenedor);
  const pendientes = panel(d, t('pendientes_titulo')), lista = nodo(d, 'ul', undefined, 'lista-comprobacion');
  dto.pendientes.forEach(p => lista.append(nodo(d, 'li', t(`actuacion.${p}`), 'pendiente'))); pendientes.cuerpo.append(lista);
  const referencias = panel(d, t('detalle_archivo')), detalle = nodo(d, 'details'), datos = nodo(d, 'dl', undefined, 'datos-clave');
  for (const [nombre, valor] of [['ejemplo_ref', dto.ejemplo_ref], ['configuracion_version', textos.numero(dto.configuracion.version)], ['huella', dto.huella_antecedente_sha256]]) {
    const par = nodo(d, 'div'); par.append(nodo(d, 'dt', t(nombre)), nodo(d, 'dd', valor)); datos.append(par);
  }
  detalle.append(nodo(d, 'summary', t('ver_detalle')), nodo(d, 'p', t('referencias_limite'), 'texto-secundario'), datos);
  referencias.cuerpo.append(detalle); raiz.replaceChildren(cambios.elemento, efecto.elemento, pendientes.elemento, referencias.elemento);
}
