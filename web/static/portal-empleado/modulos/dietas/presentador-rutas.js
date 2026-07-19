/** Estado y reglas de la herramienta de rutas, independiente del adaptador OSRM. */
import {
  ATRIBUCION_OSM_INTERNA,
  ESQUEMA_SOLICITUD_RUTA_DIETAS,
  PLANTILLA_TESELAS_OSM_INTERNA,
  copiarDietas,
  validarCalculoRutaDietas,
  validarCatalogoRutasDietas,
  validarSolicitudRutaDietas,
} from "./contrato.js";

const MAXIMO_PARADAS = 12;

function normalizarNombre(valor) {
  return String(valor ?? "").normalize("NFD").replace(/[\u0300-\u036f]/g, "")
    .toLocaleLowerCase("es").trim();
}

function textoMotivo(valor, nombre) {
  const resultado = String(valor ?? "").trim();
  if (resultado.length < 8 || resultado.length > 500
    || /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(resultado)) {
    throw new Error(`${nombre} debe contener entre 8 y 500 caracteres`);
  }
  return resultado;
}

function numeroAjuste(valor) {
  const numero = Number(valor ?? 0);
  if (!Number.isFinite(numero) || numero < 0 || numero > 1_000) throw new Error("kilometros de ajuste no validos");
  return Math.round(numero * 10) / 10;
}

function congelarModelo(valor) {
  if (!valor || typeof valor !== "object" || Object.isFrozen(valor)) return valor;
  Object.values(valor).forEach(congelarModelo);
  return Object.freeze(valor);
}

export function crearPresentadorRutasDietas({ catalogo: catalogoEntrada, permisos, demostracion } = {}) {
  if (!permisos || typeof permisos !== "object") throw new Error("permisos de rutas no validos");
  if (!permisos.consultarRutas || !permisos.gestionarRutas || !catalogoEntrada) return null;
  const catalogo = validarCatalogoRutasDietas(catalogoEntrada);
  if (catalogo.demostracion !== (demostracion === true)) throw new Error("el catalogo de rutas no coincide con el entorno");
  const puntoPorCodigo = new Map(catalogo.puntos.map((punto) => [punto.codigo, punto]));
  const codigoPorNombre = new Map(catalogo.puntos.map((punto) => [normalizarNombre(punto.nombre), punto.codigo]));
  const codigoGranada = codigoPorNombre.get("granada") || catalogo.puntos[0].codigo;
  const codigoDestino = codigoPorNombre.get("motril") || catalogo.puntos.find((punto) => punto.codigo !== codigoGranada)?.codigo;
  const estado = {
    paradas: [codigoGranada, codigoDestino, codigoGranada],
    ultimaSolicitud: null,
    calculo: null,
    alternativa: "",
    motivoAlternativa: "",
    ajustes: new Map(),
  };

  function reiniciar() {
    estado.paradas = [codigoGranada, codigoDestino, codigoGranada];
    invalidarCalculo();
    return obtenerModelo();
  }

  function invalidarCalculo() {
    estado.ultimaSolicitud = null;
    estado.calculo = null;
    estado.alternativa = "";
    estado.motivoAlternativa = "";
    estado.ajustes.clear();
  }

  function obtenerAlternativa() {
    return estado.calculo?.alternativas.find((item) => item.referencia === estado.alternativa) || null;
  }

  function obtenerModelo() {
    const alternativa = obtenerAlternativa();
    const ajustes = alternativa?.tramos.map((tramo) => {
      const ajuste = estado.ajustes.get(tramo.indice) || { kilometros: 0, motivo: "" };
      return {
        ...tramo,
        ajuste_kilometros: ajuste.kilometros,
        motivo_ajuste: ajuste.motivo,
        kilometros_liquidables_demo: Math.round((tramo.kilometros + ajuste.kilometros) * 10) / 10,
      };
    }) || [];
    const kilometrosAjuste = ajustes.reduce((total, tramo) => total + tramo.ajuste_kilometros, 0);
    const ruta = alternativa
      ? [alternativa.tramos[0].origen_nombre, ...alternativa.tramos.map((tramo) => tramo.destino_nombre)]
      : estado.paradas.map((codigo) => puntoPorCodigo.get(codigo)?.nombre || "");
    const mapaRuta = alternativa ? Object.freeze({
      vista_ref: "borrador-ruta-calculada",
      proveedor: "openstreetmap",
      despliegue: "red_interna",
      plantilla_teselas: PLANTILLA_TESELAS_OSM_INTERNA,
      atribucion: ATRIBUCION_OSM_INTERNA,
      demostracion: demostracion === true,
      geometria: alternativa.geometria,
    }) : null;
    return congelarModelo({
      catalogo: copiarDietas(catalogo),
      paradas: [...estado.paradas],
      calculado: Boolean(alternativa),
      calculo_ref: estado.calculo?.referencia || "",
      motor: estado.calculo?.motor || "",
      version_grafo: estado.calculo?.version_grafo || "",
      liquidable: false,
      alternativas: (estado.calculo?.alternativas || []).map((item) => ({
        referencia: item.referencia,
        etiqueta: item.etiqueta,
        recomendada: item.recomendada,
        kilometros: item.kilometros,
        duracion_minutos: item.duracion_minutos,
        seleccionada: item.referencia === estado.alternativa,
      })),
      alternativa_ref: alternativa?.referencia || "",
      alternativa_recomendada: alternativa?.recomendada === true,
      motivo_alternativa: estado.motivoAlternativa,
      ruta,
      tramos: ajustes,
      kilometros_base: alternativa?.kilometros || 0,
      kilometros_ajuste: Math.round(kilometrosAjuste * 10) / 10,
      kilometros_total: Math.round(((alternativa?.kilometros || 0) + kilometrosAjuste) * 10) / 10,
      duracion_minutos: alternativa?.duracion_minutos || 0,
      mapa_ruta: mapaRuta,
      lista_para_borrador: Boolean(alternativa)
        && (alternativa.recomendada || estado.motivoAlternativa.length >= 8),
    });
  }

  function establecerParada(indiceEntrada, codigoEntrada) {
    const indice = Number(indiceEntrada);
    const codigo = String(codigoEntrada ?? "").trim();
    if (!Number.isInteger(indice) || indice < 0 || indice >= estado.paradas.length) throw new Error("indice de parada no valido");
    if (codigo && !puntoPorCodigo.has(codigo)) throw new Error("parada fuera del catalogo provincial");
    estado.paradas[indice] = codigo;
    invalidarCalculo();
    return obtenerModelo();
  }

  function agregarParada() {
    if (estado.paradas.length >= MAXIMO_PARADAS) throw new Error("la ruta no admite mas paradas");
    const ultima = estado.paradas.at(-1);
    const indice = estado.paradas.length > 1 && estado.paradas[0] === ultima
      ? estado.paradas.length - 1 : estado.paradas.length;
    estado.paradas.splice(indice, 0, "");
    invalidarCalculo();
    return obtenerModelo();
  }

  function eliminarParada(indiceEntrada) {
    const indice = Number(indiceEntrada);
    if (!Number.isInteger(indice) || indice < 0 || indice >= estado.paradas.length) throw new Error("indice de parada no valido");
    if (estado.paradas.length <= 2) throw new Error("la ruta necesita salida y destino");
    estado.paradas.splice(indice, 1);
    invalidarCalculo();
    return obtenerModelo();
  }

  function prepararSolicitudCalculo() {
    const solicitud = validarSolicitudRutaDietas({
      esquema: ESQUEMA_SOLICITUD_RUTA_DIETAS,
      paradas: [...estado.paradas],
      alternativas: 3,
    }, catalogo);
    estado.ultimaSolicitud = solicitud;
    return solicitud;
  }

  function registrarCalculo(calculoEntrada) {
    if (!estado.ultimaSolicitud) throw new Error("no existe una solicitud de ruta pendiente");
    const calculo = validarCalculoRutaDietas(calculoEntrada, estado.ultimaSolicitud);
    if (calculo.demostracion !== (demostracion === true)) {
      throw new Error("el cálculo de ruta no coincide con el entorno de la sesión");
    }
    estado.calculo = calculo;
    const recomendada = estado.calculo.alternativas.find((item) => item.recomendada);
    estado.alternativa = recomendada.referencia;
    estado.motivoAlternativa = "";
    estado.ajustes.clear();
    return obtenerModelo();
  }

  function seleccionarAlternativa(referenciaEntrada, motivoEntrada = "") {
    if (!estado.calculo) throw new Error("calcule la ruta antes de elegir una alternativa");
    const referencia = String(referenciaEntrada ?? "").trim();
    const alternativa = estado.calculo.alternativas.find((item) => item.referencia === referencia);
    if (!alternativa) throw new Error("alternativa de ruta no encontrada");
    const motivo = alternativa.recomendada ? "" : textoMotivo(motivoEntrada, "el motivo de la ruta alternativa");
    estado.alternativa = referencia;
    estado.motivoAlternativa = motivo;
    estado.ajustes.clear();
    return obtenerModelo();
  }

  function ajustarTramo(indiceEntrada, kilometrosEntrada, motivoEntrada = "") {
    const alternativa = obtenerAlternativa();
    const indice = Number(indiceEntrada);
    if (!alternativa || !Number.isInteger(indice) || !alternativa.tramos.some((tramo) => tramo.indice === indice)) {
      throw new Error("tramo de ruta no encontrado");
    }
    const kilometros = numeroAjuste(kilometrosEntrada);
    if (kilometros === 0) estado.ajustes.delete(indice);
    else estado.ajustes.set(indice, { kilometros, motivo: textoMotivo(motivoEntrada, "el motivo del ajuste") });
    return obtenerModelo();
  }

  function prepararRutaBorrador() {
    const modelo = obtenerModelo();
    const alternativa = obtenerAlternativa();
    if (!modelo.lista_para_borrador || !alternativa) throw new Error("calcule y justifique la ruta antes de guardar el borrador");
    return congelarModelo({
      ruta: [...modelo.ruta],
      origen: modelo.ruta[0],
      destino: modelo.ruta.length > 2 && modelo.ruta.at(-1) === modelo.ruta[0]
        ? modelo.ruta.at(-2) : modelo.ruta.at(-1),
      kilometros: modelo.kilometros_total,
      geometria_ruta: alternativa.geometria,
      trazabilidad_ruta: {
        calculo_ref: modelo.calculo_ref,
        alternativa_ref: modelo.alternativa_ref,
        motor: modelo.motor,
        version_grafo: modelo.version_grafo,
        liquidable: false,
        kilometros_base: modelo.kilometros_base,
        kilometros_ajuste: modelo.kilometros_ajuste,
        motivo_alternativa: modelo.motivo_alternativa,
        ajustes: modelo.tramos.filter((tramo) => tramo.ajuste_kilometros > 0).map((tramo) => ({
          indice: tramo.indice,
          kilometros: tramo.ajuste_kilometros,
          motivo: tramo.motivo_ajuste,
        })),
      },
    });
  }

  return Object.freeze({
    obtenerModelo,
    reiniciar,
    establecerParada,
    agregarParada,
    eliminarParada,
    prepararSolicitudCalculo,
    registrarCalculo,
    seleccionarAlternativa,
    ajustarTramo,
    prepararRutaBorrador,
  });
}
