#!/usr/bin/env python3
"""Prepara material privado; no publica identidad ni asigna permisos."""
import argparse
import hashlib
import json
import os
import re
from pathlib import Path
import stat
import subprocess


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


def ejecutar(*argumentos, salida=False):
    r = subprocess.run(argumentos, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30, check=False)
    if r.returncode:
        rechazar('openssl')
    return r.stdout if salida else None


def comprobar(ca, clave):
    certificado = ejecutar('openssl', 'x509', '-in', str(ca), '-pubkey', '-noout', salida=True)
    publica = ejecutar('openssl', 'pkey', '-in', str(clave), '-pubout', salida=True)
    if certificado != publica:
        rechazar('ca_clave_coincidente')
    ejecutar('openssl', 'x509', '-in', str(ca), '-checkend', '86400', '-noout')


def preparar(args):
    carpeta = Path(__file__).resolve().parent
    catalogo = json.loads((carpeta / 'firmantes.json').read_text())
    if catalogo.get('version') != 1 or len(catalogo.get('personas', [])) != 3:
        rechazar('catalogo_tres_firmantes')
    material = ruta_privada(args.material, True)
    ca = material / 'ca' / 'ca.crt'
    clave = ruta_privada(str(material / 'ca' / 'ca.key'))
    if ca.is_symlink() or not ca.is_file():
        rechazar('ca_certificado')
    destino = ruta_privada(args.salida, True)
    # El destino es independiente del material activo: nunca cambia manifiesto,
    # CA/serie, identidad.json, Caddy ni los perfiles de la aplicación.
    if destino == material or material in destino.parents or destino in material.parents:
        rechazar('destino_independiente')
    comprobar(ca, clave)
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
        for persona, prevista in zip(anteriores, catalogo['personas']):
            if any(persona.get(k) != v for k, v in prevista.items()):
                rechazar('replay_persona_catalogo')
            crt = ruta_privada(str(destino / persona['usuario_web'] / 'cliente.crt'))
            der = ejecutar('openssl', 'x509', '-in', str(crt), '-outform', 'DER', salida=True)
            if hashlib.sha256(der).hexdigest() != persona['certificado_sha256']:
                rechazar('replay_certificado')
            ejecutar('openssl', 'verify', '-CAfile', str(ca), '-purpose', 'sslclient', str(crt))
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
                 '-days', '7', '-sha256', '-extfile', str(ext), '-out', str(crt))
        ejecutar('openssl', 'verify', '-CAfile', str(ca), '-purpose', 'sslclient', str(crt))
        password = directorio / 'pkcs12.password'
        password.write_bytes(ejecutar('openssl', 'rand', '-base64', '32', salida=True))
        ejecutar('openssl', 'pkcs12', '-export', '-inkey', str(key), '-in', str(crt), '-certfile', str(ca),
                 '-name', persona['nombre'], '-out', str(directorio / 'cliente.p12'), '-passout', 'file:' + str(password))
        der = ejecutar('openssl', 'x509', '-in', str(crt), '-outform', 'DER', salida=True)
        personas.append({**persona, 'certificado_sha256': hashlib.sha256(der).hexdigest()})
        csr.unlink()
        ext.unlink()
        for fichero in directorio.iterdir():
            fichero.chmod(0o600)
    # La CA existente firma una lista inicial para los certificados nuevos.
    # La admisión sigue cotejando además el vínculo vigente de CA25 en la base.
    crl_temporal = destino / 'crl-preparacion'
    crl_temporal.mkdir(mode=0o700)
    (crl_temporal / 'index.txt').write_text('')
    (crl_temporal / 'crlnumber').write_text('01\n')
    config_crl = crl_temporal / 'ca.cnf'
    config_crl.write_text('[ ca ]\ndefault_ca = desarrollo\n[ desarrollo ]\n'
                          'database = ' + str(crl_temporal / 'index.txt') + '\n'
                          'crlnumber = ' + str(crl_temporal / 'crlnumber') + '\n'
                          'private_key = ' + str(clave) + '\n'
                          'certificate = ' + str(ca) + '\n'
                          'default_md = sha256\ndefault_crl_days = 1\n')
    ejecutar('openssl', 'ca', '-gencrl', '-config', str(config_crl), '-out', str(destino / 'clientes.crl'))
    ejecutar('openssl', 'crl', '-in', str(destino / 'clientes.crl'), '-verify', '-CAfile', str(ca), '-noout')
    (destino / 'clientes.crl').chmod(0o600)
    for fichero in crl_temporal.iterdir():
        fichero.unlink()
    crl_temporal.rmdir()
    manifiesto = {'version': 1, 'catalogo_sha256': hashlib.sha256((carpeta / 'firmantes.json').read_bytes()).hexdigest(),
                  'personas': personas, 'perfiles_asignados': False}
    with manifiesto_path.open('x') as f:
        json.dump(manifiesto, f, ensure_ascii=False, indent=2)
        f.write('\n')
    manifiesto_path.chmod(0o600)
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
    print(json.dumps({'estado': 'preparado', 'personas': 3, 'perfiles_asignados': False}))


if __name__ == '__main__':
    os.umask(0o077)
    parser = argparse.ArgumentParser()
    modos = parser.add_mutually_exclusive_group(required=True)
    modos.add_argument('--comprobar', action='store_true')
    modos.add_argument('--aplicar')
    parser.add_argument('--material', required=True)
    parser.add_argument('--salida', required=True)
    argumentos = parser.parse_args()
    try:
        preparar(argumentos)
    except (OSError, ValueError, KeyError, subprocess.TimeoutExpired):
        rechazar('material_entrada_o_salida')
