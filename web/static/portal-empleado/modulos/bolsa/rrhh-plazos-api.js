export const RUTA_POLITICA_OFERTAS = "/api/vec/bolsa/politica-ofertas";
export const RUTA_CAPACIDAD_POLITICA_OFERTAS = `${RUTA_POLITICA_OFERTAS}/capacidad`;
export const ESQUEMA_POLITICA_OFERTAS = "vec.bolsa.rrhh.politica-ofertas.v1";

const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,255}$/u;
const MUNICIPIO = /^[0-9]{5}$/u;
const SHA256 = /^[a-f0-9]{64}$/u;
const UNIDADES_DIAS = new Set(["dias_habiles", "dias_naturales"]);

export function validarPoliticaEditable(politica) {
  const plazo = politica?.plazo;
  const horas = plazo?.unidad === "horas_naturales";
  if (!plazo || (!horas && !UNIDADES_DIAS.has(plazo.unidad)) || !Number.isSafeInteger(plazo.cantidad)
    || plazo.cantidad < 1 || plazo.cantidad > (horas ? 720 : 30)
    || plazo.computo !== (horas ? "continuo_utc" : "administrativo")
    || typeof plazo.municipio_sede !== "string" || !MUNICIPIO.test(plazo.municipio_sede)
    || politica?.adjudicacion?.criterio !== "orden_vigente"
    || politica.adjudicacion.elegibilidad !== "disposicion_en_plazo"
    || politica?.no_cubierta?.accion !== "llamamiento_directo"
    || politica.no_cubierta.condicion !== "sin_disposiciones_elegibles") {
    throw new TypeError("política de ofertas no válida");
  }
  return structuredClone(politica);
}

export function validarPoliticaRecibida(sobre) {
  const p = sobre?.data ?? sobre;
  if (!p || p.esquema !== ESQUEMA_POLITICA_OFERTAS
    || !REFERENCIA.test(p.bolsa_ref ?? "") || !Number.isSafeInteger(p.version)
    || p.version < 0 || typeof p.configurada !== "boolean"
    || (p.huella_sha256 !== undefined && !SHA256.test(p.huella_sha256))
    || p.ejemplo !== true || (p.recibo_ref !== undefined && !REFERENCIA.test(p.recibo_ref))
    || (p.publicada_en !== undefined && Number.isNaN(Date.parse(p.publicada_en)))
    || (p.configurada && (p.version < 1 || !p.huella_sha256 || !p.publicada_en))
    || (!p.configurada && (p.version !== 0 || p.politica !== null))
    || (Object.hasOwn(p, "puede_publicar") && typeof p.puede_publicar !== "boolean")) {
    throw new TypeError("respuesta de política de ofertas no válida");
  }
  const politica = { ...p, politica: p.configurada ? validarPoliticaEditable(p.politica) : null };
  delete politica.puede_publicar;
  return politica;
}

function validarBolsa(bolsaRef) {
  if (typeof bolsaRef !== "string" || !REFERENCIA.test(bolsaRef)) throw new TypeError("bolsa no válida");
  return bolsaRef;
}

async function codigoError(respuesta) {
  try { return (await respuesta.json())?.error?.codigo ?? ""; } catch { return ""; }
}

export function crearClientePoliticaOfertas({ fetchImpl = fetch } = {}) {
  const opciones = (signal) => ({ credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
    referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" } });
  return Object.freeze({
    async consultar(bolsaRef, { signal } = {}) {
      const bolsa = validarBolsa(bolsaRef);
      const ruta = `${RUTA_POLITICA_OFERTAS}?${new URLSearchParams({ bolsa_ref: bolsa })}`;
      const respuesta = await fetchImpl(ruta, { ...opciones(signal), method: "GET" });
      if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
      const politica = validarPoliticaRecibida(await respuesta.json());
      if (politica.bolsa_ref !== bolsa) throw new TypeError("política de otra bolsa");
      return { ok: true, politica };
    },
    async consultarCapacidad(bolsaRef, { signal } = {}) {
      const bolsa = validarBolsa(bolsaRef);
      const respuesta = await fetchImpl(RUTA_CAPACIDAD_POLITICA_OFERTAS, { ...opciones(signal), method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ bolsa_ref: bolsa }) });
      if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
      const capacidad = await respuesta.json();
      if (typeof capacidad?.puede_publicar !== "boolean") throw new TypeError("capacidad de política no válida");
      return { ok: true, puede_publicar: capacidad.puede_publicar };
    },
    async publicar({ bolsa_ref, version_esperada, clave_idempotencia, politica }, { signal } = {}) {
      validarBolsa(bolsa_ref);
      if (!Number.isSafeInteger(version_esperada) || version_esperada < 0
        || typeof clave_idempotencia !== "string" || !REFERENCIA.test(clave_idempotencia)) {
        throw new TypeError("comando de política no válido");
      }
      const cuerpo = { bolsa_ref, version_esperada, clave_idempotencia, politica: validarPoliticaEditable(politica) };
      const respuesta = await fetchImpl(RUTA_POLITICA_OFERTAS, { ...opciones(signal), method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave_idempotencia },
        body: JSON.stringify(cuerpo) });
      if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
      const recibida = validarPoliticaRecibida(await respuesta.json());
      if (recibida.bolsa_ref !== bolsa_ref || !recibida.configurada
        || recibida.version !== version_esperada + 1 || !recibida.recibo_ref) {
        throw new TypeError("recibo de política incoherente");
      }
      return { ok: true, status: respuesta.status, politica: recibida };
    },
  });
}
