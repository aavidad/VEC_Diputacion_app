"""Entrada privada de Personal B2 para el productor nativo de la copia."""

import hashlib
import json
from pathlib import Path
import re

from copia_local_personal import B2, OBLIGATORIAS

HEX = re.compile(r'[0-9a-f]{64}\Z')
PERSONA = re.compile(r'per_[0-9a-f]{32}\Z')
PERFIL = re.compile(r'prf_[0-9a-f]{32}\Z')
ORG = re.compile(r'org_[0-9a-f]{32}\Z')
MOTIVO = re.compile(r'motivo_[0-9a-f]{32}\Z')
CATALOGO = re.compile(r'[a-z][a-z0-9._-]{0,127}\Z')
ESQUEMA = 'vec.contratacion-temporal.incorporacion-servidor.v2'


def _privado(c, ruta):
    ruta = Path(ruta)
    if (not ruta.is_absolute() or ruta.is_symlink() or not ruta.is_file()
            or ruta.stat().st_mode & 0o077
            or not (ruta.resolve().is_relative_to(c.carpeta.resolve())
                    or ruta.resolve().is_relative_to(c.material.resolve())
                    or ruta.resolve().is_relative_to(c.raiz.resolve()))):
        raise ValueError('Fuente Personal ajena a la copia')
    return c.leer(ruta)


def _referencia(prefijo, sujeto, certificado, discriminador):
    """Misma derivación de referenciaAltaContratacionTemporalDesarrollo en Go."""
    base = sujeto + '\0' + certificado
    material = 'vec.ct.alta.desarrollo.v1\0' + base + '\0' + discriminador
    return prefijo + hashlib.sha256(material.encode()).hexdigest()[:32]


def _motivos(c, operaciones):
    unicos = set()
    for nombre, op in operaciones.items():
        if (not re.fullmatch(r'[a-z][a-z0-9_]{2,63}', nombre)
                or not isinstance(op, dict) or set(op) != {'motivo'}
                or not isinstance(op['motivo'], dict)):
            raise ValueError('Operación B2 con datos no autorizados')
        m = op['motivo']
        if (set(m) != {'catalogo_id', 'catalogo_version', 'catalogo_huella_sha256', 'entrada_clave'}
                or not isinstance(m['catalogo_id'], str) or not CATALOGO.fullmatch(m['catalogo_id'])
                or type(m['catalogo_version']) is not int or m['catalogo_version'] < 1
                or not isinstance(m['catalogo_huella_sha256'], str)
                or not HEX.fullmatch(m['catalogo_huella_sha256'])
                or not isinstance(m['entrada_clave'], str) or not MOTIVO.fullmatch(m['entrada_clave'])):
            raise ValueError('Referencia de motivo B2 inválida')
        unicos.add((m['catalogo_id'], m['catalogo_version'],
                    m['catalogo_huella_sha256'], m['entrada_clave']))
    if not OBLIGATORIAS <= set(operaciones) or len(operaciones) != 27:
        raise ValueError('Inventario de operaciones Personal incompleto')
    for catalogo, version, huella, clave in sorted(unicos):
        # El origen aprobado solo aporta la referencia. La copia acredita su publicación.
        sql = ("SELECT count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado c "
               "JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version) "
               "WHERE c.catalogo_id='" + catalogo + "' AND c.catalogo_version=" + str(version) +
               " AND c.catalogo_huella_publicada_sha256='" + huella + "' AND e.entrada_clave='" + clave + "' "
               "AND e.vigente_desde<=clock_timestamp() "
               "AND (e.vigente_hasta IS NULL OR clock_timestamp()<e.vigente_hasta) "
               "AND NOT EXISTS (SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r "
               "WHERE r.catalogo_id=c.catalogo_id AND r.catalogo_version=c.catalogo_version);")
        if c.consultar(sql).strip() != '1':
            raise ValueError('Motivo B2 aprobado ausente o retirado en esta copia')


def preparar(c):
    """Devuelve la entrada privada; Dirección invoca después copia.personal(ruta)."""
    estado = c.copia.leer()
    if (estado.get('recorrido_completo') is not True
            or c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();').strip()
               != estado.get('sid_copia')):
        raise ValueError('Personal exige la copia completa y su SID')
    ruta_base = Path(__file__).parent / 'datos/personal_b2.json'
    base = json.loads(ruta_base.read_bytes())
    if (set(base) != {'version', 'fuente_ref', 'esquema', 'personal_b2', 'operaciones'}
            or base['version'] != 1 or base['esquema'] != ESQUEMA
            or set(base['personal_b2']) != {
                'protocolo', 'catalogo_rpt_id', 'modulo_rpt_id', 'cese_fecha_efecto'}
            or base['personal_b2']['protocolo'] != 'personal_b2_v1'
            or base['personal_b2']['cese_fecha_efecto'] not in
               ('ultimo_dia_trabajado', 'primer_dia_sin_relacion')
            or not CATALOGO.fullmatch(base['personal_b2']['catalogo_rpt_id'])
            or not CATALOGO.fullmatch(base['personal_b2']['modulo_rpt_id'])):
        raise ValueError('Base Personal B2 incompleta o con campos ajenos')
    identidad = _privado(c, c.material / 'identidad/identidad.json')
    bback = _privado(c, c.material / 'identidad/bolsa-bback.json')
    sujeto, certificado = identidad.get('subject'), identidad.get('certificate_sha256')
    if (identidad.get('autoridad') != 'no_autoritativo' or not isinstance(sujeto, str)
            or not sujeto or '\0' in sujeto or not isinstance(certificado, str)
            or not HEX.fullmatch(certificado)
            or bback.get('sujeto') != sujeto
            or bback.get('certificado_sha256') != certificado):
        raise ValueError('Identidad RRHH original distinta del certificado de la copia')
    derivado = _referencia('per_', sujeto, certificado, 'persona')
    perfil = _referencia('prf_', sujeto, certificado, 'perfil')
    if not PERSONA.fullmatch(derivado) or not PERFIL.fullmatch(perfil):
        raise ValueError('Actor sintético de RRHH inválido')
    firmantes = _privado(c, c.carpeta / 'firmantes/metadata.json')
    unidad_acuse = _privado(c, c.raiz / 'admin/h10/unidad/acuse-primero.json')
    unidad = unidad_acuse.get('recibo', {}).get('unidad', {})
    if (firmantes.get('copia_ref') != c.copia.nombre
            or firmantes.get('organizacion_operativa_ref') != c.parametros['organizacion_operativa_ref']
            or unidad_acuse.get('estado') != 'permitido'
            or not unidad_acuse.get('recibo', {}).get('auditoria_ref')
            or unidad.get('unidad_ref') != firmantes.get('unidad_ref')
            or unidad.get('revision') != 1
            or not isinstance(unidad.get('organismo_ref'), str)
            or not ORG.fullmatch(unidad['organismo_ref'])):
        raise ValueError('Unidad Personal ajena a la semilla H10')
    _motivos(c, base['operaciones'])
    entrada = {'esquema': ESQUEMA,
               'referencias': {'PrincipalV3Ref': derivado, 'PerfilV3Ref': perfil,
                               'OrganizacionRef': c.parametros['organizacion_operativa_ref'],
                               'UnidadRef': unidad['unidad_ref'], 'ActorRef': derivado},
               'personal_b2': {**base['personal_b2'],
                               'organismo_ref': unidad['organismo_ref'],
                               'dsn_files': {k: 'b2-' + k + '.dsn' for k in B2},
                               'operaciones': base['operaciones']}}
    destino = c.carpeta / 'personal/configuracion.json'
    c.guardar(destino, entrada)
    return destino
