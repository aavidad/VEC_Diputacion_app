// Ficha propia de la persona aspirante en «Perfil y contacto».
// La identidad (nombre, apellidos y documento) viene del certificado y solo
// se muestra. El teléfono, el móvil, el domicilio y el código postal solo se
// piden si el catálogo de datos personales los pide. Los textos viven en
// textos/<idioma>/area-personal.json (claves areaPersonal.ficha.*).
import { idiomaAreaPersonal, traducir } from "./i18n.js";

export const RUTA_MI_FICHA = "/api/vec/aspirantes/area-personal/mi-ficha";
export const CAMPOS_CONTACTO = Object.freeze(["telefono", "movil", "domicilio", "codigo_postal"]);
const MAXIMO_RESPUESTA = 64 * 1024;
const CODIGOS = Object.freeze({ 401: "no_autenticado", 403: "prohibido", 404: "sin_ficha", 409: "conflicto", 422: "peticion_invalida", 503: "no_disponible" });
const CODIGOS_CONOCIDOS = new Set(["no_autenticado", "prohibido", "sin_ficha", "ficha_existente", "conflicto", "peticion_invalida", "no_disponible"]);
const TIPOS_DOCUMENTO = new Set(["dni", "nie", "pasaporte", "otro"]);
const CONDICION = /^[a-z0-9_]{1,64}$/u;
// Clases del tema para los avisos (tipo de nota → clase CSS).
const CLASES_NOTA = Object.freeze({ error: "nota error", aviso: "nota aviso", exito: "nota exito" });

const t = (clave, valores) => traducir(`areaPersonal.ficha.${clave}`, valores);

export class ErrorFicha extends Error {
  constructor(codigo, estado = 0) {
    super(codigo);
    this.name = "ErrorFicha";
    this.codigo = CODIGOS_CONOCIDOS.has(codigo) ? codigo : "no_disponible";
    this.estado = estado;
  }
}

function texto(v, maximo = 200) { return typeof v === "string" && v.length <= maximo; }

// validarVista rechaza cualquier forma distinta de la que publica la API.
export function validarVista(v) {
  if (!v || typeof v !== "object" || Array.isArray(v) || !["sin_ficha", "activa"].includes(v.estado)
    || !Number.isSafeInteger(v.version) || v.version < 0 || (v.estado === "activa") !== (v.version > 0)
    || !v.identidad || typeof v.identidad !== "object" || !texto(v.identidad.nombre) || !texto(v.identidad.apellidos)
    || !TIPOS_DOCUMENTO.has(v.identidad.tipo_documento) || !texto(v.identidad.documento, 40)
    || !v.contacto || typeof v.contacto !== "object" || Array.isArray(v.contacto)
    || Object.entries(v.contacto).some(([campo, valor]) => !CAMPOS_CONTACTO.includes(campo) || !texto(valor, 400))
    || !Array.isArray(v.exigencias) || v.exigencias.some((e) => !e || !CAMPOS_CONTACTO.includes(e.campo) || typeof e.obligatorio !== "boolean"
      || (e.condicion !== undefined && (typeof e.condicion !== "string" || !CONDICION.test(e.condicion))))
    || typeof v.catalogo_disponible !== "boolean" || typeof v.catalogo_ejemplo !== "boolean") {
    throw new ErrorFicha("no_disponible");
  }
  return v;
}

function validarRecibo(v, versionEsperada) {
  if (!v || typeof v !== "object" || Array.isArray(v)
    || !/^asprec_[0-9a-f]{32}$/u.test(v.recibo_ref)
    || !Number.isSafeInteger(v.version) || v.version !== versionEsperada + 1
    || typeof v.fecha_utc !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/u.test(v.fecha_utc)
    || Number.isNaN(Date.parse(v.fecha_utc)) || typeof v.replay !== "boolean") {
    throw new ErrorFicha("no_disponible");
  }
  return v;
}

export function crearClienteFicha({ fetchImpl = globalThis.fetch } = {}) {
  async function solicitar(metodo, cuerpo, signal) {
    let respuesta;
    try {
      respuesta = await fetchImpl(RUTA_MI_FICHA, {
        method: metodo, credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal,
        headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
        ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
      });
    } catch (error) {
      if (signal?.aborted) throw error;
      throw new ErrorFicha("no_disponible");
    }
    // Solo se lee JSON acotado: un proxy que devuelva otra cosa es «no disponible».
    const tipo = String(respuesta.headers?.get?.("Content-Type") ?? "");
    const largo = Number(respuesta.headers?.get?.("Content-Length") ?? 0);
    let envoltura = null;
    if (tipo.startsWith("application/json") && largo <= MAXIMO_RESPUESTA) {
      try {
        const bruto = await respuesta.text();
        if (bruto.length <= MAXIMO_RESPUESTA) envoltura = JSON.parse(bruto);
      } catch { envoltura = null; }
    }
    if (respuesta.status === 200 || respuesta.status === 201) {
      if (!envoltura?.data || typeof envoltura.data !== "object") throw new ErrorFicha("no_disponible", respuesta.status);
      return metodo === "POST" ? validarRecibo(envoltura.data, cuerpo.version_esperada) : envoltura.data;
    }
    throw new ErrorFicha(envoltura?.error?.codigo ?? CODIGOS[respuesta.status], respuesta.status);
  }
  return Object.freeze({
    async consultar(signal) { return validarVista(await solicitar("GET", null, signal)); },
    alta(peticion, signal) { return solicitar("POST", { operacion: "alta", ...peticion }, signal); },
    rectificar(peticion, signal) { return solicitar("POST", { operacion: "rectificar", ...peticion }, signal); },
  });
}

export function claveOperacion(azar = globalThis.crypto) {
  return `ficha-${azar.randomUUID()}`;
}

// cambiosDeContacto compara lo escrito con lo guardado. Devuelve los campos
// cambiados y si alguno ya tenía valor (entonces hay que preguntar el motivo).
export function cambiosDeContacto(valoresFormulario, guardados) {
  const cambios = {};
  let habiaValor = false;
  for (const campo of CAMPOS_CONTACTO) {
    if (!Object.hasOwn(valoresFormulario, campo)) continue;
    const nuevo = String(valoresFormulario[campo] ?? "").trim();
    const anterior = String(guardados?.[campo] ?? "").trim();
    if (nuevo === anterior) continue;
    cambios[campo] = nuevo;
    if (anterior !== "") habiaValor = true;
  }
  return { cambios, habiaValor };
}

// errorDeCampo es una comprobación previa y amable; el servidor decide.
export function errorDeCampo(campo, valor, obligatorio) {
  const v = String(valor ?? "").trim();
  if (v === "") return obligatorio ? `validacion.vacio.${campo}` : "";
  if (campo === "telefono" || campo === "movil") {
    const cifras = v.replace(/\D/gu, "").length;
    return /^\+?[0-9 ().-]+$/u.test(v) && cifras >= 9 && cifras <= 15 ? "" : `validacion.${campo}`;
  }
  if (campo === "codigo_postal") return /^[0-9]{5}$/u.test(v) ? "" : "validacion.codigo_postal";
  if (campo === "domicilio") return v.length >= 5 && v.length <= 200 ? "" : "validacion.domicilio";
  return "";
}

function fechaLegible(fecha) {
  const d = new Date(fecha);
  if (Number.isNaN(d.getTime())) return "";
  const idioma = globalThis.document?.documentElement?.lang || idiomaAreaPersonal();
  return new Intl.DateTimeFormat(idioma, { dateStyle: "long", timeStyle: "short", timeZone: "Europe/Madrid" }).format(d);
}

function nodo(documento, etiqueta, atributos = {}, ...hijos) {
  const n = documento.createElement(etiqueta);
  for (const [clave, valor] of Object.entries(atributos)) {
    if (valor === false || valor === undefined || valor === null) continue;
    if (clave === "texto") n.textContent = valor;
    else if (clave === "clase") n.className = valor;
    else n.setAttribute(clave, valor === true ? "" : String(valor));
  }
  n.append(...hijos.filter(Boolean));
  return n;
}

// Patrón del atributo HTML (se compila con la bandera v): cifras, espacios,
// guion, punto, paréntesis y un + inicial, como acepta el servidor.
const PATRON_TELEFONO = String.raw`\+?[0-9 \-\.\(\)]{9,20}`;
const TIPOS_CAMPO = Object.freeze({
  telefono: { type: "tel", autocomplete: "tel", inputmode: "tel", maxlength: "20", pattern: PATRON_TELEFONO },
  movil: { type: "tel", autocomplete: "tel", inputmode: "tel", maxlength: "20", pattern: PATRON_TELEFONO },
  domicilio: { type: "text", autocomplete: "street-address", maxlength: "200" },
  codigo_postal: { type: "text", autocomplete: "postal-code", inputmode: "numeric", maxlength: "5", pattern: "[0-9]{5}" },
});

export function montarFichaAspirante({ contenedor, fetchImpl = globalThis.fetch, cliente = null, azar = globalThis.crypto } = {}) {
  if (!contenedor) return null;
  const documento = contenedor.ownerDocument ?? globalThis.document;
  const api = cliente ?? crearClienteFicha({ fetchImpl });
  let vista = null;
  let errorCarga = null;
  let errorGuardar = null;
  let aviso = null;
  let borrador = null;
  let erroresCampo = {};
  let errorMotivo = false;
  let confirmandoQuitar = null;
  let ocupado = false;
  let operacionPendiente = null;
  let activo = true;
  let aborto = null;

  function pintar() {
    if (!activo) return;
    const hijos = [];
    if (aviso) hijos.push(nodo(documento, "p", { clase: CLASES_NOTA[aviso.tipo], role: "status", tabindex: "-1", "data-aviso-ficha": true, texto: aviso.texto }));
    if (errorCarga) {
      hijos.push(nodo(documento, "div", { clase: CLASES_NOTA.error, role: "alert", tabindex: "-1", "data-error-ficha": true },
        nodo(documento, "p", { texto: t(`error.${errorCarga}`) }),
        nodo(documento, "button", { type: "button", clase: "boton-secundario", "data-accion-ficha": "reintentar", texto: t("accion.reintentar") })));
    } else if (!vista) {
      hijos.push(nodo(documento, "p", { role: "status", texto: t("cargando") }));
    } else {
      hijos.push(bloqueIdentidad(), bloqueContacto());
    }
    contenedor.replaceChildren(...hijos);
    contenedor.querySelector?.("[data-accion-ficha='reintentar']")?.addEventListener("click", () => cargar());
    const formulario = contenedor.querySelector?.("form[data-ficha]");
    formulario?.addEventListener("submit", enviar);
    formulario?.addEventListener("input", () => actualizarMotivo(formulario));
    contenedor.querySelectorAll?.("[data-quitar]").forEach((b) => b.addEventListener("click", () => pedirQuitar(b.getAttribute("data-quitar"))));
    contenedor.querySelector?.("[data-confirmar-quitar]")?.addEventListener("click", () => quitar(confirmandoQuitar));
    contenedor.querySelector?.("[data-cancelar-quitar]")?.addEventListener("click", () => {
      conservarBorradorFormulario();
      const campo = confirmandoQuitar;
      confirmandoQuitar = null;
      pintar();
      enfocar(`[data-quitar='${campo}']`);
    });
  }

  function bloqueIdentidad() {
    const id = vista.identidad;
    const filas = [["nombre", id.nombre], ["apellidos", id.apellidos], ["documento", `${t(`tipo.${id.tipo_documento}`)} ${id.documento}`]];
    const lista = nodo(documento, "dl", { clase: "dato-lista" });
    for (const [campo, valor] of filas) lista.append(nodo(documento, "dt", { texto: t(`campo.${campo}`) }), nodo(documento, "dd", { texto: valor }));
    return nodo(documento, "section", { clase: "ficha-identidad", "aria-labelledby": "ficha-identidad-titulo" },
      nodo(documento, "h4", { id: "ficha-identidad-titulo", texto: t("identidad.titulo") }), lista);
  }

  function campoFormulario(e) {
    const id = `ficha-${e.campo}`;
    const errorClave = erroresCampo[e.campo];
    const valor = borrador && Object.hasOwn(borrador, e.campo) ? borrador[e.campo] : (vista.contacto[e.campo] ?? "");
    const pista = e.condicion ? nodo(documento, "small", { id: `${id}-pista`, texto: t(`condicion.${e.condicion}`) }) : null;
    const error = errorClave ? nodo(documento, "p", { id: `${id}-error`, clase: "error-campo", texto: t(errorClave) }) : null;
    const descrito = [pista && `${id}-pista`, error && `${id}-error`].filter(Boolean).join(" ") || undefined;
    return nodo(documento, "div", { clase: e.campo === "domicilio" ? "campo ancho-completo" : "campo" },
      nodo(documento, "label", { for: id, texto: e.obligatorio ? t(`campo.${e.campo}`) : `${t(`campo.${e.campo}`)} ${t("opcional")}` }),
      nodo(documento, "input", { id, name: e.campo, ...TIPOS_CAMPO[e.campo], value: valor, required: e.obligatorio,
        "aria-invalid": errorClave ? "true" : undefined, "aria-describedby": descrito }),
      pista, error);
  }

  function bloqueContacto() {
    const seccion = nodo(documento, "section", { clase: "ficha-contacto", "aria-labelledby": "ficha-contacto-titulo" },
      nodo(documento, "h4", { id: "ficha-contacto-titulo", texto: t("contacto.titulo") }));
    const sinFicha = vista.estado === "sin_ficha";
    if (!vista.catalogo_disponible) {
      seccion.append(nodo(documento, "p", { clase: CLASES_NOTA.aviso, texto: t("contacto.sinCatalogo") }));
      if (sinFicha) return seccion;
    } else if (vista.exigencias.length === 0) {
      seccion.append(nodo(documento, "p", { texto: t("contacto.nadaPedido") }));
    }
    const formulario = nodo(documento, "form", { "data-ficha": vista.estado, novalidate: true });
    if (errorGuardar) {
      formulario.append(nodo(documento, "p", { clase: CLASES_NOTA.error, role: "alert", tabindex: "-1", "data-error-guardar": true, texto: t(`error.${errorGuardar}`) }));
    } else if (Object.keys(erroresCampo).length) {
      formulario.append(nodo(documento, "p", { clase: CLASES_NOTA.error, role: "alert", tabindex: "-1", "data-error-guardar": true, texto: t("resumen") }));
    }
    if (vista.exigencias.length) formulario.append(nodo(documento, "div", { clase: "formulario-rejilla" }, ...vista.exigencias.map(campoFormulario)));
    if (!sinFicha) {
      const habiaValor = borrador ? cambiosDeContacto(borrador, vista.contacto).habiaValor : false;
      const grupo = nodo(documento, "fieldset", { clase: "ficha-motivo", hidden: !habiaValor && !errorMotivo, "data-motivo": true,
        "aria-describedby": errorMotivo ? "ficha-motivo-error" : undefined },
      nodo(documento, "legend", { texto: t("motivo.pregunta") }),
      errorMotivo ? nodo(documento, "p", { id: "ficha-motivo-error", clase: "error-campo", texto: t("error.motivo") }) : null,
      ...["cambio_de_dato", "correccion_de_error"].map((m, i) => nodo(documento, "label", { clase: "opcion-check" },
        nodo(documento, "input", { type: "radio", name: "motivo", value: m, required: i === 0, checked: borrador?.motivo === m }),
        nodo(documento, "span", { texto: t(`motivo.${m}`) }))));
      formulario.append(grupo);
    }
    if (vista.catalogo_ejemplo && vista.exigencias.length) formulario.append(nodo(documento, "p", { clase: "nota", texto: t("ejemplo") }));
    formulario.append(nodo(documento, "div", { clase: "fila-acciones" },
      nodo(documento, "button", { type: "submit", clase: "boton-primario", disabled: ocupado, "aria-busy": ocupado ? "true" : undefined,
        texto: t(sinFicha ? "accion.crear" : "accion.guardar") })));
    // Sin campos pedidos solo queda el botón de crear la ficha.
    if (vista.exigencias.length || sinFicha) seccion.append(formulario);
    // Datos que ya no pide ninguna convocatoria: se pueden quitar, tras confirmar.
    const pedidos = new Set(vista.exigencias.map((e) => e.campo));
    const sobrantes = CAMPOS_CONTACTO.filter((c) => !pedidos.has(c) && vista.contacto[c]);
    if (!sinFicha && sobrantes.length) {
      const lista = nodo(documento, "dl", { clase: "dato-lista" });
      for (const c of sobrantes) {
        const acciones = confirmandoQuitar === c
          ? nodo(documento, "div", { clase: CLASES_NOTA.aviso, role: "group", "aria-labelledby": `ficha-quitar-${c}` },
            nodo(documento, "p", { id: `ficha-quitar-${c}`, tabindex: "-1", "data-pregunta-quitar": true, texto: t("quitar.pregunta", { campo: t(`campo.${c}`) }) }),
            nodo(documento, "button", { type: "button", clase: "boton-secundario", "data-confirmar-quitar": true, disabled: ocupado, texto: t("accion.confirmarQuitar") }), " ",
            nodo(documento, "button", { type: "button", clase: "boton-secundario", "data-cancelar-quitar": true, texto: t("accion.cancelar") }))
          : nodo(documento, "button", { type: "button", clase: "boton-secundario", "data-quitar": c, disabled: ocupado, texto: t("accion.quitar") });
        lista.append(nodo(documento, "dt", { texto: t(`campo.${c}`) }), nodo(documento, "dd", {},
          nodo(documento, "span", { texto: vista.contacto[c] }), " ", acciones));
      }
      seccion.append(nodo(documento, "h5", { texto: t("sobrantes.titulo") }), lista);
    }
    return seccion;
  }

  function valoresDe(formulario) {
    const valores = {};
    for (const campo of CAMPOS_CONTACTO) {
      const entrada = formulario.elements?.namedItem?.(campo);
      if (entrada) valores[campo] = entrada.value;
    }
    return valores;
  }

  function conservarBorradorFormulario() {
    const formulario = contenedor.querySelector?.("form[data-ficha]");
    if (formulario) borrador = {
      ...valoresDe(formulario),
      motivo: formulario.querySelector?.("input[name='motivo']:checked")?.value ?? "",
    };
  }

  function peticionConClave(operacion, version_esperada, campos, motivo) {
    const peticion = { operacion, version_esperada, campos, ...(motivo ? { motivo } : {}) };
    const firma = JSON.stringify(peticion);
    if (operacionPendiente?.firma !== firma) operacionPendiente = { firma, clave: claveOperacion(azar) };
    return { clave_operacion: operacionPendiente.clave, version_esperada, campos, ...(motivo ? { motivo } : {}) };
  }

  function actualizarMotivo(formulario) {
    const grupo = formulario.querySelector?.("[data-motivo]");
    if (!grupo) return;
    grupo.hidden = !errorMotivo && !cambiosDeContacto(valoresDe(formulario), vista.contacto).habiaValor;
  }

  // enfocar lleva el foco al primer elemento que coincida, tras pintar.
  function enfocar(...selectores) {
    for (const s of selectores) {
      const n = contenedor.querySelector?.(s);
      if (n) { n.focus?.(); return; }
    }
  }

  function marcarOcupado(valor) {
    ocupado = valor;
    const boton = contenedor.querySelector?.("form[data-ficha]")?.querySelector?.("button[type='submit']");
    if (!boton) return;
    if (valor) { boton.setAttribute("disabled", ""); boton.setAttribute("aria-busy", "true"); }
    else { boton.removeAttribute?.("disabled"); boton.removeAttribute?.("aria-busy"); }
  }

  // operar no repinta al empezar: lo escrito sigue en el formulario hasta que
  // el servicio confirma. Un error de guardado conserva el borrador.
  async function operar(fn, textoExito, { conservarBorrador = false } = {}) {
    errorGuardar = null;
    aviso = null;
    marcarOcupado(true);
    aborto?.abort();
    aborto = new AbortController();
    const propio = aborto;
    try {
      const recibo = await fn(propio.signal);
      if (!activo) return;
      confirmandoQuitar = null;
      ocupado = false;
      await cargar();
      if (activo && !errorCarga && vista?.estado === "activa" && vista.version >= recibo.version) {
        if (!conservarBorrador) borrador = null;
        operacionPendiente = null;
        aviso = { tipo: "exito", texto: t(textoExito, { fecha: fechaLegible(recibo.fecha_utc) }) };
        pintar();
        enfocar("[data-aviso-ficha]");
      } else if (activo && !errorCarga) {
        errorGuardar = "guardar";
        pintar();
        enfocar("[data-error-guardar]");
      }
    } catch (e) {
      if (!activo || propio.signal.aborted) return;
      ocupado = false;
      const codigo = e?.codigo ?? "no_disponible";
      if (codigo === "conflicto" || codigo === "ficha_existente") {
        operacionPendiente = null;
        borrador = null;
        aviso = { tipo: "aviso", texto: t(`error.${codigo}`) };
        await cargar({ conservarAviso: true });
        enfocar("[data-aviso-ficha]");
        return;
      }
      if (codigo !== "no_disponible") operacionPendiente = null;
      errorGuardar = codigo === "no_disponible" ? "guardar" : codigo;
      pintar();
      enfocar("[data-error-guardar]");
    }
  }

  function enviar(evento) {
    evento.preventDefault();
    const formulario = evento.currentTarget ?? evento.target;
    if (ocupado || !vista) return;
    const valores = valoresDe(formulario);
    const motivoElegido = formulario.querySelector?.("input[name='motivo']:checked")?.value ?? "";
    borrador = { ...valores, motivo: motivoElegido };
    errorGuardar = null;
    errorMotivo = false;
    erroresCampo = {};
    for (const e of vista.exigencias) {
      const clave = errorDeCampo(e.campo, valores[e.campo], e.obligatorio);
      if (clave) erroresCampo[e.campo] = clave;
    }
    const primero = vista.exigencias.find((e) => erroresCampo[e.campo]);
    if (primero) {
      pintar();
      enfocar(`input[id='ficha-${primero.campo}']`);
      return;
    }
    if (vista.estado === "sin_ficha") {
      const campos = {};
      for (const [c, v] of Object.entries(valores)) if (String(v).trim() !== "") campos[c] = String(v).trim();
      const peticion = peticionConClave("alta", 0, campos);
      void operar((signal) => api.alta(peticion, signal), "creada");
      return;
    }
    const { cambios, habiaValor } = cambiosDeContacto(valores, vista.contacto);
    if (Object.keys(cambios).length === 0) {
      aviso = { tipo: "aviso", texto: t("sinCambios") };
      pintar();
      enfocar("[data-aviso-ficha]");
      return;
    }
    let motivo = "dato_nuevo";
    if (habiaValor) {
      motivo = motivoElegido;
      if (!motivo) {
        errorMotivo = true;
        pintar();
        enfocar("input[name='motivo']");
        return;
      }
    }
    const peticion = peticionConClave("rectificar", vista.version, cambios, motivo);
    void operar((signal) => api.rectificar(peticion, signal), "guardado");
  }

  function pedirQuitar(campo) {
    if (ocupado || !CAMPOS_CONTACTO.includes(campo)) return;
    conservarBorradorFormulario();
    confirmandoQuitar = campo;
    pintar();
    enfocar("[data-pregunta-quitar]");
  }

  function quitar(campo) {
    if (ocupado || !vista || !CAMPOS_CONTACTO.includes(campo)) return;
    conservarBorradorFormulario();
    const peticion = peticionConClave("rectificar", vista.version, { [campo]: "" }, "cambio_de_dato");
    void operar((signal) => api.rectificar(peticion, signal), "guardado", { conservarBorrador: true });
  }

  async function cargar({ conservarAviso = false } = {}) {
    if (!conservarAviso) aviso = null;
    errorCarga = null;
    errorGuardar = null;
    erroresCampo = {};
    errorMotivo = false;
    vista = null;
    pintar();
    const controlador = new AbortController();
    aborto?.abort();
    aborto = controlador;
    try {
      vista = await api.consultar(controlador.signal);
    } catch (e) {
      if (!activo || controlador.signal.aborted) return;
      errorCarga = e?.codigo ?? "no_disponible";
    }
    pintar();
    if (errorCarga) enfocar("[data-error-ficha]");
  }

  void cargar();
  return Object.freeze({
    recargar: () => cargar(),
    destruir() { activo = false; aborto?.abort(); },
  });
}
