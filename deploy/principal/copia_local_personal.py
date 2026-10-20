"""Personal en la copia local: material propio desde el gobierno ya existente."""
from copy import deepcopy
from datetime import datetime, timedelta, timezone
import json
import http.client
import ssl
import secrets
import subprocess
from urllib.parse import quote, urlencode, urlsplit, parse_qs

from copia_local_conexiones import _publicar, _rol
from copia_local_firma import privado, _sha256
from copia_local_usuarios import sql as sql_logins

BASE = {
    'fuente_autorizacion': 'vec_autorizacion_fuente',
    'motivos_autorizacion': 'vec_autorizacion_motivos_evaluador',
    'registro_ct': 'vec_contratacion_temporal_ejecutor',
}
B2 = {'personal_planes': 'vec_personal_ejecutor',
      'personal_actos': 'vec_personal_ejecutor',
      'personal_consultas': 'vec_personal_ejecutor',
      'bolsa_persona': 'vec_bolsa_llamamientos_ejecutor',
      'rpt': 'vec_personal_ejecutor'}
CESE = 'vec_bolsa_llamamientos_relevo_cese'
# Nombres de las 27 filas AD172 restauradas (origen_consumos_ad172/ternas.tsv).
# Como en Usuarios, el LOGIN se crea en esta copia con una contraseña nueva;
# no se altera la fila de origen ni se traslada una conexión de la principal.
LOGINS_ORIGEN = {
    'base-registro_ct': 'vec_ct_personal_b2_registro_ct',
    'b2-personal_planes': 'vec_ct_personal_b2_planes',
    'b2-personal_actos': 'vec_ct_personal_b2_actos',
    'b2-personal_consultas': 'vec_ct_personal_b2_consultas',
    'b2-bolsa_persona': 'vec_ct_personal_b2_bolsa_persona',
    'b2-rpt': 'vec_ct_personal_b2_rpt',
}
OBLIGATORIAS = {'ct_vinculo_registrar', 'rpt_listar', 'personal_hecho',
                'personal_catalogo_publicar', 'personal_catalogo_retirar'}


def configuracion(origen):
    """Conserva referencias/motivos; descarta el circuito de ejercicio y sus secretos."""
    datos = json.loads(privado(origen).read_text())
    b2 = deepcopy({k: datos['personal_b2'][k] for k in (
        'protocolo', 'organismo_ref', 'catalogo_rpt_id', 'modulo_rpt_id',
        'dsn_files', 'operaciones', 'cese_fecha_efecto') if k in datos['personal_b2']})
    if (b2.get('protocolo') != 'personal_b2_v1'
            or set(b2.get('dsn_files', {})) != set(B2)
            or not OBLIGATORIAS <= set(b2.get('operaciones', {}))
            or b2.get('cese_fecha_efecto') not in ('ultimo_dia_trabajado', 'primer_dia_sin_relacion')):
        raise ValueError('La configuración aprobada no incluye categoría RPT, catálogos y cese B2')
    # La selección del publicador solo lee este inventario. Las capacidades
    # activables las escribe después vec-preparar-material-interno cotejando BD.
    for nombre, op in b2['operaciones'].items():
        op['capacidad'] = {'file': nombre + '.cap'}
    return {'esquema': datos['esquema'], 'referencias': datos['referencias'],
            'dsn_files': {k: 'base-' + k + '.dsn' for k in BASE},
            'personal_b2': {**b2, 'dsn_files': {k: 'b2-' + k + '.dsn' for k in B2}}}


def guardar_entorno(copia, entorno, guardar):
    ruta = copia.raiz / 'entorno_modulos.json'
    guardar(ruta, entorno)
    estado = copia.leer()
    estado['entorno_modulos_sha256'] = _sha256(ruta)
    guardar(copia.estado, estado)


def conexiones(copia, carpeta, guardar):
    """Nueve LOGIN propios; reutiliza el aprovisionador nominal de la copia."""
    marca = carpeta / 'conexiones.json'
    grupos = [*BASE.values(), *B2.values(), CESE]
    campos = [*('base-' + k for k in BASE), *('b2-' + k for k in B2), 'cese']
    nombres = [LOGINS_ORIGEN.get(campo, _rol(copia.nombre, i + 90)) for i, campo in enumerate(campos)]
    if marca.exists():
        urls = json.loads(privado(marca).read_text())
        if set(urls) != set(campos):
            raise ValueError('Las conexiones de Personal no corresponden al contrato')
        for campo, login in zip(campos, nombres):
            url = urlsplit(urls[campo])
            if (url.scheme != 'postgresql' or url.hostname != 'localhost'
                    or url.port != copia.leer()['puerto_pg'] or url.path != '/postgres'
                    or url.username != login or not url.password or url.fragment
                    or parse_qs(url.query) != {'sslmode': ['verify-full'],
                        'sslrootcert': [str(copia.material / 'ca/ca.crt')]}):
                raise ValueError('Conexión de Personal ajena a esta copia o sin TLS verificado')
        return urls
    claves = [secrets.token_hex(32) for _ in grupos]
    vence = (datetime.now(timezone.utc) + timedelta(days=90)).strftime('%Y-%m-%dT%H:%M:%SZ')
    copia.sql_bytes(sql_logins(nombres, claves, grupos, vence), 'personal-conexiones.log')
    estado = copia.leer()
    urls = ['postgresql://' + quote(n) + ':' + quote(c) + '@localhost:' + str(estado['puerto_pg'])
            + '/postgres?' + urlencode({'sslmode': 'verify-full',
                'sslrootcert': str(copia.material / 'ca/ca.crt')}) for n, c in zip(nombres, claves)]
    resultado = dict(zip(campos, urls))
    # El resultado persiste antes de los archivos derivados para recuperar una
    # interrupción sin generar otras identidades ni cambiar sus credenciales.
    guardar(marca, resultado)
    return resultado


def entorno_personal(copia, carpeta, urls, repo):
    entorno = copia.entorno_modulos()
    entorno.update(VEC_CT_INCORPORACION_V2_FILE=str(carpeta / 'servidor.json'),
        VEC_PERSONAL_B2_GOBIERNO_ENABLED='true', VEC_CT_INCORPORACION_ACREDITADA_ENABLED='true',
        VEC_CT_SEGUIMIENTO_CESE_ENABLED='true', VEC_BOLSA_CESE_CT_ENABLED='true',
        VEC_BOLSA_CESE_CT_DATABASE_URL=urls['cese'])
    if entorno.get('VEC_BOLSA_CONTRATOS_CT_INTERVALO', '30s') in ('0', '0s'):
        raise ValueError('El relevo de contratos está apagado; Personal necesita su intervalo activo')
    for variable, fuente in {
        'VEC_CT_CAUSAS_CESE_SOURCE_PATH': 'data/demo/reglas/ct_causas_cese.demo.json',
        'VEC_CT_ANALISIS_RECTIFICACION_MOTIVOS_SOURCE_PATH': 'data/demo/reglas/ct_motivos_rectificacion.demo.json',
    }.items():
        entorno.setdefault(variable, str(repo / fuente))
    return entorno


def comprobar_personal(copia):
    """Comprueba consumidores reales, además del arranque general de la copia."""
    contexto = ssl.create_default_context(cafile=str(copia.material / 'ca/ca.crt'))
    contexto.load_cert_chain(str(copia.material / 'mtls/cliente.crt'), str(copia.material / 'mtls/cliente.key'))
    for ruta in ('/api/vec/contratacion-temporal/incorporacion-personal-b2/categorias-rpt/v1',
                 '/api/vec/personal/catalogos-registro-empleado?tipo=regimen&estado=publicada&limite=25'):
        conexion = http.client.HTTPSConnection('localhost', copia.leer()['puerto_app'], context=contexto, timeout=30)
        try:
            conexion.request('GET', ruta, headers={'Accept': 'application/json'})
            respuesta = conexion.getresponse()
            respuesta.read()
            if respuesta.status != 200:
                raise ValueError(f'Personal no está listo: HTTP {respuesta.status} en {ruta}')
        finally:
            conexion.close()


def preparar(copia, origen, *, repo, go, entorno_go, guardar):
    """Publica con vec-server y deriva con el productor B2; no escribe gobierno por SQL."""
    anterior = copia.entorno_modulos()
    if not anterior:
        raise ValueError('Personal necesita una copia creada con --modulos')
    datos = configuracion(origen)
    carpeta = copia.raiz / 'personal'
    carpeta.mkdir(mode=0o700, exist_ok=True)
    privado(carpeta, directorio=True)
    entrada = carpeta / 'entrada'
    entrada.mkdir(mode=0o700, exist_ok=True)
    privado(entrada, directorio=True)
    salida = carpeta / 'material'
    firma = _sha256(origen)
    marca = carpeta / 'preparada.json'
    if marca.exists():
        if json.loads(privado(marca).read_text())['origen_sha256'] != firma:
            raise ValueError('Personal ya está preparado con otra configuración; conserve la copia')
        if _sha256(privado(salida / 'servidor.json')) != json.loads(privado(marca).read_text())['servidor_sha256']:
            raise ValueError('El material de Personal cambió desde la preparación')
        copia.comprobar()
        comprobar_personal(copia)
        return
    # Conservar la configuración sana también si se interrumpe el proceso
    # durante la publicación y la siguiente llamada encuentra el borrador.
    recuperacion = carpeta / 'preparacion.json'
    if recuperacion.exists():
        previa = json.loads(privado(recuperacion).read_text())
        if previa['origen_sha256'] != firma:
            raise ValueError('La preparación pendiente usa otra configuración; conserve la copia')
        anterior = previa['entorno_anterior']
    else:
        guardar(recuperacion, {'origen_sha256': firma, 'entorno_anterior': anterior})
    # Comprobación del material antes de parar una aplicación sana.
    privado(copia.material / 'idempotencia/configuracion.json')
    binario = carpeta / 'preparar-material'
    with (carpeta / 'compilacion.log').open('wb') as log:
        subprocess.run([go, 'build', '-p', '6', '-o', str(binario), './cmd/vec-preparar-material-interno'],
                       cwd=repo, env=entorno_go, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=600)
    copia.parar()
    try:
        copia.pg()
        urls = conexiones(copia, carpeta, guardar)
        for nombre, url in urls.items():
            ruta = entrada / (nombre + '.dsn')
            if not ruta.exists():
                _publicar(ruta, url + '\n')
        for nombre, variable in {'motivos-rrhh': 'VEC_CT_MOTIVOS_RRHH_DATABASE_URL'}.items():
            guardar_dsn = entrada / (nombre + '.dsn')
            if not guardar_dsn.exists():
                _publicar(guardar_dsn, anterior[variable] + '\n')
        guardar(entrada / 'servidor.json', datos)
        copia.sql(repo / 'deploy/principal/candidata_hz39/personal/datos_plantilla.sql', 'personal-plantilla.log')
        borrador = entorno_personal(copia, entrada, urls, repo)
        guardar_entorno(copia, borrador, guardar)
        # Esta fase publica las audiencias. Aún no sirve operaciones de Personal:
        # no hay .cap hasta que el productor las coteje con el gobierno publicado.
        copia.arrancar(comprobar_modulos=False)
        if not salida.exists():
            from copia_local_conexiones import crear_preparador_gobierno, retirar_preparador_gobierno
            gobierno = entrada / 'gobierno.dsn'
            try:
                _publicar(gobierno, crear_preparador_gobierno(copia) + '\n')
                with (carpeta / 'material.log').open('ab') as log:
                    subprocess.run([str(binario), '-material-idempotencia', str(copia.material / 'idempotencia'),
                        '-incorporacion-config', str(entrada / 'servidor.json'),
                        '-incorporacion-motivos-rrhh-dsn', str(entrada / 'motivos-rrhh.dsn'),
                        '-dsn-archivo', str(gobierno), '-salida', str(salida)],
                        cwd=repo, env={'PATH': '/usr/bin:/bin', 'LANG': 'C.UTF-8'},
                        stdout=log, stderr=subprocess.STDOUT, check=True, timeout=180)
            finally:
                gobierno.unlink(missing_ok=True)
                retirar_preparador_gobierno(copia)
        privado(salida, directorio=True)
        privado(salida / 'servidor.json')
        copia.parar()
        guardar_entorno(copia, entorno_personal(copia, salida, urls, repo), guardar)
        copia.arrancar()
        comprobar_personal(copia)
        guardar(marca, {'origen_sha256': firma, 'servidor_sha256': _sha256(salida / 'servidor.json')})
        recuperacion.unlink(missing_ok=True)
    except Exception:
        # El fallo conserva los recibos/material para conciliar; recupera el
        # arranque anterior. Nunca convierte un fallo de preparación en éxito.
        copia.parar()
        guardar_entorno(copia, anterior, guardar)
        copia.arrancar()
        # Una vuelta atrás terminada deja de ser una preparación interrumpida.
        # El próximo intento debe capturar la configuración sana que haya entonces.
        recuperacion.unlink(missing_ok=True)
        raise
