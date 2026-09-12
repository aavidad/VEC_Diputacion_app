export const CORREO_ENDPOINT = '/api/vec/contratacion-temporal/configuracion-correo';

const CAMPOS = ['host', 'port', 'serverName', 'ca', 'sender', 'username', 'tlsMode', 'oauthMode', 'timeout'];

export function createCorreoClient({ fetchImpl = globalThis.fetch, endpoint = CORREO_ENDPOINT } = {}) {
  if (typeof fetchImpl !== 'function') throw new TypeError('Se requiere un cliente HTTP');
  const request = async (method, body) => {
    const response = await fetchImpl(endpoint, { method, headers: { Accept: 'application/json', ...(body ? { 'Content-Type': 'application/json' } : {}) }, ...(body ? { body: JSON.stringify(body) } : {}) });
    let payload = null;
    try { payload = await response.json(); } catch (_) { /* respuesta sin cuerpo */ }
    if (!response.ok) { const error = new Error(payload?.message || `Error HTTP ${response.status}`); error.status = response.status; error.details = payload; throw error; }
    return payload || {};
  };
  return { get: () => request('GET'), put: (config) => request('PUT', redactOutgoing(config)) };
}

export function redactOutgoing(config = {}) {
  const clean = {};
  for (const key of CAMPOS) if (config[key] !== undefined) clean[key] = config[key];
  if (typeof config.secret === 'string' && config.secret.length > 0) clean.secret = config.secret;
  return clean;
}

export function renderCorreo(container, { client, initial } = {}) {
  if (!container) throw new TypeError('Falta contenedor');
  container.innerHTML = '';
  const allowed = initial?.access?.allowed ?? initial?.allowed ?? false;
  if (!allowed) { container.append(accessDenied()); return { load: async () => {}, save: async () => false }; }
  container.append(buildForm(initial?.configuration || {}));
  const form = container.querySelector('form');
  const errors = container.querySelector('[role="alert"]');
  const load = async () => { try { const data = await client.get(); fill(form, data.configuration || data); renderMeta(container, data); } catch (error) { errors.textContent = error.message; } };
  const save = async () => { errors.textContent = ''; const data = Object.fromEntries(new FormData(form)); const secret = data.secret; delete data.secret; if (secret) data.secret = secret; try { const result = await client.put(data); form.elements.secret.value = ''; renderMeta(container, result); return true; } catch (error) { errors.textContent = error.message; return false; } };
  form.addEventListener('submit', event => { event.preventDefault(); save(); });
  return { load, save };
}

function accessDenied() { const el = document.createElement('section'); el.className = 'correo-denegado'; el.setAttribute('role', 'alert'); el.innerHTML = '<h1>Configuración de correo</h1><p>Acceso denegado.</p><p>Esta pantalla requiere acceso desde la intranet, ticket SSH vigente, autenticación con DNIe o certificado digital y permiso ADMIN explícito.</p><p>El navegador no valida estos requisitos; el servidor debe aportar la autorización compuesta.</p>'; return el; }
function field(name, label, type = 'text', help = '') { return `<label>${label}<input name="${name}" type="${type}" autocomplete="off">${help ? `<small>${help}</small>` : ''}</label>`; }
function buildForm(config) { const el = document.createElement('section'); el.className = 'correo-configuracion'; el.innerHTML = `<header><p class="eyebrow">ADMIN · Comunicaciones</p><h1>Configuración SMTP de Diputación</h1><p>Gestiona el canal institucional. La configuración se guarda pendiente hasta su validación operativa.</p></header><div class="correo-grid"><form novalidate><fieldset><legend>Servidor</legend>${field('host','Host o IP')} ${field('port','Puerto','number')} ${field('serverName','Nombre del servidor (SNI)')} ${field('ca','Certificado CA','text','PEM o referencia segura; nunca se muestra un secreto')}</fieldset><fieldset><legend>Cuenta remitente</legend>${field('sender','Remitente fijo','email')} ${field('username','Usuario')} ${field('secret','Secreto','password','Write-only: se envía sólo si introduces uno; no se conserva en navegador')}</fieldset><fieldset><legend>Seguridad y tiempo</legend><label>Modo TLS<select name="tlsMode"><option value="implicit">TLS implícito</option><option value="starttls">STARTTLS obligatorio</option></select></label><label>Modo OAuth<select name="oauthMode"><option value="disabled">OAuth no configurado</option><option value="optional">OAuth opcional</option></select></label>${field('timeout','Timeout (ms)','number')}</fieldset><div class="correo-actions"><button type="submit">Guardar configuración</button><span role="alert" aria-live="assertive"></span></div></form><aside class="correo-meta" aria-label="Estado y auditoría"><div class="status-card" data-status></div><div data-audit></div></aside></div>`; fill(el.querySelector('form'), config); return el; }
function fill(form, values) { for (const [key, value] of Object.entries(values || {})) if (key !== 'secret' && form.elements[key]) form.elements[key].value = value ?? ''; }
function renderMeta(container, data = {}) { const status = container.querySelector('[data-status]'); const configured = data.secretConfigured ?? data.configuration?.secretConfigured; status.innerHTML = `<strong>Estado</strong><span>${configured ? 'Secreto configurado' : 'Sin cuenta / secreto no configurado'}</span><small>El valor del secreto nunca se muestra.</small>`; const audit = container.querySelector('[data-audit]'); const entries = data.audit || []; audit.innerHTML = `<h2>Auditoría resumida</h2>${entries.length ? `<ul>${entries.map(item => `<li><strong>${item.action || 'Cambio'}</strong> · ${item.actor || 'Actor no informado'} · ${item.at || 'Fecha no informada'}</li>`).join('')}</ul>` : '<p>Sin movimientos registrados.</p>'}`; }
