export function resultadoPrueba(hechoRef = "hecho:propio-a", nombre = "Curso de gestión", version = 2) {
  return {
    codigo: "obtenida",
    hecho_actual: {
      referencia: hechoRef, version, tipo: "curso_superacion", concepto_ref: "concepto:curso", denominacion: nombre, horas: 24,
      procedencia: { fuente_ref: "fuente:formacion", version: "edicion:2", hecho_origen_ref: "curso:2", capturada_en: "2026-10-02T08:30:00Z" },
      vigencia: { desde: "2026-10-01" }, estado: "acreditado", evidencias: [{ id: "documento:curso", version: 1 }],
      revision: { referencia: "revision:2", motivo_ref: "motivo:verificacion", fecha: "2026-10-02T08:35:00Z" },
    },
    recibo_consulta: {
      referencia: "consulta:1", hecho_ref: hechoRef, version_consultada: version,
      decision_ref: "decision:1", consumo_huella_sha256: "a".repeat(64), auditoria_ref: "auditoria:1", correlacion_ref: "corr:1", consultada_en: "2026-10-02T08:40:00.123456Z",
    },
  };
}

export function noEncontradaPrueba(hechoRef = "hecho:propio-a") {
  const valor = resultadoPrueba(hechoRef);
  valor.codigo = "no_encontrada"; valor.hecho_actual = null; valor.recibo_consulta.version_consultada = 0;
  return valor;
}

export function respuestaPrueba(datos = resultadoPrueba()) {
  return new Response(JSON.stringify(datos), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

export function pendiente() {
  let resolver; let rechazar;
  const promesa = new Promise((resolve, reject) => { resolver = resolve; rechazar = reject; });
  return { promesa, resolver, rechazar };
}

export function raizPrueba() {
  class Nodo {
    constructor(documento) { this.ownerDocument = documento; this.children = []; this.listeners = new Map(); this.innerHTML = ""; this.parentElement = null; this.dataset = {}; this.attributes = new Map(); }
    append(nodo) { this.children.push(nodo); nodo.parentElement = this; }
    remove() { if (this.parentElement) this.parentElement.children = this.parentElement.children.filter((v) => v !== this); this.parentElement = null; }
    addEventListener(tipo, callback) { this.listeners.set(tipo, callback); }
    removeEventListener(tipo, callback) { if (this.listeners.get(tipo) === callback) this.listeners.delete(tipo); }
    contains(nodo) { return nodo?.parentElement === this; }
    querySelector() { return null; }
    focus() { this.enfocado = true; }
    setAttribute(nombre, valor) { this.attributes.set(nombre, valor); }
  }
  const documento = { createElement() { return new Nodo(documento); } };
  return new Nodo(documento);
}
