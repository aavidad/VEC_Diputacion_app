function nodo(d, etiqueta, texto, clase) {
  const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n;
}
function datos(d, pares) {
  const dl = nodo(d, 'dl', undefined, 'datos-clave');
  for (const [titulo, valor] of pares) { const div = nodo(d, 'div'); div.append(nodo(d, 'dt', titulo), nodo(d, 'dd', valor)); dl.append(div); }
  return dl;
}
function panel(d, titulo, recuento) {
  const n = nodo(d, 'section', undefined, 'panel'); const cabecera = nodo(d, 'header', undefined, 'cabecera-panel');
  const h = nodo(d, 'h2', titulo); cabecera.append(h);
  if (recuento !== undefined) cabecera.append(nodo(d, 'span', recuento, 'texto-secundario'));
  const cuerpo = nodo(d, 'div', undefined, 'cuerpo-panel pila'); n.append(cabecera, cuerpo); return { n, cuerpo };
}
function tabla(d, etiqueta, columnas) {
  const region = nodo(d, 'div', undefined, 'tabla-contenedor'); region.tabIndex = 0;
  region.setAttribute('role', 'region'); region.setAttribute('aria-label', etiqueta);
  const t = nodo(d, 'table', undefined, 'tabla-datos tabla-apilable lista-admision-tabla');
  const head = nodo(d, 'thead'); const fila = nodo(d, 'tr');
  for (const c of columnas) { const th = nodo(d, 'th', c); th.scope = 'col'; fila.append(th); }
  head.append(fila); const body = nodo(d, 'tbody'); t.append(head, body); region.append(t);
  return { region, body };
}
const ICONOS = {
  solicitudes: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 3h9l4 4v14H6zM14 3v5h5"/></svg>',
  admitidas: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m5 12 4 4 10-10"/></svg>',
  excluidas: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 6l12 12M18 6 6 18"/></svg>',
  subsanables: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 8v5l3 2"/><circle cx="12" cy="12" r="9"/></svg>',
};
function kpis(d, textos, dto) {
  const t = textos.traducir;
  const rejilla = nodo(d, 'section', undefined, 'rejilla-kpi cuatro'); rejilla.setAttribute('aria-label', t('resumen'));
  const valores = [['solicitudes', dto.resumen.solicitudes, ''], ['admitidas', dto.resumen.admitidas, 'kpi--exito'],
    ['excluidas', dto.resumen.excluidas, 'kpi--peligro'], ['subsanables', dto.resumen.excluidas_subsanables, 'kpi--advertencia']];
  for (const [clave, valor, clase] of valores) {
    const tarjeta = nodo(d, 'article', undefined, `tarjeta-kpi ${clase}`.trim());
    const icono = nodo(d, 'span', undefined, 'icono-kpi'); icono.setAttribute('aria-hidden', 'true');
    // Plantillas fijas de este módulo; ningún dato del archivo entra en innerHTML.
    icono.innerHTML = ICONOS[clave];
    const texto = nodo(d, 'div'); texto.append(nodo(d, 'strong', textos.numero(valor), 'valor-kpi'), nodo(d, 'span', t(`kpi.${clave}`), 'etiqueta-kpi'));
    tarjeta.append(icono, texto); rejilla.append(tarjeta);
  }
  return rejilla;
}

/** Pinta la lista provisional. `motivos` traduce código → texto; sin texto se muestra el código. */
export function pintarLista({ raiz, dto, textos, motivos = new Map() }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const f = d.createDocumentFragment();
  const estado = nodo(d, 'p', undefined, 'lista-admision-estado');
  estado.append(nodo(d, 'span', t('estado_borrador'), 'insignia aviso'));
  if (dto.catalogo.paquete_ejemplo) estado.append(nodo(d, 'span', t('catalogo_ejemplo'), 'insignia'));
  f.append(estado, kpis(d, textos, dto));

  const plazo = panel(d, t('plazo'));
  plazo.cuerpo.append(datos(d, [
    [t('plazo_duracion'), textos.plural(`unidades.${dto.plazo_subsanacion.unidad}`, dto.plazo_subsanacion.cantidad)],
    [t('plazo_inicio'), t('plazo_inicio_valor')],
    [t('plazo_ultimo_dia'), t('plazo_ultimo_dia_valor')],
  ]));
  f.append(plazo.n);

  const excluidas = panel(d, t('excluidas'), textos.plural('recuento', dto.excluidas.length));
  if (!dto.excluidas.length) excluidas.cuerpo.append(nodo(d, 'p', t('sin_excluidas')));
  else {
    const { region, body } = tabla(d, t('excluidas'), [t('solicitud'), t('motivos'), t('subsanacion')]);
    for (const e of dto.excluidas) {
      const tr = nodo(d, 'tr'); const ref = nodo(d, 'th', e.antecedente.preparacion_ref); ref.scope = 'row';
      const tdMotivos = nodo(d, 'td'); tdMotivos.dataset.etiqueta = t('motivos'); const ul = nodo(d, 'ul', undefined, 'lista-motivos');
      for (const m of e.motivos) ul.append(nodo(d, 'li', motivos.get(m.codigo) ?? t('motivo_sin_texto', { codigo: m.codigo })));
      tdMotivos.append(ul);
      const tdSub = nodo(d, 'td'); tdSub.dataset.etiqueta = t('subsanacion');
      const marca = nodo(d, 'span', t(e.subsanable ? 'puede_subsanar' : 'no_puede_subsanar'), 'estado-subsanacion');
      marca.dataset.subsanable = String(e.subsanable); tdSub.append(marca);
      tr.append(ref, tdMotivos, tdSub); body.append(tr);
    }
    excluidas.cuerpo.append(region);
  }
  f.append(excluidas.n);

  const admitidas = panel(d, t('admitidas'), textos.plural('recuento', dto.admitidas.length));
  if (!dto.admitidas.length) admitidas.cuerpo.append(nodo(d, 'p', t('sin_admitidas')));
  else {
    const { region, body } = tabla(d, t('admitidas'), [t('solicitud')]);
    for (const a of dto.admitidas) { const tr = nodo(d, 'tr'); const th = nodo(d, 'th', a.antecedente.preparacion_ref); th.scope = 'row'; tr.append(th); body.append(tr); }
    admitidas.cuerpo.append(region);
  }
  f.append(admitidas.n);

  const falta = panel(d, t('que_falta')); const lista = nodo(d, 'ul', undefined, 'lista-comprobacion');
  for (const p of dto.pendientes) { const li = nodo(d, 'li', undefined, 'pendiente'); li.append(nodo(d, 'p', t(`pendientes.${p.slice(p.lastIndexOf('.') + 1)}`))); lista.append(li); }
  falta.cuerpo.append(lista); f.append(falta.n);

  const tecnico = panel(d, t('detalle_archivo')); const detalle = nodo(d, 'details');
  detalle.append(nodo(d, 'summary', t('ver_detalle')), datos(d, [
    [t('lista_ref'), dto.lista_ref], [t('revision'), textos.numero(dto.revision)],
    [t('catalogo_ref'), dto.catalogo.referencia], [t('catalogo_version'), dto.catalogo.version],
    [t('bases_ref'), dto.bases.referencia], [t('bases_version'), dto.bases.version], [t('huella_bases'), dto.bases.huella_sha256],
  ]), nodo(d, 'pre', JSON.stringify(dto, null, 2), 'lista-admision-original'));
  tecnico.cuerpo.append(detalle); f.append(tecnico.n);
  raiz.replaceChildren(f);
}
