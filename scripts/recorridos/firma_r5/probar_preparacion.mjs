#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { spawnSync } from 'node:child_process';

// Ensayo propio: no modifica roles, SQL, material ajeno ni la configuración de producto.
const OWNER = 'codexe-firma-e2e-20261003';
const PG = 'codexe-firmas-frio-20261003';
const KIT = fs.realpathSync(process.argv[3] ?? path.dirname(fs.realpathSync(process.argv[1])));
if (fs.statSync(KIT).uid !== process.getuid() || (fs.statSync(KIT).mode & 0o077)) throw new Error('kit_no_privado');
const action = process.argv[2] ?? 'check';
const roles = ['app', 'grxfirma'];
const fail = (code) => { throw new Error(code); };
function privatePath(relative, directory = false) {
  const p = path.resolve(KIT, relative);
  if (!p.startsWith(`${KIT}/`) || fs.realpathSync(p) !== p) fail('ruta_privada_no_valida');
  const s = fs.lstatSync(p);
  if (s.uid !== process.getuid() || (s.mode & 0o077) || (directory ? !s.isDirectory() : !s.isFile() || s.nlink !== 1)) fail('material_no_privado');
  return p;
}
function read(relative) { return JSON.parse(fs.readFileSync(privatePath(relative), 'utf8')); }
function sha(file) { return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex'); }
function docker(args) {
  const cfg = path.join(KIT, 'runtime/docker-client');
  fs.mkdirSync(cfg, { recursive: true, mode: 0o700 });
  const r = spawnSync('/usr/bin/docker', ['--config', cfg, '--host', 'unix:///var/run/docker.sock', ...args], {
    env: { PATH: '/usr/bin:/bin', HOME: cfg }, encoding: 'utf8', maxBuffer: 1 << 20, timeout: 30000,
  });
  if (r.error || r.status !== 0) fail('docker_operacion_no_disponible');
  return r.stdout.trim();
}
function pg() {
  const [p] = JSON.parse(docker(['container', 'inspect', PG]));
  if (!p || !/^[a-f0-9]{64}$/u.test(p.Id) || p.HostConfig.NetworkMode !== 'none' || !p.State.Running) fail('clon_no_aislado');
  return p;
}
function source(config) {
  if (typeof config.source !== 'string' || !config.source.includes('/VEC_Diputacion_app/.worktrees/')) fail('fuente_no_asignada');
  const p = fs.realpathSync(config.source);
  if (p !== config.source || !fs.statSync(path.join(p, 'go.mod')).isFile()) fail('fuente_no_disponible');
  return p;
}
function variables(config, src) {
  const input = read('runtime/runtime-config.json');
  const names = new Set();
  for (const dir of ['config', 'internal/app/bootstrap']) {
    for (const f of fs.readdirSync(path.join(src, dir))) {
      if (!f.endsWith('.go') || f.endsWith('_test.go')) continue;
      const data = fs.readFileSync(path.join(src, dir, f), 'utf8');
      for (const m of data.matchAll(/"(VEC_[A-Z0-9_]+)"/gu)) names.add(m[1]);
    }
  }
  for (const [name, value] of Object.entries(input)) {
    if (!names.has(name) || typeof value !== 'string' || /[\r\n\0]/u.test(value)) fail('configuracion_no_admitida_por_fuente');
    if (name.endsWith('_DATABASE_URL') && value) {
      const u = new URL(value);
      if (!['postgres:', 'postgresql:'].includes(u.protocol) || u.hostname !== '127.0.0.1' || u.port !== '5432'
        || u.searchParams.get('sslmode') !== 'verify-full') fail('dsn_fuera_clon_tls');
      const query = [...u.searchParams.keys()];
      if (new Set(query).size !== query.length || query.some(k => !['sslmode','sslrootcert','sslcert','sslkey','connect_timeout','application_name'].includes(k))) fail('dsn_parametros_no_nominales');
      for (const k of ['sslrootcert','sslcert','sslkey']) {
        const v = u.searchParams.get(k);
        if (v && !v.startsWith(`${KIT}/material/`)) fail('dsn_material_fuera_ensayo');
      }
    } else if (name === 'VEC_FIRMA_VERIFICACION_URL') {
      if (value !== 'https://localhost:63118') fail('verificador_fuera_namespace');
    } else if (name.endsWith('_URL') && value) fail('endpoint_no_admitido');
    if (name.endsWith('_CIDRS') && value !== '127.0.0.1/32') fail('cidr_no_aislado');
    if (name.endsWith('_HOST') && value && !['localhost', '127.0.0.1'].includes(value)) fail('host_fuera_namespace');
    if (name.endsWith('_FILE') || name.endsWith('_PATH') || name.endsWith('_DIR')) {
      if (value && !value.startsWith(`${KIT}/`) && !value.startsWith(`${src}/`)) fail('ruta_config_fuera_ensayo');
    }
  }
  if (input.VEC_EXECUTION_PROFILE !== 'desarrollo' || input.VEC_AUTH_MODE !== 'desarrollo'
    || input.VEC_DEVELOPMENT_GUARD !== 'ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO'
    || input.VEC_HTTP_ADDR !== '127.0.0.1:18443') fail('perfil_ensayo_no_declarado');
  return input;
}
function common(role, p, c) {
  const rw = privatePath(`runtime/${role}`, true);
  return ['run', '--detach', '--pull=never', '--name', `${OWNER}-${role}`, '--label', `vec.firma.owner=${OWNER}`,
    '--label', `vec.firma.kit=${KIT}`, '--network', `container:${p.Id}`, '--read-only', '--cap-drop=ALL',
    '--security-opt', 'no-new-privileges:true', '--user', `${process.getuid()}:${process.getgid()}`,
    '--pids-limit', '256', '--cpus', '2', '--memory', '1g', '--ulimit', 'nofile=1024:1024',
    '--ulimit', 'fsize=67108864:67108864', '--tmpfs', '/tmp:rw,nosuid,nodev,size=128m',
    '--mount', `type=bind,src=${rw},dst=${rw}`,
    '--mount', 'type=bind,src=/usr,dst=/usr,readonly', '--mount', 'type=bind,src=/lib,dst=/lib,readonly',
    '--mount', 'type=bind,src=/lib64,dst=/lib64,readonly', '--env', `HOME=${rw}/home`, '--env', 'TZ=UTC'];
}
function plan(role, p, c) {
  const args = common(role, p, c);
  const GRX = c.grxfirma_binary;
  if (typeof GRX !== 'string' || fs.realpathSync(GRX) !== GRX || sha(GRX) !== c.grxfirma_sha256) fail('grxfirma_no_acreditado');
  if (role === 'grxfirma') {
    privatePath('pki/ancla.pem'); privatePath('pki/crl', true); privatePath('runtime/grxfirma/token');
    args.push('--mount', `type=bind,src=${GRX},dst=/work/grxfirma,readonly`,
      '--mount', `type=bind,src=${KIT}/launch_grxfirma.py,dst=${KIT}/launch_grxfirma.py,readonly`,
      '--mount', `type=bind,src=${KIT}/pki/ancla.pem,dst=${KIT}/pki/ancla.pem,readonly`,
      '--mount', `type=bind,src=${KIT}/pki/crl,dst=${KIT}/pki/crl,readonly`,
      '--entrypoint', '/usr/bin/python3', c.image,
      `${KIT}/launch_grxfirma.py`, KIT);
  } else {
    const src = source(c); variables(c, src);
    if (sha(privatePath('runtime/vec-server')) !== c.binary_sha256) fail('binario_no_acreditado');
    const material = privatePath('material', true);
    if (fs.existsSync(path.join(material, 'ca/ca.key'))) fail('clave_ca_en_servidor');
    const mtls = path.join(material, 'mtls');
    if (fs.existsSync(mtls) && fs.readdirSync(mtls).some(f => /\.(key|p12|password)$/u.test(f))) fail('clave_cliente_en_servidor');
    args.push('--mount', `type=bind,src=${src},dst=${src},readonly`,
      '--mount', `type=bind,src=${KIT}/material,dst=${KIT}/material,readonly`,
      '--mount', `type=bind,src=${KIT}/runtime/vec-server,dst=${KIT}/runtime/vec-server,readonly`,
      '--mount', `type=bind,src=${KIT}/runtime/runtime-config.json,dst=${KIT}/runtime/runtime-config.json,readonly`,
      '--mount', `type=bind,src=${KIT}/launch_app.py,dst=${KIT}/launch_app.py,readonly`,
      '--workdir', src, '--entrypoint', '/usr/bin/python3', c.image, `${KIT}/launch_app.py`, KIT);
  }
  return args;
}
function approval(c, p) {
  const a = read('runtime/arranque-aprobado.json');
  if (a.owner !== OWNER || a.pg_container_id !== p.Id || a.config_sha256 !== sha(privatePath('kit.json'))
    || a.runtime_config_sha256 !== sha(privatePath('runtime/runtime-config.json'))
    || a.binary_sha256 !== c.binary_sha256 || a.material_nominal_verificado !== true) fail('esperando_montaje_root');
}
try {
  if (!['check', 'plan-app', 'plan-grxfirma', 'start-app', 'start-grxfirma', 'stop-app', 'stop-grxfirma'].includes(action)) fail('accion_no_admitida');
  const c = read('kit.json');
  if (c.owner !== OWNER || c.datos !== 'sinteticos' || c.pg_container !== PG || c.image !== 'postgres:18.4') fail('kit_no_nominal');
  const p = pg();
  if (action === 'check') {
    const checks = {};
    for (const [key, check] of Object.entries({ fuente: () => source(c), config: () => variables(c, source(c)),
      pki: () => privatePath('pki/ancla.pem'), crl: () => privatePath('pki/crl', true), binario: () => privatePath('runtime/vec-server'),
      aprobacion: () => approval(c, p) })) {
      try { check(); checks[key] = 'preparado'; } catch { checks[key] = 'pendiente'; }
    }
    console.log(JSON.stringify({ owner: OWNER, clon_aislado: true, servicios_iniciados: false, checks }));
  } else {
    const [verb, role] = action.split('-');
    if (!roles.includes(role)) fail('servicio_no_admitido');
    if (verb === 'stop') {
      const [own] = JSON.parse(docker(['container', 'inspect', `${OWNER}-${role}`]));
      if (own.Config.Labels?.['vec.firma.owner'] !== OWNER || own.Config.Labels?.['vec.firma.kit'] !== KIT) fail('contenedor_ajeno');
      docker(['stop', '--time', '10', own.Id]); console.log(JSON.stringify({ stopped: role }));
    } else {
      const args = plan(role, p, c);
      if (verb === 'plan') console.log(JSON.stringify({ role, command: ['/usr/bin/docker', ...args], ejecutado: false }));
      else { approval(c, p); const id = docker(args); console.log(JSON.stringify({ started: role, container_id: id })); }
    }
  }
} catch (error) {
  // No diagnostics from Docker, DSN, environment, identity or target code leave this boundary.
  console.error(JSON.stringify({ preparado: false, codigo: error instanceof Error && /^[a-z_]+$/u.test(error.message) ? error.message : 'precondicion_pendiente' }));
  process.exitCode = 2;
}
