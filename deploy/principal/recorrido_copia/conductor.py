"""Secuencia versionada del recorrido de una copia con material recién generado."""

import json
import os
from pathlib import Path
import re
import shutil
import stat

from .comun import Contexto


def _parametros(copia):
    from copia_local import REPO
    fuente = REPO / 'scripts/operacion/firma_ct_tres_pasos/cargos.v2.json'
    cargos = json.loads(fuente.read_bytes())['cargos']
    organizaciones = {c['organizacion_ref'] for c in cargos}
    if len(cargos) != 16 or len(organizaciones) != 1:
        raise ValueError('La fuente de cargos no fija una organización operativa única')
    return {'organizacion_operativa_ref': organizaciones.pop()}


def provisionar_perfiles(copia):
    """Aprueba solo las preimágenes que el arranque propio declaró pendientes."""
    from copia_local import guardar, sha256
    for intento in range(3):
        log = (copia.raiz / 'app.log').read_text(errors='replace')
        marca = log.rfind('vec server listening')
        # Las advertencias preceden inmediatamente a la línea de escucha.
        if marca < 0:
            raise ValueError('El arranque propio no confirmó la escucha')
        anterior = log.rfind('vec server listening', 0, marca)
        lineas = log[anterior + 1:marca].splitlines()[-60:]
        pendientes = sorted(set(re.findall(r'asignacion_vigente_huella_sha256=([0-9a-f]{64})',
            '\n'.join(x for x in lineas if 'perfil fijo de RRHH sin asignación consumible' in x))))
        if not pendientes:
            return
        ruta = copia.raiz / 'entorno_modulos.json'
        entorno = json.loads(ruta.read_bytes())
        if sha256(ruta) != copia.leer()['entorno_modulos_sha256']:
            raise ValueError('El entorno cambió antes de provisionar perfiles')
        entorno.update(VEC_CT_PROVISION_PERFILES_RRHH_APROBACION='aprobacion:fabrica:f1-131',
                       VEC_CT_PROVISION_PERFILES_RRHH_PREIMAGENES=','.join(pendientes))
        guardar(ruta, entorno)
        estado = copia.leer()
        estado['entorno_modulos_sha256'] = sha256(ruta)
        guardar(copia.estado, estado)
        copia.parar()
        copia.arrancar(comprobar_modulos=False)
        # Siempre se retiran la aprobación y sus preimágenes después del acto.
        entorno.pop('VEC_CT_PROVISION_PERFILES_RRHH_APROBACION')
        entorno.pop('VEC_CT_PROVISION_PERFILES_RRHH_PREIMAGENES')
        guardar(ruta, entorno)
        estado = copia.leer()
        estado['entorno_modulos_sha256'] = sha256(ruta)
        guardar(copia.estado, estado)
        copia.parar()
        copia.arrancar(comprobar_modulos=False)
    raise ValueError('Persisten perfiles pendientes de provisión; se conservan los recibos')


def _verificador(c, puerto):
    software = c.copia.leer().get('software_verificador')
    if not isinstance(software, dict) or set(software) != {'binario', 'binario_sha256', 'version'}:
        raise ValueError('Falta el verificador prevalidado de esta copia')
    origen = Path(software['binario'])
    esperado = c.raiz / 'software-verificador/grxfirma'
    huella = software['binario_sha256']
    if (origen != esperado or origen.is_symlink() or not origen.is_file()
            or not re.fullmatch(r'[0-9a-f]{64}', huella)
            or not isinstance(software['version'], str) or not software['version']
            or origen.stat().st_uid != os.getuid()
            or stat.S_IMODE(origen.stat().st_mode) != 0o700
            or origen.stat().st_nlink != 1 or c.sha(origen) != huella):
        raise ValueError('El verificador de la copia cambió tras el preflight')
    destino = c.carpeta / 'grxfirma'
    if destino.exists() or destino.is_symlink():
        if (destino.is_symlink() or not destino.is_file()
                or stat.S_IMODE(destino.stat().st_mode) != 0o700
                or c.sha(destino) != huella):
            raise ValueError('La copia del verificador es divergente')
    else:
        descriptor = os.open(destino, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o700)
        with origen.open('rb') as entrada, os.fdopen(descriptor, 'wb') as salida:
            shutil.copyfileobj(entrada, salida)
            salida.flush()
            os.fsync(salida.fileno())
        if c.sha(destino) != huella:
            raise ValueError('La copia del verificador es divergente')
    manifiesto = c.carpeta / 'manifiesto-firma.json'
    c.guardar(manifiesto, {'version': 1, 'etapas': [], 'verificador': {
        'binario': str(destino), 'binario_sha256': huella, 'puerto': puerto}})
    return manifiesto


def _organizacion(c):
    from copia_local_organizacion_historica_activacion import activar
    activar(c.copia, c.fuente)
    return {'material_sha256': c.sha(c.raiz /
            'organizacion-historica/material/organizacion_historica_v3.json'),
            'semilla_sha256': c.copia.leer()['organizacion_historica_semilla_sha256']}


def preparar(copia, args):
    """Ejecuta actos nativos y conserva cada recibo, sin recetas privadas."""
    from . import catalogo, roles_admin, certificados_base, firmantes, montajes_firma, actos, plan, admisiones, configuracion_firma
    c = Contexto(copia, _parametros(copia))
    etapas = [('catalogo', catalogo.preparar), ('roles_admin', roles_admin.preparar),
              ('organizacion_historica', _organizacion),
              ('certificados_base', certificados_base.preparar), ('firmantes', firmantes.preparar),
              ('montajes_firma', montajes_firma.preparar), ('actos', actos.preparar_actos),
              ('plan', plan.preparar_plan), ('admisiones', admisiones.preparar),
              ('configuracion_firma', configuracion_firma.preparar)]
    resultados = {}
    for nombre, funcion in etapas:
        resultados[nombre] = funcion(c)
        c.guardar(c.carpeta / ('etapa-' + nombre + '.json'), {
            'version': 1, 'copia_ref': copia.nombre, 'etapa': nombre,
            'resultado': resultados[nombre]})
    from . import documentos_base
    from .documentos import preparar_documentos
    base_documental = documentos_base.preparar(c)
    preparar_documentos(copia, c.leer(base_documental), c)
    from copia_local import guardar, sha256
    entorno = copia.entorno_modulos()
    entorno['VEC_DOCUMENTOS_ENABLED'] = 'true'
    archivo = copia.raiz / 'entorno_modulos.json'
    guardar(archivo, entorno)
    estado = copia.leer()
    estado['entorno_modulos_sha256'] = sha256(archivo)
    guardar(copia.estado, estado)
    # La orden firma reutiliza la publicación de plantillas y el arranque vigilado.
    copia.firma(_verificador(c, args.puerto_verificador))
    provisionar_perfiles(copia)
    from . import personal
    copia.personal(personal.preparar(c))
    c.guardar(c.carpeta / 'recibo-fabrica.json', {
        'version': 1, 'tarea_ref': 'F1-131', 'copia_ref': copia.nombre,
        'revision': copia.leer()['revision'], 'sid_copia': copia.leer()['sid_copia'],
        'roles_compuestos': 'semilla técnica de copia de desarrollo, no gobierno',
        'plan': 'publicado por dos administradores',
        'documentos': 'configuración propia compuesta',
        'configuracion_personal': True,
        'etapas': list(resultados)})
