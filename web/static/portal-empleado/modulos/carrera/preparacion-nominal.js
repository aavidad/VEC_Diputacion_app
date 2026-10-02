// Consumidor preparado: la raíz aporta la lectura autorizada y el traductor común.
// No monta rutas ni selecciona una fuente de antecedentes.
function nodo(documento, etiqueta, texto, clase) {
  const elemento = documento.createElement(etiqueta);
  if (texto !== undefined) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}

function objeto(valor) { return valor !== null && typeof valor === 'object' && !Array.isArray(valor); }
function cadena(valor) { return typeof valor === 'string' && valor.trim().length > 0; }
function version(valor) { return Number.isSafeInteger(valor) && valor > 0; }
function versionCatalogo(valor) { return Number.isSafeInteger(valor) && valor >= 0; }
function fechaDia(valor) {
  return typeof valor === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(valor)
    && Number.isFinite(Date.parse(valor)) && new Date(valor).toISOString().slice(0, 10) === valor;
}
function instante(valor) {
  return typeof valor === 'string' && /^\d{4}-\d{2}-\d{2}T/.test(valor)
    && /(?:Z|[+-]\d{2}:\d{2})$/.test(valor) && Number.isFinite(Date.parse(valor));
}
function periodo(valor) {
  return objeto(valor) && fechaDia(valor.Desde)
    && (valor.Hasta === '' || fechaDia(valor.Hasta) && valor.Hasta > valor.Desde);
}
function procedencia(valor) {
  return objeto(valor) && ['ActoRef', 'FuenteRef', 'FuenteVersion', 'Certeza'].every(clave => typeof valor[clave] === 'string');
}
function lecturaValida(resultado) {
  if (!objeto(resultado) || resultado.estado !== 'pendiente'
    || !Array.isArray(resultado.pendientes) || resultado.pendientes.length !== 3
    || new Set(resultado.pendientes).size !== 3
    || !['grupo', 'grado', 'politica'].every(clave => resultado.pendientes.includes(clave))) return false;
  const a = resultado.antecedentes;
  if (!objeto(a) || !cadena(a.EmpleadoRef) || !cadena(a.OrganismoRef) || !version(a.Version)
    || !objeto(a.Corte) || !fechaDia(a.Corte.vigente_en) || !instante(a.Corte.conocido_en)
    || !['completa', 'parcial', 'no_acreditada'].includes(a.Cobertura)
    || !['Relaciones', 'Servicios', 'Puestos'].every(clave => Array.isArray(a[clave]) && a[clave].length <= 128)
    || !objeto(a.Evidencia)) return false;
  const e = a.Evidencia;
  if (!['recibo_ref', 'decision_ref', 'efecto_ref', 'auditoria_ref'].every(clave => cadena(e[clave]))
    || !/^[a-fA-F0-9]{64}$/.test(e.consumo_huella_sha256) || !instante(e.consultada_en)) return false;
  return a.Relaciones.every(r => objeto(r) && cadena(r.RelacionRef) && version(r.Version)
    && periodo(r.Periodo) && cadena(r.Estado) && cadena(r.RegimenRef) && versionCatalogo(r.RegimenVersion) && procedencia(r.Procedencia))
    && a.Servicios.every(s => objeto(s) && cadena(s.ServicioRef) && cadena(s.RelacionRef) && version(s.Version)
      && periodo(s.Periodo) && cadena(s.Estado) && cadena(s.ClaseRef) && versionCatalogo(s.ClaseVersion) && procedencia(s.Procedencia))
    && a.Puestos.every(p => objeto(p) && cadena(p.PuestoRef) && cadena(p.PuestoVersion) && cadena(p.RelacionRef)
      && periodo(p.Periodo) && (p.Nivel === null || Number.isSafeInteger(p.Nivel)) && procedencia(p.Procedencia));
}

function pintarAntecedentes(documento, resultado, textos) {
  const t = textos.traducir;
  const valor = dato => dato === null || dato === undefined || dato === '' ? t('sin_dato')
    : typeof dato === 'number' ? textos.numero(dato) : String(dato);
  const dia = dato => textos.fecha(`${dato}T12:00:00Z`, { dateStyle: 'medium', timeZone: 'UTC' });
  const fechaHora = dato => textos.fecha(dato, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'UTC' });
  function listaDatos(pares) {
    const dl = nodo(documento, 'dl', undefined, 'resumen-expediente');
    pares.forEach(([clave, dato]) => dl.append(nodo(documento, 'dt', t(`datos.${clave}`)), nodo(documento, 'dd', valor(dato))));
    const marco = nodo(documento, 'div', undefined, 'tabla-contenedor tabla-contenedor--prioritaria');
    marco.tabIndex = 0;
    marco.setAttribute('aria-label', t('detalle_tecnico'));
    marco.append(dl);
    return marco;
  }
  function fuentes(fila, pares) {
    const details = nodo(documento, 'details');
    const fuente = fila.Procedencia;
    let certeza;
    try { certeza = t(`certeza.${fuente.Certeza}`); } catch { certeza = fuente.Certeza; }
    details.append(nodo(documento, 'summary', t('procedencia')), listaDatos([...pares,
      ...(fila.Estado === undefined ? [] : [['estado_fuente', fila.Estado]]),
      ['acto', fuente.ActoRef], ['fuente', fuente.FuenteRef], ['version_fuente', fuente.FuenteVersion], ['certeza', certeza]]));
    return details;
  }
  const pila = nodo(documento, 'div', undefined, 'pila');
  pila.append(nodo(documento, 'h3', t('pendientes_titulo')));
  const pendientes = nodo(documento, 'ul');
  resultado.pendientes.forEach(clave => pendientes.append(nodo(documento, 'li', t(`pendientes.${clave}`))));
  pila.append(pendientes, nodo(documento, 'p', t('limite')),
    nodo(documento, 'p', t('cobertura_resumen', { cobertura: t(`cobertura.${resultado.antecedentes.Cobertura}`) })));

  function tabla(clave, filas, columnas, celdas) {
    pila.append(nodo(documento, 'h3', t(`tablas.${clave}`)));
    if (!filas.length) { pila.append(nodo(documento, 'p', t('sin_antecedentes'))); return; }
    const marco = nodo(documento, 'div', undefined, 'tabla-contenedor tabla-contenedor--prioritaria');
    marco.tabIndex = 0;
    marco.setAttribute('role', 'region');
    marco.setAttribute('aria-label', t(`tablas.${clave}`));
    const tabla = nodo(documento, 'table', undefined, 'tabla-datos');
    tabla.append(nodo(documento, 'caption', t(`tablas.${clave}`)));
    const cabecera = nodo(documento, 'thead'); const tr = nodo(documento, 'tr');
    columnas.forEach(columna => { const th = nodo(documento, 'th', t(`columnas.${columna}`)); th.setAttribute('scope', 'col'); tr.append(th); });
    cabecera.append(tr); const cuerpo = nodo(documento, 'tbody');
    filas.forEach(fila => {
      const tr = nodo(documento, 'tr');
      celdas(fila).forEach(dato => { const td = nodo(documento, 'td'); if (typeof dato === 'object') td.append(dato); else td.textContent = dato; tr.append(td); });
      cuerpo.append(tr);
    });
    tabla.append(cabecera, cuerpo); marco.append(tabla); pila.append(marco);
  }
  const periodoTexto = p => t(p.Hasta === '' ? 'periodo_abierto' : 'periodo', { desde: dia(p.Desde), hasta: p.Hasta === '' ? '' : dia(p.Hasta) });
  const estado = (tipo, codigo) => {
    try { return t(`estados_${tipo}.${codigo}`); } catch { return t('estado_sin_traduccion'); }
  };
  function valorConCerteza(dato, procedencia) {
    const bloque = nodo(documento, 'div');
    let certeza;
    try { certeza = t(`advertencia_certeza.${procedencia.Certeza}`); } catch { certeza = t('advertencia_certeza.desconocida'); }
    bloque.append(nodo(documento, 'span', dato), nodo(documento, 'p', certeza));
    return bloque;
  }
  const a = resultado.antecedentes;
  tabla('relaciones', a.Relaciones, ['periodo', 'estado', 'procedencia'], r => [periodoTexto(r.Periodo), valorConCerteza(estado('relaciones', r.Estado), r.Procedencia),
    fuentes(r, [['relacion', r.RelacionRef], ['version', r.Version], ['regimen', r.RegimenRef], ['version_regimen', r.RegimenVersion]])]);
  tabla('servicios', a.Servicios, ['periodo', 'estado', 'procedencia'], s => [periodoTexto(s.Periodo), valorConCerteza(estado('servicios', s.Estado), s.Procedencia),
    fuentes(s, [['servicio', s.ServicioRef], ['relacion', s.RelacionRef], ['version', s.Version], ['clase', s.ClaseRef], ['version_clase', s.ClaseVersion]])]);
  tabla('puestos', a.Puestos, ['periodo', 'nivel_puesto', 'procedencia'], p => [periodoTexto(p.Periodo), valorConCerteza(valor(p.Nivel), p.Procedencia),
    fuentes(p, [['puesto', p.PuestoRef], ['version_puesto', p.PuestoVersion], ['relacion', p.RelacionRef]])]);
  const detalle = nodo(documento, 'details'); const e = a.Evidencia;
  detalle.append(nodo(documento, 'summary', t('detalle_tecnico')), listaDatos([
    ['empleado', a.EmpleadoRef], ['organismo', a.OrganismoRef], ['version', a.Version],
    ['vigente_en', dia(a.Corte.vigente_en)], ['conocido_en', fechaHora(a.Corte.conocido_en)], ['cobertura', t(`cobertura.${a.Cobertura}`)],
    ['recibo', e.recibo_ref], ['decision', e.decision_ref], ['efecto', e.efecto_ref], ['huella', e.consumo_huella_sha256],
    ['auditoria', e.auditoria_ref], ['consultada_en', fechaHora(e.consultada_en)],
  ]));
  pila.append(detalle);
  return pila;
}

/** Consulta de solo lectura. `consultar` pertenece a la futura composición autorizada. */
export function montarPreparacionNominal({ raiz, consultar, textos, registrarDesmontar } = {}) {
  if (!raiz?.ownerDocument || typeof textos?.traducir !== 'function'
    || typeof textos?.numero !== 'function' || typeof textos?.fecha !== 'function') throw new TypeError('carrera.preparacion.contrato_incompleto');
  const documento = raiz.ownerDocument; const t = textos.traducir;
  const panel = nodo(documento, 'section', undefined, 'panel');
  const cabecera = nodo(documento, 'header', undefined, 'cabecera-panel');
  const boton = nodo(documento, 'button', t('reintentar'), 'boton-secundario'); boton.type = 'button';
  cabecera.append(nodo(documento, 'h2', t('titulo')), boton);
  const cuerpo = nodo(documento, 'div', undefined, 'cuerpo-panel pila');
  const estado = nodo(documento, 'p'); estado.setAttribute('role', 'status'); estado.setAttribute('aria-live', 'polite');
  const datos = nodo(documento, 'div'); cuerpo.append(estado, datos); panel.append(cabecera, cuerpo);
  if (textos.idioma) panel.setAttribute('lang', textos.idioma);
  raiz.replaceChildren(panel);
  let desmontada = false; let secuencia = 0; let controlador;
  function cambiarEstado(nombre) {
    panel.dataset.estado = nombre;
    estado.textContent = t(`estados.${nombre}`);
    panel.setAttribute('aria-busy', String(nombre === 'carga'));
    boton.hidden = typeof consultar !== 'function';
  }
  async function reintentar() {
    if (desmontada) return;
    const turno = ++secuencia;
    controlador?.abort(); controlador = new AbortController();
    datos.replaceChildren();
    if (typeof consultar !== 'function') { cambiarEstado('no_disponible'); return; }
    cambiarEstado('carga');
    try {
      const resultado = await consultar({ signal: controlador.signal });
      if (desmontada || turno !== secuencia) return;
      if (resultado?.estado === 'denegado') { cambiarEstado('denegado'); return; }
      if (!lecturaValida(resultado)) { cambiarEstado('no_disponible'); return; }
      const contenido = pintarAntecedentes(documento, resultado, textos);
      datos.replaceChildren(contenido); cambiarEstado('disponible');
    } catch (error) {
      if (desmontada || turno !== secuencia) return;
      datos.replaceChildren(); cambiarEstado(error?.estado === 'denegado' ? 'denegado' : 'no_disponible');
    }
  }
  function solicitarLectura() { void reintentar(); }
  boton.addEventListener('click', solicitarLectura);
  function desmontar() {
    if (desmontada) return;
    desmontada = true; secuencia++; controlador?.abort();
    boton.removeEventListener('click', solicitarLectura);
    // Evita borrar una vista que la raíz haya montado después.
    if (panel.parentNode === raiz) panel.remove();
  }
  registrarDesmontar?.(desmontar);
  const listo = reintentar();
  return Object.freeze({ listo, reintentar, desmontar });
}
