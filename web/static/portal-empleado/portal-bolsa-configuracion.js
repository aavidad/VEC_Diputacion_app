import { consultarJSON } from "../comun/http.js?v=20261012-sonda-frontera-v1";
import { cargarTextos } from "../comun/textos.js";
import { validarReglas, API_REGLAS, valorRegla } from "./reglas/reglas.js?v=20261012-reglas-configuracion-v1";
import { textoPresentacionRegla } from "./reglas/i18n.js?v=20261012-reglas-configuracion-v1";
import { crearClientePoliticaCeseRRHH } from "./modulos/bolsa/rrhh-politica-cese-api.js?v=20260928-rrhh-politica-cese-v1";
import { RUTA_PLANTILLA_CORREO, validarPlantillaCorreo } from "./portal-bolsas-correo.js?v=20260930-portales-i18n-integracion-v1";
import { validarCanalesLlamamiento } from "./portal-bolsas-contrato.js?v=20261013-bolsa-ranking-n4";

export const RUTA_PERFIL_PROPIO_BOLSA = "/api/vec/bolsa/bolsas?perfil=propio";
export async function consultarPerfilPropioBolsa({ signal, fetchImpl = globalThis.fetch } = {}) {
  return validarPerfilPropioBolsa(await consultarJSON(RUTA_PERFIL_PROPIO_BOLSA, { fetchImpl, signal, reintentos: 0 }));
}
export function validarPerfilPropioBolsa(sobre) {
  const d = sobre?.data;
  if (!d || Object.keys(d).sort().join(",") !== "acciones,ambito,canales,consultada_en,esquema,rol"
    || Object.keys(d.rol ?? {}).sort().join(",") !== "nombre,version"
    || Object.keys(d.ambito ?? {}).sort().join(",") !== "ambito_ref,unidad_ref") throw new TypeError("perfil_bolsa_invalido");
  if (d?.esquema !== "vec.bolsa.rrhh.perfil.v1" || !d.rol || typeof d.rol.nombre !== "string" || d.rol.nombre.length > 512
    || !Number.isSafeInteger(d.rol.version) || d.rol.version < 1 || !d.ambito
    || ![d.ambito.unidad_ref, d.ambito.ambito_ref].every((r) => typeof r === "string" && r.length > 0 && r.length <= 512)
    || !Array.isArray(d.acciones) || d.acciones.length < 1 || d.acciones.length > 256
    || d.acciones.some((a) => typeof a !== "string" || !/^[a-z][a-z0-9_.-]{1,127}$/u.test(a))
    || new Set(d.acciones).size !== d.acciones.length
    || typeof d.consultada_en !== "string" || !Number.isFinite(Date.parse(d.consultada_en))) throw new TypeError("perfil_bolsa_invalido");
  const canales = validarCanalesLlamamiento(d.canales);
  if (!Array.isArray(d.canales) || canales.length !== d.canales.length) throw new TypeError("canales_bolsa_invalidos");
  return Object.freeze({ ...d, canales });
}

export function renderizarConfiguracionBolsa({ perfil, reglas, cese, correo, ayuda }, { textos, escaparHTML: e }) {
  const t = (clave, variables) => e(textos.traducir(clave, variables));
  const fecha = (valor) => e(textos.fecha(valor, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }));
  const aviso = (estado, bloque) => estado.carga === "cargando" ? `<p role="status" aria-busy="true">${t("cargando")}</p>`
    : estado.carga === "denegado" ? `<p role="alert">${t("denegado")}</p>`
      : estado.carga === "error" ? `<p role="alert">${t("error")}</p><button type="button" class="boton-secundario" data-configuracion-reintentar="${bloque}">${t("reintentar")}</button>` : "";
  const datos = perfil.datos;
  const operaciones = datos?.acciones.map((a) => `<li>${e(textos.mensajes.acciones[a.replaceAll(".", "_")] ?? textos.traducir("operacion_adicional"))}</li>`).join("") || "";
  const fila = (etiqueta, valor) => `<div class="fila-resumen"><dt>${t(etiqueta)}</dt><dd>${valor}</dd></div>`;
  const fuente = reglas.datos?.catalogos.find((c) => c.modulo === "bolsa");
  const filasReglas = fuente?.reglas.map((r) => `<tr><th scope="row">${e(textoPresentacionRegla("bolsa", r, "etiqueta").texto)}</th><td>${e(valorRegla(r))}</td><td>${e(textos.numero(r.version))}</td></tr>`).join("") || "";
  const canales = datos?.canales.map((c) => `<li>${t(`canal_${c.canal}`)} · ${t(c.modo === "automatico" ? "automatico" : "manual")}</li>`).join("") || "";
  return `<section class="panel" aria-labelledby="configuracion-titulo"><header class="cabecera-panel"><h2 id="configuracion-titulo">${t("titulo")}</h2><button type="button" class="boton-secundario" data-configuracion-ayuda aria-label="${t("ayuda_boton")}" aria-expanded="${ayuda}" aria-controls="configuracion-ayuda">?</button></header><div class="cuerpo-panel" id="configuracion-ayuda" ${ayuda ? "" : "hidden"}>${t("ayuda")}</div></section>
  <section class="panel"><header class="cabecera-panel"><h3>${t("perfil")}</h3></header><div class="cuerpo-panel">${aviso(perfil, "perfil")}${datos ? `<dl class="resumen-expediente">${fila("rol", t("rol_rrhh"))}${fila("version", e(textos.numero(datos.rol.version)))}${fila("ambito", t("ambito_unidad"))}${fila("consulta", fecha(datos.consultada_en))}</dl><h4>${t("operaciones")}</h4><ul>${operaciones}</ul>` : ""}</div></section>
  <section class="panel"><header class="cabecera-panel"><h3>${t("reglas")}</h3><a class="boton-secundario" href="/portal-empleado/reglas/?lang=${e(textos.idioma)}">${t("ver_reglas")}</a></header><div class="cuerpo-panel">${aviso(reglas, "reglas")}${fuente && fuente.estado !== "disponible" ? `<p role="status">${t("reglas_no_disponibles")}</p>` : ""}</div>${fuente?.estado === "disponible" ? `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${t("reglas")}"><table class="tabla-datos"><thead><tr><th scope="col">${t("regla")}</th><th scope="col">${t("valor")}</th><th scope="col">${t("version")}</th></tr></thead><tbody>${filasReglas}</tbody></table></div>` : ""}</section>
  <section class="panel"><header class="cabecera-panel"><h3>${t("canales")}</h3></header><div class="cuerpo-panel">${aviso(perfil, "perfil")}${datos ? `<ul>${canales || `<li>${t("sin_canales")}</li>`}</ul>` : ""}${aviso(correo, "correo")}${correo.datos ? `<dl class="resumen-expediente">${fila("asunto", e(correo.datos.asunto))}${fila("mensaje", e(correo.datos.cuerpo).replaceAll("\n", "<br>"))}</dl>` : ""}</div></section>
  <section class="panel"><header class="cabecera-panel"><h3>${t("cese")}</h3><button type="button" class="boton-secundario" data-vista="reglas">${t("ver_cese")}</button></header><div class="cuerpo-panel">${aviso(cese, "cese")}${cese.datos ? `<dl class="resumen-expediente">${fila("version", e(textos.numero(cese.datos.version)))}${fila("cese_general", t("meses", { cantidad: textos.numero(cese.datos.meses_general) }))}${fila("cese_acumulacion", t("meses", { cantidad: textos.numero(cese.datos.meses_acumulacion) }))}</dl>` : ""}</div></section>`;
}

export async function montarConfiguracionBolsa({ raiz, signal, escaparHTML, fetchImpl = globalThis.fetch }) {
  const textos = await cargarTextos("bolsa-configuracion");
  if (signal.aborted) return;
  const bloques = Object.fromEntries(["perfil", "reglas", "cese", "correo"].map((b) => [b, { carga: "cargando", datos: null }]));
  const cargas = new Map(); let ayuda = false;
  const pintar = () => { if (!signal.aborted) raiz.innerHTML = renderizarConfiguracionBolsa({ ...bloques, ayuda }, { textos, escaparHTML }); };
  const consultar = (ruta) => consultarJSON(ruta, { fetchImpl, signal, reintentos: 0 });
  const lectores = {
    perfil: () => consultarPerfilPropioBolsa({ signal, fetchImpl }),
    reglas: async () => { const datos = validarReglas(await consultar(API_REGLAS)); for (const catalogo of datos.catalogos) if (catalogo.modulo === "bolsa") for (const regla of catalogo.reglas) valorRegla(regla); return datos; },
    cese: () => crearClientePoliticaCeseRRHH({ fetchImpl }).consultar({ signal }),
    correo: async () => { const d = validarPlantillaCorreo((await consultar(RUTA_PLANTILLA_CORREO))?.data); if (!d) throw new TypeError("plantilla_no_valida"); return d; },
  };
  function cargar(bloque) {
    if (signal.aborted || cargas.has(bloque)) return;
    bloques[bloque] = { carga: "cargando", datos: null }; pintar();
    const carga = Promise.resolve().then(lectores[bloque]).then((datos) => { if (!signal.aborted) bloques[bloque] = { carga: "listo", datos }; })
      .catch((error) => { if (!signal.aborted) bloques[bloque] = { carga: [401, 403].includes(error.estado) ? "denegado" : "error", datos: null }; })
      .finally(() => { cargas.delete(bloque); pintar(); });
    cargas.set(bloque, carga);
  }
  const click = (evento) => {
    const boton = evento.target.closest?.("[data-configuracion-ayuda], [data-configuracion-reintentar]");
    if (!boton || !raiz.contains(boton)) return;
    evento.preventDefault();
    if (boton.hasAttribute("data-configuracion-ayuda")) { ayuda = !ayuda; pintar(); raiz.querySelector("[data-configuracion-ayuda]")?.focus(); }
    else if (Object.hasOwn(lectores, boton.dataset.configuracionReintentar)) cargar(boton.dataset.configuracionReintentar);
  };
  raiz.addEventListener("click", click); pintar(); Object.keys(lectores).forEach(cargar);
  return { desmontar() { raiz.removeEventListener("click", click); } };
}
