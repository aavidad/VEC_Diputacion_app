// Analizador léxico mínimo para localizar textos visibles escritos en el código
// sin pasar por un catálogo i18n. Lo usa `i18n-bolsa-contratacion.test.mjs`.
//
// Recorre el fuente distinguiendo comentarios, cadenas, plantillas (con sus
// expresiones `${…}` anidadas) y expresiones regulares. Las expresiones de una
// plantilla se sustituyen por `\u0001`, de modo que `title="${t("x")}"` no
// cuenta como literal y `title="Ver ${n} filas"` sí.

const LETRAS = /[A-Za-zÁÉÍÓÚÜÑáéíóúüñ]{2,}/u;
const HUECO = "\u0001";
const ANTES_DE_REGEX = new Set([..."(,=:[!&|?{};+-*%<>~^"]);
const PALABRAS_ANTES_DE_REGEX = /(?:^|[^\w$])(?:return|typeof|case|do|else|in|of|void|yield|await)$/u;

/** Devuelve los literales del fuente: { tipo: "cadena"|"plantilla", texto, linea, inicio, fin }. */
export function extraerLiterales(fuente) {
  const literales = [];
  let i = 0;
  let linea = 1;
  const n = fuente.length;
  let previo = "";
  let palabraPrevia = "";

  function leerCadena(comilla) {
    const inicio = i; const lineaInicio = linea; i += 1; let texto = "";
    while (i < n && fuente[i] !== comilla) {
      if (fuente[i] === "\\") { texto += fuente[i + 1] ?? ""; i += 2; continue; }
      if (fuente[i] === "\n") linea += 1;
      texto += fuente[i]; i += 1;
    }
    i += 1;
    literales.push({ tipo: "cadena", texto, linea: lineaInicio, inicio, fin: i });
  }

  function leerPlantilla() {
    const inicio = i; const lineaInicio = linea; i += 1; let texto = "";
    while (i < n && fuente[i] !== "`") {
      if (fuente[i] === "\\") { texto += fuente[i + 1] ?? ""; i += 2; continue; }
      if (fuente[i] === "$" && fuente[i + 1] === "{") {
        i += 2; leerCodigo("}"); i += 1; texto += HUECO; continue;
      }
      if (fuente[i] === "\n") linea += 1;
      texto += fuente[i]; i += 1;
    }
    i += 1;
    literales.push({ tipo: "plantilla", texto, linea: lineaInicio, inicio, fin: i });
  }

  function leerRegex() {
    i += 1; let clase = false;
    while (i < n) {
      const c = fuente[i];
      if (c === "\\") { i += 2; continue; }
      if (c === "\n") break;
      if (c === "[") clase = true; else if (c === "]") clase = false;
      else if (c === "/" && !clase) { i += 1; break; }
      i += 1;
    }
    while (i < n && /[a-z]/u.test(fuente[i])) i += 1;
  }

  function leerCodigo(cierre) {
    let profundidad = 0;
    while (i < n) {
      const c = fuente[i];
      if (cierre && c === cierre && profundidad === 0) return;
      if (c === "\n") { linea += 1; i += 1; continue; }
      if (c === "/" && fuente[i + 1] === "/") { while (i < n && fuente[i] !== "\n") i += 1; continue; }
      if (c === "/" && fuente[i + 1] === "*") {
        const fin = fuente.indexOf("*/", i + 2); const hasta = fin < 0 ? n : fin + 2;
        linea += (fuente.slice(i, hasta).match(/\n/gu) || []).length; i = hasta; continue;
      }
      if (c === '"' || c === "'") { leerCadena(c); previo = "a"; palabraPrevia = ""; continue; }
      if (c === "`") { leerPlantilla(); previo = "a"; palabraPrevia = ""; continue; }
      if (c === "/" && (previo === "" || ANTES_DE_REGEX.has(previo) || PALABRAS_ANTES_DE_REGEX.test(palabraPrevia))) {
        leerRegex(); previo = "a"; palabraPrevia = ""; continue;
      }
      if (c === "{" || c === "(" || c === "[") profundidad += 1;
      if (c === "}" || c === ")" || c === "]") profundidad -= 1;
      if (!/\s/u.test(c)) { previo = c; palabraPrevia = /[\w$]/u.test(c) ? palabraPrevia + c : ""; if (!/[\w$]/u.test(c)) palabraPrevia = ""; }
      else if (palabraPrevia) palabraPrevia += " ";
      i += 1;
    }
  }

  leerCodigo(null);
  return literales;
}

/**
 * Rangos del fuente ocupados por catálogos i18n locales: objetos asignados a
 * constantes `MENSAJES*`, `TEXTO`, `TEXTOS`, `VOCABULARIO` o `textos*`.
 */
export function rangosCatalogo(fuente) {
  const rangos = [];
  const todos = extraerLiterales(fuente);
  const patron = /\bconst\s+(?:MENSAJES(?:_[A-Z0-9_]*)?|TEXTOS?|VOCABULARIO|textos[A-Za-z]*)\s*=\s*(?:Object\.freeze\(\s*)?\{/gu;
  for (const coincidencia of fuente.matchAll(patron)) {
    const inicio = coincidencia.index + coincidencia[0].length - 1;
    let profundidad = 0; let j = inicio;
    // Recorrido de llaves ignorando las contenidas en literales.
    const literales = todos.filter((l) => l.inicio > inicio);
    let k = 0;
    for (; j < fuente.length; j += 1) {
      while (k < literales.length && literales[k].fin <= j) k += 1;
      if (k < literales.length && literales[k].inicio <= j && j < literales[k].fin) { j = literales[k].fin - 1; continue; }
      if (fuente[j] === "{") profundidad += 1;
      if (fuente[j] === "}") { profundidad -= 1; if (profundidad === 0) break; }
    }
    rangos.push([inicio, j]);
  }
  return rangos;
}

const ATRIBUTO = /\b(title|aria-label|aria-description|aria-roledescription|aria-placeholder|placeholder|alt)\s*=\s*(["'])([^"']*)\2/gu;
const PROPIEDAD = /(?:\.(?:title|placeholder|ariaLabel|alt|textContent|innerText)\s*=|setAttribute\(\s*["'](?:title|aria-label|placeholder|alt)["']\s*,|\b(?:confirm|alert|prompt)\()\s*$/u;

/**
 * Hallazgos del fuente: atributos accesibles o de ayuda con texto literal,
 * texto visible literal en plantillas HTML, propiedades DOM asignadas con un
 * literal y diálogos del navegador con texto literal.
 */
export function hallazgosTextosLiterales(fuente) {
  const catalogos = rangosCatalogo(fuente);
  const enCatalogo = (literal) => catalogos.some(([a, b]) => literal.inicio >= a && literal.fin <= b + 1);
  const hallazgos = [];
  for (const literal of extraerLiterales(fuente)) {
    if (enCatalogo(literal)) continue;
    const texto = literal.texto;
    const esHTML = /<[a-z][\w-]*[\s>/]|<\/[a-z]/u.test(texto);
    if (esHTML || /\b(title|aria-label|placeholder|alt)\s*=/u.test(texto)) {
      for (const [, atributo, , valor] of texto.matchAll(ATRIBUTO)) {
        if (LETRAS.test(valor.replaceAll(HUECO, " "))) hallazgos.push({ linea: literal.linea, clase: "atributo", texto: `${atributo}="${valor.replaceAll(HUECO, "${…}")}"` });
      }
    }
    if (esHTML) {
      const sinEtiquetas = texto.replace(/<!--[\s\S]*?-->/gu, " ").replace(/<(script|style)[\s\S]*?<\/\1>/gu, " ");
      for (const [, contenido] of sinEtiquetas.matchAll(/>([^<>]*)</gu)) {
        for (const trozo of contenido.split(HUECO)) {
          const limpio = trozo.replace(/&[a-z]+;|&#\d+;/gu, " ").trim();
          if (LETRAS.test(limpio)) hallazgos.push({ linea: literal.linea, clase: "texto", texto: limpio });
        }
      }
      // Texto tras la última etiqueta o antes de la primera, junto a un hueco.
      const bordes = [sinEtiquetas.match(/^([^<>]*)</u)?.[1], sinEtiquetas.match(/>([^<>]*)$/u)?.[1]];
      for (const borde of bordes) {
        for (const trozo of String(borde ?? "").split(HUECO)) {
          const limpio = trozo.replace(/&[a-z]+;|&#\d+;/gu, " ").trim();
          if (LETRAS.test(limpio) && !/^[\w-]+=/u.test(limpio) && !/["=]/u.test(limpio)) hallazgos.push({ linea: literal.linea, clase: "texto", texto: limpio });
        }
      }
    }
    const antes = fuente.slice(Math.max(0, literal.inicio - 80), literal.inicio);
    if (PROPIEDAD.test(antes) && LETRAS.test(texto.replaceAll(HUECO, " "))) hallazgos.push({ linea: literal.linea, clase: "propiedad", texto: texto.replaceAll(HUECO, "${…}").slice(0, 80) });
  }
  return hallazgos;
}

const HUMANO = /[A-ZÁÉÍÓÚÑ¿¡][a-záéíóúüñ]{2,}|[A-ZÁÉÍÓÚÑ¿¡a-záéíóúüñ][a-záéíóúüñ]*\s+[a-záéíóúüñ]{2,}/u;
const CONTEXTO_TECNICO = /(?:\b(?:exigir|validar)\w*\((?:[^()]|\([^()]*\))*,\s*|\bthrow\s+new\s+\w*Error\(|\bnew\s+\w*Error\(|console\.\w+\(|\bimport\b[^;]*\bfrom\s*|\bimport\(|querySelector(?:All)?\(|closest\(|matches\(|\bError\(|RegExp\()\s*$/u;

// Identificadores, cabeceras HTTP, zonas horarias, selectores y fragmentos de
// atributos: no son texto para personas.
const TECNICA = /^(?:[A-Za-z]*[a-z][A-Z][A-Za-z0-9]*|[A-Z]{2,}[a-z][A-Za-z]*|[A-Za-z0-9+/_-]{24,}|noopener noreferrer|(?:[a-z0-9]+(?:-{1,2}[a-z0-9]*)+\s*)+|[a-z]+(?:[A-Z][a-z0-9]*)+|[A-Z][a-z]+(?:[A-Z][a-z]+)+|[A-Z][A-Za-z]*(?:-[A-Z][A-Za-z]*)+|[A-Z][a-z]+\/[A-Z][a-z_]+|[.#[].*|.*(?:=|aria-|\bdisabled\b|\brequired\b|\binert\b).*|ETag|Escape|Fetch|[A-Za-z]+(?:\.[A-Za-z]+)+)$/u;

/**
 * Cadenas con aspecto de texto humano fuera de catálogos y de contextos
 * técnicos (errores de programación lanzados, selectores CSS, consola,
 * importaciones). Son candidatas a texto visible sin traducir.
 */
export function cadenasHumanas(fuente) {
  const catalogos = rangosCatalogo(fuente);
  const enCatalogo = (literal) => catalogos.some(([a, b]) => literal.inicio >= a && literal.fin <= b + 1);
  const hallazgos = [];
  for (const literal of extraerLiterales(fuente)) {
    if (enCatalogo(literal)) continue;
    const texto = literal.texto;
    if (/<[a-z][\w-]*[\s>/]|<\/[a-z]/u.test(texto)) continue;
    const limpio = texto.replaceAll(HUECO, " ");
    if (!HUMANO.test(limpio) || TECNICA.test(limpio.trim())) continue;
    const antes = fuente.slice(Math.max(0, literal.inicio - 120), literal.inicio);
    if (CONTEXTO_TECNICO.test(antes)) continue;
    hallazgos.push({ linea: literal.linea, clase: "cadena", texto: limpio.slice(0, 90), literal });
  }
  return hallazgos;
}

const TEXTO_I18N = /\bdata-(?:i18n-portal|i18n|ct-copia|i18n-ayuda(?:-[a-z]+)*)="/u;
const ATRIBUTO_I18N = (atributo) => new RegExp(`\\bdata-(?:i18n-portal|ct-copia)-${atributo}="|\\bdata-i18n-label="`, "u");

/**
 * Hallazgos de una página HTML estática: texto o atributos accesibles con
 * texto literal en un elemento que no declara la clave i18n que lo sustituye
 * al cargar (`data-i18n-portal`, `data-i18n-portal-aria-label`…).
 */
export function hallazgosHTML(fuente) {
  const limpio = fuente.replace(/<!--[\s\S]*?-->/gu, "").replace(/<(script|style)\b[\s\S]*?<\/\1>/gu, "");
  const hallazgos = [];
  const linea = (indice) => limpio.slice(0, indice).split("\n").length;
  for (const etiqueta of limpio.matchAll(/<([a-z][\w-]*)\b([^>]*)>/gu)) {
    const atributos = etiqueta[2];
    for (const [, atributo, , valor] of atributos.matchAll(ATRIBUTO)) {
      if (LETRAS.test(valor) && !ATRIBUTO_I18N(atributo).test(atributos)) hallazgos.push({ linea: linea(etiqueta.index), clase: "atributo", texto: `${atributo}="${valor}"` });
    }
    const resto = limpio.slice(etiqueta.index + etiqueta[0].length);
    const texto = resto.slice(0, resto.search(/<|$/u)).trim();
    if (LETRAS.test(texto) && !TEXTO_I18N.test(atributos)) hallazgos.push({ linea: linea(etiqueta.index), clase: "texto", texto });
  }
  return hallazgos;
}
