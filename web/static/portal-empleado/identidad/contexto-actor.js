/**
 * Proyeccion frontend, inmutable y no autoritativa, del ContextoActor canonico.
 *
 * El dominio y la autoridad permanecen en internal/vec/domain/contexto_actor.go.
 * Esta proyeccion solo transporta las referencias opacas y el contexto visible
 * que necesitan los modulos internos. Nunca concede permisos ni sustituye la
 * comprobacion del backend.
 */

export const ESQUEMA_CONTEXTO_ACTOR_FRONTEND = "vec.identidad.contexto-actor.frontend.v1";

const MAXIMO_MODULOS_AMBITO = 128;
const METODOS_AUTENTICACION = new Set(["demo", "kerberos_ad", "certificado"]);
const GARANTIAS_AUTENTICACION = new Set(["bajo", "sustancial", "alto"]);
const CONTEXTOS_VALIDADOS = new WeakSet();

const CAMPOS_CONTEXTO = Object.freeze([
  "esquema", "revision", "demostracion", "persona_ref", "cuenta_ref",
  "perfil_ref", "actor", "rol", "ambito", "autenticacion", "resuelto_en",
]);
const CAMPOS_ACTOR = Object.freeze(["actor_ref", "nombre_visible", "iniciales"]);
const CAMPOS_ROL = Object.freeze(["clave", "etiqueta"]);
const CAMPOS_AMBITO = Object.freeze([
  "clase", "organizacion_ref", "unidad_ref", "modulos",
]);
const CAMPOS_AUTENTICACION = Object.freeze(["sesion_ref", "metodo", "garantia"]);

function esObjetoPlano(valor) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) return false;
  const prototipo = Object.getPrototypeOf(valor);
  return prototipo === Object.prototype || prototipo === null;
}

function exigirCamposExactos(objeto, campos, nombre) {
  if (!esObjetoPlano(objeto)) throw new TypeError(`${nombre} no valido`);
  const esperados = new Set(campos);
  if (Object.keys(objeto).length !== campos.length
    || Object.keys(objeto).some((campo) => !esperados.has(campo))
    || campos.some((campo) => !Object.hasOwn(objeto, campo))) {
    throw new TypeError(`${nombre} no respeta el contrato cerrado`);
  }
}

function exigirCadena(valor, nombre, maximo = 256) {
  if (typeof valor !== "string" || valor.length === 0 || valor.length > maximo
    || valor !== valor.trim() || /[\u0000-\u001f\u007f-\u009f]/u.test(valor)) {
    throw new TypeError(`${nombre} no valido`);
  }
  return valor;
}

function exigirReferenciaOpaca(valor, prefijo, nombre) {
  const referencia = exigirCadena(valor, nombre, 160);
  const cuerpo = referencia.slice(prefijo.length);
  if (!referencia.startsWith(prefijo) || cuerpo.length < 22 || cuerpo.length > 128
    || !/^[A-Za-z0-9_-]+$/u.test(cuerpo)) {
    throw new TypeError(`${nombre} no valida`);
  }
  return referencia;
}

function exigirActorRef(valor, demostracion) {
  const referencia = exigirCadena(valor, "actor.actor_ref", 160);
  if (demostracion && /^DEMO-PERFIL-[A-Z0-9-]{2,120}$/u.test(referencia)) return referencia;
  if (["per_", "prf_", "act_"].some((prefijo) => {
    try {
      exigirReferenciaOpaca(referencia, prefijo, "actor.actor_ref");
      return true;
    } catch {
      return false;
    }
  })) return referencia;
  throw new TypeError("actor.actor_ref no valida");
}

function exigirClave(valor, nombre) {
  const clave = exigirCadena(valor, nombre, 80);
  if (!/^[a-z][a-z0-9_.-]{1,79}$/u.test(clave)) throw new TypeError(`${nombre} no valida`);
  return clave;
}

function exigirInstanteUTC(valor, nombre) {
  const instante = exigirCadena(valor, nombre, 40);
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/u.test(instante)
    || !Number.isFinite(Date.parse(instante))) {
    throw new TypeError(`${nombre} no valido`);
  }
  return instante;
}

function copiarYValidarModulos(modulos) {
  if (!Array.isArray(modulos) || modulos.length === 0 || modulos.length > MAXIMO_MODULOS_AMBITO) {
    throw new TypeError("ambito.modulos no valido");
  }
  const copia = [];
  const unicos = new Set();
  for (const modulo of modulos) {
    if (typeof modulo !== "string" || !/^[a-z][a-z0-9_.-]{1,79}$/u.test(modulo)
      || unicos.has(modulo)) {
      throw new TypeError("ambito.modulos no valido");
    }
    unicos.add(modulo);
    copia.push(modulo);
  }
  return copia;
}

function congelarProfundo(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelarProfundo);
    Object.freeze(valor);
  }
  return valor;
}

/**
 * Valida una proyeccion recibida desde un adaptador y devuelve una copia
 * defensiva profundamente congelada. La validacion es estructural; la
 * autenticacion y la autorizacion se verifican siempre en el servidor.
 */
export function validarYCongelarContextoActor(datos) {
  exigirCamposExactos(datos, CAMPOS_CONTEXTO, "contexto de actor");
  exigirCamposExactos(datos.actor, CAMPOS_ACTOR, "actor");
  exigirCamposExactos(datos.rol, CAMPOS_ROL, "rol");
  exigirCamposExactos(datos.ambito, CAMPOS_AMBITO, "ambito");
  exigirCamposExactos(datos.autenticacion, CAMPOS_AUTENTICACION, "autenticacion");

  if (datos.esquema !== ESQUEMA_CONTEXTO_ACTOR_FRONTEND
    || !Number.isSafeInteger(datos.revision) || datos.revision < 1
    || typeof datos.demostracion !== "boolean") {
    throw new TypeError("contexto de actor no valido");
  }

  const demostracion = datos.demostracion;
  const metodo = exigirClave(datos.autenticacion.metodo, "autenticacion.metodo");
  const garantia = exigirClave(datos.autenticacion.garantia, "autenticacion.garantia");
  if (!METODOS_AUTENTICACION.has(metodo) || !GARANTIAS_AUTENTICACION.has(garantia)
    || (demostracion && metodo !== "demo") || (!demostracion && metodo === "demo")) {
    throw new TypeError("autenticacion no valida para el contexto");
  }
  if (datos.ambito.clase !== "personal_interno") {
    throw new TypeError("ambito.clase no valida");
  }

  const resultado = {
    esquema: datos.esquema,
    revision: datos.revision,
    demostracion,
    persona_ref: exigirReferenciaOpaca(datos.persona_ref, "per_", "persona_ref"),
    cuenta_ref: exigirReferenciaOpaca(datos.cuenta_ref, "cta_", "cuenta_ref"),
    perfil_ref: exigirReferenciaOpaca(datos.perfil_ref, "prf_", "perfil_ref"),
    actor: {
      actor_ref: exigirActorRef(datos.actor.actor_ref, demostracion),
      nombre_visible: exigirCadena(datos.actor.nombre_visible, "actor.nombre_visible", 160),
      iniciales: exigirCadena(datos.actor.iniciales, "actor.iniciales", 4),
    },
    rol: {
      clave: exigirClave(datos.rol.clave, "rol.clave"),
      etiqueta: exigirCadena(datos.rol.etiqueta, "rol.etiqueta", 200),
    },
    ambito: {
      clase: datos.ambito.clase,
      organizacion_ref: exigirReferenciaOpaca(
        datos.ambito.organizacion_ref, "org_", "ambito.organizacion_ref",
      ),
      unidad_ref: exigirReferenciaOpaca(datos.ambito.unidad_ref, "uni_", "ambito.unidad_ref"),
      modulos: copiarYValidarModulos(datos.ambito.modulos),
    },
    autenticacion: {
      sesion_ref: exigirReferenciaOpaca(datos.autenticacion.sesion_ref, "ses_", "autenticacion.sesion_ref"),
      metodo,
      garantia,
    },
    resuelto_en: exigirInstanteUTC(datos.resuelto_en, "resuelto_en"),
  };
  congelarProfundo(resultado);
  CONTEXTOS_VALIDADOS.add(resultado);
  return resultado;
}

/**
 * Comprueba el ambito sin derivar otra identidad. Todos los modulos reciben la
 * misma instancia inmutable, de modo que nunca aparecen usuarios por modulo.
 */
export function exigirContextoParaModulo(contexto, modulo) {
  if (!CONTEXTOS_VALIDADOS.has(contexto) || !Object.isFrozen(contexto)
    || !Object.isFrozen(contexto?.ambito)
    || !Object.isFrozen(contexto?.ambito?.modulos)) {
    throw new TypeError("el contexto debe estar validado e inmutable");
  }
  const claveModulo = exigirClave(modulo, "modulo");
  if (!contexto.ambito.modulos.includes(claveModulo)) {
    throw new Error("modulo fuera del ambito de la identidad interna");
  }
  return contexto;
}

/**
 * Punto de composicion comun. Conserva por referencia un unico ContextoActor y
 * no realiza E/S. Un adaptador de sesion productivo puede implementar la misma
 * funcion obtenerContexto tras resolver ContextoActor en el backend.
 */
export function crearProveedorContextoActorFijo(contexto) {
  const contextoValidado = CONTEXTOS_VALIDADOS.has(contexto)
    ? contexto : validarYCongelarContextoActor(contexto);
  return Object.freeze({
    obtenerContexto() {
      return contextoValidado;
    },
  });
}

/**
 * Entrega expresamente el mismo objeto a cada consumidor autorizado.
 */
export function compartirContextoActor(proveedor, modulos) {
  if (!proveedor || typeof proveedor.obtenerContexto !== "function") {
    throw new TypeError("proveedor de contexto no valido");
  }
  if (!Array.isArray(modulos) || modulos.length === 0) {
    throw new TypeError("modulos consumidores no validos");
  }
  const contexto = proveedor.obtenerContexto();
  const resultado = {};
  const vistos = new Set();
  for (const modulo of modulos) {
    if (vistos.has(modulo)) throw new TypeError("modulo consumidor repetido");
    vistos.add(modulo);
    resultado[modulo] = exigirContextoParaModulo(contexto, modulo);
  }
  return Object.freeze(resultado);
}
