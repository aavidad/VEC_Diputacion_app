/**
 * Servidor simulado del contrato de solicitudes de Selección (revisión 2),
 * solo para pruebas: persona (mis-solicitudes) y RRHH (bandeja y ficha).
 * Datos sintéticos. Expone `fetch(ruta, opciones)` compatible con el cliente
 * y `atender(metodo, ruta, cabeceras, cuerpo)` para un servidor HTTP de prueba.
 */

const CONVOCATORIA = Object.freeze({
  convocatoria_ref: "bolsa-operario-diputacion-2026", version: 1,
  titulo: "Bolsa de empleo de Operario de la Diputación de Granada",
  abre_en: "2026-09-01T00:00:00.000000Z", cierra_en: "2026-10-30T21:59:59.000000Z", abierta: true,
  fecha_referencia: "2026-10-30",
  turnos: [{ clave: "libre", etiqueta: "Turno libre" }, { clave: "discapacidad", etiqueta: "Reserva para personas con discapacidad" }],
  requisitos: [
    { clave: "nacionalidad", titulo: "Nacionalidad", descripcion: "Nacionalidad española o supuestos de acceso previstos en las bases.", obligatorio: true, impide_presentar: true },
    { clave: "edad", titulo: "Edad", descripcion: "Tener cumplidos dieciséis años.", obligatorio: true, impide_presentar: true },
    { clave: "titulacion", titulo: "Titulación", descripcion: "Certificado de escolaridad o equivalente.", obligatorio: true, impide_presentar: false },
  ],
  baremo: { maximo: "20", grupos: [
    { clave: "experiencia", titulo: "Experiencia profesional", maximo: "12", meritos: [
      { clave: "meses_diputacion", titulo: "Meses trabajados en la Diputación", unidad: "mes", puntos_por_unidad: "0.1", maximo: "8" },
      { clave: "meses_otras", titulo: "Meses en otras administraciones", unidad: "mes", puntos_por_unidad: "0.05", maximo: "6" },
    ] },
    { clave: "formacion", titulo: "Formación", maximo: "8", meritos: [
      { clave: "horas_curso", titulo: "Horas de cursos relacionados", unidad: "hora", puntos_por_unidad: "0.01", maximo: "8" },
    ] },
  ] },
  marca_ejemplo: true,
});
const CERRADA = Object.freeze({ ...CONVOCATORIA, convocatoria_ref: "bolsa-conserje-2026", titulo: "Bolsa de empleo de Conserje", abierta: false });

const PERSONAS_RRHH = Object.freeze([
  ["Reyes Álvarez", "Antonio", "***5678*", "libre"], ["Moreno Castillo", "Lucía", "***2211*", "libre"],
  ["Jiménez Ortega", "Francisco", "***9034*", "discapacidad"], ["Navarro Gil", "Carmen", "***4410*", "libre"],
]);

function decimal(valor) { return String(Math.round(valor * 1000) / 1000); }

function puntuacion(meritos) {
  let total = 0;
  for (const grupo of CONVOCATORIA.baremo.grupos) {
    let suma = 0;
    for (const merito of grupo.meritos) {
      const declarado = meritos.find((item) => item.clave_grupo === grupo.clave && item.clave_merito === merito.clave);
      if (declarado) suma += Math.min(Number(declarado.cantidad) * Number(merito.puntos_por_unidad), Number(merito.maximo));
    }
    total += Math.min(suma, Number(grupo.maximo));
  }
  return decimal(Math.min(total, Number(CONVOCATORIA.baremo.maximo)));
}

export function crearServidorSimulado({ ahora = () => "2026-09-26T10:15:00.000000Z", plazoAbierto = () => true, montado = true } = {}) {
  const solicitudes = new Map();
  const claves = new Map();
  const presentadas = [];
  const peticiones = [];
  let secuencia = 0;
  let numero = 0;
  // Bandeja RRHH: 30 solicitudes sintéticas presentadas más las que se presenten.
  for (let indice = 0; indice < 30; indice += 1) {
    const [apellidos, nombre, documento, turno] = PERSONAS_RRHH[indice % PERSONAS_RRHH.length];
    numero += 1;
    presentadas.push({
      solicitud_ref: `sol_rrhh${String(indice).padStart(4, "0")}`, numero_justificante: `2026/SOL-${String(numero).padStart(6, "0")}`,
      nombre_visible: `${apellidos}, ${nombre}`, documento_parcial: documento, estado: "presentada",
      presentada_en: `2026-09-${String(10 + (indice % 15)).padStart(2, "0")}T09:${String(indice).padStart(2, "0")}:00.000000Z`,
      puntuacion_autobaremo: decimal((indice % 9) * 0.7), turno,
      datos: { nombre, apellidos, documento_identidad: "00000000T", fecha_nacimiento: "1985-03-14", nacionalidad: "española", correo: "persona@example.org", telefono: "600000000", direccion: { via: "C/ Real 1", codigo_postal: "18001", municipio: "Granada", provincia: "Granada" } },
      requisitos: [{ clave: "nacionalidad", estado: "cumple" }, { clave: "edad", estado: "cumple" }, { clave: "titulacion", estado: indice % 3 ? "cumple" : "pendiente" }],
      meritos: [{ clave_grupo: "experiencia", clave_merito: "meses_diputacion", descripcion: "Peón de mantenimiento", cantidad: String(indice % 9 * 7) }],
      historia: [{ tipo: "borrador_guardado", version: 1, en: "2026-09-09T08:00:00.000000Z" }, { tipo: "presentada", version: 1, en: "2026-09-10T09:00:00.000000Z" }],
    });
  }

  const respuesta = (estado, cuerpo) => ({ estado, cuerpo });
  const error = (estado, codigo) => respuesta(estado, { error: { codigo } });
  const idempotente = (clave, huella, producir) => {
    if (!/^[A-Za-z0-9._:-]{8,128}$/u.test(clave || "")) return error(400, "peticion_no_permitida");
    const previa = claves.get(clave);
    if (previa) {
      if (previa.huella !== huella) return error(409, "clave_reutilizada");
      return respuesta(200, { data: { ...previa.resultado, repetida: true } });
    }
    const resultado = producir();
    if (resultado.estado >= 400) return resultado;
    claves.set(clave, { huella, resultado: resultado.cuerpo.data });
    return resultado;
  };
  const reglas = (ref) => ref === CONVOCATORIA.convocatoria_ref ? CONVOCATORIA : ref === CERRADA.convocatoria_ref ? CERRADA : null;

  function atender(metodo, rutaCompleta, cabeceras = {}, cuerpoTexto = "") {
    const url = new URL(rutaCompleta, "https://vec.invalid");
    const ruta = url.pathname;
    peticiones.push({ metodo, ruta: rutaCompleta, cabeceras });
    if (!montado) return error(404, "recurso_no_encontrado");
    let cuerpo = {};
    if (cuerpoTexto) { try { cuerpo = JSON.parse(cuerpoTexto); } catch { return error(400, "datos_no_validos"); } }
    const clave = cabeceras["Idempotency-Key"] || cabeceras["idempotency-key"] || "";
    const base = "/api/vec/seleccion/mis-solicitudes";
    if (metodo === "GET" && ruta === `${base}/convocatorias`) {
      return respuesta(200, { data: { convocatorias: [CONVOCATORIA, CERRADA].map(({ convocatoria_ref, titulo, abre_en, cierra_en, abierta }) => ({ convocatoria_ref, titulo, abre_en, cierra_en, abierta })) } });
    }
    if (metodo === "GET" && ruta === `${base}/convocatoria`) {
      const dato = reglas(url.searchParams.get("convocatoria_ref"));
      return dato ? respuesta(200, { data: structuredClone(dato) }) : error(404, "convocatoria_no_disponible");
    }
    if (metodo === "GET" && ruta === base) {
      return respuesta(200, { data: { solicitudes: [...solicitudes.values()].map((item) => ({
        solicitud_ref: item.solicitud_ref, convocatoria_ref: item.convocatoria_ref, convocatoria_titulo: reglas(item.convocatoria_ref).titulo,
        estado: item.estado, version: item.version, presentada_en: item.presentada_en ?? null, numero_justificante: item.numero_justificante ?? null,
        puntuacion_autobaremo: item.puntuacion_autobaremo,
      })) } });
    }
    if (metodo === "GET" && ruta === `${base}/borrador`) {
      const item = [...solicitudes.values()].find((solicitud) => solicitud.convocatoria_ref === url.searchParams.get("convocatoria_ref"));
      return item ? respuesta(200, { data: structuredClone(item) }) : error(404, "sin_borrador");
    }
    if (metodo === "PUT" && ruta === `${base}/borrador`) {
      return idempotente(clave, cuerpoTexto, () => {
        const convocatoria = reglas(cuerpo.convocatoria_ref);
        if (!convocatoria) return error(404, "convocatoria_no_disponible");
        if (!convocatoria.abierta || !plazoAbierto()) return error(409, "fuera_de_plazo");
        if (cuerpo.meritos?.some((merito) => !/^\d{1,9}(?:\.\d{1,3})?$/u.test(merito.cantidad))) return error(400, "datos_no_validos");
        const actual = [...solicitudes.values()].find((item) => item.convocatoria_ref === cuerpo.convocatoria_ref);
        if (actual?.estado === "presentada") return error(409, "ya_presentada");
        if ((actual?.version ?? 0) !== cuerpo.version_esperada) return error(409, "version_obsoleta");
        const solicitudRef = actual?.solicitud_ref ?? `sol_${String(++secuencia).padStart(6, "0")}`;
        const guardada = {
          solicitud_ref: solicitudRef, convocatoria_ref: cuerpo.convocatoria_ref, version: (actual?.version ?? 0) + 1, estado: "borrador",
          turno: cuerpo.turno ?? null, datos: cuerpo.datos ?? {}, requisitos: cuerpo.requisitos ?? [], meritos: cuerpo.meritos ?? [],
          puntuacion_autobaremo: puntuacion(cuerpo.meritos ?? []), actualizada_en: ahora(),
        };
        solicitudes.set(solicitudRef, guardada);
        return respuesta(actual ? 201 : 201, { data: { solicitud_ref: solicitudRef, version: guardada.version, estado: "borrador", puntuacion_autobaremo: guardada.puntuacion_autobaremo, repetida: false } });
      });
    }
    if (metodo === "POST" && ruta === `${base}/presentacion`) {
      return idempotente(clave, cuerpoTexto, () => {
        if (cuerpo.declaracion_responsable !== true) return error(400, "declaracion_requerida");
        const item = solicitudes.get(cuerpo.solicitud_ref);
        if (!item) return error(404, "recurso_no_encontrado");
        if (item.estado === "presentada") return error(409, "ya_presentada");
        if (item.version !== cuerpo.version_esperada) return error(409, "version_obsoleta");
        if (!plazoAbierto()) return error(409, "fuera_de_plazo");
        const convocatoria = reglas(item.convocatoria_ref);
        if (convocatoria.requisitos.some((requisito) => requisito.impide_presentar && item.requisitos.find((declarado) => declarado.clave === requisito.clave)?.estado === "no_cumple")) {
          return error(422, "requisito_no_cumplido");
        }
        numero += 1;
        Object.assign(item, { estado: "presentada", presentada_en: ahora(), numero_justificante: `2026/SOL-${String(numero).padStart(6, "0")}`, recibo_ref: `recibo:${item.solicitud_ref}` });
        presentadas.push({
          solicitud_ref: item.solicitud_ref, numero_justificante: item.numero_justificante,
          nombre_visible: `${item.datos.apellidos ?? ""}, ${item.datos.nombre ?? ""}`, documento_parcial: "***678Z*", estado: "presentada",
          presentada_en: item.presentada_en, puntuacion_autobaremo: item.puntuacion_autobaremo, turno: item.turno ?? "",
          datos: item.datos, requisitos: item.requisitos, meritos: item.meritos,
          historia: [{ tipo: "borrador_guardado", version: item.version, en: item.actualizada_en }, { tipo: "presentada", version: item.version, en: item.presentada_en }],
        });
        return respuesta(201, { data: {
          solicitud_ref: item.solicitud_ref, estado: "presentada", numero_justificante: item.numero_justificante, presentada_en: item.presentada_en,
          recibo_ref: item.recibo_ref, puntuacion_autobaremo: item.puntuacion_autobaremo, repetida: false,
          servicios: { firma: "no_disponible", registro_sede: "no_disponible", tasas: "no_disponible", notificacion: "no_disponible" },
        } });
      });
    }
    const rrhh = "/api/vec/seleccion/solicitudes";
    if (metodo === "GET" && ruta === `${rrhh}/convocatorias`) {
      return respuesta(200, { data: { convocatorias: [CONVOCATORIA, CERRADA].map(({ convocatoria_ref, titulo, abre_en, cierra_en, abierta }) => ({ convocatoria_ref, titulo, abre_en, cierra_en, abierta })) } });
    }
    if (metodo === "POST" && ruta === `${rrhh}/consultas`) {
      if (!Number.isSafeInteger(cuerpo.limite) || cuerpo.limite < 1 || cuerpo.limite > 100) return error(400, "datos_no_validos");
      const lista = cuerpo.convocatoria_ref === CONVOCATORIA.convocatoria_ref ? presentadas : [];
      const desde = cuerpo.cursor ? Number(cuerpo.cursor.slice(2)) : 0;
      const pagina = lista.slice(desde, desde + cuerpo.limite);
      const siguiente = desde + cuerpo.limite < lista.length ? `c:${desde + cuerpo.limite}` : "";
      return respuesta(200, { data: { solicitudes: pagina.map(({ datos: _d, requisitos: _r, meritos: _m, historia: _h, ...fila }) => fila), cursor_siguiente: siguiente } });
    }
    if (metodo === "POST" && ruta === `${rrhh}/detalle/consultas`) {
      const item = presentadas.find((fila) => fila.solicitud_ref === cuerpo.solicitud_ref);
      if (!item) return error(404, "recurso_no_encontrado");
      const titulos = new Map(CONVOCATORIA.requisitos.map((requisito) => [requisito.clave, requisito.titulo]));
      const meritosPublicados = new Map(CONVOCATORIA.baremo.grupos.flatMap((grupo) => grupo.meritos.map((merito) => [`${grupo.clave}|${merito.clave}`, merito])));
      return respuesta(200, { data: {
        solicitud_ref: item.solicitud_ref, convocatoria_ref: CONVOCATORIA.convocatoria_ref, convocatoria_titulo: CONVOCATORIA.titulo,
        numero_justificante: item.numero_justificante, estado: item.estado, presentada_en: item.presentada_en, turno: item.turno, datos: item.datos,
        requisitos: item.requisitos.map((requisito) => ({ ...requisito, titulo: titulos.get(requisito.clave) ?? requisito.clave, procedencia: "declarado_persona", fecha_referencia: CONVOCATORIA.fecha_referencia })),
        meritos: item.meritos.map((merito) => {
          const publicado = meritosPublicados.get(`${merito.clave_grupo}|${merito.clave_merito}`);
          return { ...merito, titulo: publicado?.titulo, unidad: publicado?.unidad ?? "", puntos: decimal(Math.min(Number(merito.cantidad) * Number(publicado?.puntos_por_unidad ?? 0), Number(publicado?.maximo ?? 0))) };
        }),
        puntuacion_autobaremo: item.puntuacion_autobaremo, historia: item.historia,
      } });
    }
    return error(404, "recurso_no_encontrado");
  }

  async function fetchSimulado(ruta, opciones = {}) {
    const { estado, cuerpo } = atender(opciones.method || "GET", ruta, opciones.headers || {}, opciones.body || "");
    const texto = JSON.stringify(cuerpo);
    return {
      status: estado, ok: estado >= 200 && estado < 300,
      headers: { get: (nombre) => (nombre.toLowerCase() === "content-type" ? "application/json; charset=utf-8" : null) },
      text: async () => texto, json: async () => JSON.parse(texto),
    };
  }

  return Object.freeze({ atender, fetch: fetchSimulado, peticiones, solicitudes, presentadas });
}
