import { escaparHTML, listaDatos, panel } from "./vistas/comunes.js";
import { TEXTO_MI_BOLSA as texto } from "./mi-bolsa-i18n.js";

const RUTA_MI_BOLSA = "/api/vec/bolsa/mi-bolsa";
const ESQUEMA_MI_BOLSA = "vec.bolsa.mi-bolsa.v1";
const MAXIMO_JSON_BYTES = 128 * 1024;
const MAXIMO_PARTICIPACIONES = 200;

export class ErrorMiBolsa extends Error {
  constructor(codigo, mensaje, causa) {
    super(mensaje, causa ? { cause: causa } : undefined);
    this.name = "ErrorMiBolsa";
    this.codigo = codigo;
  }
}

function exigirObjeto(valor, nombre) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) throw new TypeError(texto.errorObjeto(nombre));
  return valor;
}

function exigirCadena(valor, nombre, maximo = 300) {
  if (typeof valor !== "string" || valor.trim() === "" || valor.length > maximo) {
    throw new TypeError(texto.errorCadena(nombre, maximo));
  }
  return valor;
}

function exigirEntero(valor, nombre, minimo) {
  if (!Number.isSafeInteger(valor) || valor < minimo) throw new TypeError(texto.errorEntero(nombre, minimo));
  return valor;
}

function exigirFecha(valor, nombre, { nulo = false } = {}) {
  if (nulo && valor === null) return null;
  const fecha = exigirCadena(valor, nombre, 40);
  const coincidencia = /^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{3}))?Z)?$/u.exec(fecha);
  if (!coincidencia) throw new TypeError(texto.errorFecha(nombre));
  const [, anoTexto, mesTexto, diaTexto, horaTexto = "00", minutoTexto = "00", segundoTexto = "00", milisegundoTexto = "000"] = coincidencia;
  const ano = Number(anoTexto);
  const mes = Number(mesTexto);
  const dia = Number(diaTexto);
  const hora = Number(horaTexto);
  const minuto = Number(minutoTexto);
  const segundo = Number(segundoTexto);
  const milisegundo = Number(milisegundoTexto);
  const calendario = new Date(Date.UTC(ano, mes - 1, dia, hora, minuto, segundo, milisegundo));
  if (calendario.getUTCFullYear() !== ano || calendario.getUTCMonth() !== mes - 1 || calendario.getUTCDate() !== dia
    || calendario.getUTCHours() !== hora || calendario.getUTCMinutes() !== minuto || calendario.getUTCSeconds() !== segundo
    || calendario.getUTCMilliseconds() !== milisegundo) throw new TypeError(texto.errorFecha(nombre));
  return fecha;
}

export function validarMiBolsa(entrada) {
  const datos = structuredClone(exigirObjeto(entrada, "datos de mi bolsa"));
  if (exigirCadena(datos.esquema, "esquema", 80) !== ESQUEMA_MI_BOLSA) throw new TypeError(texto.errorEsquema);
  exigirFecha(datos.consultada_en, "consultada_en");
  if (!Array.isArray(datos.participaciones) || datos.participaciones.length > MAXIMO_PARTICIPACIONES) {
    throw new TypeError(texto.errorParticipaciones(MAXIMO_PARTICIPACIONES));
  }
  datos.participaciones.forEach((participacion, indice) => {
    const item = exigirObjeto(participacion, `participaciones[${indice}]`);
    exigirCadena(item.bolsa, `participaciones[${indice}].bolsa`);
    exigirCadena(item.categoria, `participaciones[${indice}].categoria`);
    exigirEntero(item.version, `participaciones[${indice}].version`, 1);
    exigirEntero(item.orden_inicial, `participaciones[${indice}].orden_inicial`, 1);
    exigirEntero(item.total_instantanea, `participaciones[${indice}].total_instantanea`, 0);
    if (item.total_instantanea < item.orden_inicial) {
      throw new TypeError(texto.errorTotal(indice));
    }
    exigirCadena(item.estado_bolsa, `participaciones[${indice}].estado_bolsa`);
    exigirFecha(item.vigente_desde, `participaciones[${indice}].vigente_desde`);
    exigirFecha(item.vigente_hasta, `participaciones[${indice}].vigente_hasta`, { nulo: true });
  });
  return congelarProfundo(datos);
}

function congelarProfundo(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelarProfundo);
    Object.freeze(valor);
  }
  return valor;
}

async function leerRespuesta(respuesta) {
  if (respuesta?.status === 403) throw new ErrorMiBolsa("acceso_denegado", texto.accesoDenegado);
  if (respuesta?.status !== 200) throw new ErrorMiBolsa("respuesta_http", texto.respuestaHTTP(respuesta?.status));
  const tipo = respuesta.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/iu.test(tipo)) throw new ErrorMiBolsa("tipo_respuesta", texto.tipoRespuesta);
  const declarada = respuesta.headers?.get?.("Content-Length");
  if (declarada && (!/^(?:0|[1-9][0-9]*)$/u.test(declarada) || Number(declarada) > MAXIMO_JSON_BYTES)) {
    throw new ErrorMiBolsa("respuesta_excesiva", texto.respuestaExcesiva);
  }
  const cuerpo = await respuesta.text();
  if (new TextEncoder().encode(cuerpo).byteLength > MAXIMO_JSON_BYTES) throw new ErrorMiBolsa("respuesta_excesiva", texto.respuestaExcesiva);
  let envelope;
  try {
    envelope = JSON.parse(cuerpo);
  } catch (causa) {
    throw new ErrorMiBolsa("json_invalido", texto.jsonInvalido, causa);
  }
  if (!envelope || typeof envelope !== "object" || Array.isArray(envelope) || !envelope.data || typeof envelope.data !== "object" || Array.isArray(envelope.data)) {
    throw new ErrorMiBolsa("respuesta_incompatible", texto.respuestaIncompatible);
  }
  return validarMiBolsa(envelope.data);
}

export function crearClienteMiBolsa({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") {
    return Object.freeze({ cargar: async () => { throw new ErrorMiBolsa("transporte_no_disponible", texto.transporteNoDisponible); } });
  }
  return Object.freeze({
    async cargar({ signal } = {}) {
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTA_MI_BOLSA, {
          method: "GET",
          headers: { Accept: "application/json" },
          credentials: "omit",
          cache: "no-store",
          redirect: "error",
          referrerPolicy: "no-referrer",
          signal,
        });
      } catch (causa) {
        if (causa?.name === "AbortError") throw causa;
        throw new ErrorMiBolsa("servicio_no_disponible", texto.servicioNoDisponible, causa);
      }
      return leerRespuesta(respuesta);
    },
  });
}

function formatoFecha(valor) {
  const fecha = new Date(valor);
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "long", timeZone: "UTC" }).format(fecha);
}

export function renderizarMiBolsa(datos) {
  if (!datos.participaciones.length) {
    return panel(texto.bolsa, texto.consultaDisponible, `<p>${texto.vacio}</p>`);
  }
  const participaciones = datos.participaciones.map((item) => panel(
    item.bolsa,
    item.categoria,
    `${listaDatos([
      [texto.ordenInicial, `${escaparHTML(String(item.orden_inicial))} de ${escaparHTML(String(item.total_instantanea))}`],
      [texto.estadoBolsa, escaparHTML(item.estado_bolsa)],
      [texto.vigenteDesde, escaparHTML(formatoFecha(item.vigente_desde))],
      [texto.vigenteHasta, item.vigente_hasta ? escaparHTML(formatoFecha(item.vigente_hasta)) : texto.sinFechaFin],
    ])}<p class="nota">${escaparHTML(texto.ordenHistorico(item.version))}</p>`,
    { estado: item.estado_bolsa },
  )).join("");
  return `<header class="encabezado-vista"><div><h2>${texto.bolsa}</h2><p>${texto.descripcion}</p></div></header>${participaciones}<p class="nota">${texto.limiteDemo}</p>`;
}

export async function iniciarMiBolsa({ cliente = crearClienteMiBolsa(), documento = document, ventana = window } = {}) {
  const carga = documento.getElementById("estado-carga");
  const espacio = documento.getElementById("espacio-trabajo");
  const titulo = documento.getElementById("titulo-vista");
  const migas = documento.getElementById("migas-pan");
  if (!carga || !espacio || !titulo || !migas) throw new Error(texto.montajeIncompleto);
  titulo.textContent = texto.bolsa;
  migas.textContent = texto.migas;
  const controlador = new AbortController();
  const cargar = async () => {
    carga.hidden = false;
    carga.className = "estado-carga";
    carga.textContent = texto.carga;
    espacio.replaceChildren();
    try {
      const datos = await cliente.cargar({ signal: controlador.signal });
      carga.hidden = true;
      espacio.innerHTML = renderizarMiBolsa(datos);
    } catch (error) {
      if (error?.name === "AbortError") return;
      carga.hidden = true;
      const mensaje = error instanceof Error ? error.message : texto.errorConsulta;
      espacio.innerHTML = `<section class="estado-error" role="alert"><h2>${texto.errorConsulta}</h2><p>${escaparHTML(mensaje)}</p><button type="button" class="boton-primario" data-reintentar-mi-bolsa>${texto.reintentar}</button></section>`;
      espacio.querySelector("[data-reintentar-mi-bolsa]")?.addEventListener("click", cargar, { once: true });
    }
  };
  ventana.addEventListener("pagehide", () => controlador.abort(), { once: true });
  await cargar();
}

export const CONTRATO_MI_BOLSA = Object.freeze({ ruta: RUTA_MI_BOLSA, esquema: ESQUEMA_MI_BOLSA });
