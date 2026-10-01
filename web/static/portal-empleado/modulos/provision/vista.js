import { VISTAS, formatearPuntos } from './modelo.js?v=20261001-provision-ciclo-v5';
function nodo(d, tag, texto, clase) { const n = d.createElement(tag); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n; }
function boton(d, texto, accion, clave, deshabilitado = false) { const b = nodo(d, 'button', texto, 'boton-secundario'); b.type = 'button'; b.disabled = deshabilitado; b.dataset.foco = clave; b.addEventListener('click', accion); return b; }
function panel(d, titulo) { const n = nodo(d, 'section', undefined, 'panel'); const h = nodo(d, 'header', undefined, 'cabecera-panel'); h.append(nodo(d, 'h2', titulo)); const cuerpo = nodo(d, 'div', undefined, 'cuerpo-panel pila'); n.append(h, cuerpo); return { n, cuerpo }; }
function tabla(d, t, titulo, columnas, filas) { const envoltura = nodo(d, 'div', undefined, 'tabla-contenedor'); envoltura.tabIndex = 0; envoltura.setAttribute('role', 'region'); envoltura.setAttribute('aria-label', titulo); const table = nodo(d, 'table', undefined, 'tabla-datos'); table.append(nodo(d, 'caption', titulo)); const head = nodo(d, 'thead'); const tr = nodo(d, 'tr'); columnas.forEach(c => { const th = nodo(d, 'th', t(`campos.${c}`)); th.scope = 'col'; tr.append(th); }); head.append(tr); const body = nodo(d, 'tbody'); filas.forEach(f => { const row = nodo(d, 'tr'); f.forEach((v, i) => { const cell = nodo(d, i === 0 ? 'th' : 'td'); if (i === 0) cell.scope = 'row'; if (v?.nodeType) cell.append(v); else cell.textContent = String(v); row.append(cell); }); body.append(row); }); table.append(head, body); envoltura.append(table); return envoltura; }
function datos(d, t, pares) { const dl = nodo(d, 'dl', undefined, 'resumen-expediente'); pares.forEach(([clave, valor]) => { const div = nodo(d, 'div'); div.append(nodo(d, 'dt', t(`campos.${clave}`)), nodo(d, 'dd', valor)); dl.append(div); }); return dl; }
export function pintarProvision({ raiz, estado, textos, proyeccion = {}, acciones }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const proyectar = (objeto, campo, fallback) => objeto?.[`${campo}_clave`] ? t(objeto[`${campo}_clave`]) : objeto?.[campo] ?? fallback; const nombre = ref => proyectar(proyeccion.puestos?.[ref], 'denominacion', ref);
  const pila = nodo(d, 'div', undefined, 'pila'); pila.dataset.provision = '';
  const cabecera = nodo(d, 'div', undefined, 'cabeza-pagina'); cabecera.append(nodo(d, 'h2', t('titulo')), nodo(d, 'span', t('preparacion'), 'estado-presentacion'));
  const ayuda = nodo(d, 'details'); ayuda.append(nodo(d, 'summary', t('ayuda_signo')), nodo(d, 'p', t('ayuda_texto'))); ayuda.firstChild.setAttribute('aria-label', t('ayuda')); cabecera.append(ayuda); pila.append(cabecera);
  const nav = nodo(d, 'nav', undefined, 'acciones-vista'); nav.setAttribute('aria-label', t('titulo'));
  VISTAS.forEach(v => { const b = boton(d, t(`tabs.${v}`), () => acciones.vista(v), `vista-${v}`); b.setAttribute('aria-current', estado.vista === v ? 'page' : 'false'); if (estado.vista === v) b.className = 'boton-primario'; nav.append(b); }); pila.append(nav);
  const live = nodo(d, 'p', estado.mensaje || t('limite'), 'texto-secundario'); live.dataset.provisionAviso = ''; live.setAttribute('role', Object.keys(estado.invalidos ?? {}).length > 0 ? 'alert' : 'status'); live.setAttribute('aria-live', 'polite'); pila.append(live);
  const p = panel(d, t(`tabs.${estado.vista}`)); pila.append(p.n);
  const enlazarValidacion = (label, input, cambio, clave) => {
    const error = nodo(d, 'p', estado.errores?.[clave] ? t(estado.errores[clave]) : '', 'texto-secundario');
    error.id = `provision-error-${clave}`; error.hidden = !estado.errores?.[clave];
    input.setAttribute('aria-describedby', error.id);
    input.addEventListener('change', () => {
      const validacion = acciones.configuracion({ ...cambio, valor: input.value });
      input.setAttribute('aria-invalid', String(!validacion.valido));
      error.textContent = validacion.mensaje; error.hidden = validacion.valido;
    });
    label.append(input, error);
  };

  if (estado.vista === 'convocatoria') {
    p.cuerpo.append(datos(d, t, [['proceso', proyectar(proyeccion, 'denominacion', t('datos.concurso'))], ['version', estado.proceso.configuracion.version], ['estado', t('estados.borrador')]]));
    p.cuerpo.append(nodo(d, 'p', t('estados.sin_bases')), nodo(d, 'p', t('estados.sin_plazo')));
    const siguiente = nodo(d, 'section', undefined, 'siguiente-paso'); siguiente.append(nodo(d, 'h3', t('siguiente')), nodo(d, 'p', t('siguiente_texto')), boton(d, t('acciones.ver_puestos'), () => acciones.vista('puestos'), 'ver-puestos')); p.cuerpo.append(siguiente);
    const form = nodo(d, 'form', undefined, 'pila');
    const campos = nodo(d, 'div', undefined, 'rejilla-dos');
    ['ventana_desde', 'fecha_corte'].forEach(campo => { const label = nodo(d, 'label', undefined, 'campo'); label.append(nodo(d, 'span', t(`configuracion.${campo}`))); const input = nodo(d, 'input'); input.type = 'date'; input.value = estado.invalidos?.[campo] ?? estado.proceso.configuracion[campo]; input.setAttribute('aria-invalid', String(Object.hasOwn(estado.invalidos ?? {}, campo))); input.dataset.foco = campo; input.required = true; enlazarValidacion(label, input, { campo }, campo); campos.append(label); });
    form.addEventListener('submit', e => e.preventDefault()); form.append(campos); p.cuerpo.append(form);
    const reglas = estado.proceso.configuracion.reglas ?? [];
    const filasReglas = reglas.map((r, i) => {
      const controles = ['coeficiente', 'maximo'].map(campo => { const label = nodo(d, 'label', undefined, 'campo'); label.append(nodo(d, 'span', t(`configuracion.${campo}`))); const input = nodo(d, 'input'); input.type = 'text'; input.inputMode = 'decimal'; input.value = estado.invalidos?.[`${campo}-${i}`] ?? formatearPuntos(r[campo], textos.localizacion).replaceAll(new Intl.NumberFormat(textos.localizacion).formatToParts(1000).find(p => p.type === 'group')?.value ?? '\uFFFF', ''); input.dataset.foco = `${campo}-${i}`; input.setAttribute('aria-invalid', String(Object.hasOwn(estado.invalidos ?? {}, `${campo}-${i}`))); if (campo === 'coeficiente' && r.tramos?.length) { input.disabled = true; input.value = t('configuracion.por_tramos'); } input.setAttribute('aria-label', t('configuracion.etiqueta', { campo: t(`configuracion.${campo}`), familia: t(`familias.${r.familia}`) })); enlazarValidacion(label, input, { campo, regla: i }, `${campo}-${i}`); return label; });
      return [t(`familias.${r.familia}`), ...controles];
    });
    p.cuerpo.append(tabla(d, t, t('configuracion.titulo'), ['regla', 'coeficiente', 'maximo'], filasReglas));
  }
  if (estado.vista === 'puestos') {
    const filas = estado.proceso.puestos.map(puesto => {
      const b = boton(d, t('acciones.seleccionar'), () => acciones.preferencia(puesto.referencia, 'seleccionar'), `seleccionar-${puesto.referencia}`, estado.preferencias.includes(puesto.referencia));
      b.setAttribute('aria-label', t('acciones.mover', { accion: t('acciones.seleccionar'), puesto: nombre(puesto.referencia) }));
      const ref = nodo(d, 'div'); ref.append(nodo(d, 'span', puesto.rpt_ref), nodo(d, 'small', t('version_rpt', { version: puesto.rpt_version })));
      return [nombre(puesto.referencia), proyectar(proyeccion.puestos?.[puesto.referencia], 'centro', t('estados.sin_dato')), textos.numero(puesto.nivel), ref, b];
    });
    if (!filas.length) p.cuerpo.append(nodo(d, 'p', t('estados.sin_puestos')));
    else p.cuerpo.append(tabla(d, t, t('tabs.puestos'), ['puesto', 'centro', 'nivel', 'rpt', 'acciones'], filas));
  }
  if (estado.vista === 'personal') {
    p.cuerpo.append(nodo(d, 'h3', t('preferencias_locales')));
    if (!estado.preferencias.length) p.cuerpo.append(nodo(d, 'p', t('estados.sin_preferencias')), boton(d, t('acciones.ver_puestos'), () => acciones.vista('puestos'), 'ver-puestos'));
    else {
      const filas = estado.preferencias.map((ref, i) => { const controles = nodo(d, 'div', undefined, 'acciones-vista'); ['arriba', 'abajo', 'quitar'].forEach(a => { const b = boton(d, t(`acciones.${a}`), () => acciones.preferencia(ref, a), `${a}-${ref}`, a === 'arriba' && i === 0 || a === 'abajo' && i === estado.preferencias.length - 1); b.setAttribute('aria-label', t('acciones.mover', { accion: t(`acciones.${a}`), puesto: nombre(ref) })); controles.append(b); }); return [textos.numero(i + 1), nombre(ref), controles]; });
      p.cuerpo.append(tabla(d, t, t('preferencias_locales'), ['preferencia', 'puesto', 'acciones'], filas));
    }
    const b = boton(d, t('acciones.presentar'), () => {}, 'presentar', true); b.setAttribute('aria-describedby', 'provision-bloqueo-presentar'); const aviso = nodo(d, 'p', t('bloqueos.presentar')); aviso.id = 'provision-bloqueo-presentar'; p.cuerpo.append(b, aviso);
  }
  if (estado.vista === 'valoracion') {
    const simular = boton(d, t('acciones.simular'), acciones.simular, 'simular', !acciones.puedeSimular || !estado.preferencias.length || estado.estado === 'cargando' || Object.keys(estado.invalidos ?? {}).length > 0); simular.className = 'boton-primario'; p.cuerpo.append(simular);
    if (Object.keys(estado.invalidos ?? {}).length > 0 || estado.estado === 'validacion') p.cuerpo.append(nodo(d, 'p', t('configuracion.errores_pendientes')), boton(d, t('configuracion.corregir'), () => acciones.vista('convocatoria', Object.keys(estado.invalidos ?? {})[0]), 'corregir-configuracion'));
    if (!acciones.puedeSimular) p.cuerpo.append(nodo(d, 'p', t('estados.sin_cliente')));
    if (estado.estado === 'cargando') p.cuerpo.append(nodo(d, 'p', t('estados.cargando')));
    if (['error', 'denegado', 'conflicto', 'validacion'].includes(estado.estado)) { const error = nodo(d, 'p', t(`estados.${estado.estado}`)); error.setAttribute('role', 'alert'); p.cuerpo.append(error); }
    if (!estado.resultado && estado.estado !== 'cargando') p.cuerpo.append(nodo(d, 'p', t('estados.sin_resultado')));
    (estado.resultado?.valoraciones ?? []).forEach(v => {
      const r = v.resultado; const detalles = nodo(d, 'details', undefined, 'campo'); detalles.open = true; const summary = nodo(d, 'summary', nombre(v.puesto_ref)); detalles.append(summary);
      detalles.append(datos(d, t, [['preferencia', textos.numero(v.orden)], ['estado', t(`estados.${r.completo ? 'completo' : 'pendiente'}`)], ['total', r.total === null ? t('estados.sin_dato') : formatearPuntos(r.total, textos.localizacion)]]));
      const reqs = (v.requisitos ?? []).map(q => [proyeccion.requisitos?.[q.requisito_ref] ?? t('requisito_grupo'), t(`estados.${q.estado}`), t(`motivos.${Object.hasOwn(textos.mensajes.motivos, q.motivo_codigo) ? q.motivo_codigo : 'sin_motivo'}`), q.fuente_ref]);
      detalles.append(tabla(d, t, t('requisitos'), ['requisito', 'estado', 'motivo', 'fuente'], reqs));
      const filas = r.desglose.map(g => [t(`familias.${g.familia}`), t(`estados.${g.estado === 'calculado' ? 'completo' : 'pendiente'}`), g.estado === 'pendiente_dato' ? t('estados.sin_dato') : formatearPuntos(g.resultado, textos.localizacion)]);
      detalles.append(tabla(d, t, t('detalle'), ['regla', 'estado', 'puntos'], filas));
      const evidencia = nodo(d, 'details', undefined, 'campo'); const huella = nodo(d, 'p'); String(r.huella_resultado ?? '').match(/.{1,8}/g)?.forEach(parte => huella.append(d.createTextNode(parte), nodo(d, 'wbr'))); evidencia.append(nodo(d, 'summary', t('detalle_tecnico')), huella); detalles.append(evidencia); p.cuerpo.append(detalles);
    });
    p.cuerpo.append(nodo(d, 'p', t('resultado_limite')));
  }
  if (estado.vista === 'adjudicacion') { const espacio = nodo(d, 'div'); espacio.dataset.ensayoAdjudicacion = ''; p.cuerpo.append(espacio); }
  if (estado.vista === 'ciclo') { const espacio = nodo(d, 'div'); espacio.dataset.ensayoCiclo = ''; p.cuerpo.append(espacio); }
  if (estado.vista === 'tramitacion') {
    const filas = ['propuesta', 'alegar', 'resolver'].map(a => { const b = boton(d, t(`acciones.${a}`), () => {}, a, true); const motivo = nodo(d, 'span', t(`bloqueos.${a}`)); motivo.id = `provision-bloqueo-${a}`; b.setAttribute('aria-describedby', motivo.id); return [t(`acciones.${a}`), motivo, b]; });
    p.cuerpo.append(nodo(d, 'p', t('estados.revision')), tabla(d, t, t('tabs.tramitacion'), ['estado', 'motivo', 'acciones'], filas));
  }
  raiz.replaceChildren(pila);
}
