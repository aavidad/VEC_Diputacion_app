/**
 * Adaptador documental exclusivo de presentación.
 *
 * Genera en memoria un PDF institucional no autoritativo. La superficie y el
 * descriptor son los mismos que consumirá el conector documental de servidor;
 * este archivo se elimina del artefacto productivo por contener
 * `presentacion` en su nombre.
 */
export function generarReciboPDFPresentacion(descriptor) {
  const datos = validarDescriptor(descriptor);
  const contenido = [
    "0.98 0.99 1 rg 36 36 523 770 re f",
    "0.78 0.84 0.88 RG 36 36 523 770 re S",
    "1 1 1 rg 45 736 505 72 re f",
    "0.72 0.80 0.88 RG 45 736 505 72 re S",
    "0.67 0.80 0.29 rg 45 736 505 4 re f",
    "0 G 0 g",
  ];
  dibujarLogoDiputacion(contenido, 58, 782, 0.7);
  contenido.push("0.09 0.23 0.31 rg");
  lineaPDF(contenido, 370, 787, "RECIBO DE ACTUACIÓN", { tamano: 10.5, negrita: true });
  lineaPDF(contenido, 370, 770, "Portal del Empleado - DEMO", { tamano: 8 });

  contenido.push("0.08 0.13 0.20 rg");
  lineaPDF(contenido, 58, 704, abreviar(datos.titulo, 62), { tamano: 13, negrita: true });
  lineaPDF(contenido, 58, 686, abreviar(datos.subtitulo, 78), { tamano: 9 });

  contenido.push("0.10 0.29 0.47 rg 58 642 474 24 re f", "1 1 1 rg");
  lineaPDF(contenido, 66, 650, "DATO", { tamano: 8.5, negrita: true });
  lineaPDF(contenido, 230, 650, "VALOR", { tamano: 8.5, negrita: true });
  let y = 610;
  datos.filas.forEach(([etiqueta, valor], indice) => {
    contenido.push(`${indice % 2 ? "0.94 0.97 0.95" : "0.97 0.99 1"} rg 58 ${y - 8} 474 24 re f`);
    contenido.push(`0.84 0.88 0.92 RG 58 ${y - 8} 474 24 re S`, "0.08 0.13 0.20 rg");
    lineaPDF(contenido, 66, y, abreviar(etiqueta, 26), { tamano: 8.6, negrita: true });
    lineaPDF(contenido, 230, y, abreviar(valor, 52), { tamano: 8.6 });
    y -= 28;
  });

  contenido.push("0.97 0.99 1 rg 58 270 474 100 re f", "0.72 0.80 0.88 RG 58 270 474 100 re S", "0.08 0.13 0.20 rg");
  lineaPDF(contenido, 66, 346, "CERTIFICA", { tamano: 10, negrita: true });
  partirTexto(datos.textoCertificacion, 94).slice(0, 5).forEach((linea, indice) => {
    lineaPDF(contenido, 66, 326 - indice * 13, linea, { tamano: 8.2 });
  });

  contenido.push("0.96 0.98 1 rg 58 174 310 70 re f", "0.72 0.80 0.88 RG 58 174 310 70 re S", "0.08 0.13 0.20 rg");
  lineaPDF(contenido, 66, 221, "Firma y verificación", { tamano: 10, negrita: true });
  lineaPDF(contenido, 66, 203, `Referencia: ${datos.referencia}`, { tamano: 8.5, negrita: true });
  lineaPDF(contenido, 66, 187, "Estado: documento de demostración sin validez administrativa", { tamano: 8.1 });
  lineaPDF(contenido, 66, 158, abreviar(datos.nota, 78), { tamano: 8 });
  dibujarQR(contenido, datos.urlVerificacion, 480, 54, 1.3);
  lineaPDF(contenido, 382, 98, "Comprobar documento", { tamano: 8, negrita: true });
  lineaPDF(contenido, 382, 84, abreviar(datos.referencia, 30), { tamano: 7 });

  return construirPDF(contenido.join("\n"));
}

export function descargarReciboPDFPresentacion(descriptor, entorno = globalThis) {
  const datos = validarDescriptor(descriptor);
  if (!entorno?.URL?.createObjectURL || !entorno?.document?.createElement) {
    throw new Error("el navegador no permite descargar el PDF");
  }
  const url = entorno.URL.createObjectURL(generarReciboPDFPresentacion(datos));
  const enlace = entorno.document.createElement("a");
  enlace.href = url;
  enlace.download = datos.nombreArchivo;
  enlace.hidden = true;
  entorno.document.body.append(enlace);
  enlace.click();
  enlace.remove();
  entorno.setTimeout(() => entorno.URL.revokeObjectURL(url), 0);
  return Object.freeze({ referencia: datos.referencia, nombre_archivo: datos.nombreArchivo,
    url_verificacion: datos.urlVerificacion, formato: "application/pdf", demostracion: true });
}

function validarDescriptor(descriptor) {
  if (!descriptor || typeof descriptor !== "object" || Array.isArray(descriptor)) throw new TypeError("descriptor documental no válido");
  const referencia = textoAcotado(descriptor.referencia, 8, 80, /^[A-Z0-9][A-Z0-9._:-]+$/);
  const titulo = textoAcotado(descriptor.titulo, 3, 120);
  const subtitulo = textoAcotado(descriptor.subtitulo || "Recibo emitido por el Portal del Empleado", 3, 160);
  const nota = textoAcotado(descriptor.nota || "El documento definitivo se generara y firmara en el servidor autorizado.", 3, 220);
  if (!Array.isArray(descriptor.filas) || descriptor.filas.length < 1 || descriptor.filas.length > 8) throw new TypeError("filas documentales no válidas");
  const filas = descriptor.filas.map((fila) => {
    if (!Array.isArray(fila) || fila.length !== 2) throw new TypeError("fila documental no válida");
    return Object.freeze([textoAcotado(fila[0], 1, 80), textoAcotado(fila[1], 1, 180)]);
  });
  const url = new URL(String(descriptor.urlVerificacion || ""), globalThis.location?.origin || "http://127.0.0.1");
  if (!/^https?:$/.test(url.protocol) || url.username || url.password || url.hash) throw new TypeError("URL de comprobación no válida");
  const nombreArchivo = String(descriptor.nombreArchivo || `recibo-${referencia.toLowerCase()}.pdf`);
  if (!/^[a-z0-9][a-z0-9._-]{1,119}\.pdf$/i.test(nombreArchivo)) throw new TypeError("nombre de PDF no válido");
  const textoCertificacion = textoAcotado(descriptor.textoCertificacion
    || "Se deja constancia de la actuación indicada y de su referencia de comprobación. El documento definitivo se emitirá desde el expediente administrativo autorizado.", 3, 430);
  return Object.freeze({ referencia, titulo, subtitulo, nota, filas: Object.freeze(filas),
    urlVerificacion: url.href, nombreArchivo, textoCertificacion });
}

function textoAcotado(valor, minimo, maximo, patron = null) {
  if (typeof valor !== "string") throw new TypeError("texto documental no válido");
  const texto = valor.trim();
  if (texto.length < minimo || texto.length > maximo || /[\u0000-\u001F\u007F]/.test(texto)
    || (patron && !patron.test(texto))) throw new TypeError("texto documental no válido");
  return texto;
}

function abreviar(valor, maximo) {
  const texto = String(valor || "");
  return texto.length > maximo ? `${texto.slice(0, maximo - 3)}...` : texto;
}

function partirTexto(valor, maximo) {
  const palabras = String(valor || "").trim().split(/\s+/u);
  const lineas = [];
  let linea = "";
  palabras.forEach((palabra) => {
    const candidata = linea ? `${linea} ${palabra}` : palabra;
    if (candidata.length <= maximo) {
      linea = candidata;
      return;
    }
    if (linea) lineas.push(linea);
    linea = palabra;
  });
  if (linea) lineas.push(linea);
  return lineas;
}

function textoComentarioPDF(valor) {
  return String(valor ?? "").normalize("NFD").replace(/[\u0300-\u036f]/g, "")
    .replace(/[^\x20-\x7E]/g, " ").replaceAll("%", " ");
}

function textoWinAnsiHex(valor) {
  const especiales = new Map([
    [0x20ac, 0x80], [0x201a, 0x82], [0x201e, 0x84], [0x2026, 0x85],
    [0x2018, 0x91], [0x2019, 0x92], [0x201c, 0x93], [0x201d, 0x94],
    [0x2013, 0x96], [0x2014, 0x97],
  ]);
  return Array.from(String(valor ?? "")).map((caracter) => {
    const codigo = caracter.codePointAt(0);
    const byte = especiales.get(codigo) ?? (codigo >= 0x20 && codigo <= 0xff ? codigo : 0x3f);
    return byte.toString(16).padStart(2, "0").toUpperCase();
  }).join("");
}

function lineaPDF(partes, x, y, texto, opciones = {}) {
  const tamano = opciones.tamano || 10;
  partes.push(`% ${textoComentarioPDF(texto)}\nBT /${opciones.negrita ? "F2" : "F1"} ${tamano} Tf ${x} ${y} Td <${textoWinAnsiHex(texto)}> Tj ET`);
}

function dibujarLogoDiputacion(partes, x, y, escala = 1) {
  const s = Number(escala || 1);
  const n = (valor) => valor.toFixed(3);
  const escalaMarca = 0.25 * s;
  const e = x - 106.13 * escalaMarca;
  const f = y + 212.04 * escalaMarca;
  partes.push(`q ${n(escalaMarca)} 0 0 ${n(-escalaMarca)} ${n(e)} ${n(f)} cm`);
  partes.push("0.67 0.80 0.29 rg 264.19 212.04 m 191.23 285.00 l 219.60 313.37 l 235.82 297.16 l 223.80 285.14 l 264.33 244.61 l 288.65 268.93 l 207.59 349.99 l 138.55 280.95 l 149.84 269.66 l 153.32 266.18 157.95 264.26 162.87 264.26 c 167.79 264.26 172.42 266.18 175.90 269.66 c 179.08 272.84 l 195.29 256.63 l 192.11 253.45 l 184.30 245.64 173.91 241.34 162.87 241.34 c 151.83 241.34 141.44 245.64 133.63 253.45 c 106.13 280.95 l 207.60 382.41 l 321.09 268.93 l 264.21 212.05 l h f Q");
  partes.push("0.09 0.23 0.31 rg");
  lineaPDF(partes, x + 62 * s, y + 11 * s, "Diputación de Granada", { tamano: 15 * s, negrita: true });
  lineaPDF(partes, x + 63 * s, y - 6 * s, "Área de Recursos Humanos y Régimen Interior", { tamano: 7.5 * s });
  partes.push(`0.67 0.80 0.29 rg ${n(x + 63 * s)} ${n(y - 12 * s)} ${n(156 * s)} ${n(2 * s)} re f`);
}

function multiplicarGF(x, y) {
  let z = 0;
  for (let i = 7; i >= 0; i--) {
    z = ((z << 1) ^ (((z >>> 7) & 1) * 0x11d)) & 0xff;
    z ^= ((y >>> i) & 1) * x;
  }
  return z;
}

function divisorReedSolomon(grado) {
  const resultado = Array(grado).fill(0);
  resultado[grado - 1] = 1;
  let raiz = 1;
  for (let i = 0; i < grado; i++) {
    for (let j = 0; j < resultado.length; j++) {
      resultado[j] = multiplicarGF(resultado[j], raiz);
      if (j + 1 < resultado.length) resultado[j] ^= resultado[j + 1];
    }
    raiz = multiplicarGF(raiz, 0x02);
  }
  return resultado;
}

function restoReedSolomon(datos, divisor) {
  const resultado = Array(divisor.length).fill(0);
  datos.forEach((valor) => {
    const factor = valor ^ resultado.shift();
    resultado.push(0);
    divisor.forEach((coeficiente, indice) => { resultado[indice] ^= multiplicarGF(coeficiente, factor); });
  });
  return resultado;
}

function agregarBits(bits, valor, longitud) {
  for (let i = longitud - 1; i >= 0; i--) bits.push((valor >>> i) & 1);
}

function bitsFormato(mascara) {
  const datos = (1 << 3) | mascara;
  let resto = datos;
  for (let i = 0; i < 10; i++) resto = (resto << 1) ^ (((resto >>> 9) & 1) * 0x537);
  return ((datos << 10) | (resto & 0x3ff)) ^ 0x5412;
}

function matrizQR(texto) {
  const version = 5;
  const tamano = version * 4 + 17;
  const bytesDatos = 108;
  const bytesCorreccion = 26;
  const bytes = Array.from(new TextEncoder().encode(String(texto)));
  if (bytes.length > bytesDatos - 2) throw new Error("la URL excede la capacidad del QR");
  const bits = [];
  agregarBits(bits, 0x4, 4);
  agregarBits(bits, bytes.length, 8);
  bytes.forEach((valor) => agregarBits(bits, valor, 8));
  agregarBits(bits, 0, Math.min(4, bytesDatos * 8 - bits.length));
  while (bits.length % 8 !== 0) bits.push(0);
  const datos = [];
  for (let i = 0; i < bits.length; i += 8) datos.push(bits.slice(i, i + 8).reduce((suma, bit) => (suma << 1) | bit, 0));
  for (let relleno = 0xec; datos.length < bytesDatos; relleno ^= 0xfd) datos.push(relleno);
  const palabras = [...datos, ...restoReedSolomon(datos, divisorReedSolomon(bytesCorreccion))];
  const modulos = Array.from({ length: tamano }, () => Array(tamano).fill(false));
  const reservados = Array.from({ length: tamano }, () => Array(tamano).fill(false));
  const fijar = (x, y, oscuro) => {
    if (x < 0 || y < 0 || x >= tamano || y >= tamano) return;
    modulos[y][x] = Boolean(oscuro);
    reservados[y][x] = true;
  };
  const localizador = (x, y) => {
    for (let dy = -1; dy <= 7; dy++) for (let dx = -1; dx <= 7; dx++) {
      const dentro = dx >= 0 && dx <= 6 && dy >= 0 && dy <= 6;
      fijar(x + dx, y + dy, dentro && (dx === 0 || dx === 6 || dy === 0 || dy === 6 || (dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4)));
    }
  };
  const formato = () => {
    const valor = bitsFormato(0);
    const bit = (indice) => ((valor >>> indice) & 1) !== 0;
    for (let i = 0; i <= 5; i++) fijar(8, i, bit(i));
    fijar(8, 7, bit(6)); fijar(8, 8, bit(7)); fijar(7, 8, bit(8));
    for (let i = 9; i < 15; i++) fijar(14 - i, 8, bit(i));
    for (let i = 0; i < 8; i++) fijar(tamano - 1 - i, 8, bit(i));
    for (let i = 8; i < 15; i++) fijar(8, tamano - 15 + i, bit(i));
    fijar(8, tamano - 8, true);
  };
  localizador(0, 0); localizador(tamano - 7, 0); localizador(0, tamano - 7);
  for (let i = 8; i < tamano - 8; i++) { fijar(6, i, i % 2 === 0); fijar(i, 6, i % 2 === 0); }
  for (let dy = -2; dy <= 2; dy++) for (let dx = -2; dx <= 2; dx++) fijar(30 + dx, 30 + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1);
  fijar(8, version * 4 + 9, true);
  formato();
  const datosBits = palabras.flatMap((valor) => Array.from({ length: 8 }, (_, indice) => (valor >>> (7 - indice)) & 1));
  let indiceBit = 0;
  let ascendente = true;
  for (let derecha = tamano - 1; derecha >= 1; derecha -= 2) {
    if (derecha === 6) derecha = 5;
    for (let vertical = 0; vertical < tamano; vertical++) {
      const y = ascendente ? tamano - 1 - vertical : vertical;
      for (let columna = 0; columna < 2; columna++) {
        const x = derecha - columna;
        if (!reservados[y][x]) modulos[y][x] = indiceBit < datosBits.length && datosBits[indiceBit++] === 1;
      }
    }
    ascendente = !ascendente;
  }
  for (let y = 0; y < tamano; y++) for (let x = 0; x < tamano; x++) {
    if (!reservados[y][x] && (x + y) % 2 === 0) modulos[y][x] = !modulos[y][x];
  }
  formato();
  return modulos;
}

function dibujarQR(partes, texto, x, y, modulo) {
  const matriz = matrizQR(texto);
  const margen = 4;
  const total = (matriz.length + margen * 2) * modulo;
  partes.push(`1 1 1 rg ${x} ${y} ${total} ${total} re f`, `0.72 0.80 0.88 RG ${x} ${y} ${total} ${total} re S`);
  const rectangulos = [];
  matriz.forEach((fila, numeroFila) => fila.forEach((oscuro, columna) => {
    if (oscuro) rectangulos.push(`${(x + (columna + margen) * modulo).toFixed(2)} ${(y + (matriz.length + margen - 1 - numeroFila) * modulo).toFixed(2)} ${modulo.toFixed(2)} ${modulo.toFixed(2)} re`);
  }));
  if (rectangulos.length) partes.push(`0 0 0 rg ${rectangulos.join(" ")} f`);
}

function construirPDF(contenido) {
  const objetos = [
    "<< /Type /Catalog /Pages 2 0 R >>",
    "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
    "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>",
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
    `<< /Length ${contenido.length} >>\nstream\n${contenido}\nendstream`,
  ];
  let pdf = "%PDF-1.4\n";
  const desplazamientos = [];
  objetos.forEach((objeto, indice) => {
    desplazamientos.push(pdf.length);
    pdf += `${indice + 1} 0 obj\n${objeto}\nendobj\n`;
  });
  const inicioReferencias = pdf.length;
  pdf += `xref\n0 ${objetos.length + 1}\n0000000000 65535 f \n`;
  pdf += desplazamientos.map((valor) => `${String(valor).padStart(10, "0")} 00000 n \n`).join("");
  pdf += `trailer\n<< /Size ${objetos.length + 1} /Root 1 0 R >>\nstartxref\n${inicioReferencias}\n%%EOF`;
  return new Blob([pdf], { type: "application/pdf" });
}
