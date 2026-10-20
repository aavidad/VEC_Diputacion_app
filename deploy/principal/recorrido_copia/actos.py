"""Ejecuta los catorce actos propios con ADMIN 1 y conserva sus recibos."""

import json
from pathlib import Path

from .materiales_actos import preparar_materiales, verificar_materiales

PROPIOS = '/api/admin/seleccion-perfil/v1/propios'
SELECCION = '/api/admin/seleccion-perfil/v1/seleccion'


def perfiles_admin(c):
    """Selecciona el perfil de Aplicación de ambas personas sin combinar perfiles."""
    perfiles = []
    for numero in (1, 2):
        estado, propios = c.admin(numero, 'GET', PROPIOS, None)
        if estado != 200 or not isinstance(propios, dict):
            raise ValueError('No se pudo consultar el perfil ADMIN propio')
        disponibles = [p['perfil_ref'] for p in propios.get('perfiles', [])
                       if p.get('categoria_admin', '').lower() == 'aplicacion']
        if len(disponibles) != 1:
            raise ValueError('El ADMIN no tiene un perfil único de Aplicación')
        elegido = disponibles[0]
        if propios.get('perfil_activo_ref') != elegido:
            estado, _ = c.admin(numero, 'POST', SELECCION,
                                {'perfil_ref': elegido, 'revision_esperada': propios['revision']})
            if estado != 200:
                raise ValueError('No se pudo seleccionar el perfil de Aplicación')
            estado, propios = c.admin(numero, 'GET', PROPIOS, None)
            if estado != 200 or propios.get('perfil_activo_ref') != elegido:
                raise ValueError('Selección del perfil ADMIN no confirmada')
        perfiles.append(elegido)
    if perfiles[0] == perfiles[1]:
        raise ValueError('Los dos ADMIN comparten perfil nominal')
    return perfiles


def _recibo(a, respuesta):
    if not isinstance(respuesta, dict) or respuesta.get('error'):
        raise ValueError('Respuesta ADMIN sin recibo')
    r = respuesta.get('recibo')
    if not isinstance(r, dict) or not r.get('recibo_ref') or not r.get('auditoria_ref') or not respuesta.get('consumo_auditoria_ref'):
        raise ValueError('Recibo ADMIN o auditoría ausente')
    if a['tipo'] == 'certificado':
        if (r.get('vinculo_ref') != a['objeto_esperado'] or
                r.get('certificado_der_sha256') != a['certificado_sha256'] or
                r.get('estado') != 'vigente'):
            raise ValueError('Recibo de certificado ajeno')
    elif r.get('objeto_ref') != a['objeto_esperado'] or r.get('version') != 1:
        raise ValueError('Recibo de cargo o enlace ajeno')
    return r


def preparar_actos(c):
    m = preparar_materiales(c)
    verificar_materiales(c, m)
    perfiles = perfiles_admin(c)
    carpeta = c.carpeta / 'actos'
    refs = []
    for a in m['acciones']:
        id_ = a['id']
        confirmado = carpeta / 'recibos' / (id_ + '.json')
        incierto = carpeta / 'recibos' / (id_ + '.sin-confirmar.json')
        if confirmado.exists():
            recibo = _recibo(a, c.leer(confirmado))
            refs.append(recibo['recibo_ref'])
            continue
        if incierto.exists():
            raise ValueError(f'Acto {id_} incierto: conciliar con la misma clave antes de continuar')
        # La marca durable precede al POST. Un error de transporte nunca autoriza otra clave.
        c.guardar(incierto, {'accion': id_, 'material_sha256': a['material_sha256'],
                             'estado': 'pendiente_de_conciliacion'})
        try:
            estado, respuesta = c.admin(1, 'POST', a['ruta'], c.leer(Path(a['peticion'])))
        except Exception as error:
            raise ValueError(f'Acto {id_} incierto: consultar su operación original') from error
        if estado != 200:
            raise ValueError(f'Acto {id_} sin confirmación HTTP {estado}: conciliar la operación original')
        recibo = _recibo(a, respuesta)
        c.guardar(confirmado, respuesta)
        refs.append(recibo['recibo_ref'])
    meta = {'version': 1, 'copia_ref': m['copia_ref'], 'actos_confirmados': len(refs),
            'admin_personas': m['admin_personas'], 'admin_perfiles': perfiles,
            'fuente_sha256': m['fuente_sha256'], 'recibos': refs,
            'pasos': m['pasos'], 'gobierno_tecnico_roles': False}
    c.guardar(carpeta / 'metadata.json', meta)
    return meta
