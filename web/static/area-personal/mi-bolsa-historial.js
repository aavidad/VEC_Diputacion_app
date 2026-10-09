// Histórico propio de Bolsa. El servidor obtiene la identidad y las participaciones
// de la sesión; el navegador solo elige un número de página.
import { localizacionAreaPersonal, traducir } from "./i18n.js";
import { escaparHTML, listaDatos, panel } from "./vistas/comunes.js";
import { nombreCategoria } from "./mi-bolsa-campos.js";

export const RUTA_HISTORIAL_MI_BOLSA = "/api/vec/bolsa/mi-bolsa/historial";
export const ESQUEMA_HISTORIAL_MI_BOLSA = "vec.bolsa.mi-bolsa.historial.v1";
// Nombres de campo del contrato del servidor (puertos de Bolsa y concesión V3).
const CAMPOS = ["contratos_propios", "llamamientos_propios", "renuncias_propias"];
const CLASE_CAMPO = { contrato_bolsa: "contratos_propios", llamamiento: "llamamientos_propios", renuncia: "renuncias_propias" };
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u;
const TIPO = /^[a-z][a-z0-9_]{1,39}$/u;
const MAXIMO_BYTES = 256 * 1024;
// Estados de consulta que tienen mensaje propio; cualquier otro se muestra como «error».
const ESTADOS_CON_MENSAJE = new Set(["vacio", "sinCampos", "error", "autenticacion", "denegado"]);


function t(clave, variables = {}) {
  return traducir(`areaPersonal.miBolsa.historial.${clave}`, variables);
}

function objeto(valor, nombre) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) throw new TypeError(`${nombre} no es un objeto.`);
  return valor;
}

function clavesExactas(valor, esperadas, nombre) {
  if (Object.keys(valor).sort().join() !== [...esperadas].sort().join()) throw new TypeError(`${nombre} contiene campos no autorizados.`);
}

function instante(valor, nombre) {
  if (typeof valor !== "string" || !INSTANTE.test(valor) || Number.isNaN(Date.parse(valor))) throw new TypeError(`${nombre} no es un instante UTC válido.`);
}

function texto(valor, nombre, maximo = 200) {
  if (typeof valor !== "string" || !valor.trim() || valor.length > maximo) throw new TypeError(`${nombre} no es válido.`);
}

export function validarHistorialMiBolsa(entrada, paginaSolicitada) {
  objeto(entrada, "respuesta");
  clavesExactas(entrada, ["data"], "respuesta");
  const datos = objeto(entrada.data, "data");
  clavesExactas(datos, ["esquema", "consultada_en", "campos_visibles", "historial"], "data");
  if (datos.esquema !== ESQUEMA_HISTORIAL_MI_BOLSA) throw new TypeError("Esquema de histórico incompatible.");
  instante(datos.consultada_en, "consultada_en");
  if (!Array.isArray(datos.campos_visibles) || datos.campos_visibles.some((campo, i) => !CAMPOS.includes(campo) || CAMPOS.indexOf(campo) <= CAMPOS.indexOf(datos.campos_visibles[i - 1] ?? ""))) {
    throw new TypeError("Campos visibles no válidos.");
  }
  const historial = objeto(datos.historial, "historial");
  clavesExactas(historial, ["pagina", "tamano", "hay_mas", "items"], "historial");
  if (!Number.isSafeInteger(historial.pagina) || historial.pagina !== paginaSolicitada || historial.tamano !== 20 || typeof historial.hay_mas !== "boolean" || !Array.isArray(historial.items) || historial.items.length > 20 || (historial.hay_mas && historial.items.length !== 20)) {
    throw new TypeError("Paginación del histórico no válida.");
  }
  for (const item of historial.items) {
    objeto(item, "actuación");
    const comunes = ["clase", "bolsa", "categoria", "ocurrido_en"];
    if (!Object.hasOwn(CLASE_CAMPO, item.clase) || !datos.campos_visibles.includes(CLASE_CAMPO[item.clase])) throw new TypeError("Actuación no autorizada.");
    texto(item.bolsa, "bolsa"); texto(item.categoria, "categoría"); instante(item.ocurrido_en, "ocurrido_en");
    if (Date.parse(item.ocurrido_en) > Date.parse(datos.consultada_en)) throw new TypeError("Actuación posterior a la consulta.");
    if (item.clase === "contrato_bolsa") {
      clavesExactas(item, [...comunes, "tipo", "inicio", "fin_previsto", "modalidad_clave", "procedencia"], "contrato");
      if (!TIPO.test(item.tipo) || item.procedencia !== "evento_ct_recibido") throw new TypeError("Contrato no válido.");
      if (item.inicio !== null) instante(item.inicio, "inicio");
      if (item.fin_previsto !== null) instante(item.fin_previsto, "fin_previsto");
      if (item.modalidad_clave !== null) texto(item.modalidad_clave, "modalidad_clave", 80);
    } else if (item.clase === "llamamiento") {
      clavesExactas(item, [...comunes, "canal", "resultado"], "llamamiento");
      if (item.canal !== "correo" || !["enviado", "no_enviado"].includes(item.resultado)) throw new TypeError("Llamamiento no válido.");
    } else {
      clavesExactas(item, [...comunes, "respuesta", "modo", "estado"], "renuncia");
      if (!["renuncia", "renuncia_justificada"].includes(item.respuesta) || !["firme", "propuesta_rrhh"].includes(item.modo) || !["respuesta_registrada", "propuesta_pendiente_rrhh"].includes(item.estado)) throw new TypeError("Renuncia no válida.");
    }
  }
  for (let i = 1; i < historial.items.length; i++) {
    if (historial.items[i].ocurrido_en > historial.items[i - 1].ocurrido_en) throw new TypeError("El histórico no está ordenado.");
  }
  return structuredClone(datos);
}

export async function cargarHistorialMiBolsa({ pagina = 1, fetchImpl = globalThis.fetch, signal } = {}) {
  if (!Number.isSafeInteger(pagina) || pagina < 1 || pagina > 10000 || typeof fetchImpl !== "function") throw new TypeError("Página no válida.");
  const respuesta = await fetchImpl(`${RUTA_HISTORIAL_MI_BOLSA}?pagina=${pagina}`, {
    method: "GET", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
    headers: { Accept: "application/json" }, signal,
  });
  if (respuesta?.status !== 200) {
    const error = new Error("Consulta de histórico rechazada.");
    error.codigo = respuesta?.status === 401 ? "autenticacion" : respuesta?.status === 403 ? "denegado" : "error";
    throw error;
  }
  const cuerpo = await respuesta.text();
  if (new TextEncoder().encode(cuerpo).byteLength > MAXIMO_BYTES) throw new TypeError("Respuesta excesiva.");
  return validarHistorialMiBolsa(JSON.parse(cuerpo), pagina);
}

function fecha(valor) {
  return new Intl.DateTimeFormat(localizacionAreaPersonal(), { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

function tarjeta(item) {
  const nombre = item.clase === "contrato_bolsa" ? t(item.tipo === "incorporacion" ? "incorporacion" : "contrato") : t(item.clase);
  const pares = [[t("categoria"), escaparHTML(nombreCategoria(item))], [t("fecha"), escaparHTML(fecha(item.ocurrido_en))]];
  if (item.clase === "contrato_bolsa") {
    pares.push([t("inicio"), escaparHTML(item.inicio ? fecha(item.inicio) : t("sinFecha"))]);
    pares.push([t("fin"), escaparHTML(item.fin_previsto ? fecha(item.fin_previsto) : t("sinFecha"))]);
  } else if (item.clase === "llamamiento") {
    pares.push([t("resultado"), `<span class="estado-chip ${item.resultado === "enviado" ? "info" : "aviso"}">${escaparHTML(t(item.resultado))}</span>`]);
  } else {
    pares.push([t("respuesta"), escaparHTML(t(item.respuesta === "renuncia" ? "renuncia_simple" : "renuncia_justificada"))]);
    pares.push([t("estado"), `<span class="estado-chip aviso">${escaparHTML(t(item.estado))}</span>`]);
    pares.push([t("modo"), escaparHTML(t(item.modo))]);
  }
  return `<li><article class="portal-mi-bolsa__bolsa"><h4>${escaparHTML(nombre)}</h4>${listaDatos(pares)}</article></li>`;
}

export function renderizarHistorialMiBolsa(datos = null, { pagina = 1, estado = "cargando" } = {}) {
  let contenido;
  if (estado === "cargando") contenido = `<p role="status">${escaparHTML(t("cargando"))}</p>`;
  else if (estado !== "correcto") {
    const clave = ESTADOS_CON_MENSAJE.has(estado) ? estado : "error";
    // Si el servicio no responde no es culpa de la persona: estado neutro, sin alarma.
    const identidad = clave === "autenticacion" || clave === "denegado";
    contenido = `<div class="${identidad ? "estado-error" : "estado-vacio"}" role="${identidad ? "alert" : "status"}"><p>${escaparHTML(t(clave))}</p><button type="button" class="boton-secundario" data-historial-accion="reintentar">${escaparHTML(t("reintentar"))}</button></div>`;
  } else if (datos.campos_visibles.length === 0) contenido = `<p class="nota aviso">${escaparHTML(t("sinCampos"))}</p>`;
  else if (datos.historial.items.length === 0) contenido = `<p class="estado-vacio">${escaparHTML(t("vacio"))}</p>`;
  else contenido = `<ol class="linea-tiempo">${datos.historial.items.map(tarjeta).join("")}</ol>`;
  if (estado === "correcto" && datos.campos_visibles.length && (pagina > 1 || datos.historial.hay_mas)) {
    contenido += `<nav class="paginacion-participaciones" aria-label="${escaparHTML(t("titulo"))}"><span>${escaparHTML(t("pagina", { pagina }))}</span><button type="button" class="boton-secundario" data-historial-accion="anterior" ${pagina === 1 ? "disabled" : ""}>${escaparHTML(t("anterior"))}</button><button type="button" class="boton-secundario" data-historial-accion="siguiente" ${!datos.historial.hay_mas || pagina >= 10000 ? "disabled" : ""}>${escaparHTML(t("siguiente"))}</button></nav>`;
  }
  return panel(t("titulo"), t("subtitulo"), `${contenido}<p class="nota">${escaparHTML(t("limite"))}</p>`, { clase: "historial-mi-bolsa" });
}

export function montarHistorialMiBolsa({ contenedor, fetchImpl = globalThis.fetch } = {}) {
  if (!contenedor) return null;
  let vigente = true;
  let pagina = 1;
  let peticion = null;
  let secuencia = 0;
  async function cargar(nuevaPagina) {
    if (!vigente || nuevaPagina < 1 || nuevaPagina > 10000) return;
    peticion?.abort();
    peticion = new AbortController();
    const propia = ++secuencia;
    pagina = nuevaPagina;
    contenedor.innerHTML = renderizarHistorialMiBolsa(null, { pagina });
    try {
      const datos = await cargarHistorialMiBolsa({ pagina, fetchImpl, signal: peticion.signal });
      if (vigente && propia === secuencia) contenedor.innerHTML = renderizarHistorialMiBolsa(datos, { pagina, estado: "correcto" });
    } catch (error) {
      if (vigente && propia === secuencia && !peticion.signal.aborted) contenedor.innerHTML = renderizarHistorialMiBolsa(null, { pagina, estado: error?.codigo || "error" });
    }
  }
  function pulsar(evento) {
    const accion = evento.target?.closest?.("[data-historial-accion]")?.dataset.historialAccion;
    if (accion === "reintentar") void cargar(pagina);
    else if (accion === "anterior") void cargar(pagina - 1);
    else if (accion === "siguiente") void cargar(pagina + 1);
  }
  contenedor.addEventListener("click", pulsar);
  void cargar(1);
  return { destruir() { vigente = false; secuencia++; peticion?.abort(); contenedor.removeEventListener("click", pulsar); } };
}
