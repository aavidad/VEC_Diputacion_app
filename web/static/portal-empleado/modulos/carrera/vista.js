function nodo(d, tag, texto, clase) { const n = d.createElement(tag); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n; }
function panel(d, titulo) { const p = nodo(d, 'section', undefined, 'panel'); const h = nodo(d, 'header', undefined, 'cabecera-panel'); h.append(nodo(d, 'h2', titulo)); const cuerpo = nodo(d, 'div', undefined, 'cuerpo-panel pila'); p.append(h, cuerpo); return { p, cuerpo }; }
function seguro(t, clave, respaldo) { try { return t(clave); } catch { return t(respaldo); } }
function boton(d, titulo, accion, clase = 'boton-secundario') { const b = nodo(d, 'button', titulo, clase); b.type = 'button'; b.addEventListener('click', accion); return b; }
export function pintarCarrera({ raiz, dto, textos, acciones }) {
  const d = raiz.ownerDocument; const t = textos.traducir;
  const nombrePersona = c => c.persona_nombre.trim() ? c.persona_nombre : t('detalle.persona_pendiente');
  const aviso = nodo(d, 'p', t('limite'), 'borrador-aviso');
  const ayuda = panel(d, t('ayuda.titulo')); ayuda.p.id = 'carrera-ayuda-contenido'; ayuda.p.hidden = true;
  const pasos = nodo(d, 'ol'); ['elegir', 'revisar', 'descargar'].forEach(k => pasos.append(nodo(d, 'li', t(`ayuda.${k}`)))); ayuda.cuerpo.append(pasos);
  const rejilla = nodo(d, 'div', undefined, 'carrera-rejilla'); const lista = panel(d, t('lista.titulo')); const detalle = panel(d, t('detalle.titulo')); detalle.p.id = 'carrera-detalle'; detalle.p.tabIndex = -1;
  const label = nodo(d, 'label', undefined, 'campo'); label.append(nodo(d, 'span', t('lista.filtro'))); const select = nodo(d, 'select'); select.id = 'carrera-via';
  ['todas', 'grado', 'progresion', 'promocion'].forEach(v => { const o = nodo(d, 'option', t(`vias.${v}`)); o.value = v; select.append(o); }); label.append(select);
  const cuenta = nodo(d, 'p'); cuenta.setAttribute('role', 'status'); const ul = nodo(d, 'ul', undefined, 'carrera-lista'); lista.cuerpo.append(label, cuenta, ul);
  let seleccion = null;
  function mostrarCaso(c, moverFoco = false) {
    seleccion = c.referencia;
    ul.querySelectorAll('button').forEach(b => b.setAttribute('aria-pressed', String(b.dataset.referencia === seleccion)));
    detalle.cuerpo.replaceChildren(nodo(d, 'h3', nombrePersona(c)), nodo(d, 'p', t(`vias.${c.via}`)), nodo(d, 'span', t('estados.global'), 'estado-chip'));
    const dl = nodo(d, 'dl', undefined, 'resumen-expediente');
    const valor = v => v === null || v === undefined || typeof v === 'string' && !v.trim() ? t('detalle.sin_dato') : typeof v === 'number' ? textos.numero(v) : String(v);
    [['regimen', seguro(t, `regimen.${c.regimen}`, 'detalle.regimen_pendiente')], ['nivel_puesto', valor(c.nivel_puesto)], ['grado_personal', valor(c.grado_personal)]].forEach(([k, v]) => dl.append(nodo(d, 'dt', t(`detalle.${k}`)), nodo(d, 'dd', v)));
    detalle.cuerpo.append(dl, nodo(d, 'p', t('detalle.separacion')), nodo(d, 'h3', t('detalle.comprobaciones')));
    const checks = nodo(d, 'ul', undefined, 'carrera-comprobaciones');
    c.comprobaciones.forEach(x => { const li = nodo(d, 'li'); const head = nodo(d, 'div', undefined, 'carrera-comprobacion-titulo'); head.append(nodo(d, 'strong', seguro(t, x.clave, 'detalle.comprobacion_pendiente')), nodo(d, 'span', t(`estados.${x.estado}`), `estado-chip${x.estado === 'disponible' ? ' info' : x.estado === 'incompatible' ? ' peligro' : ''}`)); li.append(head, nodo(d, 'p', seguro(t, x.motivo_clave, 'detalle.motivo_pendiente'))); checks.append(li); });
    if (!c.comprobaciones.length) checks.append(nodo(d, 'li', t('detalle.sin_comprobaciones'))); detalle.cuerpo.append(checks);
    detalle.cuerpo.append(nodo(d, 'h3', t('detalle.siguiente')), nodo(d, 'p', t('detalle.responsable')));
    const deps = nodo(d, 'ul'); [...new Set(c.pendientes)].forEach(k => deps.append(nodo(d, 'li', seguro(t, k, 'detalle.dependencia_pendiente')))); if (!deps.children.length) deps.append(nodo(d, 'li', t('detalle.dependencia_pendiente'))); detalle.cuerpo.append(deps);
    const fuentes = nodo(d, 'details'); fuentes.append(nodo(d, 'summary', t('detalle.fuentes'))); const refs = nodo(d, 'dl', undefined, 'resumen-expediente');
    refs.append(nodo(d, 'dt', t('detalle.version_ejercicio')), nodo(d, 'dd', valor(dto.version)), nodo(d, 'dt', t('detalle.fuente_ejercicio')), nodo(d, 'dd', valor(dto.fuente.referencia)), nodo(d, 'dt', t('detalle.version_fuente')), nodo(d, 'dd', valor(dto.fuente.version)), nodo(d, 'dt', t('detalle.referencia')), nodo(d, 'dd', c.referencia));
    c.fuentes.forEach(f => refs.append(nodo(d, 'dt', valor(f.referencia)), nodo(d, 'dd', t('detalle.version', { version: valor(f.version) })))); fuentes.append(refs); detalle.cuerpo.append(fuentes);
    const bloqueo = nodo(d, 'p', t('detalle.bloqueo')); bloqueo.id = 'carrera-bloqueo'; const oficial = boton(d, t('acciones.reconocer'), () => {}); oficial.disabled = true; oficial.setAttribute('aria-describedby', bloqueo.id); detalle.cuerpo.append(oficial, bloqueo);
    if (moverFoco) { detalle.p.focus({ preventScroll: true }); detalle.p.scrollIntoView({ block: 'nearest' }); }
  }
  function filtrar() {
    const casos = dto.casos.filter(c => select.value === 'todas' || c.via === select.value); cuenta.textContent = t('lista.cuenta', { cuenta: textos.numero(casos.length) }); ul.replaceChildren();
    casos.forEach(c => { const li = nodo(d, 'li'); const b = boton(d, '', () => mostrarCaso(c, true), 'boton-secundario carrera-caso'); b.dataset.referencia = c.referencia; b.setAttribute('aria-controls', detalle.p.id); b.setAttribute('aria-pressed', String(seleccion === c.referencia)); b.append(nodo(d, 'strong', nombrePersona(c)), nodo(d, 'span', t(`vias.${c.via}`)), nodo(d, 'span', t('estados.global'))); li.append(b); ul.append(li); });
    if (!casos.length) { ul.append(nodo(d, 'li', t(dto.casos.length ? 'lista.sin_filtro' : 'lista.vacia'))); detalle.cuerpo.replaceChildren(nodo(d, 'p', t('detalle.elegir'))); seleccion = null; }
    else mostrarCaso(casos.find(c => c.referencia === seleccion) ?? casos[0]);
  }
  select.addEventListener('change', filtrar); rejilla.append(lista.p, detalle.p);
  const pie = panel(d, t('exportar.titulo')); pie.cuerpo.append(nodo(d, 'p', t('exportar.limite')), boton(d, t('acciones.descargar'), acciones.descargar, 'boton-primario'));
  const estado = nodo(d, 'p'); estado.id = 'carrera-descarga-estado'; estado.setAttribute('role', 'status'); pie.cuerpo.append(estado);
  const pila = nodo(d, 'div', undefined, 'pila'); pila.append(aviso, ayuda.p, rejilla, pie.p); raiz.replaceChildren(pila); filtrar();
}
export function pintarEstado({ raiz, textos, clave, reintentar }) {
  const d = raiz.ownerDocument; const p = panel(d, textos.traducir('titulo')); const estado = nodo(d, 'p', textos.traducir(clave)); estado.setAttribute('role', reintentar ? 'alert' : 'status'); p.cuerpo.append(estado); if (reintentar) p.cuerpo.append(boton(d, textos.traducir('acciones.reintentar'), reintentar)); raiz.replaceChildren(p.p);
}
