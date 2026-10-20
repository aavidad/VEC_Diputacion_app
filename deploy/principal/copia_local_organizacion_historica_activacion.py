"""Monta la consulta OH después de publicar su rol por Administración."""

import json
import os
import subprocess

import copia_local_organizacion_historica as oh


def activar(copia, repo):
    from copia_local import ejecutar, guardar, memoria, sha256, verificar_fuente

    estado = copia.leer()
    if (not estado.get('preparada') or not estado.get('entorno_modulos_sha256')
            or estado.get('portal_externo_preparado') or not copia.existe()):
        raise ValueError('La consulta requiere una copia interna preparada con --modulos')
    if estado.get('organizacion_historica_semilla_sha256') or (copia.raiz / oh.SEMILLA).exists():
        raise ValueError('Los datos OH ya están preparados o su preparación quedó interrumpida; concilie la copia')
    verificar_fuente(estado['revision'])
    oh.comprobar_rol_publicado(copia, repo)
    memoria()
    go = subprocess.check_output([repo / 'scripts/seleccionar_toolchain_go_local.sh'], text=True).strip()
    entorno = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off',
               'GOSUMDB': 'off', 'GOENV': 'off', 'GOCACHE': '/dev/shm/vec/go-build'}
    with (copia.raiz / 'compilacion-oh.log').open('ab') as log:
        ejecutar([go, 'build', '-p', '6', '-o', copia.raiz / 'vec-preparar-material-interno',
                  './cmd/vec-preparar-material-interno'], cwd=repo, salida=log, entorno=entorno)
    # sembrar coteja el acto de los dos administradores antes de cualquier
    # dato de identidad o asignación; nunca publica un rol ni su control.
    huella = oh.sembrar(copia, repo)
    estado = copia.leer()
    estado['organizacion_historica_semilla_sha256'] = huella
    guardar(copia.estado, estado)
    copia.parar()
    copia.arrancar()
    if estado.get('rrhh_adicional_semilla_sha256'):
        from copia_local_organizacion_historica_actores import extender_contextos
        segunda = extender_contextos(copia, repo)
        estado = copia.leer()
        estado['organizacion_historica_segunda_semilla_sha256'] = segunda
        guardar(copia.estado, estado)
    oh.preparar_material(copia, copia.raiz / 'vec-preparar-material-interno')
    copia.parar()
    copia.arrancar()
    copia.comprobar()
    print(json.dumps({'consulta_activada': True, 'rol_publicado_por_administracion': oh.ROL_REF,
                      'asignacion': 'dato_ficticio_de_copia',
                      'material_sha256': sha256(copia.raiz / oh.DIRECTORIO / 'material/organizacion_historica_v3.json')}))
