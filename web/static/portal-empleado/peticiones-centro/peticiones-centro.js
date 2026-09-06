import {
  crearBorradorAlta,
  crearComandoAlta,
  validarBorradorAlta,
  validarCatalogosAlta,
} from "../modulos/contratacion-temporal/contrato.js";
import {
  extraerBorradorPeticionCentro,
  renderizarFormularioPeticionCentro,
  renderizarRevisionPeticionCentro,
} from "../modulos/contratacion-temporal/vista.js";
import { MENSAJES_CONTRATACION_TEMPORAL_ES } from "../modulos/contratacion-temporal/i18n.js";

const RUTAS = Object.freeze({
  contexto: "/api/vec/contratacion-temporal/peticiones-centro/contexto",
  bandeja: "/api/vec/contratacion-temporal/peticiones-centro/bandeja",
  operaciones: "/api/vec/contratacion-temporal/peticiones-centro/operaciones",
});
const MAX_BODY = 2 * 1024 * 1024;
const TIMEOUT_MS = 15_000;
const TEXTO = Object.freeze({
  sobrelinea: "Contratación temporal · circuito previo",
  titulo: "Petición del centro y ratificación",
  descripcion: "Una petición previa reúne la necesidad del centro antes de que RRHH la transfiera al expediente de contratación.",
  ficticio: "Datos ficticios de desarrollo",
  pendienteEntrada: "Transferencia a RRHH: pendiente de entrada en el circuito siguiente",
  solicitante: "Presentar petición",
  ratificador: "Bandeja de ratificación",
  peticiones: "Peticiones del centro",
  detalle: "Detalle revisable",
  sinPeticiones: "No hay peticiones disponibles para este actor.",
  cargar: "Cargando contexto y peticiones…",
  recargar: "Recargar bandeja",
  seleccionar: "Revisar",
  volver: "Volver a Contratación",
  confirmarPresentar: "Confirmar presentación de esta petición",
  confirmarRatificar: "Confirmar ratificación de esta petición",
  confirmarPregunta: "Revise todos los datos y confirme expresamente para continuar.",
  motivo: "Motivo de la ratificación",
  motivoAyuda: "Explique brevemente la revisión realizada.",
  ratificar: "Ratificar petición",
  cancelar: "Volver a la bandeja",
  estadoPendiente: "Resultado pendiente: conserve esta pantalla y reintente la misma operación.",
  error: "No se pudo completar la operación.",
  conflicto: "La petición cambió. Se ha recargado la bandeja; revise antes de continuar.",
  exito: "Operación registrada",
  peticionRef: "Referencia de petición",
  reciboRef: "Referencia del recibo",
  version: "Versión",
  actor: "Referencia de quien registró",
  registrado: "Registrado en",
  estado: "Estado",
  solicitanteDatos: "Solicitante y cargo",
  transferencia: "La transferencia a RRHH todavía no forma parte de este circuito.",
  identidadAviso: "Identidades de prueba. La ratificación queda registrada; no se firma electrónicamente un documento. Para uso real: identidad y circuito de firma corporativos según el procedimiento que se establezca.",
  peticionNoEnviada: "Petición previa: todavía no enviada a Recursos Humanos",
  confirmacion: "Confirmo expresamente esta operación",
  volverEditar: "Volver a editar",
  datosNoDisponibles: "No hay detalle seleccionado.",
  nombre: "Nombre",
  cargo: "Cargo",
  centro: "Centro",
  pendiente: "pendiente de ratificación",
  ratificada: "ratificada",
  enviando: "Registrando la operación. Espere el recibo antes de cerrar.",
  bandejaNoActualizada: "El registro está confirmado. No se pudo actualizar la bandeja; puede recargarla sin volver a registrar.",
  motivoInvalido: "Escriba el motivo sin saltos de línea (máximo 1000 bytes) y marque la confirmación.",
});
const MENSAJES = Object.freeze({
  ...MENSAJES_CONTRATACION_TEMPORAL_ES,
  sobrelinea: TEXTO.sobrelinea,
  titulo: TEXTO.solicitante,
  descripcion: TEXTO.descripcion,
  alcance: TEXTO.pendienteEntrada,
  progreso_etiqueta: "Progreso de la petición",
  progreso_datos: "Datos",
  progreso_revision: "Revisión",
  progreso_recibo: "Registro",
  revision_titulo: "Revise la petición antes de presentarla",
  revision_aviso: "La confirmación registrará una petición previa; no crea un expediente.",
  confirmar: "Confirmar presentación",
  revisar: "Revisar petición",
  estado_disponible: "Petición preparada para revisión",
  resumen_contacto: "Contacto del centro (no quien presenta)",
  contacto_ref: "Contacto del centro",
});

function esc(value) {
  return String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function claveUUID() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
  throw new Error("El navegador no permite generar una clave UUIDv4");
}

async function leerCuerpoLimitado(respuesta) {
  if (!respuesta.body?.getReader) {
    const texto = await respuesta.text();
    if (texto.length > MAX_BODY) throw new Error("respuesta demasiado grande");
    return texto;
  }
  const lector = respuesta.body.getReader();
  const partes = [];
  let total = 0;
  try {
    while (true) {
      const parte = await lector.read();
      if (parte.done) break;
      total += parte.value.byteLength;
      if (total > MAX_BODY) throw new Error("respuesta demasiado grande");
      partes.push(parte.value);
    }
  } finally { await lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const parte of partes) { bytes.set(parte, offset); offset += parte.byteLength; }
  return new TextDecoder().decode(bytes);
}

export async function pedir(ruta, { method = "GET", cuerpo, signal } = {}) {
  const controlador = new AbortController();
  const temporizador = setTimeout(() => controlador.abort("timeout"), TIMEOUT_MS);
  const abortar = () => controlador.abort(signal?.reason || "cancelado");
  signal?.addEventListener("abort", abortar, { once: true });
  try {
    const respuesta = await fetch(ruta, {
      // Igual que el cliente CT existente: Firefox necesita same-origin para
      // presentar el certificado TLS. El servidor no usa cookies.
      method, credentials: "same-origin", mode: "same-origin", cache: "no-store",
      redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
      headers: cuerpo === undefined ? {} : { "Content-Type": "application/json; charset=utf-8" },
      body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo),
    });
    const texto = await leerCuerpoLimitado(respuesta);
    let data = null;
    try { data = texto ? JSON.parse(texto) : null; } catch { throw Object.assign(new Error(TEXTO.error), { indeterminado: true }); }
    if (!respuesta.ok) throw Object.assign(new Error(data?.error?.codigo || TEXTO.error), { status: respuesta.status, payload: data });
    return data?.data;
  } catch (error) {
    if (controlador.signal.aborted || !error?.status) {
      throw Object.assign(new Error(TEXTO.estadoPendiente), { indeterminado: true });
    }
    throw error;
  } finally {
    clearTimeout(temporizador); signal?.removeEventListener("abort", abortar);
  }
}

function estadoBase(catalogos, borrador, extra = {}) {
  return { disponible: true, ocupado: false, fase: "edicion", borrador, catalogos,
    errores: {}, mensaje_clave: "estado_disponible", tipo_mensaje: "informacion", ...extra };
}

function fecha(valor, hora = false) {
  if (!valor || !Number.isFinite(Date.parse(valor))) return "—";
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", ...(hora ? { timeStyle: "medium" } : {}), timeZone: hora ? "Europe/Madrid" : "UTC" }).format(new Date(valor));
}

function detallePeticion(peticion, contexto) {
  if (!peticion) return `<p>${esc(TEXTO.datosNoDisponibles)}</p>`;
  const s = peticion.solicitud || {};
  const c = peticion.configuracion?.solicitante;
  const actor = contexto?.actor;
  const solicitante = contexto?.intervinientes?.[c?.actor_ref];
  const catalogos = contexto?.catalogos;
  const centro = catalogos?.centros.find((v) => v.referencia === s.centro_ref);
  const etiqueta = (opciones, referencia) => opciones?.find((v) => v.referencia === referencia)?.etiqueta || referencia || "—";
  const rc = s.rc?.existe
    ? `${s.rc.numero} · ${fecha(s.rc.fecha)} · ${new Intl.NumberFormat("es-ES", { style: "currency", currency: "EUR" }).format(s.rc.importe.centimos / 100)} · ${s.rc.documento_ref}`
    : "Sin retención de crédito aportada";
  const filas = [[TEXTO.peticionRef, peticion.referencia], [TEXTO.estado, peticion.estado], [TEXTO.version, peticion.version],
    ["Centro", centro?.etiqueta || s.centro_ref], ["Contacto", etiqueta(centro?.contactos, s.contacto_ref)],
    ["Categoría", etiqueta(catalogos?.categorias, s.categoria_ref)], ["Grupo o subgrupo", s.grupo_subgrupo], ["Motivo", s.motivo_clave],
    ["Detalle", s.detalle], ["Periodo", `${fecha(s.periodo?.inicio)} — ${fecha(s.periodo?.fin)}`], ["Observaciones", s.observaciones || "—"],
    ["Retención de crédito", rc], ["Documentos aportados", (s.documentos_adjuntos || []).map((ref) => etiqueta(catalogos?.documentos, ref)).join(" · ") || "Ninguno"],
    [TEXTO.solicitanteDatos, solicitante?.puesto_ref === c?.puesto_ref
      ? `${solicitante.nombre} · ${solicitante.cargo}`
      : `${c?.actor_ref || actor?.referencia || "—"} · ${c?.puesto_ref || "cargo resuelto por identidad"}`],
    ["Creada en", fecha(peticion.creada_en, true)]];
  if (peticion.estado === "ratificada") filas.push(["Motivo de ratificación", peticion.motivo_ratificacion], ["Ratificada en", fecha(peticion.ratificada_en, true)]);
  return `<dl>${filas.map(([k, v]) => `<dt>${esc(k)}</dt><dd>${esc(v)}</dd>`).join("")}</dl>`;
}

function reciboHTML(recibo) {
  return `<section class="pc-panel pc-recibo" role="status"><h2>${esc(TEXTO.exito)}</h2><dl>
    <dt>${esc(TEXTO.reciboRef)}</dt><dd>${esc(recibo.recibo_ref)}</dd><dt>${esc(TEXTO.peticionRef)}</dt><dd>${esc(recibo.peticion_ref)}</dd>
    <dt>${esc(TEXTO.version)}</dt><dd>${esc(recibo.version)}</dd><dt>${esc(TEXTO.estado)}</dt><dd>${esc(recibo.estado)}</dd>
    <dt>${esc(TEXTO.actor)}</dt><dd>${esc(recibo.actor_ref)}</dd><dt>${esc(TEXTO.registrado)}</dt><dd>${esc(fecha(recibo.registrado_en, true))}</dd></dl>
    <p>${esc(TEXTO.peticionNoEnviada)}. ${esc(TEXTO.transferencia)}</p></section>`;
}

export function validarReciboPeticionCentro(recibo, objetivo) {
  if (!recibo || recibo.peticion_ref !== objetivo.peticionRef || recibo.version !== objetivo.version
    || recibo.estado !== objetivo.estado || recibo.actor_ref !== objetivo.actorRef
    || !recibo.recibo_ref || !recibo.registrado_en || !Number.isFinite(Date.parse(recibo.registrado_en))
    || !["registrado", "replay_confirmado"].includes(recibo.estado_local)) throw new Error(TEXTO.error);
  return recibo;
}

function tabla(peticiones, seleccionada) {
  if (!peticiones.length) return `<p>${esc(TEXTO.sinPeticiones)}</p>`;
  return `<div class="pc-tabla-wrap"><table class="pc-tabla"><caption class="solo-lectura">${esc(TEXTO.peticiones)}</caption><thead><tr><th>Referencia</th><th>Centro</th><th>Estado</th><th>Creada</th><th>Acción</th></tr></thead><tbody>${peticiones.map((p) => `<tr${p.referencia === seleccionada ? ' aria-selected="true"' : ""}><td>${esc(p.referencia)}</td><td>${esc(p.solicitud?.centro_ref)}</td><td><span class="pc-estado pc-estado-${esc(p.estado === "ratificada" ? "ratificada" : "pendiente")}">${esc(p.estado === "ratificada" ? TEXTO.ratificada : TEXTO.pendiente)}</span></td><td>${esc(fecha(p.creada_en))}</td><td><button type="button" data-seleccionar="${esc(p.referencia)}">${esc(TEXTO.seleccionar)}</button></td></tr>`).join("")}</tbody></table></div>`;
}

function formularioHTML(contexto, estado, revision) {
  const contenido = revision ? renderizarRevisionPeticionCentro(estado, { mensajes: MENSAJES }) : renderizarFormularioPeticionCentro(estado, { mensajes: MENSAJES });
  return `<section class="pc-panel ct-alta"><h2>${esc(TEXTO.solicitante)}</h2><p class="pc-aviso">${esc(TEXTO.confirmarPregunta)}</p>${contenido}<button type="button" class="boton-secundario" data-accion="cancelar-ratificacion">${esc(TEXTO.cancelar)}</button></section>`;
}

export function renderizarPeticionCentro({ contexto, peticiones = [], peticion = null, modo = "bandeja", estado = null, recibo = null, mensaje = "", motivo = "", confirmado = false } = {}) {
  const actor = contexto?.actor;
  const esSolicitante = actor?.puede_presentar && !actor?.puede_ratificar;
  const cabecera = `<section class="pc-cabecera"><p class="sobrelinea">${esc(TEXTO.sobrelinea)}</p><h1>${esc(TEXTO.titulo)}</h1><p>${esc(TEXTO.descripcion)}</p><div class="pc-etiquetas"><span class="pc-etiqueta">${esc(TEXTO.ficticio)}</span><span class="pc-etiqueta">${esc(TEXTO.pendienteEntrada)}</span></div><p>${esc(TEXTO.identidadAviso)}</p><p>${esc(actor?.nombre || "—")} · ${esc(actor?.cargo || "—")} · ${esc(actor?.centro || "—")}</p></section>`;
  const error = mensaje ? `<p class="pc-error" role="alert">${esc(mensaje)}</p>` : "";
  if (modo === "formulario") return `${cabecera}${error}${formularioHTML(contexto, estado, false)}`;
  if (modo === "revision") return `${cabecera}${error}${formularioHTML(contexto, estado, true)}`;
  if (modo === "ratificacion") return `${cabecera}${error}<section class="pc-panel pc-detalle"><h2>${esc(TEXTO.ratificador)}</h2>${detallePeticion(peticion, contexto)}<p class="pc-aviso">${esc(TEXTO.confirmarPregunta)}</p><label for="motivo-ratificacion">${esc(TEXTO.motivo)}</label><input id="motivo-ratificacion" name="motivo_ratificacion" value="${esc(motivo)}" maxlength="1000" required aria-describedby="motivo-ratificacion-ayuda"><small id="motivo-ratificacion-ayuda">${esc(TEXTO.motivoAyuda)}</small><label class="pc-confirmacion"><input type="checkbox" name="confirmacion_ratificacion"${confirmado ? " checked" : ""}> ${esc(TEXTO.confirmacion)}</label><div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="cancelar-ratificacion">${esc(TEXTO.cancelar)}</button><button type="button" class="boton-primario" data-accion="confirmar-ratificar">${esc(TEXTO.confirmarRatificar)}</button></div></section>`;
  if (modo === "pendiente") return `${cabecera}<section class="pc-panel pc-pendiente" role="status"><h2>${esc(TEXTO.estadoPendiente)}</h2><p>${esc(TEXTO.peticionNoEnviada)}</p><div class="pc-acciones"><button type="button" class="boton-primario" data-accion="reintentar">${esc("Reintentar la misma operación")}</button></div></section>`;
  const detalle = `<aside class="pc-panel pc-detalle"><h2>${esc(TEXTO.detalle)}</h2>${detallePeticion(peticion, contexto)}${peticion?.estado === "pendiente_ratificacion" && peticion.version === 1 && actor?.puede_ratificar ? `<div class="pc-acciones"><button type="button" class="boton-primario" data-accion="abrir-ratificacion">${esc(TEXTO.ratificador)}</button></div>` : ""}</aside>`;
  return `${cabecera}${error}${recibo ? reciboHTML(recibo) : ""}<div class="pc-layout"><section class="pc-panel"><h2>${esc(esSolicitante ? TEXTO.peticiones : TEXTO.ratificador)}</h2>${tabla(peticiones, peticion?.referencia)}<p>Últimas 50 peticiones visibles para su identidad.</p><div class="pc-acciones">${esSolicitante ? `<button type="button" class="boton-primario" data-accion="nueva">${esc(TEXTO.solicitante)}</button>` : ""}<button type="button" class="boton-secundario" data-accion="recargar">${esc(TEXTO.recargar)}</button><button type="button" class="boton-secundario" data-accion="volver-contratacion">${esc(TEXTO.volver)}</button></div></section>${detalle}</div>`;
}

export async function registrarOperacionPeticionCentro(cliente, comando, actorRef) {
  const objetivo = {
    peticionRef: comando.operacion === "presentar" ? `peticion:centro:${comando.clave_idempotencia}` : comando.peticion_ref,
    version: comando.operacion === "presentar" ? 1 : 2,
    estado: comando.operacion === "presentar" ? "pendiente_ratificacion" : "ratificada", actorRef,
  };
  try {
    const recibo = await cliente(RUTAS.operaciones, { method: "POST", cuerpo: structuredClone(comando) });
    return validarReciboPeticionCentro(recibo, objetivo);
  } catch (error) {
    // Solo un rechazo explícito permite descartar la clave. Transporte, lectura
    // o recibo inválido no prueban que PostgreSQL no haya confirmado.
    if ([400, 403, 409].includes(error?.status)) throw error;
    throw Object.assign(new Error(TEXTO.estadoPendiente), { indeterminado: true });
  }
}

export async function iniciarPeticionCentro({ raiz = document.querySelector("#aplicacion"), cliente = pedir } = {}) {
  if (!raiz) throw new TypeError("falta la raíz de la aplicación");
  let contexto; let peticiones = []; let peticion = null; let modo = "bandeja";
  let estado = null; let recibo = null; let mensaje = "";
  let ocupado = false; let operacionPendiente = null; let motivo = ""; let confirmado = false;
  const dibujar = () => {
    raiz.innerHTML = renderizarPeticionCentro({ contexto, peticiones, peticion, modo, estado, recibo, mensaje, motivo, confirmado });
    raiz.setAttribute("aria-busy", String(ocupado));
    if (ocupado) {
      raiz.querySelectorAll("button, input, select, textarea").forEach((control) => { control.disabled = true; });
      raiz.insertAdjacentHTML("afterbegin", `<p class="pc-aviso" role="status">${esc(TEXTO.enviando)}</p>`);
    }
  };
  const cargarBandeja = async () => {
    const bandeja = await cliente(RUTAS.bandeja);
    if (!bandeja || !Array.isArray(bandeja.peticiones) || bandeja.peticiones.length > 50
      || bandeja.peticiones.some((p) => !p?.referencia || !p.solicitud || ![1, 2].includes(p.version)
        || !["pendiente_ratificacion", "ratificada"].includes(p.estado))) throw new Error(TEXTO.error);
    peticiones = bandeja.peticiones;
    peticion = peticiones.find((p) => p.referencia === (recibo?.peticion_ref || peticion?.referencia)) || null;
  };
  const cargar = async () => {
    if (ocupado || operacionPendiente) return;
    ocupado = true; mensaje = "";
    try {
      const nuevo = await cliente(RUTAS.contexto);
      if (!nuevo?.actor?.referencia || nuevo.actor.puede_presentar === nuevo.actor.puede_ratificar) throw new Error(TEXTO.error);
      validarCatalogosAlta(nuevo.catalogos);
      contexto = nuevo;
      await cargarBandeja();
    } catch (error) {
      mensaje = [401, 403].includes(error?.status)
        ? "Acceso reservado al certificado de prueba del solicitante o del ratificador del centro. El certificado de RRHH no sustituye estas identidades."
        : TEXTO.error;
    } finally { ocupado = false; dibujar(); }
  };
  const ejecutar = async (comando) => {
    if (ocupado) return;
    const cuerpo = structuredClone(comando);
    ocupado = true; mensaje = ""; dibujar();
    try {
      recibo = await registrarOperacionPeticionCentro(cliente, cuerpo, contexto.actor.referencia);
      operacionPendiente = null; modo = "bandeja";
      // El fallo de una consulta posterior nunca convierte un recibo válido en
      // escritura pendiente ni invita a registrar otra vez.
      try { await cargarBandeja(); } catch { mensaje = TEXTO.bandejaNoActualizada; }
    } catch (error) {
      if (error.indeterminado) {
        operacionPendiente = cuerpo; modo = "pendiente"; mensaje = TEXTO.estadoPendiente;
      } else {
        operacionPendiente = null;
        modo = cuerpo.operacion === "presentar" ? "formulario" : "bandeja";
        mensaje = error.status === 409 ? TEXTO.conflicto : TEXTO.error;
        if (error.status === 409) {
          modo = "bandeja";
          try { await cargarBandeja(); } catch { mensaje = TEXTO.error; }
        }
      }
    } finally { ocupado = false; dibujar(); }
  };
  raiz.addEventListener("click", async (event) => {
    const control = event.target.closest?.("[data-accion], [data-seleccionar], [data-ct-accion], [data-ct-enfocar]");
    if (!control || ocupado || (operacionPendiente && control.dataset.accion !== "reintentar")) return;
    event.preventDefault();
    if (control.dataset.ctEnfocar) { raiz.querySelector(`#ct-${control.dataset.ctEnfocar}`)?.focus(); return; }
    const accion = control.dataset.accion || ({ volver: "editar", confirmar: "confirmar-presentar" })[control.dataset.ctAccion];
    try {
    if (control.dataset.seleccionar) { peticion = peticiones.find((p) => p.referencia === control.dataset.seleccionar) || null; dibujar(); return; }
    if (accion === "nueva" && contexto?.actor.puede_presentar) {
      modo = "formulario"; recibo = null;
      estado = estadoBase(contexto.catalogos, { ...crearBorradorAlta(), centro_ref: contexto.catalogos.centros[0]?.referencia || "" });
      mensaje = ""; dibujar(); return;
    }
    if (accion === "recargar") { await cargar(); return; }
    if (accion === "volver-contratacion") { globalThis.location.href = "/portal-empleado/#contratacion-temporal"; return; }
    if (accion === "editar") { modo = "formulario"; dibujar(); return; }
    if (accion === "abrir-ratificacion" && contexto?.actor.puede_ratificar && peticion?.version === 1) { modo = "ratificacion"; recibo = null; motivo = ""; confirmado = false; dibujar(); return; }
    if (accion === "cancelar-ratificacion") { modo = "bandeja"; dibujar(); return; }
    if (accion === "reintentar" && operacionPendiente) { await ejecutar(operacionPendiente); return; }
    if (accion === "confirmar-presentar" && modo === "revision" && contexto?.actor.puede_presentar) {
      const comando = crearComandoAlta(estado.borrador, contexto.catalogos, claveUUID());
      await ejecutar({ operacion: "presentar", ...comando }); return;
    }
    if (accion === "confirmar-ratificar" && modo === "ratificacion" && contexto?.actor.puede_ratificar) {
      motivo = raiz.querySelector("[name=motivo_ratificacion]")?.value.trim() || "";
      confirmado = raiz.querySelector("[name=confirmacion_ratificacion]")?.checked === true;
      if (!motivo || !confirmado || !peticion || new TextEncoder().encode(motivo).length > 1000 || /\p{Cc}/u.test(motivo)) { mensaje = TEXTO.motivoInvalido; dibujar(); return; }
      await ejecutar({ operacion: "ratificar", clave_idempotencia: claveUUID(), peticion_ref: peticion.referencia, version_esperada: 1, motivo });
    }
    } catch { mensaje = TEXTO.error; dibujar(); }
  });
  raiz.addEventListener("submit", (event) => {
    if (!event.target.matches("[data-ct-form]")) return;
    event.preventDefault();
    if (ocupado || operacionPendiente) return;
    const borrador = extraerBorradorPeticionCentro(event.target);
    const validacion = validarBorradorAlta(borrador, contexto.catalogos);
    estado = { ...estadoBase(contexto.catalogos, borrador), errores: validacion.errores, fase: validacion.valido ? "revision" : "edicion" };
    modo = validacion.valido ? "revision" : "formulario"; dibujar();
    raiz.querySelector(validacion.valido ? "#ct-revision-titulo" : "[data-ct-error-general]")?.focus();
  });
  raiz.addEventListener("change", (event) => {
    const campo = event.target.name;
    const formulario = event.target.closest?.("[data-ct-form]");
    if (ocupado || operacionPendiente || !formulario || !["centro_ref", "categoria_ref", "rc_existe"].includes(campo)) return;
    const borrador = extraerBorradorPeticionCentro(formulario);
    if (campo === "centro_ref") borrador.contacto_ref = "";
    if (campo === "categoria_ref") borrador.grupo_subgrupo = "";
    estado = estadoBase(contexto.catalogos, borrador); dibujar();
    raiz.querySelector(campo === "rc_existe" ? `[name="rc_existe"][value="${borrador.rc_existe ? "si" : "no"}"]` : `#ct-${campo}`)?.focus();
  });
  globalThis.addEventListener?.("beforeunload", (event) => { if (ocupado || operacionPendiente) { event.preventDefault(); event.returnValue = ""; } });
  raiz.innerHTML = `<p class="pc-cargando">${esc(TEXTO.cargar)}</p>`;
  await cargar();
  return { recargar: cargar };
}

if (typeof document !== "undefined" && document.querySelector("#aplicacion")) iniciarPeticionCentro();
