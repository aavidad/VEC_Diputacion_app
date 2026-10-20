"""Secuencia versionada del recorrido de una copia con material recién generado."""

import hashlib
import json
from pathlib import Path
import re
import shutil

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
    origen = Path('/usr/bin/grxfirma')
    if origen.is_symlink() or not origen.is_file():
        raise ValueError('Falta el verificador GrxFirma instalado en el equipo')
    destino = c.carpeta / 'grxfirma'
    with origen.open('rb') as entrada, destino.open('xb') as salida:
        shutil.copyfileobj(entrada, salida)
    destino.chmod(0o700)
    huella = hashlib.sha256(destino.read_bytes()).hexdigest()
    if huella != hashlib.sha256(origen.read_bytes()).hexdigest():
        raise ValueError('El verificador copiado no coincide con el instalado')
    manifiesto = c.carpeta / 'manifiesto-firma.json'
    c.guardar(manifiesto, {'version': 1, 'etapas': [], 'verificador': {
        'binario': str(destino), 'binario_sha256': huella, 'puerto': puerto}})
    return manifiesto


def preparar(copia, args):
    """Ejecuta actos nativos y conserva cada recibo, sin recetas privadas."""
    from . import catalogo, roles_admin, firmantes, actos, plan, admisiones, configuracion_firma
    c = Contexto(copia, _parametros(copia))
    etapas = [('catalogo', catalogo.preparar), ('roles_admin', roles_admin.preparar),
              ('firmantes', firmantes.preparar), ('actos', actos.preparar_actos),
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
    # La orden firma reutiliza la publicación de plantillas y el arranque vigilado.
    copia.firma(_verificador(c, args.puerto_verificador))
    provisionar_perfiles(copia)
    if args.config_personal:
        copia.personal(args.config_personal)
    c.guardar(c.carpeta / 'recibo-fabrica.json', {
        'version': 1, 'tarea_ref': 'F1-131', 'copia_ref': copia.nombre,
        'revision': copia.leer()['revision'], 'sid_copia': copia.leer()['sid_copia'],
        'roles_compuestos': 'semilla técnica de copia de desarrollo, no gobierno',
        'plan': 'publicado por dos administradores',
        'documentos': 'configuración propia compuesta',
        'configuracion_personal': bool(args.config_personal),
        'etapas': list(resultados)})
