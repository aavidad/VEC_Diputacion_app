/**
 * Atlas común, exclusivamente visual, para la presentación de RRHH.
 *
 * No contiene identificadores personales, contacto, credenciales ni datos de
 * efectos administrativos. Cada consumidor recibe una copia congelada: este
 * fichero no es un directorio ni una fuente de autorización.
 */

export const TEXTO_DATOS_FICTICIOS_RRHH = "Datos ficticios de presentación";
export const ESQUEMA_ATLAS_SINTETICO_RRHH = "vec.presentacion.rrhh.atlas.v1";

const USO = "presentacion_rrhh";
const CAMPOS_MARCA = ["sintetico", "uso"];

function congelarProfundo(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelarProfundo);
    Object.freeze(valor);
  }
  return valor;
}

function copiar(valor) {
  return structuredClone(valor);
}

function objetoPlano(valor) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) return false;
  const prototipo = Object.getPrototypeOf(valor);
  return prototipo === Object.prototype || prototipo === null;
}

function exigirCampos(objeto, campos, nombre) {
  if (!objetoPlano(objeto) || Object.keys(objeto).length !== campos.length
    || Object.keys(objeto).some((campo) => !campos.includes(campo))) {
    throw new TypeError(`${nombre} no respeta el esquema cerrado`);
  }
}

function exigirMarca(objeto, nombre) {
  if (objeto.sintetico !== true || objeto.uso !== USO) {
    throw new TypeError(`${nombre} debe declararse como dato sintético de presentación`);
  }
}

function exigirTexto(valor, nombre, maximo = 180) {
  if (typeof valor !== "string" || valor.length === 0 || valor.length > maximo
    || valor !== valor.trim() || /[\u0000-\u001f\u007f-\u009f]/u.test(valor)) {
    throw new TypeError(`${nombre} no es un texto válido`);
  }
}

function exigirReferencia(valor, prefijo, nombre) {
  exigirTexto(valor, nombre, 160);
  if (!new RegExp(`^${prefijo}_[A-Za-z0-9_-]{16,128}$`, "u").test(valor)) {
    throw new TypeError(`${nombre} no es una referencia opaca válida`);
  }
}

function exigirColeccion(valor, nombre, maximo = 16) {
  if (!Array.isArray(valor) || valor.length === 0 || valor.length > maximo) {
    throw new TypeError(`${nombre} no es una colección válida`);
  }
}

function validarPersona(valor, nombre) {
  exigirCampos(valor, ["persona_ref", "nombre_visible", "iniciales", ...CAMPOS_MARCA], nombre);
  exigirMarca(valor, nombre);
  exigirReferencia(valor.persona_ref, "per", `${nombre}.persona_ref`);
  exigirTexto(valor.nombre_visible, `${nombre}.nombre_visible`);
  exigirTexto(valor.iniciales, `${nombre}.iniciales`, 4);
}

function validarEmpleado(valor) {
  exigirCampos(valor, ["empleado_ref", "persona_ref", "puesto_ref", "unidad_ref", "centro_ref", ...CAMPOS_MARCA], "empleado");
  exigirMarca(valor, "empleado");
  exigirReferencia(valor.empleado_ref, "emp", "empleado.empleado_ref");
  exigirReferencia(valor.persona_ref, "per", "empleado.persona_ref");
  exigirReferencia(valor.puesto_ref, "pue", "empleado.puesto_ref");
  exigirReferencia(valor.unidad_ref, "uni", "empleado.unidad_ref");
  exigirReferencia(valor.centro_ref, "cen", "empleado.centro_ref");
}

function validarRelacion(valor) {
  exigirCampos(valor, ["relacion_ref", "empleado_ref", "unidad_ref", "puesto_ref", "situacion_visible", ...CAMPOS_MARCA], "relacion");
  exigirMarca(valor, "relacion");
  exigirReferencia(valor.relacion_ref, "rel", "relacion.relacion_ref");
  exigirReferencia(valor.empleado_ref, "emp", "relacion.empleado_ref");
  exigirReferencia(valor.unidad_ref, "uni", "relacion.unidad_ref");
  exigirReferencia(valor.puesto_ref, "pue", "relacion.puesto_ref");
  exigirTexto(valor.situacion_visible, "relacion.situacion_visible");
}

function validarUnidad(valor) {
  exigirCampos(valor, ["unidad_ref", "centro_ref", "nombre_visible", "area_visible", ...CAMPOS_MARCA], "unidad");
  exigirMarca(valor, "unidad");
  exigirReferencia(valor.unidad_ref, "uni", "unidad.unidad_ref");
  exigirReferencia(valor.centro_ref, "cen", "unidad.centro_ref");
  exigirTexto(valor.nombre_visible, "unidad.nombre_visible");
  exigirTexto(valor.area_visible, "unidad.area_visible");
}

function validarCentro(valor) {
  exigirCampos(valor, ["centro_ref", "nombre_visible", "localidad_ref", ...CAMPOS_MARCA], "centro");
  exigirMarca(valor, "centro");
  exigirReferencia(valor.centro_ref, "cen", "centro.centro_ref");
  exigirReferencia(valor.localidad_ref, "loc", "centro.localidad_ref");
  exigirTexto(valor.nombre_visible, "centro.nombre_visible");
}

function validarElementoCatalogo(valor, nombre, prefijo) {
  exigirCampos(valor, ["referencia", "nombre_visible", ...CAMPOS_MARCA], nombre);
  exigirMarca(valor, nombre);
  exigirReferencia(valor.referencia, prefijo, `${nombre}.referencia`);
  exigirTexto(valor.nombre_visible, `${nombre}.nombre_visible`);
}

function validarSinTerminosNoPresentables(valor) {
  const texto = JSON.stringify(valor).toLocaleLowerCase("es");
  if (/(?:demo|prueba|usuario)/u.test(texto)) {
    throw new TypeError("el atlas contiene un término no presentable");
  }
}

/** Valida estructura, marcas y coherencia de las referencias del atlas. */
export function validarAtlasSinteticoRRHH(atlas) {
  exigirCampos(atlas, [
    "esquema", "aviso_visible", "persona_principal", "responsable", "tecnica_rrhh",
    "empleado", "relacion", "unidad", "centro", "companeros", "proveedores", "localidades",
    ...CAMPOS_MARCA,
  ], "atlas");
  exigirMarca(atlas, "atlas");
  if (atlas.esquema !== ESQUEMA_ATLAS_SINTETICO_RRHH || atlas.aviso_visible !== TEXTO_DATOS_FICTICIOS_RRHH) {
    throw new TypeError("cabecera del atlas no válida");
  }

  validarPersona(atlas.persona_principal, "persona_principal");
  validarPersona(atlas.responsable, "responsable");
  validarPersona(atlas.tecnica_rrhh, "tecnica_rrhh");
  validarEmpleado(atlas.empleado);
  validarRelacion(atlas.relacion);
  validarUnidad(atlas.unidad);
  validarCentro(atlas.centro);

  exigirColeccion(atlas.companeros, "companeros");
  exigirColeccion(atlas.proveedores, "proveedores");
  exigirColeccion(atlas.localidades, "localidades");
  atlas.companeros.forEach((valor, indice) => validarPersona(valor, `companeros[${indice}]`));
  atlas.proveedores.forEach((valor, indice) => validarElementoCatalogo(valor, `proveedores[${indice}]`, "pro"));
  atlas.localidades.forEach((valor, indice) => validarElementoCatalogo(valor, `localidades[${indice}]`, "loc"));

  if (atlas.empleado.persona_ref !== atlas.persona_principal.persona_ref
    || atlas.relacion.empleado_ref !== atlas.empleado.empleado_ref
    || atlas.relacion.unidad_ref !== atlas.unidad.unidad_ref
    || atlas.relacion.puesto_ref !== atlas.empleado.puesto_ref
    || atlas.empleado.unidad_ref !== atlas.unidad.unidad_ref
    || atlas.empleado.centro_ref !== atlas.centro.centro_ref
    || atlas.unidad.centro_ref !== atlas.centro.centro_ref
    || !atlas.localidades.some((localidad) => localidad.referencia === atlas.centro.localidad_ref)) {
    throw new TypeError("las referencias Persona, Empleado, relación y unidad no son coherentes");
  }
  validarSinTerminosNoPresentables(atlas);
  return congelarProfundo(copiar(atlas));
}

const ATLAS_BASE = validarAtlasSinteticoRRHH({
  esquema: ESQUEMA_ATLAS_SINTETICO_RRHH,
  aviso_visible: TEXTO_DATOS_FICTICIOS_RRHH,
  persona_principal: {
    persona_ref: "per_AntonioLpezFernandez_7K2mX9Qa",
    nombre_visible: "Antonio López Fernández",
    iniciales: "ALF",
    sintetico: true,
    uso: USO,
  },
  responsable: {
    persona_ref: "per_MariaCarmenRuizSoto_4V8pN6Ld",
    nombre_visible: "María del Carmen Ruiz Soto",
    iniciales: "MRS",
    sintetico: true,
    uso: USO,
  },
  tecnica_rrhh: {
    persona_ref: "per_ElenaMartinRojas_9C3hW5Tf",
    nombre_visible: "Elena Martín Rojas",
    iniciales: "EMR",
    sintetico: true,
    uso: USO,
  },
  empleado: {
    empleado_ref: "emp_J4sR8qV2nT6kD9mP",
    persona_ref: "per_AntonioLpezFernandez_7K2mX9Qa",
    puesto_ref: "pue_TecnicoGestionProvincial_6Y2f",
    unidad_ref: "uni_InfraestructurasProvinciales_8N4c",
    centro_ref: "cen_SedeProvincialGranada_5H7b",
    sintetico: true,
    uso: USO,
  },
  relacion: {
    relacion_ref: "rel_VinculoServicio_3Q9wK6eR",
    empleado_ref: "emp_J4sR8qV2nT6kD9mP",
    unidad_ref: "uni_InfraestructurasProvinciales_8N4c",
    puesto_ref: "pue_TecnicoGestionProvincial_6Y2f",
    situacion_visible: "Relación activa mostrada únicamente para presentación",
    sintetico: true,
    uso: USO,
  },
  unidad: {
    unidad_ref: "uni_InfraestructurasProvinciales_8N4c",
    centro_ref: "cen_SedeProvincialGranada_5H7b",
    nombre_visible: "Servicio de Infraestructuras Provinciales",
    area_visible: "Área de Obras y Servicios",
    sintetico: true,
    uso: USO,
  },
  centro: {
    centro_ref: "cen_SedeProvincialGranada_5H7b",
    nombre_visible: "Sede provincial",
    localidad_ref: "loc_GranadaCapital_2M6x",
    sintetico: true,
    uso: USO,
  },
  companeros: [
    { persona_ref: "per_JavierMorenoGil_5B8nS3uQ", nombre_visible: "Javier Moreno Gil", iniciales: "JMG", sintetico: true, uso: USO },
    { persona_ref: "per_LuciaSerranoVega_8T4dP7yL", nombre_visible: "Lucía Serrano Vega", iniciales: "LSV", sintetico: true, uso: USO },
  ],
  proveedores: [
    { referencia: "pro_DesplazamientosAndaluces_7F3r", nombre_visible: "Desplazamientos Andaluces", sintetico: true, uso: USO },
    { referencia: "pro_AlojamientosSierraSur_9J5k", nombre_visible: "Alojamientos Sierra Sur", sintetico: true, uso: USO },
  ],
  localidades: [
    { referencia: "loc_GranadaCapital_2M6x", nombre_visible: "Granada", sintetico: true, uso: USO },
    { referencia: "loc_MotrilCosta_4R8v", nombre_visible: "Motril", sintetico: true, uso: USO },
    { referencia: "loc_GuadixAltiplano_6L1z", nombre_visible: "Guadix", sintetico: true, uso: USO },
  ],
  sintetico: true,
  uso: USO,
});

/** Atlas inmutable para quien no necesite mutarlo. */
export const ATLAS_SINTETICO_RRHH = ATLAS_BASE;

/** Devuelve una copia defensiva profundamente congelada para cada consumidor. */
export function obtenerAtlasSinteticoRRHH() {
  return congelarProfundo(copiar(ATLAS_BASE));
}
