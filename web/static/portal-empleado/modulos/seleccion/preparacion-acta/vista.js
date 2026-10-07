import { nodo, panel, tabla } from '../dom.js?v=20261001-codexa-selectivos-s0-n2-v5';

function datos(d, pares) {
  const dl = nodo(d, 'dl', undefined, 'datos-clave');
  for (const [titulo, valor] of pares) { const div = nodo(d, 'div'); div.append(nodo(d, 'dt', titulo), nodo(d, 'dd', valor)); dl.append(div); }
  return dl;
}
export function pintarSalida({ raiz, dto, textos }) {
  const d = raiz.ownerDocument, t = textos.traducir, m = dto.preparacion.material_propuesto;
  const fragmento = d.createDocumentFragment();
  const contexto = panel(d, t('contexto_material'));
  contexto.cuerpo.append(nodo(d, 'p', t('borrador_local'), 'insignia aviso'), nodo(d, 'p', t('sesion_limite')),
    datos(d, [[t('fecha'), m.fecha_propuesta ? textos.fecha(m.fecha_propuesta, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Europe/Madrid' }) : t('sin_fecha')],
      [t('fase'), t('fase_pendiente')], [t('antecedente'), t('antecedente_pendiente')]]));
  fragmento.append(contexto.elemento);
  const propuestas = nodo(d, 'div', undefined, 'rejilla-dos');
  for (const [nombre, lista] of [['agenda', m.orden_dia_propuesto], ['acuerdos', m.acuerdos_propuestos]]) {
    const p = panel(d, t(nombre));
    p.cuerpo.append(nodo(d, 'p', t(`${nombre}_limite`), 'texto-secundario'));
    if (!lista.length) p.cuerpo.append(nodo(d, 'p', t(`${nombre}_vacio`)));
    else {
      const tab = tabla(d, ['punto', 'texto_propuesto'].map(c => t(c)), t(nombre)); tab.contenedor.querySelector('table').classList.add('tabla-apilable');
      for (const [i, item] of lista.entries()) {
        const numero = nombre === 'agenda' ? i + 1 : m.orden_dia_propuesto.findIndex(punto => punto.punto_ref === item.punto_ref) + 1;
        const fila = nodo(d, 'tr'), th = nodo(d, 'th', textos.numero(numero)); th.scope = 'row';
        const td = nodo(d, 'td', item.texto_propuesto.trim() ? item.texto_propuesto : t('sin_texto')); td.dataset.etiqueta = t('texto_propuesto');
        fila.append(th, td); tab.cuerpo.append(fila);
      }
      p.cuerpo.append(tab.contenedor);
    }
    propuestas.append(p.elemento);
  }
  fragmento.append(propuestas);
  const pendientes = panel(d, t('pendientes'));
  const lista = nodo(d, 'ul', undefined, 'lista-comprobacion');
  for (const p of dto.preparacion.pendientes) {
    const li = nodo(d, 'li', undefined, 'pendiente'); li.append(nodo(d, 'strong', t(`campos.${p.campo}`)), nodo(d, 'p', t(`motivos.${p.codigo}`))); lista.append(li);
  }
  pendientes.cuerpo.append(lista); fragmento.append(pendientes.elemento);
  const archivo = panel(d, t('detalle_archivo')), detalle = nodo(d, 'details'); detalle.append(nodo(d, 'summary', t('ver_detalle')),
    nodo(d, 'p', t('referencias_limite'), 'texto-secundario'), datos(d, [[t('identidad_material'), m.identidad_material], [t('version_material'), textos.numero(m.version_material)],
      [t('sesion_ref'), m.sesion_ref || t('sin_dato')], [t('fase_ref'), m.fase_propuesta],
      [t('antecedente_ref'), m.antecedente_tribunal.identidad_material], [t('antecedente_version'), textos.numero(m.antecedente_tribunal.version_material)],
      [t('antecedente_huella'), m.antecedente_tribunal.huella_aportada_sha256]]), nodo(d, 'pre', JSON.stringify(dto, null, 2), 'acta-original'));
  archivo.cuerpo.append(detalle); fragmento.append(archivo.elemento); raiz.replaceChildren(fragmento);
}
