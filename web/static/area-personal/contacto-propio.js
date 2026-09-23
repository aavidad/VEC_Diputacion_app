import { crearClienteOperacionesContactoPropio, ErrorOperacionContacto, referenciaOperacionContactoValida } from "./cliente-http.js";
import { textoContactoPropio as t } from "./i18n-contacto-propio.js";

export function capturarCorreoEnviado(entrada) { return String(entrada?.value ?? "").trim(); }

function autorizado(contexto) {
  return contexto?.capacidad === true && Number.isSafeInteger(contexto.version)
    && contexto.version >= 0 && contexto.version < Number.MAX_SAFE_INTEGER;
}

function errorVisible(error) {
  if (error?.codigo === "peticion_invalida") return t("errorEntrada");
  if (error?.codigo === "acceso_denegado" || error?.codigo === "no_encontrada") return t("errorPermiso");
  if (error?.codigo === "operacion_preparada") return t("conflictoPreparada");
  if (error?.codigo === "conflicto") return t("conflicto");
  if (error?.codigo === "confirmacion_incierta") return t("confirmacionIncierta");
  return t("errorServicio");
}

export function crearControladorContactoPropio({ autorizacionServidor = null, fetchImpl = globalThis.fetch,
  presentacion = false, clienteOperaciones = null, alConfirmar = null, alDenegar = null } = {}) {
  const cliente = clienteOperaciones ?? crearClienteOperacionesContactoPropio({ fetchImpl });
  const suscriptores = new Set();
  let version = autorizacionServidor?.version ?? 0;
  let operaciones = [];
  let siguienteDesde = "";
  let seleccion = null;
  let cargado = false;
  let ocupado = false;
  let aviso = "";
  let tipoAviso = "info";
  let generacion = 0;
  let activo = true;
  let denegado = false;
  const abortos = new Set();
  const emitir = () => { if (activo) for (const fn of suscriptores) fn(); };
  const decir = (texto, tipo = "info") => { if (!activo) return; aviso = texto; tipoAviso = tipo; emitir(); };
  const limpiar = () => { operaciones = []; siguienteDesde = ""; seleccion = null; cargado = false; version = 0; generacion++; };
  const bloquear = () => { if (!activo || denegado) return; denegado = true; limpiar(); aviso = t("errorPermiso"); tipoAviso = "error"; alDenegar?.(); emitir(); };
  const exigir = () => { if (!activo || denegado || !autorizado(autorizacionServidor) || presentacion) throw new ErrorOperacionContacto("acceso_denegado", 403); };
  const actualizar = (op) => {
    if (!activo || denegado) return;
    operaciones = [op, ...operaciones.filter((item) => item.operacion_ref !== op.operacion_ref)];
    if (seleccion?.operacion_ref === op.operacion_ref) seleccion = op;
    emitir();
  };
  async function ejecutar(fn) {
    exigir();
    if (ocupado) throw new ErrorOperacionContacto("operacion_en_curso");
    const aborto = new AbortController();
    abortos.add(aborto);
    ocupado = true;
    emitir();
    try {
      const resultado = await fn(aborto.signal);
      if (!activo) throw new ErrorOperacionContacto("operacion_desmontada");
      return resultado;
    } catch (error) {
      if (activo && ["acceso_denegado", "no_encontrada"].includes(error?.codigo)) {
        bloquear();
      }
      throw error;
    } finally {
      abortos.delete(aborto);
      if (activo) { ocupado = false; emitir(); }
    }
  }
  async function cargar({ despuesDe = "" } = {}) {
    return ejecutar(async (signal) => {
      try {
        const lista = await cliente.listar(20, despuesDe, { signal });
        if (!activo) return null;
        operaciones = despuesDe ? [...operaciones, ...lista.operaciones] : [...lista.operaciones];
        siguienteDesde = lista.siguiente_desde;
        cargado = true;
        if (!despuesDe) { seleccion = null; generacion++; }
        decir(operaciones.length ? t("seleccionExplicita") : t("historialVacio"));
        return operaciones;
      } catch (error) { decir(errorVisible(error), "error"); throw error; }
    });
  }
  async function seleccionar(ref) {
    return ejecutar(async (signal) => {
      if (!referenciaOperacionContactoValida(ref)) throw new ErrorOperacionContacto("peticion_invalida");
      const turno = ++generacion;
      seleccion = null;
      emitir();
      try {
        const detalle = await cliente.detalle(ref, { signal });
        if (!activo || turno !== generacion) return null;
        seleccion = detalle;
        actualizar(detalle);
        decir(detalle.estado === "confirmada" ? t("reciboSeleccionado", { recibo: detalle.recibo_ref })
          : detalle.estado === "preparada" ? t("preparadaSeleccionada") : t("canceladaSeleccionada"),
        detalle.estado === "confirmada" ? "exito" : "info");
        return detalle;
      } catch (error) {
        if (turno === generacion) decir(errorVisible(error), "error");
        throw error;
      }
    });
  }
  async function preparar(correo) {
    return ejecutar(async (signal) => {
      const limpio = String(correo ?? "").trim();
      if (!limpio || limpio.length > 254) throw new ErrorOperacionContacto("peticion_invalida");
      if (operaciones.some((op) => op.estado === "preparada")) {
        decir(t("preparadaPendiente"), "aviso");
        throw new ErrorOperacionContacto("operacion_preparada", 409);
      }
      try {
        const op = await cliente.preparar(limpio, version, { signal });
        if (!activo) return null;
        seleccion = op;
        generacion++;
        actualizar(op);
        decir(t("preparadaRevisar"), "aviso");
        return op;
      } catch (error) {
        if (error?.codigo === "operacion_preparada") {
          // Otra pestaña pudo preparar una intención. Su referencia no autoriza nada.
          try { const lista = await cliente.listar(20, "", { signal }); if (activo) { operaciones = [...lista.operaciones]; siguienteDesde = lista.siguiente_desde; cargado = true; } }
          catch { /* Se conserva el conflicto; se podrá consultar de nuevo. */ }
        }
        decir(errorVisible(error), "error");
        throw error;
      }
    });
  }
  async function confirmar(correo) {
    return ejecutar(async (signal) => {
      const op = seleccion;
      if (op?.estado !== "preparada") throw new ErrorOperacionContacto("peticion_invalida");
      const limpio = String(correo ?? "").trim();
      if (!limpio || limpio.length > 254) throw new ErrorOperacionContacto("peticion_invalida");
      const turno = generacion;
      try {
        const confirmado = await cliente.confirmar(op.operacion_ref, limpio, op.version_esperada, { signal });
        if (!activo || turno !== generacion) return null;
        version = confirmado.version;
        actualizar(confirmado);
        decir(t("correcto", { recibo: confirmado.recibo_ref }), "exito");
        alConfirmar?.({ reciboRef: confirmado.recibo_ref, version: confirmado.version, correo: limpio });
        return confirmado;
      } catch (error) {
        if (error?.codigo === "confirmacion_incierta") {
          decir(t("confirmacionIncierta"), "aviso");
          // La referencia exacta se consulta. Nunca se confirma de nuevo automáticamente.
          try {
            const detalle = await cliente.detalle(op.operacion_ref, { signal });
            if (activo && turno === generacion) {
              actualizar(detalle);
              if (detalle.estado === "confirmada") {
                version = detalle.version;
                decir(t("reciboSeleccionado", { recibo: detalle.recibo_ref }), "exito");
                alConfirmar?.({ reciboRef: detalle.recibo_ref, version: detalle.version, correo: limpio });
              } else decir(t("resultadoNoConfirmado"), "aviso");
            }
          } catch (consultaError) {
            if (["acceso_denegado", "no_encontrada"].includes(consultaError?.codigo)) bloquear();
            else if (activo && turno === generacion) decir(t("consultaSinConfirmacion"), "aviso");
          }
        } else decir(errorVisible(error), "error");
        throw error;
      }
    });
  }
  async function cancelar() {
    return ejecutar(async (signal) => {
      const op = seleccion;
      if (op?.estado !== "preparada") throw new ErrorOperacionContacto("peticion_invalida");
      const turno = generacion;
      try {
        const cancelada = await cliente.cancelar(op.operacion_ref, { signal });
        if (activo && turno === generacion) { actualizar(cancelada); decir(t("canceladaSeleccionada")); }
        return cancelada;
      } catch (error) {
        if (error?.codigo === "conflicto" || error?.codigo === "confirmacion_incierta") {
          try { const detalle = await cliente.detalle(op.operacion_ref, { signal }); if (activo && turno === generacion) actualizar(detalle); }
          catch (consultaError) { if (["acceso_denegado", "no_encontrada"].includes(consultaError?.codigo)) bloquear(); }
        }
        decir(errorVisible(error), "error");
        throw error;
      }
    });
  }
  return Object.freeze({
    get autorizado() { return activo && !denegado && !presentacion && autorizado(autorizacionServidor); },
    get ocupado() { return ocupado; }, get cargado() { return cargado; },
    get operaciones() { return [...operaciones]; }, get siguienteDesde() { return siguienteDesde; },
    get seleccion() { return seleccion; }, get aviso() { return aviso; }, get tipoAviso() { return tipoAviso; },
    cargar, seleccionar, preparar, confirmar, cancelar,
    suscribir(fn) { suscriptores.add(fn); return () => suscriptores.delete(fn); },
    destruir() { if (!activo) return; activo = false; for (const aborto of abortos) aborto.abort(); abortos.clear(); limpiar(); aviso = ""; suscriptores.clear(); },
  });
}

export function montarContactoPropio({ contenedor, autorizacionServidor = null, fetchImpl,
  presentacion = false, reciboAnterior = null, alGuardar = null, controlador: externo = null } = {}) {
  if (!contenedor?.replaceChildren) return null;
  const controlador = externo ?? crearControladorContactoPropio({ autorizacionServidor, fetchImpl, presentacion, alConfirmar: alGuardar });
  const doc = contenedor.ownerDocument;
  const nodo = (tipo, clase = "", texto = "") => { const el = doc.createElement(tipo); el.className = clase; el.textContent = texto; return el; };
  const formulario = nodo("form", "contacto-operacion-formulario"); formulario.noValidate = true;
  const campo = nodo("div", "campo");
  const etiqueta = nodo("label", "", t("etiquetaCorreo")); etiqueta.htmlFor = "correo-contacto-propio";
  const entrada = nodo("input"); entrada.id = "correo-contacto-propio"; entrada.type = "email";
  entrada.name = "correo"; entrada.autocomplete = "email"; entrada.required = true; entrada.maxLength = 254;
  const ayuda = nodo("details", "contacto-ayuda");
  const abrirAyuda = nodo("summary", "", "?"); abrirAyuda.setAttribute("aria-label", t("ayudaTitulo"));
  ayuda.append(abrirAyuda, nodo("p", "", t("ayuda"))); campo.append(etiqueta, entrada, ayuda);
  const preparar = nodo("button", "boton-primario", t("preparar")); preparar.type = "submit";
  const confirmar = nodo("button", "boton-primario", t("confirmar")); confirmar.type = "button";
  const cancelar = nodo("button", "boton-peligro", t("cancelar")); cancelar.type = "button";
  const acciones = nodo("div", "fila-acciones"); acciones.append(preparar, confirmar, cancelar);
  const estado = nodo("p", "nota"); estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite"); estado.setAttribute("tabindex", "-1");
  const anterior = nodo("p", "nota", reciboAnterior?.reciboRef ? t("correctoAnterior", { recibo: reciboAnterior.reciboRef }) : "");
  anterior.hidden = !reciboAnterior?.reciboRef;
  const listaPanel = nodo("section", "contacto-operaciones-panel");
  const cabecera = nodo("div", "contacto-operaciones-cabecera");
  const titulo = nodo("h4", "", t("historialTitulo"));
  const actualizar = nodo("button", "boton-secundario", t("actualizarHistorial")); actualizar.type = "button";
  cabecera.append(titulo, actualizar);
  const lista = nodo("ul", "contacto-operaciones-lista");
  const mas = nodo("button", "boton-secundario", t("masOperaciones")); mas.type = "button";
  listaPanel.append(cabecera, lista, mas);
  formulario.append(campo, acciones, estado, anterior);
  contenedor.replaceChildren(formulario, listaPanel);
  let activa = true;
  let listaClave = "";
  const sincronizar = () => {
    if (!activa) return;
    const op = controlador.seleccion;
    const habilitado = controlador.autorizado && !controlador.ocupado;
    if (!controlador.autorizado) { entrada.value = ""; anterior.textContent = ""; anterior.hidden = true; }
    entrada.disabled = !habilitado;
    preparar.disabled = !habilitado || Boolean(controlador.operaciones.find((item) => item.estado === "preparada"));
    confirmar.hidden = op?.estado !== "preparada";
    cancelar.hidden = op?.estado !== "preparada";
    confirmar.disabled = !habilitado; cancelar.disabled = !habilitado;
    actualizar.disabled = !habilitado; mas.hidden = !controlador.siguienteDesde; mas.disabled = !habilitado;
    estado.textContent = controlador.autorizado ? controlador.aviso || t("seleccionExplicita") : t("sinAutorizacion");
    estado.className = `nota ${controlador.tipoAviso === "error" ? "error" : controlador.tipoAviso === "aviso" ? "aviso" : ""}`.trim();
    const items = controlador.operaciones;
    const claveActual = JSON.stringify([controlador.cargado, items]);
    if (claveActual !== listaClave) {
      const teniaFoco = lista.contains?.(doc.activeElement);
      listaClave = claveActual;
      lista.replaceChildren();
    for (const [indice, item] of items.entries()) {
      const li = nodo("li", `contacto-operacion ${item.estado}`);
      const boton = nodo("button", "boton-secundario", t(`estado.${item.estado}`));
      boton.type = "button"; boton.disabled = !habilitado;
      boton.setAttribute("aria-label", t("seleccionarOperacion", { estado: t(`estado.${item.estado}`), numero: indice + 1, referencia: item.operacion_ref }));
      boton.addEventListener("click", async () => { try { await controlador.seleccionar(item.operacion_ref); entrada.value = ""; } catch { /* Mensaje del controlador. */ } finally { if (activa) estado.focus(); } });
      const abreviada = `${item.operacion_ref.slice(0, 8)}…${item.operacion_ref.slice(-6)}`;
      const meta = nodo("small", "", t("operacionResumen", { numero: indice + 1, referencia: abreviada, version: item.version_esperada }));
      li.append(boton, meta); lista.append(li);
    }
    if (controlador.cargado && items.length === 0) {
      lista.append(nodo("li", "contacto-operacion-vacia", t("historialVacio")));
    }
      if (teniaFoco && activa) estado.focus();
    } else {
      lista.querySelectorAll?.("button").forEach((boton) => { boton.disabled = !habilitado; });
    }
  };
  const desuscribir = controlador.suscribir(sincronizar);
  const alPreparar = async (evento) => {
    evento.preventDefault();
    if (!entrada.checkValidity()) { estado.textContent = t("errorEntrada"); entrada.focus(); return; }
    try { await controlador.preparar(capturarCorreoEnviado(entrada)); estado.focus(); }
    catch { if (activa) estado.focus(); }
  };
  const alConfirmarClick = async () => {
    if (!entrada.checkValidity()) { estado.textContent = t("errorEntrada"); entrada.focus(); return; }
    const correo = capturarCorreoEnviado(entrada);
    try { const op = await controlador.confirmar(correo); if (op?.estado === "confirmada") alGuardar?.({ reciboRef: op.recibo_ref, version: op.version, correo }); }
    catch { if (activa) estado.focus(); }
  };
  const alCancelar = async () => { if (globalThis.confirm?.(t("confirmarCancelacion")) === false) return;
    try { await controlador.cancelar(); entrada.value = ""; estado.focus(); } catch { if (activa) estado.focus(); } };
  const alActualizar = async () => { try { await controlador.cargar(); } catch { if (activa) estado.focus(); } };
  const alMas = async () => { try { await controlador.cargar({ despuesDe: controlador.siguienteDesde }); } catch { if (activa) estado.focus(); } };
  formulario.addEventListener("submit", alPreparar); confirmar.addEventListener("click", alConfirmarClick);
  cancelar.addEventListener("click", alCancelar); actualizar.addEventListener("click", alActualizar); mas.addEventListener("click", alMas);
  sincronizar();
  if (controlador.autorizado && !controlador.cargado) void controlador.cargar().catch(() => {});
  return Object.freeze({ controlador, destruir() { activa = false; controlador.destruir(); desuscribir();
    formulario.removeEventListener("submit", alPreparar); confirmar.removeEventListener("click", alConfirmarClick);
    cancelar.removeEventListener("click", alCancelar); actualizar.removeEventListener("click", alActualizar); mas.removeEventListener("click", alMas); } });
}
