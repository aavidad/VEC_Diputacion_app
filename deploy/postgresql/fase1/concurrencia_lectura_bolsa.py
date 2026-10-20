"""Comprueba que la lectura de Bolsa y la decisión V3 no se interbloquean (CA48).

La decisión V3 de una petición acredita el contexto bloqueando los punteros de
la persona en el orden común (cuenta, perfil, vínculo de contexto, persona).
La lectura de datos de contacto de otra petición de la misma persona coteja el
mismo contexto con FOR SHARE. Antes de CA48 la lectura tomaba primero el
vínculo y después la cuenta: con la decisión ya dueña de la cuenta, cada una
esperaba a la otra y PostgreSQL abortaba una con 40P01.

El guion reproduce ese cruce con dos sesiones reales: A bloquea la cuenta como
la acreditación, B coteja (y debe esperar sin retener nada), A bloquea el
vínculo y confirma, y B termina con el cotejo positivo. Sin CA48 una de las
dos termina en 40P01.
"""

import re
import time

from deploy.postgresql.fase1.concurrencia_contexto import Sesion, bloqueado_por, comprobar_contenedor, psql

CUENTA = "cta_ca48_aaaaaaaaaaaaaaaaaaaaaa"
PERSONA = "per_ca48_aaaaaaaaaaaaaaaaaaaaaa"
PERFIL = "prf_ca48_aaaaaaaaaaaaaaaaaaaaaa"
VINCULO = "vca_ca48_aaaaaaaaaaaaaaaaaaaaaa"
PID = re.compile(rb"PID ([1-9][0-9]*)\n")
A_CUENTA = re.compile(rb"A_CUENTA\n")
A_FIN = re.compile(rb"A_FIN ([0-9A-Z]{5})\n")
B_FIN = re.compile(rb"B_FIN ([0-9A-Z]{5}) (\S+)\n")

INICIO = """BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='12s';
SET LOCAL statement_timeout='18s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL application_name='fase1_ca48_{lado}';
SET LOCAL ROLE {rol};
SELECT pg_backend_pid() AS pid \\gset
\\warn PID :pid
"""

# Primer paso de acreditar_uso_registro_contexto_interno_v2 (CA2).
A_BLOQUEA_CUENTA = f"""SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual
 WHERE cuenta_ref='{CUENTA}' FOR UPDATE;
\\warn A_CUENTA
"""

# Tercer paso de la acreditación: el vínculo de contexto, también FOR UPDATE.
A_BLOQUEA_VINCULO = f"""\\set ON_ERROR_STOP off
SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref='{PERFIL}' FOR UPDATE;
SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref='{VINCULO}' FOR UPDATE;
\\warn A_FIN :SQLSTATE
COMMIT;
"""

B_COTEJA = f"""\\set ON_ERROR_STOP off
SELECT vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(convert_to(jsonb_build_object(
 'esquema','vec.contexto-actor.vinculado.v2','estado','activo','cuenta_ref','{CUENTA}',
 'persona_ref','{PERSONA}','perfil_activo_ref','{PERFIL}','contexto_actor_ref','{VINCULO}',
 'cuenta_version',1,'persona_version',1,'perfil_version',1,'contexto_version',1,
 'vigente_hasta',to_char(clock_timestamp()+interval '30 minutes','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
 'UTF8'))::text AS resultado \\gset
\\warn B_FIN :SQLSTATE :resultado
COMMIT;
"""


def esperar_bloqueo(nombre, pid_a, pid_b, sesion_b, plazo=7):
    limite = time.monotonic() + plazo
    while time.monotonic() < limite:
        if sesion_b.proceso.poll() is not None:
            raise ValueError("El cotejo terminó sin esperar a la acreditación")
        if bloqueado_por(nombre, pid_a, pid_b):
            return
        time.sleep(0.1)
    raise ValueError("No se observó la espera real del cotejo")


def preparar(nombre, fixture):
    """Conserva el contexto sintético del fixture CA48 (exige CA48 instalada)."""
    if not isinstance(fixture, bytes) or b"CA48-PRUEBA-OK" not in fixture:
        raise ValueError("Falta el fixture versionado de CA48")
    preparado = psql(nombre, "-v", "ca48_persistir_para_concurrencia=1", entrada=fixture, timeout=40)
    if b"CA48-PRUEBA-OK" not in preparado.stdout:
        raise ValueError("La preparación del contexto CA48 no terminó")


def cruzar(nombre):
    """Reproduce el cruce de la acreditación y el cotejo con dos sesiones."""
    a = b = None
    try:
        a = Sesion(nombre)
        b = Sesion(nombre)
        a.enviar(INICIO.format(lado="a", rol="vec_contexto_actor_v1_propietario"))
        pid_a = int(a.esperar(PID).group(1))
        b.enviar(INICIO.format(lado="b", rol="vec_autorizacion_atestada_v3_propietario"))
        pid_b = int(b.esperar(PID).group(1))
        a.enviar(A_BLOQUEA_CUENTA)
        a.esperar(A_CUENTA)
        b.enviar(B_COTEJA)
        esperar_bloqueo(nombre, pid_a, pid_b, b)
        a.enviar(A_BLOQUEA_VINCULO)
        estado_a = a.esperar(A_FIN).group(1)
        fin_b = b.esperar(B_FIN)
        if estado_a != b"00000":
            raise ValueError(f"La acreditación terminó con SQLSTATE {estado_a.decode()}")
        if fin_b.group(1) != b"00000" or fin_b.group(2) != b"true":
            raise ValueError(f"El cotejo terminó con SQLSTATE {fin_b.group(1).decode()}")
        a.proceso.stdin.close()
        b.proceso.stdin.close()
        if a.proceso.wait(timeout=3) != 0 or b.proceso.wait(timeout=3) != 0:
            raise ValueError("Una sesión SQL terminó con error")
    finally:
        for sesion in (a, b):
            if sesion is not None:
                sesion.terminar()


def ensayar(nombre, etiqueta, fixture):
    comprobar_contenedor(nombre, etiqueta)
    preparar(nombre, fixture)
    cruzar(nombre)
