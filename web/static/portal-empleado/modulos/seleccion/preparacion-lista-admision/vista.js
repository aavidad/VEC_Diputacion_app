import { ESQUEMA_DEFINITIVA } from './modelo.js?v=20261005-s4-lista-v2';
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
  const t = nodo(d, 'table', undefined, 'tabla-datos tabla-apilable lista-admision-tabla'); t.append(nodo(d, 'caption', etiqueta));
  const head = nodo(d, 'thead'); const fila = nodo(d, 'tr');
  for (const c of columnas) { const th = nodo(d, 'th', c); th.scope = 'col'; fila.append(th); }
  head.append(fila); const body = nodo(d, 'tbody'); t.append(head, body); region.append(t);
  return { region, body };
}
// Iconos decorativos: marcado SVG, no textos visibles.
const ICONOS = new Map([
  ['solicitudes', '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 3h9l4 4v14H6zM14 3v5h5"/></svg>'],
  ['admitidas', '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m5 12 4 4 10-10"/></svg>'],
  ['excluidas', '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 6l12 12M18 6 6 18"/></svg>'],
  ['tras_escrito', '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 3h9l4 4v14H6zM14 3v5h5"/><path d="m9 14 2 2 4-4"/></svg>'],
  ['subsanables', '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 8v5l3 2"/><circle cx="12" cy="12" r="9"/></svg>'],
]);
// Número de solicitud legible: la referencia sin su prefijo técnico («preparacion:»).
const numeroSolicitud = ref => ref.slice(ref.indexOf(':') + 1);
const esDefinitiva = dto => dto.esquema === ESQUEMA_DEFINITIVA;
function kpis(d, textos, dto) {
  const t = textos.traducir; const r = dto.resumen;
  const rejilla = nodo(d, 'section', undefined, 'rejilla-kpi cuatro'); rejilla.setAttribute('aria-label', t('resumen'));
  const cuarta = esDefinitiva(dto) ? ['tras_escrito', r.admitidas_tras_escrito, 'kpi--exito', 'tras_escrito'] : ['subsanables', r.excluidas_subsanables, 'kpi--advertencia', 'subsanables'];
  const valores = [['solicitudes', r.solicitudes, '', 'solicitudes'], ['admitidas', r.admitidas, 'kpi--exito', 'admitidas'],
    ['excluidas', r.excluidas, 'kpi--peligro', 'excluidas'], cuarta];
  for (const [clave, valor, clase, icono] of valores) {
    const tarjeta = nodo(d, 'article', undefined, `tarjeta-kpi ${clase}`.trim());
    const marca = nodo(d, 'span', undefined, 'icono-kpi'); marca.setAttribute('aria-hidden', 'true');
    // Plantillas fijas de este módulo; ningún dato del archivo entra en innerHTML.
    marca.innerHTML = ICONOS.get(icono);
    const texto = nodo(d, 'div'); texto.append(nodo(d, 'strong', textos.numero(valor), 'valor-kpi'), nodo(d, 'span', t(`kpi.${clave}`), 'etiqueta-kpi'));
    tarjeta.append(marca, texto); rejilla.append(tarjeta);
  }
  return rejilla;
}
// En la definitiva la subsanación ya terminó: no se marca el motivo no subsanable.
function listaMotivos(d, t, motivosDeLaFila, motivos, marcarNoSubsanables) {
  const ul = nodo(d, 'ul', undefined, 'lista-motivos');
  for (const m of motivosDeLaFila) {
    const texto = motivos.get(m.codigo) ?? t('motivo_sin_texto', { codigo: m.codigo });
    ul.append(nodo(d, 'li', m.subsanable || !marcarNoSubsanables ? texto : t('motivo_no_subsanable', { motivo: texto })));
  }
  return ul;
}
function celda(d, etiqueta, ...hijos) { const td = nodo(d, 'td'); td.dataset.etiqueta = etiqueta; td.append(...hijos); return td; }
function cabeceraFila(d, ref) { const th = nodo(d, 'th', numeroSolicitud(ref)); th.scope = 'row'; return th; }
/** Clave de texto de un pendiente: los de la definitiva tienen su propia sección. */
function clavePendiente(p, definitiva) {
  const final = p.slice(p.lastIndexOf('.') + 1);
  return definitiva ? `pendientes_definitiva.${final}` : `pendientes.${final}`;
}

/**
 * Pinta la lista provisional o la definitiva (borradores). `motivos` traduce
 * código → texto; sin texto se muestra el código.
 */
export function pintarLista({ raiz, dto, textos, motivos = new Map() }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const f = d.createDocumentFragment(); const definitiva = esDefinitiva(dto);
  const estado = nodo(d, 'p', undefined, 'lista-admision-estado');
  estado.append(nodo(d, 'span', t(definitiva ? 'tipo_definitiva' : 'tipo_provisional'), 'insignia'),
    nodo(d, 'span', t('estado_borrador'), 'insignia aviso'));
  if (dto.catalogo.paquete_ejemplo) estado.append(nodo(d, 'span', t(definitiva ? 'catalogo_ejemplo_definitiva' : 'catalogo_ejemplo'), 'insignia'));
  f.append(estado, kpis(d, textos, dto));

  if (!definitiva) {
    const plazo = panel(d, t('plazo'));
    plazo.cuerpo.append(datos(d, [
      [t('plazo_duracion'), textos.plural(`unidades.${dto.plazo_subsanacion.unidad}`, dto.plazo_subsanacion.cantidad)],
      [t('plazo_inicio'), t('plazo_inicio_valor')],
      [t('plazo_ultimo_dia'), t('plazo_ultimo_dia_valor')],
    ]));
    f.append(plazo.n);
  }

  const excluidas = panel(d, t('excluidas'), textos.plural('recuento', dto.excluidas.length));
  if (!dto.excluidas.length) excluidas.cuerpo.append(nodo(d, 'p', t('sin_excluidas')));
  else {
    const columnas = definitiva ? [t('solicitud'), t('motivos_persistentes'), t('resolucion')] : [t('solicitud'), t('motivos'), t('subsanacion')];
    const { region, body } = tabla(d, t('excluidas'), columnas);
    for (const e of dto.excluidas) {
      const tr = nodo(d, 'tr');
      let ultima;
      if (definitiva) {
        ultima = celda(d, columnas[2], nodo(d, 'span', t(e.resolucion === 'no_presentada' ? 'resoluciones.no_presentada' : `resoluciones.desestimada_${e.via}`)));
      } else {
        const marca = nodo(d, 'span', t(e.subsanable ? 'puede_subsanar' : 'no_puede_subsanar'), 'estado-subsanacion');
        marca.dataset.subsanable = String(e.subsanable); ultima = celda(d, columnas[2], marca);
      }
      tr.append(cabeceraFila(d, e.antecedente.preparacion_ref), celda(d, columnas[1], listaMotivos(d, t, e.motivos, motivos, !definitiva)), ultima);
      body.append(tr);
    }
    excluidas.cuerpo.append(region);
  }
  f.append(excluidas.n);

  const admitidas = panel(d, t('admitidas'), textos.plural('recuento', dto.admitidas.length));
  if (!dto.admitidas.length) admitidas.cuerpo.append(nodo(d, 'p', t('sin_admitidas')));
  else {
    const columnas = definitiva ? [t('solicitud'), t('origen')] : [t('solicitud')];
    const { region, body } = tabla(d, t('admitidas'), columnas);
    for (const a of dto.admitidas) {
      const tr = nodo(d, 'tr'); tr.append(cabeceraFila(d, a.antecedente.preparacion_ref));
      if (definitiva) tr.append(celda(d, columnas[1], nodo(d, 'span', t(`origenes.${a.origen}`))));
      body.append(tr);
    }
    admitidas.cuerpo.append(region);
  }
  f.append(admitidas.n);

  const falta = panel(d, t('que_falta')); const lista = nodo(d, 'ul', undefined, 'lista-comprobacion');
  for (const p of dto.pendientes) { const li = nodo(d, 'li', undefined, 'pendiente'); li.append(nodo(d, 'p', t(clavePendiente(p, definitiva)))); lista.append(li); }
  falta.cuerpo.append(lista); f.append(falta.n);

  const tecnico = panel(d, t('detalle_archivo')); const detalle = nodo(d, 'details');
  const pares = [
    [t('lista_ref'), dto.lista_ref], [t('revision'), textos.numero(dto.revision)],
    [t('catalogo_ref'), dto.catalogo.referencia], [t('catalogo_version'), dto.catalogo.version],
    [t('bases_ref'), dto.bases.referencia], [t('bases_version'), dto.bases.version], [t('huella_bases'), dto.bases.huella_sha256],
  ];
  if (definitiva) pares.push([t('provisional_ref'), dto.provisional.lista_ref], [t('provisional_revision'), textos.numero(dto.provisional.revision)],
    [t('huella_provisional'), dto.provisional.huella_sha256]);
  detalle.append(nodo(d, 'summary', t('ver_detalle')), datos(d, pares), nodo(d, 'pre', JSON.stringify(dto, null, 2), 'lista-admision-original'));
  tecnico.cuerpo.append(detalle); f.append(tecnico.n);
  raiz.replaceChildren(f);
}
