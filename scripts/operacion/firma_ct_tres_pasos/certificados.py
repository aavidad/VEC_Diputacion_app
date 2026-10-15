#!/usr/bin/env python3
"""Prepara material privado; no publica identidad ni asigna permisos."""
import argparse
from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
import re
from pathlib import Path
import stat
import subprocess


ARCHIVOS_PRIVADOS = ('cliente.key', 'cliente.p12', 'pkcs12.password')


def rechazar(clave):
    raise SystemExit(json.dumps({"estado": "rechazado", "clave": clave}, ensure_ascii=False))


def ruta_privada(texto, directorio=False):
    ruta = Path(texto)
    if not ruta.is_absolute() or '..' in ruta.parts or not re.fullmatch(r'[A-Za-z0-9_./:-]+', str(ruta)):
        rechazar('ruta_absoluta')
    for componente in [*reversed(ruta.parents), ruta]:
        if componente.is_symlink():
            rechazar('ruta_sin_enlaces')
    info = ruta.stat()
    if info.st_uid != os.getuid() or (stat.S_ISDIR(info.st_mode) if directorio else stat.S_ISREG(info.st_mode)) is False:
        rechazar('propietario_tipo')
    if stat.S_IMODE(info.st_mode) & 0o077:
        rechazar('permisos_privados')
    return ruta


def ejecutar(*argumentos, salida=False, codigo_error='openssl'):
    r = subprocess.run(argumentos, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30, check=False)
    if r.returncode:
        rechazar(codigo_error)
    return r.stdout if salida else None


def verificar_crl(crl, ca):
    resultado = subprocess.run(('openssl', 'crl', '-in', str(crl), '-verify',
                                '-CAfile', str(ca), '-noout'), stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, timeout=30, check=False)
    mensajes = resultado.stdout.splitlines() + resultado.stderr.splitlines()
    if resultado.returncode or not any(linea.strip() == b'verify OK' for linea in mensajes):
        rechazar('crl_firma')


def fecha_openssl(valor):
    return datetime.strptime(valor.split('=', 1)[1].strip(), '%b %d %H:%M:%S %Y %Z').replace(tzinfo=timezone.utc)


def vigencia_certificado(crt, dias):
    fechas = ejecutar('openssl', 'x509', '-in', str(crt), '-noout', '-dates', salida=True).decode().splitlines()
    if len(fechas) != 2 or fechas[0].split('=', 1)[0] != 'notBefore' or fechas[1].split('=', 1)[0] != 'notAfter':
        rechazar('vigencia_certificado')
    desde, hasta = map(fecha_openssl, fechas)
    if desde > datetime.now(timezone.utc) or hasta - desde < timedelta(days=dias) - timedelta(minutes=1):
        rechazar('vigencia_certificado')


def vigencia_crl(crl, ca, dias):
    verificar_crl(crl, ca)
    fechas = ejecutar('openssl', 'crl', '-in', str(crl), '-noout', '-lastupdate', '-nextupdate', salida=True).decode().splitlines()
    if len(fechas) != 2 or fechas[0].split('=', 1)[0] != 'lastUpdate' or fechas[1].split('=', 1)[0] != 'nextUpdate':
        rechazar('vigencia_crl')
    desde, hasta = map(fecha_openssl, fechas)
    if desde > datetime.now(timezone.utc) or hasta <= datetime.now(timezone.utc) or hasta - desde < timedelta(days=dias) - timedelta(minutes=1):
        rechazar('vigencia_crl')


def revocados(crl):
    texto = ejecutar('openssl', 'crl', '-in', str(crl), '-noout', '-text', salida=True).decode()
    return {int(numero.replace(':', ''), 16) for numero in re.findall(r'Serial Number:\s*([0-9A-Fa-f:]+)', texto)}


def numero_crl(crl):
    valor = ejecutar('openssl', 'crl', '-in', str(crl), '-noout', '-crlnumber', salida=True).decode().strip()
    if not valor.startswith('crlNumber=0x'):
        rechazar('crl_origen_numero')
    return int(valor.split('=', 1)[1], 16)


def comprobar(ca, clave, dias):
    certificado = ejecutar('openssl', 'x509', '-in', str(ca), '-pubkey', '-noout', salida=True)
    publica = ejecutar('openssl', 'pkey', '-in', str(clave), '-pubout', salida=True)
    if certificado != publica:
        rechazar('ca_clave_coincidente')
    ejecutar('openssl', 'x509', '-in', str(ca), '-checkend', str((dias + 1) * 86400),
            '-noout', codigo_error='ca_vigencia')


def preparar(args):
    if not 1 <= args.cert_dias <= 3650 or not 1 <= args.crl_dias <= 3650:
        rechazar('vigencia_parametros')
    carpeta = Path(__file__).resolve().parent
    catalogo = json.loads((carpeta / 'firmantes.json').read_text())
    if catalogo.get('version') != 1 or len(catalogo.get('personas', [])) != 3:
        rechazar('catalogo_tres_firmantes')
    material = ruta_privada(args.material, True)
    ca = material / 'ca' / 'ca.crt'
    clave = ruta_privada(str(material / 'ca' / 'ca.key'))
    indice = ruta_privada(str(material / 'ca' / 'index.txt'))
    crl_origen = ruta_privada(str(material / 'identidad' / 'clientes.crl'))
    if ca.is_symlink() or not ca.is_file():
        rechazar('ca_certificado')
    destino = ruta_privada(args.salida, True)
    # El destino es independiente del material activo: nunca cambia manifiesto,
    # CA/serie, identidad.json, Caddy ni los perfiles de la aplicación.
    if destino == material or material in destino.parents or destino in material.parents:
        rechazar('destino_independiente')
    comprobar(ca, clave, max(args.cert_dias, args.crl_dias))
    verificar_crl(crl_origen, ca)
    siguiente_crl = numero_crl(crl_origen) + 1
    revocados_origen = revocados(crl_origen)
    revocados_indice = set()
    indice_datos = indice.read_bytes()
    try:
        lineas_indice = indice_datos.decode('utf-8').splitlines()
    except UnicodeDecodeError:
        rechazar('indice_revocaciones')
    for linea in lineas_indice:
        campos = linea.split('\t')
        if campos and campos[0] == 'R':
            if len(campos) < 4 or not re.fullmatch(r'[0-9A-Fa-f]+', campos[3]):
                rechazar('indice_revocaciones')
            revocados_indice.add(int(campos[3], 16))
    if not revocados_origen.issubset(revocados_indice):
        rechazar('indice_revocaciones')
    revocados_esperados = revocados_origen | revocados_indice
    if args.comprobar:
        print(json.dumps({'estado': 'comprobado', 'personas': 3, 'perfiles_asignados': False}))
        return
    if args.aplicar != 'PREPARA-CERTIFICADOS-CT':
        rechazar('confirmacion')
    manifiesto_path = destino / 'certificados.json'
    if manifiesto_path.exists():
        manifiesto = json.loads(ruta_privada(str(manifiesto_path)).read_text())
        if manifiesto.get('catalogo_sha256') != hashlib.sha256((carpeta / 'firmantes.json').read_bytes()).hexdigest():
            rechazar('replay_catalogo')
        anteriores = manifiesto.get('personas', [])
        if len(anteriores) != 3 or manifiesto.get('version') != 1:
            rechazar('replay_tres_firmantes')
        if manifiesto.get('cert_dias') != args.cert_dias or manifiesto.get('crl_dias') != args.crl_dias:
            rechazar('replay_vigencias')
        for persona, prevista in zip(anteriores, catalogo['personas']):
            if any(persona.get(k) != v for k, v in prevista.items()):
                rechazar('replay_persona_catalogo')
            directorio = destino / persona['usuario_web']
            crt = ruta_privada(str(directorio / 'cliente.crt'))
            der = ejecutar('openssl', 'x509', '-in', str(crt), '-outform', 'DER', salida=True)
            if hashlib.sha256(der).hexdigest() != persona['certificado_sha256']:
                rechazar('replay_certificado')
            ejecutar('openssl', 'verify', '-CAfile', str(ca), '-purpose', 'sslclient', str(crt))
            vigencia_certificado(crt, args.cert_dias)
            huellas = persona.get('archivos_sha256')
            if not isinstance(huellas, dict) or set(huellas) != set(ARCHIVOS_PRIVADOS):
                rechazar('replay_archivos_privados')
            for nombre in ARCHIVOS_PRIVADOS:
                archivo = directorio / nombre
                if not archivo.exists():
                    rechazar('replay_archivos_privados')
                ruta_privada(str(archivo))
                if (stat.S_IMODE(archivo.stat().st_mode) != 0o600 or
                        hashlib.sha256(archivo.read_bytes()).hexdigest() != huellas[nombre]):
                    rechazar('replay_archivos_privados')
            clave_publica = ejecutar('openssl', 'pkey', '-in', str(directorio / 'cliente.key'),
                                    '-pubout', salida=True)
            certificado_publico = ejecutar('openssl', 'x509', '-in', str(crt), '-pubkey', '-noout', salida=True)
            if clave_publica != certificado_publico:
                rechazar('replay_clave_certificado')
        crl_replay = ruta_privada(str(destino / 'clientes.crl'))
        vigencia_crl(crl_replay, ca, args.crl_dias)
        if numero_crl(crl_replay) < siguiente_crl or not revocados_esperados.issubset(revocados(crl_replay)):
            rechazar('replay_revocaciones')
        fragmento_path = destino / 'caddy-firmantes.caddy'
        if not fragmento_path.exists():
            rechazar('replay_fragmento')
        fragmento_replay = ruta_privada(str(fragmento_path))
        if (stat.S_IMODE(fragmento_replay.stat().st_mode) != 0o600 or
                manifiesto.get('fragmento_sha256') != hashlib.sha256(fragmento_replay.read_bytes()).hexdigest()):
            rechazar('replay_fragmento')
        print(json.dumps({'estado': 'replay', 'personas': 3, 'perfiles_asignados': False}))
        return
    if list(destino.iterdir()):
        rechazar('destino_vacio')
    personas = []
    for persona in catalogo['personas']:
        usuario = persona['usuario_web']
        if usuario not in ('rrhh_jefatura', 'rrhh_diputacion', 'rrhh_secretaria'):
            rechazar('usuario_catalogo')
        directorio = destino / usuario
        directorio.mkdir(mode=0o700)
        key = directorio / 'cliente.key'
        csr = directorio / 'cliente.csr'
        crt = directorio / 'cliente.crt'
        ext = directorio / 'extensiones.cnf'
        ext.write_text('basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=clientAuth\nsubjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n')
        ejecutar('openssl', 'genpkey', '-algorithm', 'RSA', '-pkeyopt', 'rsa_keygen_bits:3072', '-out', str(key))
        ejecutar('openssl', 'req', '-new', '-utf8', '-key', str(key), '-subj', '/CN=' + persona['nombre'], '-out', str(csr))
        serial = '0x' + os.urandom(16).hex()
        ejecutar('openssl', 'x509', '-req', '-in', str(csr), '-CA', str(ca), '-CAkey', str(clave), '-set_serial', serial,
                 '-days', str(args.cert_dias), '-sha256', '-extfile', str(ext), '-out', str(crt))
        ejecutar('openssl', 'verify', '-CAfile', str(ca), '-purpose', 'sslclient', str(crt))
        vigencia_certificado(crt, args.cert_dias)
        password = directorio / 'pkcs12.password'
        password.write_bytes(ejecutar('openssl', 'rand', '-base64', '32', salida=True))
        ejecutar('openssl', 'pkcs12', '-export', '-inkey', str(key), '-in', str(crt), '-certfile', str(ca),
                 '-name', persona['nombre'], '-out', str(directorio / 'cliente.p12'), '-passout', 'file:' + str(password))
        der = ejecutar('openssl', 'x509', '-in', str(crt), '-outform', 'DER', salida=True)
        huella_certificado = hashlib.sha256(der).hexdigest()
        csr.unlink()
        ext.unlink()
        for fichero in directorio.iterdir():
            fichero.chmod(0o600)
        personas.append({**persona, 'certificado_sha256': huella_certificado,
                         'archivos_sha256': {nombre: hashlib.sha256((directorio / nombre).read_bytes()).hexdigest()
                                             for nombre in ARCHIVOS_PRIVADOS}})
    # La nueva lista conserva las revocaciones de la CA existente.
    # La admisión sigue cotejando además el vínculo vigente de CA25 en la base.
    crl_temporal = destino / 'crl-preparacion'
    crl_temporal.mkdir(mode=0o700)
    (crl_temporal / 'index.txt').write_bytes(indice_datos)
    numero_hex = format(siguiente_crl, 'X')
    if len(numero_hex) % 2:
        numero_hex = '0' + numero_hex
    (crl_temporal / 'crlnumber').write_text(numero_hex + '\n')
    config_crl = crl_temporal / 'ca.cnf'
    config_crl.write_text('[ ca ]\ndefault_ca = desarrollo\n[ desarrollo ]\n'
                          'database = ' + str(crl_temporal / 'index.txt') + '\n'
                          'crlnumber = ' + str(crl_temporal / 'crlnumber') + '\n'
                          'private_key = ' + str(clave) + '\n'
                          'certificate = ' + str(ca) + '\n'
                          'default_md = sha256\ndefault_crl_days = ' + str(args.crl_dias) + '\n')
    ejecutar('openssl', 'ca', '-gencrl', '-config', str(config_crl), '-out', str(destino / 'clientes.crl'))
    vigencia_crl(destino / 'clientes.crl', ca, args.crl_dias)
    if not revocados_esperados.issubset(revocados(destino / 'clientes.crl')):
        rechazar('crl_revocaciones')
    (destino / 'clientes.crl').chmod(0o600)
    for fichero in crl_temporal.iterdir():
        fichero.unlink()
    crl_temporal.rmdir()
    caddy = []
    for persona in personas:
        usuario = persona['usuario_web']
        carpeta_usuario = destino / usuario
        # Este fragmento se inserta DESPUÉS del basic_auth vigente. No escribe
        # ni recarga Caddy y no convierte el usuario web en permiso de VEC.
        caddy.append('@ct_' + usuario + ' expression `{http.auth.user.id} == "' + usuario + '"`\n'
                     'handle @ct_' + usuario + ' {\n'
                     '    reverse_proxy https://127.0.0.1:18443 {\n'
                     '        header_up -Authorization\n'
                     '        transport http {\n'
                     '            tls_server_name localhost\n'
                     '            tls_client_auth ' + str(carpeta_usuario / 'cliente.crt') + ' ' + str(carpeta_usuario / 'cliente.key') + '\n'
                     '            tls_trust_pool file {\n'
                     '                pem_file ' + str(ca) + '\n'
                     '            }\n'
                     '        }\n'
                     '    }\n'
                     '}\n')
    fragmento = destino / 'caddy-firmantes.caddy'
    with fragmento.open('x') as f:
        f.write('\n'.join(caddy))
    fragmento.chmod(0o600)
    manifiesto = {'version': 1, 'cert_dias': args.cert_dias, 'crl_dias': args.crl_dias,
                  'catalogo_sha256': hashlib.sha256((carpeta / 'firmantes.json').read_bytes()).hexdigest(),
                  'fragmento_sha256': hashlib.sha256(fragmento.read_bytes()).hexdigest(),
                  'personas': personas, 'perfiles_asignados': False}
    with manifiesto_path.open('x') as f:
        json.dump(manifiesto, f, ensure_ascii=False, indent=2)
        f.write('\n')
    manifiesto_path.chmod(0o600)
    print(json.dumps({'estado': 'preparado', 'personas': 3, 'perfiles_asignados': False}))


if __name__ == '__main__':
    os.umask(0o077)
    parser = argparse.ArgumentParser()
    modos = parser.add_mutually_exclusive_group(required=True)
    modos.add_argument('--comprobar', action='store_true')
    modos.add_argument('--aplicar')
    parser.add_argument('--material', required=True)
    parser.add_argument('--salida', required=True)
    parser.add_argument('--cert-dias', type=int, default=400)
    parser.add_argument('--crl-dias', type=int, default=90)
    argumentos = parser.parse_args()
    try:
        preparar(argumentos)
    except (OSError, ValueError, KeyError, subprocess.TimeoutExpired):
        rechazar('material_entrada_o_salida')
