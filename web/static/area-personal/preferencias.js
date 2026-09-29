import { escaparHTML } from "./vistas/comunes.js";
import { traducir } from "./i18n.js";

const P = (clave, variables) => traducir(`areaPersonal.preferencias.${clave}`, variables);
const CAMPOS = Object.freeze([
  ["idioma", "idiomas"], ["tamano_texto", "tamanos_texto"],
  ["tema", "temas"], ["inicio", "inicios"], ["filas", "filas"],
]);
const OPCIONES = Object.freeze({
  idioma: ["navegador", "es", "en"], tamano_texto: ["normal", "grande", "muy_grande"],
  tema: ["sistema", "claro", "oscuro"], inicio: ["cuadro", "peticiones", "bolsas"],
});

export function errorPreferencias(error) {
  return P(`error.${["autenticacion", "denegado", "conflicto", "validacion", "servicio", "respuesta"].includes(error?.codigo)
    ? error.codigo : "servicio"}`);
}

function opcionesCampo(catalogo, campo, lista, elegido) {
  const elementos = lista === "filas"
    ? catalogo.filas.map((numero) => ({ codigo: numero, nombre: String(numero) }))
    : catalogo[lista].filter((opcion) => OPCIONES[campo].includes(opcion.codigo))
      .map((opcion) => ({ codigo: opcion.codigo, nombre: P(`opcion.${campo}.${opcion.codigo}`) }));
  return elementos.map(({ codigo, nombre }) => `<option value="${escaparHTML(codigo)}"${codigo === elegido ? " selected" : ""}>${escaparHTML(nombre)}</option>`).join("");
}

export function renderizarPreferencias(preferencias) {
  const { catalogo, estado, error, guardando, recibo, pendiente, avisoInicio, borrador } = preferencias || {};
  const mensaje = error ? `<p class="preferencias-estado preferencias-error" role="alert" tabindex="-1" data-pref-resultado>${escaparHTML(errorPreferencias(error))}</p>` : "";
  if (!catalogo || !estado) return `<section class="panel preferencias-panel"><header><div><h2>${escaparHTML(P("titulo"))}</h2><p>${escaparHTML(P("subtitulo"))}</p></div></header><div class="panel-contenido">${mensaje}<p>${escaparHTML(P("sinDatos"))}</p><button type="button" class="boton-secundario" data-accion="recargar-preferencias">${escaparHTML(P("recargar"))}</button></div></section>`;
  const valores = borrador || estado.valores;
  const selects = CAMPOS.map(([campo, lista]) => `<div class="preferencias-campo"><label for="preferencia-${campo}">${escaparHTML(P(`campo.${campo}`))}</label><select id="preferencia-${campo}" name="${campo}" ${guardando ? "disabled" : ""}>${opcionesCampo(catalogo, campo, lista, valores[campo])}</select></div>`).join("");
  const booleanos = ["alto_contraste", "aviso_correo_tareas", "aviso_correo_plazos"].map((campo) => `<label class="preferencias-casilla"><input type="checkbox" name="${campo}" ${valores[campo] ? "checked" : ""} ${guardando ? "disabled" : ""}><span>${escaparHTML(P(`campo.${campo}`))}</span></label>`).join("");
  const confirmacion = recibo ? `<p class="preferencias-estado preferencias-exito" role="status" tabindex="-1" data-pref-resultado>${escaparHTML(P("guardado", { recibo: recibo.recibo_ref }))}</p>` : "";
  const incierto = pendiente ? `<p class="preferencias-estado preferencias-aviso" role="alert">${escaparHTML(P("incierto"))}</p><button type="button" class="boton-secundario" data-accion="reintentar-preferencias">${escaparHTML(P("reintentarExacto"))}</button>` : "";
  return `<section class="panel preferencias-panel"><header><div><h2>${escaparHTML(P("titulo"))}</h2><p>${escaparHTML(P("subtitulo"))}</p></div><button type="button" class="boton-ayuda" data-accion="ayuda-preferencias" aria-expanded="false" aria-controls="ayuda-preferencias" aria-label="${escaparHTML(P("ayudaAbrir"))}">?</button></header><div class="panel-contenido"><div id="ayuda-preferencias" class="preferencias-ayuda" hidden><p>${escaparHTML(P("ayuda"))}</p></div>${avisoInicio ? `<p class="preferencias-estado preferencias-aviso">${escaparHTML(P("inicioAjeno"))}</p>` : ""}${mensaje}${confirmacion}${incierto}<form id="formulario-preferencias"><div class="preferencias-rejilla">${selects}<div class="preferencias-campo preferencias-opciones">${booleanos}</div></div><p class="preferencias-nota">${escaparHTML(P("avisosLimite"))}</p><div class="fila-acciones"><button type="submit" class="boton-primario" ${guardando || pendiente ? "disabled" : ""}>${escaparHTML(guardando ? P("guardando") : P("guardar"))}</button><button type="button" class="boton-secundario" data-accion="recargar-preferencias" ${guardando ? "disabled" : ""}>${escaparHTML(P("recargar"))}</button></div></form></div></section>`;
}

export function valoresDelFormulario(formulario) {
  const datos = new FormData(formulario);
  return Object.freeze({
    idioma: String(datos.get("idioma") || ""),
    tamano_texto: String(datos.get("tamano_texto") || ""),
    alto_contraste: datos.has("alto_contraste"),
    tema: String(datos.get("tema") || ""),
    inicio: String(datos.get("inicio") || ""),
    filas: Number(datos.get("filas")),
    aviso_correo_tareas: datos.has("aviso_correo_tareas"),
    aviso_correo_plazos: datos.has("aviso_correo_plazos"),
  });
}

export function crearOperacionPreferencias(preferencias, valores, cryptoImpl = globalThis.crypto) {
  if (!preferencias?.estado || !preferencias.catalogo || typeof cryptoImpl?.randomUUID !== "function") {
    throw new Error("No se puede crear una operación segura de preferencias.");
  }
  return Object.freeze({ version_esperada: preferencias.estado.version,
    catalogo_version_ref: preferencias.catalogo.version_ref,
    clave_operacion: `web-pref-${cryptoImpl.randomUUID()}`, valores });
}
