#!/usr/bin/env node
"use strict";

const childProcess = require("node:child_process");
const crypto = require("node:crypto");
const fs = require("node:fs");
const http = require("node:http");
const net = require("node:net");
const os = require("node:os");
const path = require("node:path");
const { EventEmitter } = require("node:events");

const TARGET_URL = process.env.SMOKE_NOMINAS_URL || "http://127.0.0.1:18180/#nominas";
const DEFAULT_TIMEOUT_MS = Number(process.env.SMOKE_NOMINAS_TIMEOUT_MS || 30000);
const CDP_COMMAND_TIMEOUT_MS = Number(process.env.SMOKE_NOMINAS_CDP_TIMEOUT_MS || 15000);

const OPERATIONAL_SCREEN_SPECS = [
  {
    label: "Capitulo I, RPT y plazas",
    texts: ["Organizacion, RPT y presupuestos", "Capitulo I", "Aplicaciones presupuestarias", "Puesto RPT"],
  },
  {
    label: "Expediente empleado publico",
    texts: ["Expediente empleado publico", "Regimen", "Puesto / plaza", "Trienios"],
  },
  {
    label: "Resumen y estadisticas",
    texts: ["Resumen operativo de nominas", "Estado de plantilla"],
  },
  {
    label: "Checklist y avisos",
    texts: ["Checklist y tablero de avisos", "Gestiones pendientes"],
  },
  {
    label: "Contratos y vencimientos",
    texts: ["Contratos, vencimientos", "Proximos vencimientos"],
  },
  {
    label: "Incapacidades y ausencias",
    texts: ["Incapacidades, ausencias", "IT comun"],
  },
  {
    label: "Bajas por areas",
    texts: ["Estadisticas de bajas por areas", "Area / servicio", "Personas actualmente en baja"],
  },
  {
    label: "Portal empleado Peoplenet",
    texts: ["Portal empleado Peoplenet", "Ultimos recibos", "Certificados"],
  },
];

const HR_FUNCTIONAL_SCREEN_SPECS = [
  {
    label: "Inspector de nomina",
    texts: ["Inspector de nomina", "Controles automaticos", "SLD", "Ejecutar inspector"],
  },
  {
    label: "Tablas, valores y conceptos",
    texts: ["Pagas, tablas, valores y conceptos", "Tablas y valores", "Conceptos"],
  },
  {
    label: "Retroactividad y revision salarial",
    texts: ["Retroactividad y revision salarial", "Revision salarial", "Atrasos"],
  },
];

const SENSITIVE_HR_SCREEN_SPECS = [
  {
    label: "Cotizacion RED/SLD",
    texts: ["Comunicaciones AFI", "SLD", "SILTRA", "Liquidacion directa"],
  },
  {
    label: "IRPF cotizacion y acumulados",
    texts: ["IRPF cotizacion y acumulados", "IRPF", "Acumulados"],
  },
  {
    label: "Pagos y remesas",
    texts: ["Pagos y remesas", "Remesas", "SEPA"],
  },
  {
    label: "Informes certificados y 190",
    texts: ["Informes", "Modelo 190", "AEAT"],
  },
  {
    label: "Prestamos embargos y fondo social",
    texts: ["Prestamos", "Embargos", "Fondo social"],
  },
];

const SERVICE_USERS_SCREEN_SPEC = {
  label: "Centro de servicio y usuarios",
  texts: ["Centro de servicio y usuarios", "Usuarios"],
};

const FULL_HR_SCREEN_SPECS = [
  ...HR_FUNCTIONAL_SCREEN_SPECS,
  ...OPERATIONAL_SCREEN_SPECS,
  ...SENSITIVE_HR_SCREEN_SPECS,
  SERVICE_USERS_SCREEN_SPEC,
];

const ADMIN_GLOBAL_BUTTONS = FULL_HR_SCREEN_SPECS.map((spec) => spec.label).filter((label) => label !== "Portal empleado Peoplenet");
const EMPLOYEE_BUTTONS = [
  "Portal empleado Peoplenet",
  "Nomina mensual",
  "Historico y evolucion",
  "Certificado retenciones 10T",
];
const EMPLOYEE_PAYROLL_SIMULATION_LABELS = ["IRPF editable", "Trienios", "Complemento", "Productividad"];

const {
  SmokeError,
  delay,
  normalizeText,
  normalizedKey,
  slug,
  findChromium,
  requestText,
  assertAppReachable,
  getFreePort,
  startChromium,
  waitForDevTools,
  CDPSocket,
  createPage,
  evaluate,
  waitForEval,
} = require("./smoke_nominas_cdp.js");

function browserNormalizeSource() {
  return `function normalize(value) {
    return String(value || "")
      .normalize("NFD")
      .replace(/[\\u0300-\\u036f]/g, "")
      .toLowerCase()
      .replace(/\\s+/g, " ")
      .trim();
  }
  function normalizedKey(value) {
    return normalize(value).replace(/[^a-z0-9]+/g, " ").trim();
  }
  function textMatches(actual, expected) {
    return normalize(actual) === normalize(expected) || normalizedKey(actual) === normalizedKey(expected);
  }`;
}

async function pageState(cdp) {
  return evaluate(cdp, `(() => {
    ${browserNormalizeSource()}
    function isVisible(element) {
      const style = window.getComputedStyle(element);
      return style.visibility !== "hidden" && style.display !== "none" && element.getClientRects().length > 0;
    }
    const rawText = document.body ? document.body.innerText : "";
    const buttons = Array.from(document.querySelectorAll("button"))
      .filter(isVisible)
      .map((button) => normalize(button.textContent))
      .filter(Boolean);
    return {
      hash: window.location.hash,
      selectedUser: document.querySelector("#demo-user-select")?.value || "",
      status: normalize(document.querySelector("#connection-status")?.textContent || ""),
      title: document.title,
      text: normalize(rawText),
      buttons,
      buttonCount: buttons.length,
    };
  })()`);
}

async function switchRole(cdp, roleID) {
  const expectedTitle = roleID === "empleado"
    ? "portal del empleado - consulta de nominas"
    : roleID === "administrativo"
      ? "gestion operativa de personal y nominas"
      : "control de nominas peoplenet aapp";
  const result = await evaluate(cdp, `(async () => {
    if (typeof applyDemoUser !== "function" || typeof setActiveModule !== "function") {
      return { ok: false, reason: "Funciones globales applyDemoUser/setActiveModule no disponibles" };
    }
    await applyDemoUser(${JSON.stringify(roleID)}, { reload: true, switchModule: false });
    setActiveModule("nominas");
    return {
      ok: true,
      hash: window.location.hash,
      selectedUser: document.querySelector("#demo-user-select")?.value || "",
      status: document.querySelector("#connection-status")?.textContent || ""
    };
  })()`);

  if (!result || !result.ok) {
    throw new SmokeError("ROLE_SWITCH_FAILED", `No se pudo cambiar al rol ${roleID}.`, result || {});
  }

  await waitForEval(cdp, `(() => {
    ${browserNormalizeSource()}
    const text = normalize(document.body ? document.body.innerText : "");
    return document.querySelector("#demo-user-select")?.value === ${JSON.stringify(roleID)}
      && window.location.hash === "#nominas"
      && text.includes(${JSON.stringify(expectedTitle)});
  })()`, `nominas renderizado para ${roleID}`, 10000);
}

function hasButton(state, label) {
  const wanted = normalizeText(label);
  const wantedKey = normalizedKey(label);
  return state.buttons.some((button) => button === wanted || normalizedKey(button) === wantedKey);
}

function hasText(state, text) {
  return state.text.includes(normalizeText(text)) || normalizedKey(state.text).includes(normalizedKey(text));
}

function buttonCheck(name, role, state, requiredButtons) {
  const missing = requiredButtons.filter((label) => !hasButton(state, label));
  return {
    name,
    role,
    ok: missing.length === 0,
    kind: "buttons_visible",
    missing,
    required: requiredButtons,
    diagnostics: {
      hash: state.hash,
      status: state.status,
      button_count: state.buttonCount,
    },
  };
}

function forbiddenButtonCheck(name, role, state, forbiddenButtons) {
  const present = forbiddenButtons.filter((label) => hasButton(state, label));
  return {
    name,
    role,
    ok: present.length === 0,
    kind: "buttons_absent",
    present,
    forbidden: forbiddenButtons,
    diagnostics: {
      hash: state.hash,
      status: state.status,
      button_count: state.buttonCount,
    },
  };
}

function roleScreenSpecs(role) {
  if (role === "administrativo") return OPERATIONAL_SCREEN_SPECS;
  return FULL_HR_SCREEN_SPECS;
}

function roleForbiddenScreenSpecs(role) {
  if (role === "administrativo") return [...HR_FUNCTIONAL_SCREEN_SPECS, ...SENSITIVE_HR_SCREEN_SPECS, SERVICE_USERS_SCREEN_SPEC];
  return [];
}

function textCheck(name, role, state, requiredTexts) {
  const missing = requiredTexts.filter((text) => !hasText(state, text));
  return {
    name,
    role,
    ok: missing.length === 0,
    kind: "text_visible",
    missing,
    required: requiredTexts,
    diagnostics: {
      hash: state.hash,
      status: state.status,
      text_length: state.text.length,
    },
  };
}

async function clickButton(cdp, label) {
  return evaluate(cdp, `(() => {
    ${browserNormalizeSource()}
    function isVisible(element) {
      const style = window.getComputedStyle(element);
      return style.visibility !== "hidden" && style.display !== "none" && element.getClientRects().length > 0;
    }
    const wanted = ${JSON.stringify(label)};
    const buttons = Array.from(document.querySelectorAll("button")).filter(isVisible);
    const button = buttons.find((candidate) => textMatches(candidate.textContent, wanted));
    if (!button) {
      return {
        ok: false,
        reason: "Boton no encontrado",
        available: buttons.map((candidate) => normalize(candidate.textContent)).filter(Boolean).slice(0, 100)
      };
    }
    button.click();
    return { ok: true, clicked: normalize(button.textContent) };
  })()`);
}

async function maybeWaitForText(cdp, text, timeoutMs = 3000) {
  try {
    await waitForEval(cdp, `(() => {
      ${browserNormalizeSource()}
      const bodyText = document.body ? document.body.innerText : "";
      const wanted = ${JSON.stringify(text)};
      return normalize(bodyText).includes(normalize(wanted)) || normalizedKey(bodyText).includes(normalizedKey(wanted));
    })()`, `texto ${text}`, timeoutMs);
    return true;
  } catch {
    return false;
  }
}

async function employeePayrollNoSimulationControlsCheck(cdp) {
  const result = await evaluate(cdp, `(() => {
    ${browserNormalizeSource()}
    const forbiddenLabels = ${JSON.stringify(EMPLOYEE_PAYROLL_SIMULATION_LABELS)};
    const forbiddenKeys = forbiddenLabels.map(normalizedKey).filter(Boolean);
    const simulationKeys = ["simulador", "simulacion", "simular"];

    function isVisible(element) {
      const style = window.getComputedStyle(element);
      return style.visibility !== "hidden" && style.display !== "none" && element.getClientRects().length > 0;
    }

    function clip(value) {
      return normalize(value).slice(0, 180);
    }

    function matchesForbiddenLabel(value) {
      const key = normalizedKey(value);
      return forbiddenKeys.some((forbidden) => key.includes(forbidden));
    }

    function associatedText(element) {
      const values = [];
      if (element.labels) {
        for (const label of Array.from(element.labels)) {
          values.push(label.textContent || "");
        }
      }
      const wrappingLabel = element.closest("label");
      if (wrappingLabel) values.push(wrappingLabel.textContent || "");

      const labelledBy = element.getAttribute("aria-labelledby") || "";
      for (const id of labelledBy.split(/\\s+/).filter(Boolean)) {
        const label = document.getElementById(id);
        if (label) values.push(label.textContent || "");
      }

      values.push(
        element.getAttribute("aria-label") || "",
        element.getAttribute("placeholder") || "",
        element.getAttribute("name") || "",
        element.id || "",
      );

      return values.filter(Boolean).join(" ");
    }

    function nearestContextText(element) {
      const context = element.closest("[data-screen], [data-panel], section, article, form, fieldset, main");
      return context ? context.innerText || "" : "";
    }

    const labelViolations = Array.from(document.querySelectorAll("label"))
      .filter(isVisible)
      .map((label) => ({
        kind: "label",
        text: clip(label.textContent || ""),
      }))
      .filter((violation) => matchesForbiddenLabel(violation.text));

    const controlViolations = Array.from(document.querySelectorAll("input, select, textarea"))
      .filter(isVisible)
      .map((element) => {
        const tag = element.tagName.toLowerCase();
        const type = (element.getAttribute("type") || (tag === "input" ? "text" : tag)).toLowerCase();
        const role = normalizedKey(element.getAttribute("role") || "");
        const inputMode = normalizedKey(element.getAttribute("inputmode") || "");
        const isNumeric = tag === "input" && (type === "number" || type === "range" || role === "spinbutton" || inputMode === "numeric");
        const isSelect = tag === "select";
        if ((!isNumeric && !isSelect) || element.disabled || element.readOnly) return null;

        const label = associatedText(element);
        const context = nearestContextText(element);
        const descriptor = normalizedKey([label, context].join(" "));
        const isSimulationControl = simulationKeys.some((key) => descriptor.includes(key)) || matchesForbiddenLabel(label);
        if (!isSimulationControl) return null;

        return {
          kind: "control",
          tag,
          type,
          id: element.id || "",
          name: element.getAttribute("name") || "",
          label: clip(label),
          context: clip(context),
        };
      })
      .filter(Boolean);

    const violations = [...labelViolations, ...controlViolations];
    return {
      hash: window.location.hash,
      status: normalize(document.querySelector("#connection-status")?.textContent || ""),
      violationCount: violations.length,
      violations: violations.slice(0, 25),
    };
  })()`);

  return {
    name: "empleado_nomina_mensual_no_ve_controles_simulacion",
    role: "empleado",
    ok: result.violationCount === 0,
    kind: "simulation_controls_absent",
    forbidden: {
      labels: EMPLOYEE_PAYROLL_SIMULATION_LABELS,
      controls: ["input[type=number] de simulador", "select de simulador"],
    },
    present: result.violations,
    diagnostics: {
      hash: result.hash,
      status: result.status,
      violation_count: result.violationCount,
    },
  };
}

async function prepareBrowserPage(port, targetURL) {
  const page = await createPage(port);
  const cdp = new CDPSocket(page.webSocketDebuggerUrl, CDP_COMMAND_TIMEOUT_MS);
  cdp.on("error", () => {});
  await cdp.connect();
  await cdp.command("Page.enable");
  await cdp.command("Runtime.enable");
  await cdp.command("Page.navigate", { url: targetURL });
  await waitForEval(cdp, `(() => {
    return typeof applyDemoUser === "function"
      && typeof setActiveModule === "function"
      && Boolean(document.querySelector("#demo-user-select"));
  })()`, "bootstrap de la app VEC", DEFAULT_TIMEOUT_MS);
  await waitForEval(cdp, `(() => {
    ${browserNormalizeSource()}
    const status = normalize(document.querySelector("#connection-status")?.textContent || "");
    return status.includes("vec conectado");
  })()`, "estado VEC conectado", DEFAULT_TIMEOUT_MS);
  return { page, cdp };
}

async function checkRoleScreens(cdp, role) {
  const checks = [];
  const requiredSpecs = roleScreenSpecs(role);
  const forbiddenSpecs = roleForbiddenScreenSpecs(role);
  await switchRole(cdp, role);
  const menuState = await pageState(cdp);
  checks.push(buttonCheck(`${role}_ve_pantallas_nominas_peoplenet`, role, menuState, requiredSpecs.map((spec) => spec.label)));
  if (forbiddenSpecs.length) {
    checks.push(forbiddenButtonCheck(
      `${role}_no_ve_pantallas_rrhh_pleno`,
      role,
      menuState,
      forbiddenSpecs.map((spec) => spec.label),
    ));
  }

  for (const spec of requiredSpecs) {
    const click = await clickButton(cdp, spec.label);
    const rendered = click.ok ? await maybeWaitForText(cdp, spec.texts[0]) : false;
    const state = await pageState(cdp);
    const textResult = textCheck(`${role}_${slug(spec.label)}_textos`, role, state, spec.texts);
    checks.push({
      ...textResult,
      ok: Boolean(click.ok && rendered && textResult.ok),
      kind: "screen_render",
      click,
      missing: [
        ...(!rendered ? [spec.texts[0]] : []),
        ...textResult.missing.filter((item) => item !== spec.texts[0]),
      ],
    });
  }
  return checks;
}

async function checkEmployee(cdp) {
  const checks = [];
  await switchRole(cdp, "empleado");
  let state = await pageState(cdp);
  checks.push(forbiddenButtonCheck("empleado_no_ve_pantallas_admin_globales", "empleado", state, ADMIN_GLOBAL_BUTTONS));
  checks.push(buttonCheck("empleado_ve_portal_recibos_certificados", "empleado", state, EMPLOYEE_BUTTONS));
  checks.push(textCheck("empleado_ve_portal_peoplenet", "empleado", state, [
    "Portal empleado Peoplenet",
    "Ultimos recibos",
    "Certificados",
  ]));

  const payrollClick = await clickButton(cdp, "Nomina mensual");
  const payrollRendered = payrollClick.ok ? await maybeWaitForText(cdp, "Recibo de salarios") : false;
  state = await pageState(cdp);
  checks.push({
    name: "empleado_ve_nomina_mensual",
    role: "empleado",
    ok: Boolean(payrollClick.ok && payrollRendered && hasText(state, "Liquido a recibir") && hasButton(state, "Imprimir PDF")),
    kind: "screen_render",
    click: payrollClick,
    required: ["Recibo de salarios", "Liquido a recibir", "Imprimir PDF"],
    missing: [
      ...(!payrollRendered ? ["Recibo de salarios"] : []),
      ...(!hasText(state, "Liquido a recibir") ? ["Liquido a recibir"] : []),
      ...(!hasButton(state, "Imprimir PDF") ? ["Imprimir PDF"] : []),
    ],
    diagnostics: { hash: state.hash, status: state.status, button_count: state.buttonCount },
  });
  checks.push(await employeePayrollNoSimulationControlsCheck(cdp));

  const historyClick = await clickButton(cdp, "Historico y evolucion");
  const historyRendered = historyClick.ok ? await maybeWaitForText(cdp, "Historico de Recibos de Nomina") : false;
  state = await pageState(cdp);
  checks.push({
    name: "empleado_ve_recibos_historicos",
    role: "empleado",
    ok: Boolean(historyClick.ok && historyRendered && hasButton(state, "Ver Recibo")),
    kind: "screen_render",
    click: historyClick,
    required: ["Historico de Recibos de Nomina", "Ver Recibo"],
    missing: [
      ...(!historyRendered ? ["Historico de Recibos de Nomina"] : []),
      ...(!hasButton(state, "Ver Recibo") ? ["Ver Recibo"] : []),
    ],
    diagnostics: { hash: state.hash, status: state.status, button_count: state.buttonCount },
  });

  const certClick = await clickButton(cdp, "Certificado retenciones 10T");
  const certRendered = certClick.ok ? await maybeWaitForText(cdp, "Certificado de Retenciones") : false;
  state = await pageState(cdp);
  checks.push({
    name: "empleado_ve_certificados",
    role: "empleado",
    ok: Boolean(certClick.ok && certRendered && hasButton(state, "Descargar certificado firmado")),
    kind: "screen_render",
    click: certClick,
    required: ["Certificado de Retenciones", "Descargar certificado firmado"],
    missing: [
      ...(!certRendered ? ["Certificado de Retenciones"] : []),
      ...(!hasButton(state, "Descargar certificado firmado") ? ["Descargar certificado firmado"] : []),
    ],
    diagnostics: { hash: state.hash, status: state.status, button_count: state.buttonCount },
  });

  return checks;
}

async function runChecks(cdp) {
  return [
    ...(await checkRoleScreens(cdp, "administrador")),
    ...(await checkRoleScreens(cdp, "tecnico_rrhh")),
    ...(await checkRoleScreens(cdp, "administrativo")),
    ...(await checkEmployee(cdp)),
  ];
}

function summarize(checks) {
  const failed = checks.filter((check) => !check.ok);
  return {
    total: checks.length,
    passed: checks.length - failed.length,
    failed: failed.length,
    failed_checks: failed.map((check) => check.name),
  };
}

async function closeTarget(port, page) {
  if (!page || !page.id) return;
  try {
    await requestText(`http://127.0.0.1:${port}/json/close/${encodeURIComponent(page.id)}`, { timeoutMs: 1000 });
  } catch {
    // Best effort target close.
  }
}

async function stopChromium(chrome) {
  if (!chrome || chrome.exitCode !== null) return;
  chrome.kill("SIGTERM");
  await Promise.race([
    new Promise((resolve) => chrome.once("exit", resolve)),
    delay(2000).then(() => {
      if (chrome.exitCode === null) chrome.kill("SIGKILL");
    }),
  ]);
}

function errorJSON(error, context = {}) {
  return {
    ok: false,
    url: TARGET_URL,
    ...context,
    error: {
      code: error.code || "UNEXPECTED_ERROR",
      message: error.message || String(error),
      ...(error.extra || {}),
    },
  };
}

async function main() {
  const { executable, candidates } = findChromium();
  await assertAppReachable(TARGET_URL);

  const port = await getFreePort();
  const userDataDir = fs.mkdtempSync(path.join(os.tmpdir(), "nominas-peoplenet-cdp-"));
  let chrome = null;
  let page = null;
  let cdp = null;
  let version = null;

  try {
    chrome = startChromium(executable, port, userDataDir);
    version = await waitForDevTools(port, chrome, DEFAULT_TIMEOUT_MS);
    const prepared = await prepareBrowserPage(port, TARGET_URL);
    page = prepared.page;
    cdp = prepared.cdp;
    const checks = await runChecks(cdp);
    const summary = summarize(checks);
    return {
      ok: summary.failed === 0,
      url: TARGET_URL,
      browser: {
        executable,
        version: version.Browser || "",
      },
      checks,
      summary,
      ...(summary.failed === 0
        ? {}
        : {
            error: {
              code: "SMOKE_ASSERTION_FAILED",
              message: `${summary.failed} checks fallaron.`,
              failed_checks: summary.failed_checks,
            },
          }),
    };
  } catch (error) {
    if (chrome && chrome.stderrText && error instanceof SmokeError && error.code === "CHROMIUM_LAUNCH_FAILED") {
      error.extra.stderr = chrome.stderrText;
    }
    return errorJSON(error, {
      browser: {
        executable,
        candidates,
        version: version?.Browser || "",
      },
    });
  } finally {
    if (cdp) cdp.close();
    await closeTarget(port, page);
    await stopChromium(chrome);
    try {
      fs.rmSync(userDataDir, { recursive: true, force: true });
    } catch {
      // Best effort temp cleanup.
    }
  }
}

main()
  .then((result) => {
    console.log(JSON.stringify(result, null, 2));
    process.exit(result.ok ? 0 : 1);
  })
  .catch((error) => {
    console.log(JSON.stringify(errorJSON(error), null, 2));
    process.exit(1);
  });
