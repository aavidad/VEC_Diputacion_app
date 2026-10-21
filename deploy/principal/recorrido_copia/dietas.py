"""Conexiones privadas de Dietas; no concede permisos ni activa el módulo."""

from datetime import timedelta
import secrets
from urllib.parse import parse_qs, quote, urlencode, urlsplit

from copia_local_conexiones import _nombre_instancia, _privado, _rol
from copia_local_usuarios import sql as sql_logins


# Contrato de las once identidades acreditadas por comprobar-dietas.
# Índices 110–120, posteriores a Usuarios, Organización y Personal.
ENTORNO = {
    'VEC_DIETAS_BORRADORES_DATABASE_URL': 'vec_dietas_ejecutor',
    'VEC_DIETAS_PERSONAL_RELACIONES_DATABASE_URL': 'vec_dietas_ejecutor',
    'VEC_DIETAS_PERSONAL_ASIGNACION_DATABASE_URL': 'vec_personal_d7_ejecutor',
    'VEC_DIETAS_PERSONAL_AUDITORIA_FRONTERA_DATABASE_URL': 'vec_personal_registrador_frontera',
}
MANIFIESTO = {
    'dsn_registro_identidad': 'vec_identidad_sesiones_v1_registrador',
    'dsn_revalidacion_identidad': 'vec_identidad_sesiones_v1_revalidador',
    'dsn_contexto': 'vec_contexto_actor_v1_runtime',
    'dsn_fuente_autorizacion': 'vec_autorizacion_fuente',
    'dsn_registro_autorizacion': 'vec_autorizacion_registro',
    'dsn_motivos': 'vec_autorizacion_motivos_evaluador',
    'dsn_auditoria_frontera': 'vec_dietas_registrador_frontera',
}


def preparar(c):
    """Guarda las conexiones sin atribuirles activación ni autoridad funcional."""
    from . import catalogo
    admisiones = catalogo.preparar_dietas(c)
    conexiones(c)
    resultado = {'version': 1, 'copia_ref': c.copia.nombre,
                 'sid_copia': c.copia.leer()['sid_copia'],
                 'conexiones_entorno': len(ENTORNO),
                 'conexiones_manifiesto': len(MANIFIESTO),
                 'admisiones': admisiones, 'activada': False,
                 'pendiente': ['perfil_gobernado', 'asignacion_personal',
                               'manifiesto', 'cartografia', 'recorrido_web']}
    c.guardar(c.carpeta / 'etapa-dietas.json', resultado)
    return resultado


def conexiones(c):
    """Crea una vez los LOGIN propios o coteja los datos privados conservados.

    Reutiliza el aprovisionador de Usuarios y Personal: una membresía directa,
    INHERIT TRUE, SET FALSE, ADMIN FALSE y contraseña nueva por conexión.
    Una respuesta SQL incierta exige conciliar; nunca repite el alta.
    """
    copia = c.copia
    estado = copia.leer()
    nombre = _nombre_instancia(copia)
    _privado(c.raiz, directorio=True)
    _privado(c.material, directorio=True)
    if (estado.get('recorrido_completo') is not True
            or c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();')
               != estado.get('sid_copia')):
        raise ValueError('Dietas requiere la copia completa y su identificador PostgreSQL')
    ca = c.material / 'ca/ca.crt'
    _privado(ca, directorio=False)
    puerto = estado.get('puerto_pg')
    if type(puerto) is not int or not 1024 <= puerto <= 65535:
        raise ValueError('Puerto PostgreSQL de Dietas inválido')
    grupos = ENTORNO | MANIFIESTO
    nombres = [_rol(nombre, 110 + i) for i in range(len(grupos))]
    carpeta = c.carpeta / 'dietas'
    carpeta.mkdir(mode=0o700, exist_ok=True)
    _privado(carpeta, directorio=True)
    resultado = carpeta / 'conexiones.json'
    pendiente = carpeta / 'conexiones-pendientes.json'
    if resultado.exists() or resultado.is_symlink():
        valores = c.leer(resultado)
        if set(valores) != set(grupos):
            raise ValueError('Inventario de conexiones Dietas divergente')
        for campo, login in zip(grupos, nombres):
            u = urlsplit(valores[campo])
            if (u.scheme != 'postgresql' or u.hostname != 'localhost' or u.port != puerto
                    or u.path != '/postgres' or u.username != login or not u.password
                    or u.fragment or parse_qs(u.query, strict_parsing=True) != {
                        'sslmode': ['verify-full'], 'sslrootcert': [str(ca)]}):
                raise ValueError('Conexión Dietas ajena a la copia o sin TLS verificado')
        return valores
    if pendiente.exists() or pendiente.is_symlink():
        raise ValueError('Alta Dietas incierta; conciliar las once cuentas originales')
    claves = [secrets.token_hex(32) for _ in nombres]
    vence = (c.ahora() + timedelta(days=90)).strftime('%Y-%m-%dT%H:%M:%SZ')
    c.guardar(pendiente, {'version': 1, 'copia_ref': copia.nombre,
                         'sid_copia': estado['sid_copia'], 'logins': nombres})
    c.sql(sql_logins(nombres, claves, list(grupos.values()), vence), 'dietas-conexiones.log')
    valores = {campo: 'postgresql://' + quote(login) + ':' + quote(clave)
               + '@localhost:' + str(puerto) + '/postgres?' + urlencode({
                   'sslmode': 'verify-full', 'sslrootcert': str(ca)})
               for campo, login, clave in zip(grupos, nombres, claves)}
    c.guardar(resultado, valores)
    return valores
