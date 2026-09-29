// Ficha propia de la persona aspirante en «Perfil y contacto».
// La identidad (nombre, apellidos y documento) viene del certificado y solo
// se muestra. El teléfono, el móvil, el domicilio y el código postal solo se
// piden si el catálogo de datos personales los pide. Los textos viven en
// locales/<idioma>.json (claves areaPersonal.ficha.*).
import { traducir } from "./i18n.js";

export const RUTA_MI_FICHA = "/api/vec/aspirantes/area-personal/mi-ficha";
export const CAMPOS_CONTACTO = Object.freeze(["telefono", "movil", "domicilio", "codigo_postal"]);
const MAXIMO_RESPUESTA = 64 * 1024;
const CODIGOS = Object.freeze({ 401: "no_autenticado", 403: "prohibido", 404: "sin_ficha", 409: "conflicto", 422: "peticion_invalida", 503: "no_disponible" });
const CODIGOS_CONOCIDOS = new Set(["no_autenticado", "prohibido", "sin_ficha", "ficha_existente", "conflicto", "peticion_invalida", "no_disponible"]);
const TIPOS_DOCUMENTO = new Set(["dni", "nie", "pasaporte", "otro"]);

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
    || !Array.isArray(v.exigencias) || v.exigencias.some((e) => !e || !CAMPOS_CONTACTO.includes(e.campo) || typeof e.obligatorio !== "boolean")
    || typeof v.catalogo_disponible !== "boolean" || typeof v.catalogo_ejemplo !== "boolean") {
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
    let envoltura = null;
    try {
      const bruto = await respuesta.text();
      if (bruto.length <= MAXIMO_RESPUESTA) envoltura = JSON.parse(bruto);
    } catch { envoltura = null; }
    if (respuesta.status === 200 || respuesta.status === 201) {
      if (!envoltura?.data || typeof envoltura.data !== "object") throw new ErrorFicha("no_disponible", respuesta.status);
      return envoltura.data;
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

function fechaLegible(fecha) {
  const d = new Date(fecha);
  if (Number.isNaN(d.getTime())) return "";
  const idioma = globalThis.document?.documentElement?.lang || "es";
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

const TIPOS_CAMPO = Object.freeze({
  telefono: { type: "tel", autocomplete: "tel", inputmode: "tel", maxlength: "20" },
  movil: { type: "tel", autocomplete: "tel", inputmode: "tel", maxlength: "20" },
  domicilio: { type: "text", autocomplete: "street-address", maxlength: "200" },
  codigo_postal: { type: "text", autocomplete: "postal-code", inputmode: "numeric", maxlength: "5", pattern: "[0-9]{5}" },
});

export function montarFichaAspirante({ contenedor, fetchImpl = globalThis.fetch, cliente = null, azar = globalThis.crypto } = {}) {
  if (!contenedor) return null;
  const documento = contenedor.ownerDocument ?? globalThis.document;
  const api = cliente ?? crearClienteFicha({ fetchImpl });
  let vista = null;
  let error = null;
  let aviso = null;
  let ocupado = false;
  let activo = true;
  let aborto = null;

  function pintar() {
    if (!activo) return;
    const hijos = [];
    if (aviso) hijos.push(nodo(documento, "p", { clase: "nota exito", role: "status", tabindex: "-1", "data-aviso-ficha": true, texto: aviso }));
    if (error) {
      hijos.push(nodo(documento, "div", { clase: "nota error", role: "alert" },
        nodo(documento, "p", { texto: t(`error.${error}`) }),
        nodo(documento, "button", { type: "button", clase: "boton-secundario", "data-accion-ficha": "reintentar", texto: t("accion.reintentar") })));
    } else if (!vista) {
      hijos.push(nodo(documento, "p", { role: "status", texto: t("cargando") }));
    } else {
      hijos.push(bloqueIdentidad(), bloqueContacto());
    }
    contenedor.replaceChildren(...hijos);
    const botonReintentar = contenedor.querySelector?.("[data-accion-ficha='reintentar']");
    botonReintentar?.addEventListener("click", () => cargar());
    const formulario = contenedor.querySelector?.("form[data-ficha]");
    formulario?.addEventListener("submit", enviar);
    formulario?.addEventListener("input", () => actualizarMotivo(formulario));
    contenedor.querySelectorAll?.("[data-quitar]").forEach((b) => b.addEventListener("click", () => quitar(b.getAttribute("data-quitar"))));
  }

  function bloqueIdentidad() {
    const id = vista.identidad;
    const filas = [["nombre", id.nombre], ["apellidos", id.apellidos], ["documento", `${t(`tipo.${id.tipo_documento}`)} ${id.documento}`]];
    const lista = nodo(documento, "dl", { clase: "dato-lista" });
    for (const [campo, valor] of filas) lista.append(nodo(documento, "dt", { texto: t(`campo.${campo}`) }), nodo(documento, "dd", { texto: valor }));
    return nodo(documento, "section", { clase: "ficha-identidad", "aria-labelledby": "ficha-identidad-titulo" },
      nodo(documento, "h4", { id: "ficha-identidad-titulo", texto: t("identidad.titulo") }), lista);
  }

  function bloqueContacto() {
    const seccion = nodo(documento, "section", { clase: "ficha-contacto", "aria-labelledby": "ficha-contacto-titulo" },
      nodo(documento, "h4", { id: "ficha-contacto-titulo", texto: t("contacto.titulo") }));
    const sinFicha = vista.estado === "sin_ficha";
    if (!vista.catalogo_disponible) {
      seccion.append(nodo(documento, "p", { clase: "nota aviso", texto: t("contacto.sinCatalogo") }));
      if (sinFicha) return seccion;
    }
    const pedidos = new Set(vista.exigencias.map((e) => e.campo));
    const formulario = nodo(documento, "form", { "data-ficha": vista.estado, novalidate: false });
    if (vista.catalogo_disponible && vista.exigencias.length === 0) seccion.append(nodo(documento, "p", { texto: t("contacto.nadaPedido") }));
    const rejilla = nodo(documento, "div", { clase: "formulario-rejilla" });
    for (const e of vista.exigencias) {
      const id = `ficha-${e.campo}`;
      const pista = e.condicion ? nodo(documento, "small", { id: `${id}-pista`, texto: t(`condicion.${e.condicion}`) }) : null;
      rejilla.append(nodo(documento, "div", { clase: e.campo === "domicilio" ? "campo ancho-completo" : "campo" },
        nodo(documento, "label", { for: id, texto: e.obligatorio ? t(`campo.${e.campo}`) : `${t(`campo.${e.campo}`)} ${t("opcional")}` }),
        nodo(documento, "input", { id, name: e.campo, ...TIPOS_CAMPO[e.campo], value: vista.contacto[e.campo] ?? "", required: e.obligatorio,
          "aria-describedby": pista ? `${id}-pista` : undefined }),
        pista));
    }
    if (vista.exigencias.length) formulario.append(rejilla);
    if (!sinFicha) {
      formulario.append(nodo(documento, "fieldset", { clase: "ficha-motivo", hidden: true, "data-motivo": true },
        nodo(documento, "legend", { texto: t("motivo.pregunta") }),
        ...["cambio_de_dato", "correccion_de_error"].map((m) => nodo(documento, "label", { clase: "opcion-check" },
          nodo(documento, "input", { type: "radio", name: "motivo", value: m }), nodo(documento, "span", { texto: t(`motivo.${m}`) })))));
    }
    if (vista.catalogo_ejemplo && vista.exigencias.length) formulario.append(nodo(documento, "p", { clase: "nota", texto: t("ejemplo") }));
    formulario.append(nodo(documento, "div", { clase: "fila-acciones" },
      nodo(documento, "button", { type: "submit", clase: "boton-primario", disabled: ocupado, texto: t(sinFicha ? "accion.crear" : "accion.guardar") })));
    // Sin campos pedidos solo queda el botón de crear la ficha.
    if (vista.exigencias.length || sinFicha) seccion.append(formulario);
    // Datos que ya no pide ninguna convocatoria: se pueden quitar.
    const sobrantes = CAMPOS_CONTACTO.filter((c) => !pedidos.has(c) && vista.contacto[c]);
    if (!sinFicha && sobrantes.length) {
      const lista = nodo(documento, "dl", { clase: "dato-lista" });
      for (const c of sobrantes) {
        lista.append(nodo(documento, "dt", { texto: t(`campo.${c}`) }), nodo(documento, "dd", {},
          nodo(documento, "span", { texto: vista.contacto[c] }), " ",
          nodo(documento, "button", { type: "button", clase: "boton-secundario", "data-quitar": c, disabled: ocupado, texto: t("accion.quitar") })));
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

  function actualizarMotivo(formulario) {
    const grupo = formulario.querySelector?.("[data-motivo]");
    if (!grupo) return;
    grupo.hidden = !cambiosDeContacto(valoresDe(formulario), vista.contacto).habiaValor;
  }

  async function operar(fn, textoExito) {
    ocupado = true;
    error = null;
    aviso = null;
    pintar();
    aborto?.abort();
    aborto = new AbortController();
    try {
      const recibo = await fn(aborto.signal);
      if (!activo) return;
      aviso = t(textoExito, { fecha: fechaLegible(recibo?.fecha_utc) });
      ocupado = false;
      await cargar({ conservarAviso: true });
      contenedor.querySelector?.("[data-aviso-ficha]")?.focus?.();
    } catch (e) {
      if (!activo || aborto?.signal.aborted) return;
      ocupado = false;
      const codigo = e?.codigo ?? "no_disponible";
      if (codigo === "conflicto" || codigo === "ficha_existente") {
        aviso = t(`error.${codigo}`);
        await cargar({ conservarAviso: true });
        return;
      }
      error = codigo;
      pintar();
    }
  }

  function enviar(evento) {
    evento.preventDefault();
    const formulario = evento.currentTarget ?? evento.target;
    if (ocupado || !vista || formulario?.checkValidity?.() === false) {
      formulario?.reportValidity?.();
      return;
    }
    const valores = valoresDe(formulario);
    if (vista.estado === "sin_ficha") {
      const campos = {};
      for (const [c, v] of Object.entries(valores)) if (String(v).trim() !== "") campos[c] = String(v).trim();
      void operar((signal) => api.alta({ clave_operacion: claveOperacion(azar), version_esperada: 0, campos }, signal), "creada");
      return;
    }
    const { cambios, habiaValor } = cambiosDeContacto(valores, vista.contacto);
    if (Object.keys(cambios).length === 0) {
      aviso = t("sinCambios");
      pintar();
      return;
    }
    let motivo = "dato_nuevo";
    if (habiaValor) {
      motivo = formulario.querySelector?.("input[name='motivo']:checked")?.value ?? "";
      if (!motivo) {
        const grupo = formulario.querySelector?.("[data-motivo]");
        if (grupo) grupo.hidden = false;
        formulario.querySelector?.("input[name='motivo']")?.focus?.();
        return;
      }
    }
    void operar((signal) => api.rectificar({ clave_operacion: claveOperacion(azar), version_esperada: vista.version, motivo, campos: cambios }, signal), "guardado");
  }

  function quitar(campo) {
    if (ocupado || !vista || !CAMPOS_CONTACTO.includes(campo)) return;
    void operar((signal) => api.rectificar({ clave_operacion: claveOperacion(azar), version_esperada: vista.version, motivo: "cambio_de_dato", campos: { [campo]: "" } }, signal), "guardado");
  }

  async function cargar({ conservarAviso = false } = {}) {
    if (!conservarAviso) aviso = null;
    error = null;
    vista = null;
    pintar();
    const controlador = new AbortController();
    aborto?.abort();
    aborto = controlador;
    try {
      vista = await api.consultar(controlador.signal);
    } catch (e) {
      if (!activo || controlador.signal.aborted) return;
      error = e?.codigo ?? "no_disponible";
    }
    pintar();
  }

  void cargar();
  return Object.freeze({
    recargar: () => cargar(),
    destruir() { activo = false; aborto?.abort(); },
  });
}
