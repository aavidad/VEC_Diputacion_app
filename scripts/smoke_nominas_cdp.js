"use strict";

const childProcess = require("node:child_process");
const crypto = require("node:crypto");
const fs = require("node:fs");
const http = require("node:http");
const net = require("node:net");
const path = require("node:path");
const { EventEmitter } = require("node:events");

class SmokeError extends Error {
  constructor(code, message, extra = {}) {
    super(message);
    this.name = "SmokeError";
    this.code = code;
    this.extra = extra;
  }
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function normalizeText(value) {
  return String(value || "")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/\s+/g, " ")
    .trim();
}

function normalizedKey(value) {
  return normalizeText(value).replace(/[^a-z0-9]+/g, " ").trim();
}

function slug(value) {
  return normalizeText(value).replace(/[^a-z0-9]+/g, "_").replace(/^_|_$/g, "");
}

function commandCandidates() {
  return [
    process.env.CHROME_PATH,
    process.env.CHROMIUM_PATH,
    process.env.BROWSER_PATH,
    "chromium",
    "chromium-browser",
    "google-chrome-stable",
    "google-chrome",
    "chrome",
    "microsoft-edge",
  ].filter(Boolean);
}

function isExecutable(filePath) {
  try {
    fs.accessSync(filePath, fs.constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

function findOnPath(command) {
  if (command.includes(path.sep)) {
    return isExecutable(command) ? command : "";
  }
  for (const dir of String(process.env.PATH || "").split(path.delimiter)) {
    if (!dir) continue;
    const candidate = path.join(dir, command);
    if (isExecutable(candidate)) return candidate;
  }
  return "";
}

function findChromium() {
  const candidates = commandCandidates();
  for (const candidate of candidates) {
    const executable = findOnPath(candidate);
    if (executable) return { executable, candidates };
  }
  throw new SmokeError(
    "CHROMIUM_NOT_FOUND",
    "No se encontro Chromium/Chrome. Instala chromium o define CHROME_PATH/CHROMIUM_PATH con la ruta del binario.",
    { candidates },
  );
}

function requestText(urlString, options = {}) {
  const url = new URL(urlString);
  if (url.protocol !== "http:") {
    return Promise.reject(new SmokeError("UNSUPPORTED_URL", `Solo se soporta HTTP en este smoke: ${urlString}`));
  }

  return new Promise((resolve, reject) => {
    const req = http.request(
      url,
      {
        method: options.method || "GET",
        headers: options.headers || {},
        timeout: options.timeoutMs || 5000,
      },
      (res) => {
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => {
          resolve({
            statusCode: res.statusCode || 0,
            headers: res.headers,
            body: Buffer.concat(chunks).toString("utf8"),
          });
        });
      },
    );

    req.on("timeout", () => req.destroy(new Error(`Timeout HTTP tras ${options.timeoutMs || 5000} ms`)));
    req.on("error", reject);
    if (options.body) req.write(options.body);
    req.end();
  });
}

async function requestJSON(urlString, options = {}) {
  const response = await requestText(urlString, options);
  if (response.statusCode < 200 || response.statusCode >= 300) {
    throw new SmokeError("HTTP_ERROR", `HTTP ${response.statusCode} en ${urlString}`, {
      statusCode: response.statusCode,
      body: response.body.slice(0, 500),
    });
  }
  try {
    return JSON.parse(response.body);
  } catch (error) {
    throw new SmokeError("INVALID_JSON", `Respuesta JSON invalida en ${urlString}`, {
      cause: error.message,
      body: response.body.slice(0, 500),
    });
  }
}

async function assertAppReachable(targetURL) {
  const url = new URL(targetURL);
  url.hash = "";
  try {
    const response = await requestText(url.href, { timeoutMs: 4000 });
    if (response.statusCode < 200 || response.statusCode >= 400) {
      throw new SmokeError("APP_UNAVAILABLE", `La app responde HTTP ${response.statusCode} en ${url.href}`, {
        statusCode: response.statusCode,
      });
    }
  } catch (error) {
    if (error instanceof SmokeError) throw error;
    throw new SmokeError(
      "APP_UNAVAILABLE",
      `No se pudo conectar a ${url.href}. Arranca el servidor antes de ejecutar este smoke.`,
      { cause: error.message },
    );
  }
}

function getFreePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      const port = address && typeof address === "object" ? address.port : 0;
      server.close(() => resolve(port));
    });
  });
}

function startChromium(executable, port, userDataDir) {
  const args = [
    "--remote-debugging-address=127.0.0.1",
    `--remote-debugging-port=${port}`,
    `--user-data-dir=${userDataDir}`,
    "--headless=new",
    "--disable-gpu",
    "--disable-background-networking",
    "--disable-default-apps",
    "--disable-dev-shm-usage",
    "--no-first-run",
    "--no-default-browser-check",
    "--no-sandbox",
    "--window-size=1440,1000",
    "about:blank",
  ];
  const chrome = childProcess.spawn(executable, args, { stdio: ["ignore", "ignore", "pipe"] });
  chrome.stderrText = "";
  chrome.stderr.on("data", (chunk) => {
    chrome.stderrText = `${chrome.stderrText}${chunk.toString("utf8")}`.slice(-12000);
  });
  return chrome;
}

async function waitForDevTools(port, chrome, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let lastError = null;
  while (Date.now() < deadline) {
    if (chrome.exitCode !== null) {
      throw new SmokeError("CHROMIUM_LAUNCH_FAILED", "Chromium salio antes de abrir el puerto CDP.", {
        exitCode: chrome.exitCode,
      });
    }
    try {
      const version = await requestJSON(`http://127.0.0.1:${port}/json/version`, { timeoutMs: 1000 });
      if (version.webSocketDebuggerUrl) return version;
    } catch (error) {
      lastError = error;
    }
    await delay(100);
  }
  throw new SmokeError("CDP_UNAVAILABLE", "Chromium no publico el endpoint CDP a tiempo.", {
    port,
    cause: lastError ? lastError.message : "",
  });
}

function parseHttpHeaders(headerText) {
  const lines = headerText.split(/\r?\n/);
  const statusLine = lines.shift() || "";
  const headers = {};
  for (const line of lines) {
    const index = line.indexOf(":");
    if (index === -1) continue;
    headers[line.slice(0, index).trim().toLowerCase()] = line.slice(index + 1).trim();
  }
  return { statusLine, headers };
}

class CDPSocket extends EventEmitter {
  constructor(wsURL, timeoutMs) {
    super();
    this.wsURL = wsURL;
    this.timeoutMs = timeoutMs;
    this.socket = null;
    this.buffer = Buffer.alloc(0);
    this.fragments = [];
    this.nextId = 1;
    this.pending = new Map();
    this.closed = false;
  }

  connect() {
    const url = new URL(this.wsURL);
    const key = crypto.randomBytes(16).toString("base64");
    const accept = crypto
      .createHash("sha1")
      .update(`${key}258EAFA5-E914-47DA-95CA-C5AB0DC85B11`)
      .digest("base64");

    return new Promise((resolve, reject) => {
      const socket = net.connect(Number(url.port || 80), url.hostname);
      let handshakeBuffer = Buffer.alloc(0);
      let settled = false;
      const fail = (error) => {
        if (settled) return;
        settled = true;
        socket.destroy();
        reject(error);
      };

      socket.on("connect", () => {
        const pathAndQuery = `${url.pathname}${url.search}`;
        const request = [
          `GET ${pathAndQuery} HTTP/1.1`,
          `Host: ${url.host}`,
          "Upgrade: websocket",
          "Connection: Upgrade",
          `Sec-WebSocket-Key: ${key}`,
          "Sec-WebSocket-Version: 13",
          "",
          "",
        ].join("\r\n");
        socket.write(request);
      });

      const onHandshakeData = (chunk) => {
        handshakeBuffer = Buffer.concat([handshakeBuffer, chunk]);
        const headerEnd = handshakeBuffer.indexOf("\r\n\r\n");
        if (headerEnd === -1) return;

        socket.removeListener("data", onHandshakeData);
        socket.removeListener("error", fail);

        const headerText = handshakeBuffer.slice(0, headerEnd).toString("utf8");
        const remaining = handshakeBuffer.slice(headerEnd + 4);
        const { statusLine, headers } = parseHttpHeaders(headerText);

        if (!statusLine.includes(" 101 ")) {
          fail(new SmokeError("CDP_HANDSHAKE_FAILED", `Upgrade WebSocket fallo: ${statusLine}`));
          return;
        }
        if (headers["sec-websocket-accept"] !== accept) {
          fail(new SmokeError("CDP_HANDSHAKE_FAILED", "Sec-WebSocket-Accept invalido."));
          return;
        }

        this.socket = socket;
        this.socket.on("data", (dataChunk) => this.onData(dataChunk));
        this.socket.on("error", (error) => this.onError(error));
        this.socket.on("close", () => this.onClose());

        settled = true;
        resolve();
        if (remaining.length) this.onData(remaining);
      };

      socket.on("data", onHandshakeData);
      socket.on("error", fail);
    });
  }

  command(method, params = {}) {
    if (this.closed || !this.socket) {
      return Promise.reject(new SmokeError("CDP_CLOSED", "La conexion CDP esta cerrada."));
    }
    const id = this.nextId++;
    const payload = { id, method };
    if (params && Object.keys(params).length) payload.params = params;
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new SmokeError("CDP_COMMAND_TIMEOUT", `Timeout CDP en ${method}.`, { method }));
      }, this.timeoutMs);
      this.pending.set(id, { resolve, reject, timer, method });
      this.sendText(JSON.stringify(payload));
    });
  }

  sendText(text) {
    this.sendFrame(0x1, Buffer.from(text, "utf8"));
  }

  sendFrame(opcode, payload) {
    if (!this.socket || this.closed) return;
    const length = payload.length;
    let header;
    if (length < 126) {
      header = Buffer.alloc(2);
      header[1] = 0x80 | length;
    } else if (length < 65536) {
      header = Buffer.alloc(4);
      header[1] = 0x80 | 126;
      header.writeUInt16BE(length, 2);
    } else {
      header = Buffer.alloc(10);
      header[1] = 0x80 | 127;
      header.writeBigUInt64BE(BigInt(length), 2);
    }
    header[0] = 0x80 | opcode;
    const mask = crypto.randomBytes(4);
    const masked = Buffer.alloc(length);
    for (let index = 0; index < length; index += 1) {
      masked[index] = payload[index] ^ mask[index % 4];
    }
    this.socket.write(Buffer.concat([header, mask, masked]));
  }

  onData(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);
    while (this.buffer.length >= 2) {
      const first = this.buffer[0];
      const second = this.buffer[1];
      const fin = Boolean(first & 0x80);
      const opcode = first & 0x0f;
      const masked = Boolean(second & 0x80);
      let length = second & 0x7f;
      let offset = 2;

      if (length === 126) {
        if (this.buffer.length < offset + 2) return;
        length = this.buffer.readUInt16BE(offset);
        offset += 2;
      } else if (length === 127) {
        if (this.buffer.length < offset + 8) return;
        const bigLength = this.buffer.readBigUInt64BE(offset);
        if (bigLength > BigInt(Number.MAX_SAFE_INTEGER)) {
          this.onError(new SmokeError("CDP_WS_FRAME_TOO_LARGE", "Frame WebSocket demasiado grande."));
          return;
        }
        length = Number(bigLength);
        offset += 8;
      }

      let mask = null;
      if (masked) {
        if (this.buffer.length < offset + 4) return;
        mask = this.buffer.slice(offset, offset + 4);
        offset += 4;
      }
      if (this.buffer.length < offset + length) return;

      let payload = this.buffer.slice(offset, offset + length);
      this.buffer = this.buffer.slice(offset + length);
      if (mask) {
        const unmasked = Buffer.alloc(payload.length);
        for (let index = 0; index < payload.length; index += 1) {
          unmasked[index] = payload[index] ^ mask[index % 4];
        }
        payload = unmasked;
      }
      this.onFrame(opcode, fin, payload);
    }
  }

  onFrame(opcode, fin, payload) {
    if (opcode === 0x8) {
      this.close();
      return;
    }
    if (opcode === 0x9) {
      this.sendFrame(0xA, payload);
      return;
    }
    if (opcode === 0xA) return;
    if (opcode === 0x1) {
      if (fin) {
        this.onMessage(payload.toString("utf8"));
      } else {
        this.fragments = [payload];
      }
      return;
    }
    if (opcode === 0x0) {
      this.fragments.push(payload);
      if (fin) {
        this.onMessage(Buffer.concat(this.fragments).toString("utf8"));
        this.fragments = [];
      }
    }
  }

  onMessage(text) {
    let message;
    try {
      message = JSON.parse(text);
    } catch (error) {
      this.emit("protocolError", error);
      return;
    }
    if (message.id !== undefined && this.pending.has(message.id)) {
      const pending = this.pending.get(message.id);
      this.pending.delete(message.id);
      clearTimeout(pending.timer);
      if (message.error) {
        pending.reject(new SmokeError("CDP_COMMAND_FAILED", `CDP ${pending.method}: ${message.error.message}`, {
          method: pending.method,
          error: message.error,
        }));
      } else {
        pending.resolve(message.result);
      }
      return;
    }
    this.emit("event", message);
    if (message.method) this.emit(message.method, message.params || {});
  }

  onError(error) {
    for (const [, pending] of this.pending) {
      clearTimeout(pending.timer);
      pending.reject(error);
    }
    this.pending.clear();
    this.emit("error", error);
  }

  onClose() {
    this.closed = true;
    for (const [, pending] of this.pending) {
      clearTimeout(pending.timer);
      pending.reject(new SmokeError("CDP_CLOSED", "La conexion CDP se cerro."));
    }
    this.pending.clear();
  }

  close() {
    if (this.closed) return;
    this.closed = true;
    try {
      this.sendFrame(0x8, Buffer.alloc(0));
    } catch {
      // Best effort close.
    }
    if (this.socket) this.socket.destroy();
    this.onClose();
  }
}

async function createPage(port) {
  const page = await requestJSON(`http://127.0.0.1:${port}/json/new?${encodeURIComponent("about:blank")}`, {
    method: "PUT",
    timeoutMs: 5000,
  });
  if (!page.webSocketDebuggerUrl) {
    throw new SmokeError("CDP_TARGET_FAILED", "Chromium no devolvio webSocketDebuggerUrl para la pagina.", { page });
  }
  return page;
}

function runtimeExceptionMessage(exceptionDetails) {
  if (!exceptionDetails) return "";
  return (
    exceptionDetails.exception?.description ||
    exceptionDetails.exception?.value ||
    exceptionDetails.text ||
    "Excepcion Runtime.evaluate"
  );
}

async function evaluate(cdp, expression, options = {}) {
  const result = await cdp.command("Runtime.evaluate", {
    expression,
    awaitPromise: true,
    returnByValue: options.returnByValue !== false,
    userGesture: true,
  });
  if (result.exceptionDetails) {
    throw new SmokeError("BROWSER_EVALUATION_FAILED", runtimeExceptionMessage(result.exceptionDetails), {
      exceptionDetails: result.exceptionDetails,
      expression: expression.slice(0, 500),
    });
  }
  return result.result ? result.result.value : undefined;
}

async function waitForEval(cdp, expression, description, timeoutMs = 30000) {
  const deadline = Date.now() + timeoutMs;
  let lastError = null;
  while (Date.now() < deadline) {
    try {
      const value = await evaluate(cdp, expression);
      if (value) return value;
    } catch (error) {
      lastError = error;
    }
    await delay(125);
  }
  throw new SmokeError("WAIT_TIMEOUT", `Timeout esperando: ${description}`, {
    cause: lastError ? lastError.message : "",
  });
}

module.exports = {
  SmokeError,
  delay,
  normalizeText,
  normalizedKey,
  slug,
  commandCandidates,
  isExecutable,
  findOnPath,
  findChromium,
  requestText,
  requestJSON,
  assertAppReachable,
  getFreePort,
  startChromium,
  waitForDevTools,
  parseHttpHeaders,
  CDPSocket,
  createPage,
  runtimeExceptionMessage,
  evaluate,
  waitForEval,
};
