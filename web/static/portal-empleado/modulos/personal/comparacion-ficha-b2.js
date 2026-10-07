import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const textos = await cargarTextos("personal-comparacion-b2");
const familias = Object.freeze({ relaciones: "relacion_ref", servicios: "servicio_ref", situaciones: "situacion_ref" });
const referencia = /^[A-Za-z0-9:_-]{1,256}$/u;
const positivo = (n) => Number.isSafeInteger(n) && n > 0;
const seguro = (s, max = 300) => typeof s === "string" && s.length <= max && !/[\u0000-\u001f\u007f]/u.test(s);
const fecha = (s) => typeof s === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(s) && s.slice(0, 4) !== "0000" && Number.isFinite(Date.parse(`${s}T12:00:00Z`)) && new Date(`${s}T12:00:00Z`).toISOString().slice(0, 10) === s;
const instante = (s) => typeof s === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(s) && fecha(s.slice(0, 10)) && Number.isFinite(Date.parse(s)) && new Date(s).toISOString().slice(0, 19) === s.slice(0, 19);
const corteValido = (c) => c && fecha(c.vigente_en) && instante(c.conocido_en);
const normalizarInstante = (s) => `${s.slice(0, 19)}.${(s.split(".")[1]?.slice(0, -1) ?? "").padEnd(6, "0")}Z`;
const mismoCorte = (a, b) => a.vigente_en === b.vigente_en && normalizarInstante(a.conocido_en) === normalizarInstante(b.conocido_en);
const fallo = () => { throw new TypeError("personal.comparacion.respuesta_incompatible"); };
const ref = (s) => typeof s === "string" && referencia.test(s);
function canonico(v) { return JSON.stringify(v, (_k, x) => x && typeof x === "object" && !Array.isArray(x) ? Object.fromEntries(Object.keys(x).sort().map((k) => [k, x[k]])) : x); }

/** La hora introducida corresponde a Madrid. Se rechazan horas inexistentes o
 * ambiguas del cambio de horario; nunca se elige silenciosamente un instante. */
export function fechaHoraMadrid(iso, localizacion = LOCALIZACION_ACTUAL) {
  const p = new Intl.DateTimeFormat(localizacion, { timeZone: "Europe/Madrid", numberingSystem: "latn", year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23" }).formatToParts(new Date(iso));
  const v = (k) => p.find((x) => x.type === k).value;
  return `${v("year")}-${v("month")}-${v("day")}T${v("hour")}:${v("minute")}:${v("second")}`;
}
export function instanteDesdeMadrid(local) {
  if (typeof local !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(?::\d{2})?$/u.test(local) || !fecha(local.slice(0, 10))) return "";
  const normal = local.length === 16 ? `${local}:00` : local;
  const encontrados = [60, 120].map((minutos) => new Date(Date.parse(`${normal}Z`) - minutos * 60_000)).filter((d) => Number.isFinite(d.getTime()) && fechaHoraMadrid(d.toISOString()) === normal);
  return encontrados.length === 1 ? encontrados[0].toISOString().replace(/Z$/u, "000Z") : "";
}
function validarRespuesta(respuesta, empleadoRef, corte) {
  const f = respuesta?.ficha; const e = respuesta?.evidencia;
  if (!f || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(empleadoRef) || f.empleado_ref !== empleadoRef ||
      !/^per_[A-Za-z0-9_-]{22,128}$/u.test(f.persona_ref) || !ref(f.organismo_ref) || !positivo(f.version) ||
      f.eficacia_administrativa !== false || f.firma_oficial !== false || !corteValido(f.corte) || !mismoCorte(f.corte, corte) ||
      !e || e.efecto_ref !== empleadoRef || !["recibo_ref", "decision_ref", "efecto_ref", "auditoria_ref"].every((k) => ref(e[k])) ||
      !/^[0-9a-f]{64}$/u.test(e.consumo_huella_sha256) || !instante(e.consultada_en) ||
      (f.cobertura !== undefined && !["completa", "parcial", "no_acreditada"].includes(f.cobertura))) fallo();
  for (const [familia, clave] of Object.entries(familias)) {
    if (!Array.isArray(f[familia]) || f[familia].length > 200) fallo();
    const vistos = new Set();
    const estados = familia === "relaciones" ? ["vigente", "suspendida", "finalizada"] : familia === "servicios" ? ["declarado", "comprobado", "reconocido"] : ["vigente", "finalizada", "rectificada"];
    for (const row of f[familia]) {
      const z = row?.traza; const id = `${row?.[clave]}:${z?.version}`;
      if (!row || !ref(row[clave]) || (familia !== "relaciones" && !/^rel_[A-Za-z0-9_-]{22,128}$/u.test(row.relacion_ref)) ||
          !z || !positivo(z.version) || !positivo(z.fuente_version) || !ref(z.fuente_ref) || !ref(z.acto_ref) ||
          !fecha(z.desde) || (z.hasta && (!fecha(z.hasta) || z.hasta <= z.desde)) || !instante(z.registrada_en) ||
          Date.parse(z.registrada_en) > Date.parse(f.corte.conocido_en) || vistos.has(id) || !estados.includes(row.estado) ||
          (familia === "servicios" && (!Number.isSafeInteger(row.dias_reconocidos) || row.dias_reconocidos < 0 || !fecha(row.periodo_desde) || !fecha(row.periodo_hasta) || row.periodo_hasta < row.periodo_desde))) fallo();
      if (familia === "relaciones" && (row.organismo_ref !== f.organismo_ref || !/^rel_[A-Za-z0-9_-]{22,128}$/u.test(row.relacion_ref))) fallo();
      for (const k of ["unidad_denominacion", "puesto_denominacion"]) if (row[k] !== undefined && !seguro(row[k])) fallo();
      if (row.catalogo_snapshot !== undefined && (!row.catalogo_snapshot || typeof row.catalogo_snapshot !== "object" || Array.isArray(row.catalogo_snapshot))) fallo();
      for (const k of ["regimen", "modalidad", "situacion", "clase_servicio"]) {
        const s = row.catalogo_snapshot?.[k];
        if (s !== undefined && (!s || !seguro(s.denominacion) || !positivo(s.version))) fallo();
      }
      vistos.add(id);
    }
  }
  // Sólo se conserva el contrato que se compara y su evidencia; no las
  // preparaciones de otros módulos ni campos adicionales de la respuesta.
  return JSON.parse(JSON.stringify({ empleado_ref: f.empleado_ref, persona_ref: f.persona_ref, organismo_ref: f.organismo_ref,
    version: f.version, corte: f.corte, cobertura: f.cobertura ?? "no_acreditada", evidencia: e,
    ...Object.fromEntries(Object.keys(familias).map((k) => [k, f[k]])) }));
}
export function compararFichasB2(primera, segunda, { empleadoRef, cortePrimero, corteSegundo }) {
  if (!corteValido(cortePrimero) || !corteValido(corteSegundo) || mismoCorte(cortePrimero, corteSegundo)) fallo();
  const a = validarRespuesta(primera, empleadoRef, cortePrimero); const b = validarRespuesta(segunda, empleadoRef, corteSegundo);
  if (a.persona_ref !== b.persona_ref || a.organismo_ref !== b.organismo_ref) fallo();
  const diferencias = {};
  for (const [familia, clave] of Object.entries(familias)) {
    const agrupar = (filas) => { const m = new Map(); for (const r of filas) { if (!m.has(r[clave])) m.set(r[clave], []); m.get(r[clave]).push(r); } for (const v of m.values()) v.sort((x, y) => x.traza.version - y.traza.version); return m; };
    const x = agrupar(a[familia]); const y = agrupar(b[familia]);
    diferencias[familia] = [...new Set([...x.keys(), ...y.keys()])].sort().flatMap((id) => {
      const primero = x.get(id) ?? []; const segundo = y.get(id) ?? [];
      return canonico(primero) === canonico(segundo) ? [] : [{ primero, segundo, estado: !primero.length ? "solo_segundo" : !segundo.length ? "solo_primero" : "distinto" }];
    });
  }
  return { primero: a, segundo: b, diferencias };
}
function nodo(d, tag, texto) { const n = d.createElement(tag); if (texto !== undefined) n.textContent = texto; return n; }
function fechaVisible(iso, locale) { return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(`${iso}T12:00:00Z`)); }
function instanteVisible(iso, locale) { return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(iso)); }
function detalleDatos(d, t, datos) { const dl = nodo(d, "dl"); for (const [k, v] of datos) dl.append(nodo(d, "dt", t(k)), nodo(d, "dd", String(v))); return dl; }
function tituloDato(familia, filas, t) {
  const r = filas.at(-1); const snap = r.catalogo_snapshot;
  return (familia === "relaciones" ? r.unidad_denominacion || snap?.modalidad?.denominacion : familia === "servicios" ? snap?.clase_servicio?.denominacion : snap?.situacion?.denominacion) || t("sin_denominacion");
}
function pintarResultado(d, destino, modelo, t, locale) {
  const cortes = nodo(d, "div"); cortes.className = "personal-comparacion-cortes";
  for (const [lado, foto] of [["primero", modelo.primero], ["segundo", modelo.segundo]]) {
    const bloque = nodo(d, "section"); bloque.className = "personal-comparacion-corte";
    bloque.append(nodo(d, "h4", t(lado)), detalleDatos(d, t, [["efectos", fechaVisible(foto.corte.vigente_en, locale)], ["conocimiento", instanteVisible(foto.corte.conocido_en, locale)], ["version_ficha", new Intl.NumberFormat(locale).format(foto.version)]]));
    const detalles = nodo(d, "details"); detalles.append(nodo(d, "summary", t("evidencia")), detalleDatos(d, t, [["recibo", foto.evidencia.recibo_ref], ["decision", foto.evidencia.decision_ref], ["efecto", foto.evidencia.efecto_ref], ["corte_exacto", foto.corte.conocido_en], ["auditoria", foto.evidencia.auditoria_ref], ["consumo", foto.evidencia.consumo_huella_sha256], ["consultada", instanteVisible(foto.evidencia.consultada_en, locale)]])); bloque.append(detalles); cortes.append(bloque);
  }
  destino.append(cortes, nodo(d, "p", t("limite")));
  const total = Object.values(modelo.diferencias).reduce((n, filas) => n + filas.length, 0);
  const estado = nodo(d, "p", total ? t("recuento", { n: new Intl.NumberFormat(locale).format(total) }) : t("sin_diferencias")); estado.setAttribute("role", "status"); destino.append(estado);
  for (const [familia, diferencias] of Object.entries(modelo.diferencias)) {
    if (!diferencias.length) continue;
    const region = nodo(d, "div"); region.className = "tabla-contenedor personal-comparacion-tabla"; region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t(familia));
    const tabla = nodo(d, "table"); tabla.className = "tabla-datos"; tabla.append(nodo(d, "caption", t(familia)));
    const head = nodo(d, "thead"); const tr = nodo(d, "tr"); for (const k of ["dato", "primero", "segundo", "resultado"]) { const th = nodo(d, "th", t(k)); th.setAttribute("scope", "col"); tr.append(th); } head.append(tr); tabla.append(head);
    const body = nodo(d, "tbody");
    for (const diff of diferencias) {
      const fila = nodo(d, "tr"); fila.append(nodo(d, "td", tituloDato(familia, diff.segundo.length ? diff.segundo : diff.primero, t)));
      for (const lado of ["primero", "segundo"]) {
        const celda = nodo(d, "td"); if (!diff[lado].length) celda.append(nodo(d, "p", t("no_devuelto")));
        for (const r of diff[lado]) {
          const dato = nodo(d, "div"); dato.className = "personal-comparacion-version";
          dato.append(nodo(d, "strong", t("version_dato", { n: new Intl.NumberFormat(locale).format(r.traza.version) })), nodo(d, "p", t(`estado_${r.estado}`)), nodo(d, "p", t("periodo", { desde: fechaVisible(familia === "servicios" ? r.periodo_desde : r.traza.desde, locale), hasta: (familia === "servicios" ? r.periodo_hasta : r.traza.hasta) ? fechaVisible(familia === "servicios" ? r.periodo_hasta : r.traza.hasta, locale) : t("abierto") })));
          if (familia === "servicios") dato.append(nodo(d, "p", t("valor_etiqueta", { etiqueta: t("dias_reconocidos"), valor: new Intl.NumberFormat(locale).format(r.dias_reconocidos) })));
          if (r.unidad_denominacion) dato.append(nodo(d, "p", t("valor_etiqueta", { etiqueta: t("unidad"), valor: r.unidad_denominacion })));
          for (const k of ["regimen", "modalidad", "situacion", "clase_servicio"]) {
            const snap = r.catalogo_snapshot?.[k];
            if (snap) dato.append(nodo(d, "p", t("valor_etiqueta", { etiqueta: t(k), valor: snap.denominacion })));
          }
          const origen = nodo(d, "details"); origen.append(nodo(d, "summary", t("origen")), detalleDatos(d, t, [["fuente", t("sin_denominacion")], ["efectos_version", fechaVisible(r.traza.desde, locale)], ["fin_efectos_version", r.traza.hasta ? fechaVisible(r.traza.hasta, locale) : t("abierto")], ["registrada", instanteVisible(r.traza.registrada_en, locale)], ["version_fuente", new Intl.NumberFormat(locale).format(r.traza.fuente_version)]]));
          for (const k of ["regimen", "modalidad", "situacion", "clase_servicio"]) {
            const snap = r.catalogo_snapshot?.[k];
            if (snap) origen.append(nodo(d, "p", t("version_catalogo", { catalogo: t(k), version: new Intl.NumberFormat(locale).format(snap.version) })));
          }
          const tecnicas = nodo(d, "details"); tecnicas.append(nodo(d, "summary", t("referencias")), detalleDatos(d, t, [["fuente_ref", r.traza.fuente_ref], ["acto_ref", r.traza.acto_ref]])); origen.append(tecnicas); dato.append(origen); celda.append(dato);
        }
        fila.append(celda);
      }
      const celda = nodo(d, "td"); const etiqueta = nodo(d, "span", t(diff.estado)); etiqueta.className = "personal-comparacion-estado"; celda.append(etiqueta); fila.append(celda); body.append(fila);
    }
    tabla.append(body); region.append(tabla); destino.append(nodo(d, "h4", t(familia)), region);
  }
}

/** Dos lecturas frescas del mismo cliente autorizado; ninguna foto previa
 * acredita acceso actual. El montaje retira también la ficha al iniciar. */
export function montarComparacionFichaB2({ raiz, cliente, empleadoRef, corteBase, alIniciar = () => {}, alError = () => {}, anunciar = () => {}, traducir = textos.traducir, localizacion = LOCALIZACION_ACTUAL }) {
  if (!raiz?.ownerDocument?.createElement || !cliente?.consultarFicha || !corteValido(corteBase)) fallo();
  const d = raiz.ownerDocument; const t = (k, v) => traducir(`general.${k}`, v); const base = { ...corteBase, conocido_en: normalizarInstante(corteBase.conocido_en) };
  const elemento = nodo(d, "section"); elemento.className = "panel personal-comparacion";
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-panel"; cabecera.append(nodo(d, "h3", t("titulo")));
  const ayuda = nodo(d, "details"); ayuda.className = "personal-comparacion-ayuda"; const signo = nodo(d, "summary", "?"); signo.setAttribute("aria-label", t("ayuda_abrir")); ayuda.append(signo, nodo(d, "p", t("ayuda"))); cabecera.append(ayuda);
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel"; const form = nodo(d, "form"); form.className = "personal-comparacion-formulario";
  const efectos = nodo(d, "input"); efectos.type = "date"; efectos.value = base.vigente_en; efectos.required = true; efectos.dataset.comparacionEfectos = "";
  const conocimiento = nodo(d, "input"); conocimiento.type = "datetime-local"; conocimiento.step = "1"; conocimiento.value = fechaHoraMadrid(base.conocido_en); conocimiento.required = true; conocimiento.dataset.comparacionConocimiento = "";
  for (const [k, input] of [["efectos_otro", efectos], ["conocimiento_otro", conocimiento]]) { const label = nodo(d, "label", t(k)); label.append(input); form.append(label); }
  const comparar = nodo(d, "button", t("comparar")); comparar.type = "submit"; comparar.dataset.comparacionConsultar = "";
  const cancelar = nodo(d, "button", t("cancelar")); cancelar.type = "button"; cancelar.hidden = true;
  form.append(comparar, cancelar); const salida = nodo(d, "div"); salida.className = "personal-comparacion-resultados"; salida.setAttribute("aria-live", "polite"); cuerpo.append(form, salida); elemento.append(cabecera, cuerpo); raiz.append(elemento);
  let vivo = true; let vuelo; let secuencia = 0;
  const limpiar = () => { secuencia++; vuelo?.abort(); vuelo = undefined; salida.replaceChildren(); };
  const mensaje = (k, alerta = false) => { const p = nodo(d, "p", t(k)); p.setAttribute("role", alerta ? "alert" : "status"); salida.replaceChildren(p); anunciar(t(k), alerta ? "error" : "status"); };
  const disponible = () => { comparar.disabled = false; cancelar.hidden = true; };
  cancelar.addEventListener("click", () => { limpiar(); disponible(); mensaje("cancelada"); comparar.focus?.(); });
  form.addEventListener("submit", async (evento) => {
    evento.preventDefault(); if (!vivo) return; limpiar();
    const otro = { vigente_en: efectos.value, conocido_en: instanteDesdeMadrid(conocimiento.value) };
    efectos.setAttribute("aria-invalid", String(!fecha(otro.vigente_en))); conocimiento.setAttribute("aria-invalid", String(!otro.conocido_en));
    if (!corteValido(otro) || mismoCorte(base, otro)) { mensaje("fecha_invalida", true); disponible(); return; }
    const conservarFoco = [comparar, efectos, conocimiento].includes(d.activeElement);
    alIniciar(); const actual = new AbortController(); vuelo = actual; const turno = secuencia; comparar.disabled = true; cancelar.hidden = false; mensaje("cargando");
    if (conservarFoco) cancelar.focus?.();
    try {
      const consultar = (c) => cliente.consultarFicha({ empleadoRef, vigenteEn: c.vigente_en, conocidoEn: c.conocido_en, signal: actual.signal });
      const primera = await consultar(base);
      if (!vivo || actual.signal.aborted || turno !== secuencia) return;
      validarRespuesta(primera, empleadoRef, base);
      const segunda = await consultar(otro);
      if (!vivo || actual.signal.aborted || turno !== secuencia) return;
      const modelo = compararFichasB2(primera, segunda, { empleadoRef, cortePrimero: base, corteSegundo: otro });
      salida.replaceChildren(); pintarResultado(d, salida, modelo, t, localizacion);
    } catch (e) {
      if (!vivo || actual.signal.aborted || turno !== secuencia) return;
      alError(); mensaje([401, 403].includes(e?.estado) ? "denegado" : "error", true);
    } finally {
      if (vivo && vuelo === actual) {
        const devolverFoco = d.activeElement === cancelar;
        vuelo = undefined; disponible();
        if (devolverFoco) comparar.focus?.();
      }
    }
  });
  return Object.freeze({ elemento, desmontar() { vivo = false; limpiar(); elemento.remove?.(); } });
}
