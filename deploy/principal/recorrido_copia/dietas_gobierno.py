"""Publica los dos roles AUT107 por los actos ADMIN de la copia."""

import json
import secrets

from . import catalogo, roles_admin

# Preflight de instalación; los actos siguientes conservan toda su autoridad.
# Proceden de los cuerpos exactos del corte AUT107, sin otra función consumidora.
SQL_AUT107 = """SELECT (count(p.oid)=2 AND bool_and(
 encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')=x.huella))
FROM (VALUES
 ('vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)',
  '45c953d58cdb8046d9c7b403d9c275b5546e094c1d86bcf4e41ca008a0f0492a'),
 ('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)',
  'c675cfcd7bba20d2c35421ffe7a2d424c3413922ccba1f5c09656247229300b9')
) x(firma,huella) LEFT JOIN pg_catalog.pg_proc p
 ON p.oid=pg_catalog.to_regprocedure(x.firma);"""


def exigir_instalacion(c):
    if c.consultar(SQL_AUT107).strip() != 't':
        raise ValueError('Falta AUT107 exacta; no se prepara Dietas')


def publicar(c, admisiones):
    """Sin asignaciones ni SQL de escritura: propone ADMIN1 y cierra ADMIN2."""
    exigir_instalacion(c)
    ultimo = admisiones['actos'][-1]
    _, publicado = roles_admin._cabeza(c, ultimo)
    fuentes = [json.loads((c.fuente / 'deploy/principal/fuentes' / spec[2]).read_bytes())
               for spec in catalogo.ACTOS_DIETAS]
    datos = json.loads((c.fuente / 'deploy/principal/recorrido_copia/datos/dietas_roles.v1.json').read_bytes())
    if datos.get('version') != 1 or [r['rol_id'] for r in datos['roles']] != [
            'persona_dietas_comisiones', 'persona_dietas_rutas']:
        raise ValueError('Contratos de roles Dietas divergentes')
    motivo = c.leer(c.raiz / 'admin/h10/entrada-vec-admin.json')['motivo']
    archivo = c.carpeta / 'dietas/roles-publicados.json'
    anterior = c.leer(archivo) if archivo.exists() else None
    if anterior is not None and (anterior.get('copia_ref') != c.copia.nombre
            or anterior.get('sid_copia') != c.copia.leer()['sid_copia']):
        raise ValueError('Recibos de roles Dietas de otra copia')
    resultados = []
    for contrato in datos['roles']:
        rol_id = contrato['rol_id']
        selecciones, concesiones = [], []
        for accion in contrato['acciones']:
            candidatas = [f for f in fuentes if any(e['concesion']['accion'] == accion
                                                    for e in f['entradas'])]
            if len(candidatas) != 1:
                raise ValueError('Acción Dietas sin fuente nominal única')
            fuente = candidatas[0]
            selecciones.append(roles_admin._entrada(publicado, fuente, accion))
            concesiones.append(next(e['concesion'] for e in fuente['entradas']
                                    if e['concesion']['accion'] == accion))
        carpeta = c.carpeta / 'dietas/roles' / rol_id
        carpeta.mkdir(mode=0o700, parents=True, exist_ok=True)
        existente = roles_admin._rol(c, rol_id)
        guardado = None
        if anterior is not None:
            propios = [r for r in anterior['roles'] if r['rol_ref'] == 'rol:' + rol_id + ':v1']
            if len(propios) != 1 or not existente or propios[0]['huella_sha256'] != existente['huella']:
                raise ValueError('Rol Dietas distinto del recibo original')
            guardado = propios[0]
        cierre = None
        if existente is None:
            ruta = carpeta / 'roles-propuesta.json'
            propuesta = c.leer(ruta) if ruta.exists() else {
                'operacion_ref': 'propuesta_admin:' + secrets.token_hex(16),
                'catalogo_ref': publicado['referencia'], 'catalogo_version': publicado['version'],
                'catalogo_huella_sha256': ultimo['catalogo_sha256'],
                'rol_id': rol_id, 'nombre': contrato['nombre'], 'selecciones': selecciones,
                'motivo': motivo, 'referencia_acto': 'acto:dietas:copia:' + rol_id}
            cierre = roles_admin._proponer_y_cerrar(c, carpeta, 'roles', propuesta)
            existente = roles_admin._rol(c, rol_id)
        if (not existente or existente['version'] != 1
                or existente['ref'] != 'rol:' + rol_id + ':v1'
                or existente['control_estado'] != 'habilitada'
                or existente['documento']['concesiones'] != concesiones):
            raise ValueError('Rol Dietas no publicado con su contrato exacto')
        resultados.append(guardado if guardado is not None else {'rol_ref': existente['ref'], 'huella_sha256': existente['huella'],
                           'publicado_en_esta_etapa': cierre is not None, 'cierre': cierre})
    resultado = {'copia_ref': c.copia.nombre, 'sid_copia': c.copia.leer()['sid_copia'],
                 'roles': resultados}
    c.guardar(archivo, resultado)
    return resultado
