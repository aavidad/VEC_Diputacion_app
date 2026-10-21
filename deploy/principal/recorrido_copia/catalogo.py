"""Admite dos fuentes AUT58 en la copia antes de preparar firmantes."""

import hashlib
import json
import os
from pathlib import Path
import re
import stat
import sys


CATALOGO = 'catalogo:bolsa:carga_convoca:b1'
ENCARGO = 'F1-131'
APROBADOR = 'fabrica:copias:desarrollo'
ACTOS = (
    (11, 'fuentes', 'personal_organizacion_historica_rrhh_v1.json',
     'c04d3041921c03ec32d36970d479ad8b4ec729ac9161d79652b1afeba547dfe7'),
    (12, 'sustituir', 'bolsa_origen_ct_rrhh_v2.json',
     '239b0dd9dab236396f675247f344c91863c839ab56d2d068165bd45cd930bd9b'),
)
DIRECTORIOS = {11: 'organizacion', 12: 'origen_ct', 13: 'identidad_documental'}
HEX = re.compile(r'[0-9a-f]{64}\Z')
SID = re.compile(r'[0-9]{15,20}\Z')
LOGIN = re.compile(r'vec_caa_[0-9a-f]{16}\Z')


def exigir(ok, motivo):
    if not ok:
        raise ValueError(motivo)


def privado(path):
    path = Path(path)
    st = path.lstat()
    exigir(path.is_absolute() and not path.is_symlink() and stat.S_ISREG(st.st_mode)
           and st.st_uid == os.getuid() and stat.S_IMODE(st.st_mode) == 0o600
           and st.st_nlink == 1 and 0 < st.st_size <= 2_097_152,
           'Material privado AUT58 inseguro')
    return path.read_bytes()


def ejecutar(c, args, log):
    return c.ejecutar(args, log, env={'VEC_COPIA_RAIZ': str(c.raiz)})


def preparador(c, *args, log):
    script = c.fuente / 'deploy/principal/admin_copia/catalogo_acciones.py'
    return ejecutar(c, [sys.executable, script, *map(str, args)], log)


def respuesta(resultado):
    try:
        value = json.loads(resultado.stdout)
    except (ValueError, UnicodeDecodeError, TypeError) as error:
        raise ValueError('Respuesta AUT58 no verificable') from error
    exigir(isinstance(value, dict), 'Respuesta AUT58 inválida')
    return value


def copia_y_fuentes(c):
    estado = c.copia.leer()
    sid = estado.get('sid_copia')
    exigir(estado.get('admin_preparada') is True and estado.get('nombre') == c.copia.nombre
           and isinstance(sid, str) and SID.fullmatch(sid) and int(sid) > 0,
           'ADMIN o SID de la copia ausentes')
    actual = c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();')
    exigir(actual == sid, 'SID de PostgreSQL ajeno a la copia')
    run = c.raiz / 'admin/h10'
    config = c.leer(run / 'config.json')
    contract = c.leer(run / 'contrato-arranque.json')
    preimagen = c.raiz / 'admin/preimagen-admin.dump'
    preimagen_stat = preimagen.lstat()
    backup = config.get('backup', {})
    exigir(isinstance(backup, dict), 'Preimagen ADMIN sin contrato')
    backup_sha = backup.get('sha256')
    exigir(contract.get('sid_copia') == sid and contract.get('revision') == estado.get('revision')
           and contract.get('config_sha256') == c.sha(run / 'config.json')
           and isinstance(backup.get('ref'), str) and backup['ref']
           and isinstance(backup_sha, str) and HEX.fullmatch(backup_sha)
           and stat.S_ISREG(preimagen_stat.st_mode) and not preimagen.is_symlink()
           and preimagen_stat.st_uid == os.getuid()
           and stat.S_IMODE(preimagen_stat.st_mode) == 0o600
           and preimagen_stat.st_nlink == 1
           and c.sha(preimagen) == backup_sha,
           'Preimagen ADMIN o contrato ajenos a la copia')
    fuentes = c.fuente / 'deploy/principal/fuentes'
    for _, _, nombre, huella in ACTOS:
        path = fuentes / nombre
        exigir(path.is_file() and not path.is_symlink() and c.sha(path) == huella,
               'Fuente AUT58 alterada o ausente')
    return estado, sid, backup, fuentes


def directorio(c, numero):
    path = c.carpeta / 'catalogo' / DIRECTORIOS[numero]
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    st = path.lstat()
    exigir(stat.S_ISDIR(st.st_mode) and not path.is_symlink()
           and st.st_uid == os.getuid() and stat.S_IMODE(st.st_mode) == 0o700,
           'Directorio AUT58 inseguro')
    return path


def cotejar_preparado(c, base, spec, sid, backup, anterior, encargo=ENCARGO):
    numero, modo, nombre, huella = spec
    meta = c.leer(base / 'preparado.json')
    exigir(str(meta.get('acto')) == str(numero) and meta.get('modo') == modo
           and meta.get('sueltas') == [[nombre, huella]]
           and meta.get('sid_copia') == sid and meta.get('encargo_ref') == encargo
           and meta.get('backup_ref') == backup['ref']
           and meta.get('backup_sha256') == backup['sha256']
           and HEX.fullmatch(meta.get('catalogo_sha256', ''))
           and HEX.fullmatch(meta.get('paquete_sha256', ''))
           and HEX.fullmatch(meta.get('prev_huella', ''))
           and HEX.fullmatch(meta.get('censo_preparado_sha256', ''))
           and c.sha(base / 'catalogo.json') == meta['catalogo_sha256']
           and c.sha(base / 'paquete.json') == meta['paquete_sha256'],
           'Preparación AUT58 divergente')
    if anterior is not None:
        exigir(meta.get('prev_version') == anterior['catalogo_version']
               and meta['prev_huella'] == anterior['catalogo_sha256'],
               'El segundo acto no parte del recibo primero')
    return meta


def aprobacion(c, base, meta, numero, encargo=ENCARGO):
    body = {'esquema': 'vec.admin.catalogo-acciones.aprobacion-tecnica-copia.v1',
            'aprobacion_ref': f'aprobacion:{encargo}:catalogo:{numero}:{c.copia.nombre}',
            'aprobador_ref': APROBADOR,
            'catalogo_sha256': meta['catalogo_sha256'],
            'paquete_sha256': meta['paquete_sha256'],
            'backup_ref': meta['backup_ref'], 'backup_sha256': meta['backup_sha256'],
            'encargo_ref': encargo, 'sid_copia': meta['sid_copia'],
            'esperado_version': meta['prev_version'],
            'esperado_sha256': meta['prev_huella'],
            'censo_preparado_sha256': meta['censo_preparado_sha256']}
    path = base / 'aprobacion-tecnica.json'
    c.guardar(path, body)
    exigir(c.leer(path) == body, 'Aprobación técnica de otra copia')
    return path, body


def cotejar_configuracion(c, base, meta, aprobacion_doc, encargo=ENCARGO):
    cfg = c.leer(base / 'configurado.json')
    plan = c.leer(base / 'plan.json')
    exigir(HEX.fullmatch(cfg.get('plan_sha256', ''))
           and cfg['plan_sha256'] == c.sha(base / 'plan.json')
           and cfg.get('sid_copia') == meta['sid_copia']
           and cfg.get('encargo_ref') == encargo
           and cfg.get('catalogo_sha256') == meta['catalogo_sha256']
           and cfg.get('paquete_sha256') == meta['paquete_sha256']
           and cfg.get('aprobacion_ref') == aprobacion_doc['aprobacion_ref']
           and cfg.get('aprobacion_sha256') == c.sha(base / 'aprobacion-tecnica.json')
           and cfg.get('aprobador_ref') == APROBADOR
           and cfg.get('preimagen') == {'catalogo_ref': CATALOGO,
                                         'version': meta['prev_version'],
                                         'huella_sha256': meta['prev_huella'],
                                         'censo_preparado_sha256': meta['censo_preparado_sha256']}
           and LOGIN.fullmatch(cfg.get('login', ''))
           and plan.get('operacion_ref') == cfg.get('operacion_ref')
           and plan.get('catalogo_sha256') == meta['catalogo_sha256']
           and plan.get('aprobacion_sha256') == cfg['aprobacion_sha256'],
           'Configuración AUT58 divergente')
    privado(base / 'conexion.json')
    privado(base / 'configurar.sql')
    return cfg, plan


def configurar_sql(c, base, cfg, numero):
    pending = base / 'sql-pendiente.json'
    done = base / 'sql-confirmado.json'
    expected = {'plan_sha256': cfg['plan_sha256'],
                'sql_sha256': c.sha(base / 'configurar.sql')}
    if done.exists():
        exigir(c.leer(done) == expected, 'SQL AUT58 confirmado para otro plan')
        return
    exigir(not pending.exists(), 'SQL AUT58 incierto: cotejar antes de continuar')
    c.guardar(pending, expected)
    c.sql(privado(base / 'configurar.sql'), f'catalogo-{numero}-configurar.log')
    c.guardar(done, expected)


def recibo_valido(doc, cfg, plan, meta, replay):
    r = doc.get('recibo') if isinstance(doc, dict) else None
    exigir(isinstance(r, dict) and doc.get('codigo') == 'catalogo_admitido'
           and doc.get('estado') == 'permitido' and doc.get('replay', False) is replay
           and r.get('esquema') == 'vec.admin.catalogo-acciones.recibo.v1'
           and r.get('operacion_ref') == plan['operacion_ref']
           and r.get('plan_sha256') == cfg['plan_sha256']
           and r.get('catalogo_ref') == CATALOGO
           and str(r.get('catalogo_version')) == str(meta['prev_version'] + 1)
           and r.get('catalogo_sha256') == meta['catalogo_sha256']
           and r.get('paquete_sha256') == meta['paquete_sha256']
           and r.get('aprobacion_ref') == cfg['aprobacion_ref']
           and r.get('aprobacion_sha256') == cfg['aprobacion_sha256']
           and r.get('aprobador_ref') == APROBADOR
           and bool(r.get('auditoria_ref')) and bool(r.get('confirmado_en')),
           'Recibo AUT58 ausente o ajeno')
    return r


def consultar(c, cli, base, meta, numero):
    result = ejecutar(c, [cli, '--fase', 'consultar', '--config', base / 'conexion.json',
                          '--ref', CATALOGO, '--version', meta['prev_version'] + 1,
                          '--catalogo-sha', meta['catalogo_sha256'], '--idioma', 'es'],
                       f'catalogo-{numero}-consultar.log')
    doc = respuesta(result)
    catalogo = doc.get('catalogo')
    expected = c.leer(base / 'catalogo.json')
    exigir(doc.get('codigo') == 'catalogo_consultado' and isinstance(catalogo, dict)
           and catalogo == expected, 'Consulta AUT58 distinta del catálogo admitido')
    c.guardar(base / 'consulta.json', doc)


def aplicar(c, cli, base, cfg, plan, meta, numero, replay):
    result = ejecutar(c, [cli, '--fase', 'aplicar', '--config', base / 'conexion.json',
                          '--plan', base / 'plan.json', '--sha', cfg['plan_sha256'],
                          '--idioma', 'es'], f'catalogo-{numero}-aplicar-{int(replay)}.log')
    doc = respuesta(result)
    recibo_valido(doc, cfg, plan, meta, replay)
    c.guardar(base / ('acuse-replay.json' if replay else 'acuse-primero.json'), doc)
    return doc


def acto(c, cli, spec, sid, backup, fuentes, anterior, encargo=ENCARGO):
    numero, modo, nombre, huella = spec
    exigir((spec in ACTOS and encargo == ENCARGO)
           or (numero == 13 and encargo == 'F1-27' and modo == 'fuentes'
               and nombre == 'bolsa_identidad_documental_ct_rrhh_v1.json'),
           'Acto AUT58 fuera del encargo')
    base = directorio(c, numero)
    preparado = base / 'preparado.json'
    if not preparado.exists():
        preparador(c, 'preparar', '--dir', base, '--fuentes', fuentes,
                  '--modo', modo, '--acto', numero, '--encargo-ref', encargo,
                  '--sid-copia', sid, '--backup-ref', backup['ref'],
                  '--backup-sha', backup['sha256'], '--suelta', f'{nombre}:{huella}',
                  log=f'catalogo-{numero}-preparar.log')
    meta = cotejar_preparado(c, base, spec, sid, backup, anterior, encargo)
    approval_path, approval = aprobacion(c, base, meta, numero, encargo)
    first_path = base / 'acuse-primero.json'
    if not first_path.exists():
        # Antes de configurar o aplicar, el preparador vuelve a cotejar cabeza y censo.
        if not (base / 'sql-pendiente.json').exists():
            preparador(c, 'revisar', '--dir', base, '--fuentes', fuentes,
                      '--huella', meta['catalogo_sha256'],
                      log=f'catalogo-{numero}-revisar.log')
        if not (base / 'plan.json').exists():
            estado = c.copia.leer()
            preparador(c, 'configurar', '--dir', base, '--fuentes', fuentes,
                      '--aprobacion', approval_path, '--pg-ca', c.material / 'ca/ca.crt',
                      '--pg-puerto', estado['puerto_pg'], '--ttl', 120,
                      log=f'catalogo-{numero}-configurar.log')
    cfg, plan = cotejar_configuracion(c, base, meta, approval, encargo)
    configurar_sql(c, base, cfg, numero)
    if first_path.exists():
        first = c.leer(first_path)
        receipt = recibo_valido(first, cfg, plan, meta, False)
    else:
        exigir(not (base / 'aplicar-pendiente.json').exists(),
               'Aplicación AUT58 incierta: consultar operación original')
        c.guardar(base / 'aplicar-pendiente.json', {'operacion_ref': plan['operacion_ref'],
                                                   'plan_sha256': cfg['plan_sha256']})
        first = aplicar(c, cli, base, cfg, plan, meta, numero, False)
        receipt = first['recibo']
    consultar(c, cli, base, meta, numero)
    second_path = base / 'acuse-replay.json'
    if second_path.exists():
        second = c.leer(second_path)
        recibo_valido(second, cfg, plan, meta, True)
    else:
        second = aplicar(c, cli, base, cfg, plan, meta, numero, True)
    exigir(second['recibo'] == receipt, 'El replay AUT58 cambió el recibo')
    preparador(c, 'acuses', '--primero', first_path, '--replay', second_path,
              log=f'catalogo-{numero}-acuses.log')
    recibo_bytes = json.dumps(receipt, ensure_ascii=False, sort_keys=True,
                              separators=(',', ':')).encode()
    result = {'acto': numero, 'nombre': DIRECTORIOS[numero],
              'copia_ref': c.copia.nombre, 'sid_copia': sid,
              'catalogo_ref': CATALOGO, 'catalogo_version': meta['prev_version'] + 1,
              'catalogo_sha256': meta['catalogo_sha256'],
              'paquete_sha256': meta['paquete_sha256'],
              'plan_sha256': cfg['plan_sha256'], 'operacion_ref': plan['operacion_ref'],
              'auditoria_ref': receipt.get('auditoria_ref'),
              'recibo': receipt, 'recibo_sha256': hashlib.sha256(recibo_bytes).hexdigest(),
              'acuse_primero_sha256': c.sha(first_path),
              'acuse_replay_sha256': c.sha(second_path),
              'consulta_sha256': c.sha(base / 'consulta.json')}
    c.guardar(base / 'metadata.json', result)
    return result


def preparar(c):
    """Deja dos admisiones con recibo repetido idéntico y consulta cotejada."""
    _, sid, backup, fuentes = copia_y_fuentes(c)
    cli = c.cli('vec-catalogo-acciones-admitir')
    first = acto(c, cli, ACTOS[0], sid, backup, fuentes, None)
    second = acto(c, cli, ACTOS[1], sid, backup, fuentes, first)
    exigir(first['recibo']['operacion_ref'] != second['recibo']['operacion_ref']
           and first['catalogo_version'] + 1 == second['catalogo_version'],
           'Las dos admisiones AUT58 no son consecutivas')
    metadata = {'version': 1, 'copia_ref': c.copia.nombre, 'sid_copia': sid,
                'actos': [first, second], 'recibos_reales': 2,
                'replays_cotejados': 2}
    c.guardar(c.carpeta / 'catalogo/admisiones.json', metadata)
    return metadata
