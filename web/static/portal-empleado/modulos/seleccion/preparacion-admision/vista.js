function nodo(d, etiqueta, texto, clase) {
  const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n;
}
function datos(d, pares) {
  const dl = nodo(d, 'dl', undefined, 'datos-clave');
  for (const [titulo, valor] of pares) { const div = nodo(d, 'div'); div.append(nodo(d, 'dt', titulo), nodo(d, 'dd', valor)); dl.append(div); }
  return dl;
}
function panel(d, titulo) {
  const n = nodo(d, 'section', undefined, 'panel'); const cabecera = nodo(d, 'header', undefined, 'cabecera-panel');
  cabecera.append(nodo(d, 'h2', titulo)); const cuerpo = nodo(d, 'div', undefined, 'cuerpo-panel pila'); n.append(cabecera, cuerpo); return { n, cuerpo, cabecera };
}
const claveFinal = clave => clave.slice(clave.lastIndexOf('.') + 1);

export function pintarSalida({ raiz, dto, textos }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const fragmento = d.createDocumentFragment();
  const contexto = panel(d, t('contexto_material'));
  contexto.cabecera.append(nodo(d, 'span', t('pendiente'), 'insignia aviso'));
  contexto.cuerpo.append(nodo(d, 'p', t('universo_pendiente')), datos(d, [
    [t('version_bases'), dto.bases.version],
    [t('solicitud'), t(dto.solicitud_contexto ? 'solicitud_sin_presentar' : 'sin_solicitud')],
    [t('identidad_registro'), t('identidad_registro_pendiente')],
  ]));
  fragmento.append(contexto.n);

  const requisitos = panel(d, t('requisitos'));
  requisitos.cabecera.append(nodo(d, 'span', textos.plural('recuento', dto.requisitos.length), 'texto-secundario'));
  if (!dto.requisitos.length) requisitos.cuerpo.append(nodo(d, 'p', t('sin_requisitos')));
  else {
    const region = nodo(d, 'div', undefined, 'tabla-contenedor'); region.tabIndex = 0;
    region.setAttribute('role', 'region'); region.setAttribute('aria-label', t('requisitos'));
    const tabla = nodo(d, 'table', undefined, 'tabla-datos tabla-apilable admision-tabla');
    const head = nodo(d, 'thead'); const fila = nodo(d, 'tr'); const columnas = ['requisito', 'motivo', 'accion'];
    for (const c of columnas) { const th = nodo(d, 'th', t(c)); th.scope = 'col'; fila.append(th); } head.append(fila);
    const body = nodo(d, 'tbody');
    dto.requisitos.forEach((r, i) => {
      const tr = nodo(d, 'tr'); const titulo = nodo(d, 'th'); titulo.scope = 'row';
      titulo.append(nodo(d, 'strong', r.requisito.titulo_propuesto || t('requisito_numero', { numero: textos.numero(i + 1) })),
        nodo(d, 'p', t('pendiente'), 'insignia aviso'));
      const motivo = nodo(d, 'td'); motivo.dataset.etiqueta = t('motivo');
      const causas = nodo(d, 'ul'); for (const c of r.causas) causas.append(nodo(d, 'li', t(`causas.${claveFinal(c)}`))); motivo.append(causas);
      const accion = nodo(d, 'td'); accion.dataset.etiqueta = t('accion');
      accion.append(nodo(d, 'p', t(`acciones.${claveFinal(r.accion_propuesta)}`)),
        nodo(d, 'p', t('fecha_propuesta', { fecha: textos.fecha(`${r.requisito.hito_fecha}T12:00:00Z`, { dateStyle: 'medium', timeZone: 'UTC' }) }), 'texto-secundario'));
      const tecnico = nodo(d, 'details'); tecnico.append(nodo(d, 'summary', t('ver_soportes')),
        nodo(d, 'p', t('soportes_limite'), 'texto-secundario'), datos(d, [
          [t('referencia_requisito'), r.requisito.referencia], [t('version_requisito'), r.requisito.version],
          [t('representacion'), t(`representaciones.${r.requisito.representacion}`)], [t('hito'), r.requisito.hito_ref],
          [t('regla'), r.requisito.regla_ref || t('sin_dato')], [t('version_regla'), r.requisito.regla_version || t('sin_dato')],
        ]));
      if (!r.soportes.length) tecnico.append(nodo(d, 'p', t('sin_soportes')));
      for (const s of r.soportes) tecnico.append(datos(d, [
        [t('hecho'), s.referencia], [t('version_hecho'), textos.numero(s.version)],
        [t('fuente'), s.fuente_ref], [t('version_fuente'), s.fuente_version],
        [t('estado_aportado'), t(`estados_soporte.${s.estado_aportado}`)], [t('evidencias'), textos.numero(s.evidencias_aportadas)],
        [t('desde'), textos.fecha(`${s.desde}T12:00:00Z`, { dateStyle: 'medium', timeZone: 'UTC' })],
        [t('hasta'), s.hasta ? textos.fecha(`${s.hasta}T12:00:00Z`, { dateStyle: 'medium', timeZone: 'UTC' }) : t('sin_dato')],
      ]));
      accion.append(tecnico); tr.append(titulo, motivo, accion); body.append(tr);
    });
    tabla.append(head, body); region.append(tabla); requisitos.cuerpo.append(region);
  }
  fragmento.append(requisitos.n);
  const siguientes = panel(d, t('siguientes'));
  const lista = nodo(d, 'ul', undefined, 'lista-comprobacion');
  for (const p of dto.pendientes) { const li = nodo(d, 'li', undefined, 'pendiente'); li.append(nodo(d, 'p', t(`pendientes.${claveFinal(p)}`))); lista.append(li); }
  siguientes.cuerpo.append(lista); fragmento.append(siguientes.n);

  const archivo = panel(d, t('detalle_archivo')); const detalle = nodo(d, 'details'); detalle.append(nodo(d, 'summary', t('ver_detalle')),
    nodo(d, 'p', t('referencias_limite'), 'texto-secundario'), datos(d, [
      [t('preparacion_ref'), dto.preparacion_ref], [t('revision'), textos.numero(dto.revision)],
      [t('bases_ref'), dto.bases.referencia], [t('huella_bases'), dto.bases.huella_sha256],
      [t('paquete_hechos'), dto.version_paquete_hechos || t('sin_dato')],
    ]));
  if (dto.solicitud_contexto) detalle.append(datos(d, [
    [t('solicitud_ref'), dto.solicitud_contexto.identificador_publico],
    [t('huella_solicitud'), dto.solicitud_contexto.huella_archivo_original_sha256],
  ]));
  detalle.append(nodo(d, 'pre', JSON.stringify(dto, null, 2), 'admision-original')); archivo.cuerpo.append(detalle); fragmento.append(archivo.n);
  raiz.replaceChildren(fragmento);
}
