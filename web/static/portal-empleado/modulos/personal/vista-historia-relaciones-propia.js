import { intervaloHistoriaRelacionesValido, validarRespuestaHistoriaRelaciones } from "./cliente-http-historia-relaciones-propia.js?v=20261004-personal-relaciones-v1";
import { traducirHistoriaRelaciones as t, formatearFechaHistoriaRelaciones as fecha, formatearInstanteHistoriaRelaciones as instante, formatearNumeroHistoriaRelaciones as numero } from "./i18n-historia-relaciones-propia.js?v=20261004-personal-relaciones-v1";

function nodo(d, tipo, texto, clase) {
  const elemento = d.createElement(tipo);
  if (texto !== undefined) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}
function mensaje(d, clave, alerta = false) {
  const p = nodo(d, "p", t(`general.${clave}`), "personal-ficha-mensaje");
  p.setAttribute("role", alerta ? "alert" : "status"); return p;
}
function detalle(d, titulo, datos) {
  const contenedor = nodo(d, "details"); contenedor.append(nodo(d, "summary", t(`general.${titulo}`)));
  const lista = nodo(d, "dl");
  for (const [clave, valor] of datos) lista.append(nodo(d, "dt", t(`general.${clave}`)), nodo(d, "dd", valor));
  contenedor.append(lista); return contenedor;
}
function tabla(d, revisiones) {
  const region = nodo(d, "div", undefined, "tabla-contenedor personal-ficha-tabla");
  region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("general.titulo"));
  const tablaDatos = nodo(d, "table", undefined, "tabla-datos"); tablaDatos.append(nodo(d, "caption", t("general.titulo")));
  const head = nodo(d, "thead"), cabecera = nodo(d, "tr");
  for (const clave of ["desde", "hasta", "estado", "regimen", "modalidad", "unidad", "puesto", "situacion", "registrada", "version", "procedencia"]) {
    const th = nodo(d, "th", t(`columnas.${clave}`)); th.setAttribute("scope", "col"); cabecera.append(th);
  }
  head.append(cabecera); const cuerpo = nodo(d, "tbody");
  for (const revision of revisiones) {
    const tr = nodo(d, "tr"), r = revision.traza;
    for (const valor of [fecha(r.desde), r.hasta ? fecha(r.hasta) : t("general.abierto"), t(`estados.${revision.estado}`), ...[revision.regimen, revision.modalidad, revision.unidad, revision.puesto, revision.situacion].map((v) => v || t("general.no_consta")), instante(r.registrada_en), numero(r.version)]) tr.append(nodo(d, "td", valor));
    const procedencia = nodo(d, "td");
    procedencia.append(detalle(d, "procedencia", [["fuente", r.fuente_ref], ["version_fuente", numero(r.fuente_version)], ["acto", r.acto_ref], ["relacion", revision.relacion_ref]]));
    tr.append(procedencia); cuerpo.append(tr);
  }
  tablaDatos.append(head, cuerpo); region.append(tablaDatos); return region;
}

/** Vista independiente: sin cliente nominal permanece inactiva; no consulta al montar. */
export function montarVistaHistoriaRelacionesPropia({ raiz, cliente, registrarDesmontar, anunciar = () => {}, alCaducarSesion = () => {}, efectosDesde = "", efectosHasta = "" } = {}) {
  const d = raiz?.ownerDocument;
  if (!d?.createElement || typeof raiz.append !== "function" || typeof anunciar !== "function" || typeof alCaducarSesion !== "function" ||
      (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista_historia_relaciones_no_disponible");
  const disponible = typeof cliente?.consultar === "function";
  const panel = nodo(d, "section", undefined, "panel personal-ficha-panel"); panel.dataset.personalHistoriaRelaciones = "";
  const cabecera = nodo(d, "header", undefined, "cabecera-panel"); cabecera.append(nodo(d, "h3", t("general.titulo")));
  const ayuda = nodo(d, "details", undefined, "ayuda-contextual"), abrirAyuda = nodo(d, "summary", "?");
  abrirAyuda.setAttribute("aria-label", t("general.ayuda")); ayuda.append(abrirAyuda);
  for (const clave of ["periodos", "conocimiento", "versiones"]) ayuda.append(nodo(d, "p", t(`ayuda.${clave}`)));
  cabecera.append(ayuda);
  const cuerpo = nodo(d, "div", undefined, "cuerpo-panel"), formulario = nodo(d, "form", undefined, "personal-ficha-corte");
  const campos = [];
  for (const [clave, valor] of [["desde", efectosDesde], ["hasta", efectosHasta]]) {
    const grupo = nodo(d, "div", undefined, "personal-ficha-corte-campo"), etiqueta = nodo(d, "label", t(`general.${clave}`));
    const input = nodo(d, "input"); input.type = "date"; input.required = true; input.id = `personal-historia-relaciones-${clave}`; input.name = `efectos_${clave}`; input.value = valor;
    input.dataset.personalHistoriaFecha = clave; input.disabled = !disponible;
    etiqueta.setAttribute("for", input.id); input.setAttribute("aria-describedby", "personal-historia-relaciones-error-fechas");
    grupo.append(etiqueta, input); formulario.append(grupo); campos.push(input);
  }
  const [desde, hasta] = campos;
  const errorFechas = nodo(d, "p"); errorFechas.id = "personal-historia-relaciones-error-fechas"; errorFechas.setAttribute("role", "alert");
  const consultar = nodo(d, "button", t("general.consultar"), "boton-primario"); consultar.type = "submit"; consultar.disabled = !disponible; consultar.dataset.personalHistoriaConsultar = "";
  const cancelar = nodo(d, "button", t("general.cancelar"), "boton-secundario"); cancelar.type = "button"; cancelar.disabled = true; cancelar.dataset.personalHistoriaCancelar = "";
  formulario.append(consultar, cancelar);
  const resultado = nodo(d, "div", undefined, "personal-ficha-tabla-conjunto"); resultado.dataset.personalHistoriaResultado = ""; resultado.setAttribute("aria-live", "polite"); resultado.setAttribute("tabindex", "-1");
  resultado.append(mensaje(d, disponible ? "sin_consulta" : "no_configurado"));
  cuerpo.append(formulario, errorFechas, resultado); panel.append(cabecera, cuerpo); raiz.append(panel);
  let activa = true, turno = 0, vuelo;
  const validar = () => {
    const valido = intervaloHistoriaRelacionesValido(desde.value, hasta.value);
    for (const campo of campos) campo.setAttribute("aria-invalid", String(!valido));
    errorFechas.textContent = valido ? "" : t("general.intervalo_invalido"); return valido;
  };
  for (const campo of campos) {
    campo.addEventListener("blur", validar);
    campo.addEventListener("input", () => { if (intervaloHistoriaRelacionesValido(desde.value, hasta.value)) validar(); });
  }
  const ocupada = (valor) => { consultar.disabled = !disponible || valor; cancelar.disabled = !valor; resultado.setAttribute("aria-busy", String(valor)); };
  const enfocar = () => { if ((d.activeElement === consultar || d.activeElement === d.body) && (typeof d.hasFocus !== "function" || d.hasFocus())) resultado.focus?.(); };
  cancelar.addEventListener("click", () => {
    if (!activa || !vuelo) return;
    turno += 1; vuelo.abort(); vuelo = undefined; ocupada(false);
    resultado.replaceChildren(mensaje(d, "cancelada")); consultar.focus?.();
  });
  formulario.addEventListener("submit", async (evento) => {
    evento.preventDefault();
    if (!activa || !disponible || vuelo) return;
    if (!validar()) { desde.focus?.(); return; }
    const filtros = { efectosDesde: desde.value, efectosHasta: hasta.value };
    const controlador = new AbortController(), actual = ++turno; vuelo = controlador;
    resultado.replaceChildren(mensaje(d, "cargando")); ocupada(true);
    const vigente = () => activa && turno === actual && !controlador.signal.aborted;
    try {
      const respuesta = await cliente.consultar({ ...filtros, signal: controlador.signal });
      if (!vigente()) return;
      const datos = validarRespuestaHistoriaRelaciones({ data: respuesta }, filtros), h = datos.historia;
      const cobertura = nodo(d, "p", t(`cobertura.${h.cobertura}`), "personal-ficha-mensaje");
      resultado.replaceChildren(nodo(d, "p", t("general.intervalo", { desde: fecha(h.corte.efectos_desde), hasta: fecha(h.corte.efectos_hasta) })), nodo(d, "p", t("general.conocido", { fecha: instante(h.corte.conocido_en) })), nodo(d, "p", t("general.consultada", { fecha: instante(datos.consultada_en) })), cobertura, mensaje(d, "alcance"));
      if (h.revisiones.length) resultado.append(nodo(d, "p", t("general.desplazar"), "personal-ficha-desplazar"), tabla(d, h.revisiones));
      else resultado.append(mensaje(d, "vacio"));
      resultado.append(detalle(d, "detalle", [["recibo", datos.recibo_ref]]));
    } catch (causa) {
      if (!vigente()) return;
      if (causa?.estado === 401 || causa?.codigo === "sesion_caducada") { alCaducarSesion(); if (!activa) return; }
      const codigo = ["intervalo_invalido", "sesion_caducada", "denegado", "no_configurado", "excede_limite", "respuesta_no_valida", "no_disponible"].includes(causa?.codigo) ? causa.codigo : "no_disponible";
      resultado.replaceChildren(mensaje(d, codigo, true)); anunciar(t(`general.${codigo}`), "error");
    } finally { if (vigente()) { vuelo = undefined; ocupada(false); enfocar(); } }
  });
  const desmontar = () => { if (!activa) return; activa = false; turno += 1; vuelo?.abort(); vuelo = undefined; panel.remove?.(); };
  registrarDesmontar?.(desmontar); return Object.freeze({ desmontar });
}
