import { causasBaja, consultarReglasSituacion, hoyCivil, instalarPropuestaReposicion, motivoConCausa, renderizarCausasBaja } from "./portal-bolsas-reglas-situacion.js?v=20260926-integracion-bolsa-ct-v1";
import { traducirReglasSituacion } from "./portal-bolsas-reglas-situacion-i18n.js?v=20260926-integracion-bolsa-ct-v1";
import { cargarContratosFicha, manejarClickContratos } from "./portal-bolsas-contratos.js?v=20260926-integracion-bolsa-ct-v1";
import { renderizarTrazaValores, validarCambiosTraza } from "./portal-bolsas-traza-valores.js?v=20260926-integracion-bolsa-ct-v1";
import { traducirPortal } from "./portal-i18n.js?v=20260926-integracion-bolsa-ct-v1";
import { justificanteTraducido } from "./portal-justificante.js";

const BASE = "/api/vec/bolsa/bolsas";
const TIPOS_JUSTIFICANTE = Object.freeze(["solicitud_candidato", "informe_medico", "resolucion", "correo", "acta_bolsa", "otro"]);
const HEX_SHA256 = /^[a-f0-9]{64}$/;
// Espejo de patronDocumentoIdentidadEnReferencia y patronEtiquetaDocumentoIdentidad
// del dominio: evita enviar referencias que la API rechaza con HTTP 400.
const DOCUMENTO_IDENTIDAD = /((?:[0-9][._:/#-]?){8}|[XYZ][._:/#-]?(?:[0-9][._:/#-]?){7})[A-Z]/i;
const ETIQUETA_DOCUMENTO_IDENTIDAD = /(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)/i;
// Espejo de ReferenciaPropiaSistema del dominio de Bolsa: las referencias que
// emite el sistema (espacio de nombres alfabético y huella SHA-256 en
// hexadecimal) no pueden llevar un documento escrito por una persona, pero sus
// cifras casan por azar con los patrones de DNI o teléfono.
const REFERENCIA_PROPIA_SISTEMA = /^[a-z_]+(?::[a-z_]+)*:[0-9a-f]{64}$/;
const MENSAJE_REFERENCIA_IDENTIDAD = "La referencia no puede contener un DNI o NIE; use el número de registro o de expediente";

export function referenciaContieneDocumentoIdentidad(referencia) {
  return typeof referencia === "string" && ((!REFERENCIA_PROPIA_SISTEMA.test(referencia) && DOCUMENTO_IDENTIDAD.test(referencia)) || ETIQUETA_DOCUMENTO_IDENTIDAD.test(referencia));
}

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaOperacionesSituacion(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/operaciones`;
}

function respuestaInvalida(mensaje) {
  return { ok: false, status: 0, codigo: "respuesta_invalida", mensaje };
}

export async function consultarOperacionesSituacion(bolsa, participacion, { fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(rutaOperacionesSituacion(bolsa, participacion), {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", signal, headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) {
      const cuerpoError = await respuesta.json().catch(() => ({}));
      return errorHttp(respuesta.status, cuerpoError?.error?.codigo);
    }
    const cuerpo = await respuesta.json();
    if (cuerpo?.data?.esquema !== "vec.bolsa.rrhh.operaciones_situacion.v1" || !Array.isArray(cuerpo.data.items)) {
      return respuestaInvalida("La respuesta del historial de operaciones no respeta su contrato.");
    }
    const items = cuerpo.data.items;
    const valido = items.every((item) => item && typeof item === "object"
      && ["desde", "operacion", "situacion", "motivo", "actor", "validador", "validada_en"].every((campo) => typeof item[campo] === "string")
      && item.justificante && TIPOS_JUSTIFICANTE.includes(item.justificante.tipo)
      && typeof item.justificante.referencia === "string" && HEX_SHA256.test(item.justificante.sha256));
    const cambios = validarCambiosTraza(cuerpo.data.cambios);
    return valido && cambios ? { ok: true, datos: items, cambios } : respuestaInvalida("Un registro del historial no respeta su contrato.");
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: "No se pudo comunicar con el historial de operaciones." };
  }
}

function errorHttp(status, codigoServidor = "") {
  const errores = {
    400: ["solicitud_invalida", "La solicitud no es válida. Revise los campos del formulario."],
    403: ["acceso_denegado", "La sesión no dispone de permiso para registrar estas operaciones."],
    404: ["recurso_no_encontrado", "Operación no disponible todavía."],
    409: codigoServidor === "clave_reutilizada"
      ? ["clave_reutilizada", "La clave de idempotencia ya se usó con otros datos. Revise el historial antes de iniciar un intento nuevo."]
      : ["transicion_no_valida", "La operación no puede aplicarse a la situación vigente. Actualice la ficha y revise el historial."],
    503: ["servicio_no_disponible", "El servicio no está disponible ahora. Puede reintentar esta misma operación."],
  };
  const [codigo, mensaje] = errores[status] || ["error_servidor", `No se pudo completar la operación (HTTP ${status}).`];
  return { ok: false, status, codigo, mensaje };
}

export async function registrarOperacionSituacion(bolsa, participacion, comando, clave, { fetchImpl = fetch } = {}) {
  if (referenciaContieneDocumentoIdentidad(comando?.justificante?.referencia)) {
    return { ok: false, status: 400, codigo: "referencia_identidad", mensaje: MENSAJE_REFERENCIA_IDENTIDAD };
  }
  if (!bolsa || !participacion || !comando || !clave
    || !["pausar", "reactivar", "excluir"].includes(comando.operacion)
    || typeof comando.motivo !== "string" || !comando.motivo.trim()
    || typeof comando.validador !== "string" || !comando.validador.trim()
    || !TIPOS_JUSTIFICANTE.includes(comando.justificante?.tipo)
    || typeof comando.justificante?.referencia !== "string" || !comando.justificante.referencia.trim()
    || !HEX_SHA256.test(comando.justificante?.sha256 || "")) {
    return { ok: false, ...errorHttp(400) };
  }
  try {
    const respuesta = await fetchImpl(rutaOperacionesSituacion(bolsa, participacion), {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave },
      body: JSON.stringify(comando),
    });
    const cuerpo = await respuesta.json().catch(() => ({}));
    if ((respuesta.status === 200 || respuesta.status === 201) && respuesta.ok
      && typeof cuerpo?.data?.recibo_ref === "string" && typeof cuerpo.data.situacion === "string"
      && typeof cuerpo.data.desde === "string" && typeof cuerpo.data.reutilizada === "boolean") {
      return { ok: true, datos: cuerpo.data };
    }
    if (respuesta.ok) return respuestaInvalida("La respuesta de la operación no contiene un recibo válido.");
    return errorHttp(respuesta.status, cuerpo?.error?.codigo);
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: "No se pudo comunicar con el servicio. Puede reintentar esta misma operación." };
  }
}

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

const OPERACIONES = Object.freeze({ pausar: "Pausar", reactivar: "Reactivar", excluir: "Excluir" });
const TIPOS_ETIQUETA = Object.freeze({ solicitud_candidato: "Solicitud del candidato", informe_medico: "Informe médico", resolucion: "Resolución", correo: "Correo", acta_bolsa: "Acta de Bolsa", otro: "Otro" });

// Situación que produce cada operación (espejo de destinoOperacionSituacion).
const DESTINO_OPERACION = Object.freeze({ pausar: "no_disponible", reactivar: "disponible", excluir: "excluido" });

/**
 * Operaciones que se ofrecen desde la situación vigente. Sin reglas del
 * servidor, la selección de siempre. Con ellas, solo las que el servidor
 * admitirá; «pausar» se ofrece además donde admita pasar a no disponible
 * (renuncia justificada, art. 10, cuando la política publicada lo permite).
 */
export function operacionesDisponibles(estadoClave, transiciones) {
  const base = estadoClave === "disponible" ? ["pausar", "excluir"]
    : ["no_disponible", "trabajando"].includes(estadoClave) ? ["reactivar", "excluir"]
      : estadoClave === "excluido" ? [] : ["excluir"];
  const destinos = transiciones?.[estadoClave];
  if (!Array.isArray(destinos)) return base;
  return Object.keys(DESTINO_OPERACION)
    .filter((operacion) => (base.includes(operacion) || operacion === "pausar") && destinos.includes(DESTINO_OPERACION[operacion]));
}

export function renderizarOperacionesSituacion({ candidato, estado = {}, escaparHTML = html }) {
  const actual = estado.carga || "cargando";
  const disponibles = operacionesDisponibles(candidato.estado_clave, estado.transiciones);
  const acciones = disponibles.map((operacion) => `<button type="button" class="boton-secundario" data-b8-accion="seleccionar" data-operacion="${operacion}">${OPERACIONES[operacion]}</button>`).join("");
  const botones = estado.paso > 0 ? `<button type="button" class="boton-secundario" data-b8-accion="cancelar" ${estado.enviando ? "disabled" : ""}>Cancelar</button>`
    : estado.noDisponible ? "" : acciones;
  let contenido = "";
  if (actual === "cargando") contenido = '<p class="vacio-controlado" role="status" aria-busy="true">Cargando historial de operaciones…</p>';
  else if (actual === "error") contenido = `<p class="mensaje-error" role="alert">${escaparHTML(estado.error || "No se pudo consultar el historial.")}</p><button type="button" class="boton-secundario" data-b8-accion="reintentar">Reintentar historial</button>`;
  else if (!estado.items?.length) contenido = '<p class="vacio-controlado" role="status">No hay operaciones registradas.</p>';
  else {
    const total = estado.items.length;
    const paginas = Math.max(1, Math.ceil(total / 6));
    const pagina = Math.min(Math.max(0, Number(estado.paginaHistorial) || 0), paginas - 1);
    const visibles = estado.items.slice(pagina * 6, (pagina + 1) * 6);
    contenido = `<div class="tabla-contenedor"><table class="tabla-datos"><caption>Historial de operaciones</caption><thead><tr><th>Desde</th><th>Operación</th><th>Situación</th><th>Motivo</th><th>Justificante</th><th>Actor / validador</th></tr></thead><tbody>${visibles.map((item) => `<tr><td>${escaparHTML(item.desde)}</td><td>${escaparHTML(OPERACIONES[item.operacion] || item.operacion)}</td><td>${escaparHTML(item.situacion)}</td><td>${escaparHTML(item.motivo)}</td><td>${escaparHTML(TIPOS_ETIQUETA[item.justificante.tipo] || item.justificante.tipo)} · ${escaparHTML(item.justificante.referencia)}</td><td>${escaparHTML(item.actor)} / ${escaparHTML(item.validador)}<br>${escaparHTML(item.validada_en)}</td></tr>`).join("")}</tbody></table></div>${paginas > 1 ? `<nav class="paginacion-bolsa" aria-label="Paginación del historial de operaciones"><span>Mostrando ${pagina * 6 + 1} a ${Math.min((pagina + 1) * 6, total)} de ${total}</span><button type="button" class="boton-secundario" data-b8-accion="pagina" data-pagina="${pagina - 1}" ${pagina === 0 ? "disabled" : ""}>Anterior</button><button type="button" class="boton-secundario" data-b8-accion="pagina" data-pagina="${pagina + 1}" ${pagina + 1 >= paginas ? "disabled" : ""}>Siguiente</button></nav>` : `<p>Mostrando 1 a ${total} de ${total}</p>`}`;
  }
  const flujo = estado.paso > 0 ? renderizarPaso(estado, escaparHTML) : "";
  return `<section class="panel panel-separado" data-b8-raiz="true"><div class="cabecera-panel"><div><h4>Pausa, reactivación y exclusión</h4></div></div><div class="cuerpo-panel"><div class="acciones-vista">${botones}</div>${estado.recibo ? `<p class="mensaje-exito" role="status">Operación registrada. ${justificanteTraducido(estado.recibo, escaparHTML, (clave) => traducirPortal(`panel_${clave}`))}${estado.reutilizada ? " (respuesta recuperada)" : ""}</p>` : ""}${estado.errorOperacion ? `<p class="mensaje-error" role="alert">${escaparHTML(estado.errorOperacion)}</p>` : ""}${flujo}<h4>Historial de operaciones</h4>${contenido}${actual === "listo" ? renderizarTrazaValores({ cambios: estado.cambios || [], pagina: estado.paginaTraza, escaparHTML }) : ""}</div></section>`;
}

function renderizarPaso(estado, escaparHTML) {
  const datos = estado.formulario || {};
  const operacion = estado.operacion;
  const exclusiones = operacion === "excluir" ? `<label><input type="checkbox" name="confirma_validador_distinto" required ${datos.confirma_validador_distinto ? "checked" : ""}> Confirmo que el validador es otra persona.</label>` : "";
  const etapa = estado.paso;
  const causas = operacion === "excluir" ? estado.causasBaja || [] : [];
  const campos = etapa === 1 && causas.length
    ? renderizarCausasBaja({ causas, seleccion: datos.causa, detalle: datos.detalle, escaparHTML })
    : etapa === 1
    ? `<label>Motivo <textarea name="motivo" required minlength="2" maxlength="1000">${escaparHTML(datos.motivo || "")}</textarea></label>`
    : etapa === 2
      ? `<label>Tipo de justificante <select name="tipo" required><option value="">Seleccione un tipo</option>${TIPOS_JUSTIFICANTE.map((tipo) => `<option value="${tipo}" ${datos.tipo === tipo ? "selected" : ""}>${TIPOS_ETIQUETA[tipo]}</option>`).join("")}</select></label><label>Referencia del documento en su custodia <input name="referencia" required minlength="2" maxlength="240" value="${escaparHTML(datos.referencia || "")}"></label><label>SHA-256 del documento <input name="sha256" required pattern="[a-fA-F0-9]{64}" minlength="64" maxlength="64" value="${escaparHTML(datos.sha256 || "")}" autocomplete="off"></label><p class="nota-seguridad" role="note">El documento permanece en su custodia y no se sube a VEC. Se registra únicamente su referencia y huella SHA-256.</p>`
        : `<label>Persona validadora <input name="validador" required minlength="2" maxlength="200" value="${escaparHTML(datos.validador || "")}"></label>${exclusiones}`;
  return `<form data-b8-form="operacion" data-b8-paso="${etapa}"><p><strong>Operación seleccionada:</strong> ${OPERACIONES[operacion]}</p><h5>Paso ${etapa} de 3 · ${etapa === 1 ? "Motivo" : etapa === 2 ? "Justificante" : "Validación"}</h5>${campos}<p class="mensaje-error" role="alert">${escaparHTML(estado.errorFormulario || "")}</p><div class="acciones-vista"><button type="button" class="boton-secundario" data-b8-accion="anterior" ${etapa === 1 || estado.enviando ? "disabled" : ""}>Anterior</button><button type="submit" class="boton-primario" ${estado.enviando ? "disabled" : ""}>${estado.enviando ? "Registrando…" : etapa < 3 ? "Continuar" : `Confirmar ${OPERACIONES[operacion]}`}</button></div></form>`;
}

export function crearControladorOperacionesSituacion({ estado, renderizar, recargar, consultarReglas = consultarReglasSituacion }) {
  async function cargar(modalFicha) {
    // B13: el histórico de contratos se carga junto a la ficha, en paralelo.
    void cargarContratosFicha(modalFicha, { estado, renderizar, renderizarAlIniciar: false });
    const controlador = new AbortController();
    modalFicha.controladorOperaciones?.abort();
    modalFicha.controladorOperaciones = controlador;
    modalFicha.operacionesB8 = { ...modalFicha.operacionesB8, carga: "cargando", items: [], cambios: [] };
    renderizar();
    // Las reglas del catálogo (destinos, causas y reposición) se piden con el
    // historial; si fallan, la ficha sigue como sin catálogo.
    const finRelacion = modalFicha.candidato.estado_clave === "trabajando" ? hoyCivil() : "";
    const [res, reglas] = await Promise.all([
      consultarOperacionesSituacion(estado.bolsaSeleccionada, modalFicha.candidato.participacion_ref, { signal: controlador.signal }),
      consultarReglas({ finRelacion, signal: controlador.signal }),
    ]);
    if (controlador.signal.aborted || estado.modalFicha !== modalFicha) return;
    modalFicha.reglasSituacion = reglas.ok ? reglas.datos : null;
    const causas = causasBaja(modalFicha.reglasSituacion);
    const transiciones = modalFicha.reglasSituacion?.transiciones ?? null;
    modalFicha.operacionesB8 = res.ok
      ? { ...modalFicha.operacionesB8, carga: "listo", noDisponible: false, items: res.datos, cambios: res.cambios, causasBaja: causas, transiciones }
      : { ...modalFicha.operacionesB8, carga: "error", noDisponible: res.status === 404, error: res.mensaje, items: [], cambios: [], causasBaja: causas, transiciones };
    renderizar();
  }

  function manejarClick(evento) {
    const control = evento.target?.closest?.("[data-b8-accion]");
    if (!control || !estado.modalFicha) return false;
    evento.preventDefault();
    const modal = estado.modalFicha;
    const flujo = modal.operacionesB8 || (modal.operacionesB8 = { carga: "listo", items: [] });
    if (flujo.enviando || (flujo.noDisponible && control.dataset.b8Accion !== "reintentar")) return true;
    if (control.dataset.b8Accion === "seleccionar") {
      flujo.operacion = control.dataset.operacion;
      flujo.paso = 1;
      flujo.formulario = {};
      flujo.errorFormulario = "";
      flujo.errorOperacion = "";
    } else if (control.dataset.b8Accion === "cancelar") {
      delete flujo.operacion; delete flujo.paso; delete flujo.formulario; delete flujo.clave; delete flujo.huella;
    } else if (control.dataset.b8Accion === "anterior") {
      flujo.paso = Math.max(1, Number(flujo.paso || 1) - 1);
    } else if (control.dataset.b8Accion === "reintentar") {
      void cargar(modal);
      return true;
    } else if (control.dataset.b8Accion === "pagina-traza") {
      flujo.paginaTraza = Math.max(0, Number(control.dataset.pagina) || 0);
    } else if (control.dataset.b8Accion === "pagina") {
      flujo.paginaHistorial = Math.max(0, Number(control.dataset.pagina) || 0);
    }
    renderizar();
    return true;
  }

  async function enviar(modal, flujo) {
    const comando = { operacion: flujo.operacion, motivo: flujo.formulario.motivo, validador: flujo.formulario.validador,
      justificante: { tipo: flujo.formulario.tipo, referencia: flujo.formulario.referencia, sha256: flujo.formulario.sha256 } };
    const huella = JSON.stringify(comando);
    if (flujo.huella !== huella) {
      flujo.huella = huella;
      flujo.clave = globalThis.crypto?.randomUUID?.() || `b8-${Date.now()}-${Math.random().toString(16).slice(2)}`;
    }
    flujo.enviando = true; flujo.errorOperacion = ""; renderizar();
    const respuesta = await registrarOperacionSituacion(estado.bolsaSeleccionada, modal.candidato.participacion_ref, comando, flujo.clave);
    if (estado.modalFicha !== modal) return;
    flujo.enviando = false;
    if (!respuesta.ok) {
      flujo.errorOperacion = respuesta.mensaje;
      if (respuesta.status === 409) {
        delete flujo.operacion; delete flujo.paso; delete flujo.formulario; delete flujo.clave; delete flujo.huella;
        await recargar(modal.candidato.participacion_ref);
        if (estado.modalFicha === modal) void cargar(modal);
      }
      if (estado.modalFicha === modal) renderizar();
      return;
    }
    flujo.recibo = respuesta.datos.recibo_ref;
    flujo.reutilizada = respuesta.datos.reutilizada;
    modal.candidato = { ...modal.candidato, estado_clave: respuesta.datos.situacion, estado_desde: respuesta.datos.desde };
    delete flujo.operacion; delete flujo.paso; delete flujo.formulario; delete flujo.clave; delete flujo.huella;
    await recargar(modal.candidato.participacion_ref);
    if (estado.modalFicha !== modal) return;
    modal.operacionesB8 = { ...modal.operacionesB8, carga: "cargando", items: [] };
    renderizar();
    const controlador = new AbortController(); modal.controladorOperaciones = controlador;
    const historial = await consultarOperacionesSituacion(estado.bolsaSeleccionada, modal.candidato.participacion_ref, { signal: controlador.signal });
    if (controlador.signal.aborted || estado.modalFicha !== modal) return;
    modal.operacionesB8 = historial.ok
      ? { ...modal.operacionesB8, carga: "listo", items: historial.datos }
      : { ...modal.operacionesB8, carga: "error", error: historial.mensaje, items: [] };
    renderizar();
  }

  function manejarSubmit(evento) {
    const formulario = evento.target?.closest?.('[data-b8-form="operacion"]');
    if (!formulario || !estado.modalFicha) return false;
    evento.preventDefault();
    const modal = estado.modalFicha;
    const flujo = modal.operacionesB8;
    const datos = new FormData(formulario);
    if (flujo.enviando) return true;
    if (Number(flujo.paso) < 3) {
      const causaElegida = flujo.paso === 1 ? datos.get("causa") : null;
      if (causaElegida !== null && causaElegida !== undefined) {
        const causa = String(causaElegida);
        const detalle = String(datos.get("motivo") || "").trim();
        const motivo = motivoConCausa(flujo.causasBaja || [], causa, detalle);
        flujo.formulario = { ...flujo.formulario, causa, detalle, motivo };
        if (!motivo) { flujo.errorFormulario = traducirReglasSituacion("causa_incompleta"); renderizar(); return true; }
      } else if (flujo.paso === 1) flujo.formulario = { ...flujo.formulario, motivo: String(datos.get("motivo") || "").trim() };
      else flujo.formulario = { ...flujo.formulario, tipo: String(datos.get("tipo") || ""), referencia: String(datos.get("referencia") || "").trim(), sha256: String(datos.get("sha256") || "").trim().toLowerCase() };
      flujo.errorFormulario = flujo.paso === 2 && referenciaContieneDocumentoIdentidad(flujo.formulario.referencia)
        ? MENSAJE_REFERENCIA_IDENTIDAD : "";
      if (!flujo.errorFormulario) flujo.paso += 1;
      renderizar(); return true;
    }
    flujo.formulario = { ...flujo.formulario, validador: String(datos.get("validador") || "").trim(), confirma_validador_distinto: datos.has("confirma_validador_distinto") };
    void enviar(modal, flujo);
    return true;
  }

  function instalar(documento = globalThis.document) {
    documento.addEventListener("click", (evento) => { if (!manejarClickContratos(evento, { estado, renderizar })) manejarClick(evento); });
    documento.addEventListener("submit", (evento) => { manejarSubmit(evento); });
    instalarPropuestaReposicion(documento, () => estado.modalFicha);
  }

  return Object.freeze({ cargar, instalar, manejarClick, manejarSubmit });
}
