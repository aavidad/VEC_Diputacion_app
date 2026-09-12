import { traducirPortal } from "../portal-empleado/portal-i18n.js";
export const CORREO_ENDPOINT = "/api/vec/administracion/configuracion-correo";

const campos = ["configurada", "host", "puerto", "server_name", "referencia_ca", "remitente_fijo", "usuario", "modo_tls", "modo_autenticacion", "tiempo_maximo_ms", "secreto_configurado", "version_esperada"];

// No usa cookies ni almacenamiento. El secreto solamente entra en el PUT si
// el operador lo escribe de nuevo; la respuesta nunca puede poblar ese campo.
export function crearClienteCorreo({ fetchImpl = globalThis.fetch, endpoint = CORREO_ENDPOINT } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente HTTP no disponible");
  async function pedir(method, cuerpo, signal) {
    const respuesta = await fetchImpl(endpoint, { method, credentials: "omit", mode: "same-origin", redirect: "error", cache: "no-store", headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) }, ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}), ...(signal ? { signal } : {}) });
    let datos = {}; try { datos = await respuesta.json(); } catch (_) {}
    if (!respuesta.ok) { const error = new Error(datos?.error?.codigo || "error_http"); error.status = respuesta.status; error.codigo = datos?.error?.codigo; throw error; }
    return datos;
  }
  return { consultar: ({ signal } = {}) => pedir("GET", undefined, signal), actualizar: (entrada, { signal } = {}) => pedir("PUT", normalizarEntrada(entrada), signal) };
}

export function normalizarEntrada(entrada = {}) {
  const salida = {};
  for (const campo of campos) if (entrada[campo] !== undefined) salida[campo] = entrada[campo];
  if (typeof entrada.secreto === "string" && entrada.secreto.length > 0) salida.secreto = entrada.secreto;
  return salida;
}

export function montarConfiguracionCorreo(contenedor, { cliente, traducir = traducirPortal } = {}) {
  if (!contenedor || !cliente) throw new TypeError("montaje de correo invalido");
  const t = (clave) => traducir("admin_correo_" + clave);
  const h = (clave) => t(clave).replace(/[&<>"']/g, (c) => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;", "'":"&#39;"}[c]));
  contenedor.replaceChildren();
  let activo = true, guardando = false;
  const peticiones = new AbortController();
  const desmontar = () => { activo = false; peticiones.abort(); contenedor.replaceChildren(); };
  contenedor.innerHTML = `<section class="correo-denegado" aria-busy="true"><h1>${h("titulo")}</h1><p>${h("cargando")}</p></section>`;
  const cargar = async () => {
    let respuesta;
    try { respuesta = await cliente.consultar({ signal: peticiones.signal }); } catch (_) { if (activo && !peticiones.signal.aborted) contenedor.innerHTML = `<section class="correo-denegado" role="alert"><h1>${h("titulo")}</h1><p>${h("denegado")}</p></section>`; return false; }
    if (!activo) return false;
    const inicial = respuesta.configuracion || {};
  contenedor.innerHTML = `<section class="correo-configuracion"><h1>${h("titulo_formulario")}</h1><p>${h("cuenta_pendiente")}</p><form><label>${h("host")}<input name="host" required></label><label>${h("puerto")}<input name="puerto" type="number" required></label><label>${h("servidor")}<input name="server_name" required></label><label>${h("ca")}<input name="referencia_ca" required></label><label>${h("remitente")}<input name="remitente_fijo" type="email" required></label><label>${h("usuario")}<input name="usuario"></label><label>${h("secreto")}<input name="secreto" type="password" autocomplete="new-password"></label><label>${h("tls")}<select name="modo_tls"><option value="tls_implicito">${h("tls_implicito")}</option><option value="starttls_obligatorio">${h("starttls")}</option></select></label><label>${h("autenticacion")}<select name="modo_autenticacion"><option value="ninguna">${h("ninguna")}</option><option value="plain">${h("plain")}</option><option value="xoauth2">${h("oauth")}</option></select></label><label>${h("tiempo")}<input name="tiempo_maximo_ms" type="number" required></label><input name="version_esperada" type="hidden"><p data-estado></p><button class="boton-primario">${h("guardar")}</button><p role="alert"></p></form></section>`;
  const formulario = contenedor.querySelector("form"), alerta = formulario.querySelector("[role=alert]");
  const pintar = (vista) => { for (const campo of campos) { const control = formulario.elements[campo]; if (!control || campo === "secreto_configurado" || campo === "configurada") continue; const valor = campo === "version_esperada" ? vista.version : vista[campo]; if (control.type === "checkbox") control.checked = Boolean(valor); else control.value = valor ?? ""; } formulario.querySelector("[data-estado]").textContent = vista.secreto_configurado ? t("secreto_presente") : t("secreto_ausente"); };
  pintar(inicial);
  formulario.addEventListener("submit", async (evento) => { evento.preventDefault(); if (guardando || peticiones.signal.aborted) return; guardando = true; alerta.textContent = ""; const datos = Object.fromEntries(new FormData(formulario)); datos.configurada = true; datos.secreto_configurado = datos.modo_autenticacion !== "ninguna"; datos.puerto = Number(datos.puerto); datos.tiempo_maximo_ms = Number(datos.tiempo_maximo_ms); datos.version_esperada = Number(datos.version_esperada); try { const resultado = await cliente.actualizar(datos, { signal: peticiones.signal }); if (!activo) return; formulario.elements.secreto.value = ""; pintar(resultado.configuracion || {}); } catch (e) { if (activo && !peticiones.signal.aborted) alerta.textContent = e.codigo === "conflicto_version" ? t("conflicto") : t("error_guardado"); } finally { guardando = false; } });
  return true;
  };
  return { cargar, desmontar };
}
