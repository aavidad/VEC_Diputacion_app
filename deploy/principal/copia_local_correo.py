"""Buzón local sin reenvío, con STARTTLS y destinatarios reservados."""
import json
import os
import smtplib
import ssl
import subprocess
import time

IMAGEN = 'axllent/mailpit:v1.27.8'


def existente(copia):
    nombre = copia.nombre + '-correo'
    resultado = subprocess.run(['docker', 'container', 'inspect', nombre],
                               capture_output=True, check=False)
    if resultado.returncode:
        copia.docker('info', salida=subprocess.DEVNULL)
        return False
    datos = json.loads(resultado.stdout)[0]
    if (datos['Config'].get('Labels', {}).get('vec.copia') != str(copia.raiz)
            or datos['Config'].get('Labels', {}).get('vec.funcion') != 'correo-local'):
        raise ValueError('El buzón pertenece a otra copia')
    return True


def arrancar(copia):
    estado = copia.leer()
    puerto = estado.get('puerto_smtp')
    if not puerto:
        return {}
    if not existente(copia):
        copia.docker('run', '--pull=never', '-d', '--rm', '--restart=no',
                     '--name', copia.nombre + '-correo', '--memory=128m', '--cpus=1',
                     '--user', f'{os.getuid()}:{os.getgid()}',
                     '--pids-limit=64', '--read-only', '--cap-drop=ALL',
                     '--security-opt=no-new-privileges', '--tmpfs', '/tmp:rw,noexec,nosuid,size=32m',
                     '--label', f'vec.copia={copia.raiz}', '--label', 'vec.funcion=correo-local',
                     '-p', f'127.0.0.1:{puerto}:1025',
                     '-v', f'{copia.material}/tls:/tls:ro', IMAGEN,
                     '--disable-version-check', '--block-remote-css-and-fonts',
                     '--smtp-disable-rdns', '--smtp-require-starttls',
                     '--smtp-tls-cert', '/tls/servidor.crt', '--smtp-tls-key', '/tls/servidor.key',
                     '--smtp-allowed-recipients', r'^[A-Za-z0-9._+\-]+@example\.test$',
                     '--database', '/tmp/mailpit.db', '--max', '100', '--quiet',
                     salida=subprocess.DEVNULL)
    contexto = ssl.create_default_context(cafile=str(copia.material / 'ca/ca.crt'))
    for _ in range(40):
        try:
            with smtplib.SMTP('127.0.0.1', puerto, timeout=2) as cliente:
                cliente.starttls(context=contexto)
                cliente.ehlo()
                if cliente.mail('rrhh@example.test')[0] != 250:
                    raise ValueError('El buzón rechazó el remitente local')
                if cliente.rcpt('destino@example.invalid')[0] < 500:
                    raise ValueError('El buzón acepta destinatarios ajenos a la copia')
                if cliente.rcpt('destino@example.test')[0] != 250:
                    raise ValueError('El buzón rechazó el destinatario local')
            return dict(VEC_SMTP_HOST='127.0.0.1', VEC_SMTP_PORT=str(puerto),
                        VEC_SMTP_FROM='rrhh@example.test', VEC_SMTP_MODO_TLS='starttls',
                        VEC_SMTP_CA_FILE=str(copia.material / 'ca/ca.crt'))
        except (OSError, smtplib.SMTPException):
            time.sleep(.25)
    raise ValueError('El buzón local no confirmó STARTTLS')


def parar(copia):
    if copia.leer().get('puerto_smtp') and existente(copia):
        copia.docker('stop', '--time', '10', copia.nombre + '-correo',
                     salida=subprocess.DEVNULL)
