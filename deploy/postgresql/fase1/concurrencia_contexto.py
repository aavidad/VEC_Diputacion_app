"""Comprueba el CAS concurrente del contexto externo en el contenedor F0-29."""

import json
import os
import re
import select
import subprocess
import time

CASO = "pue_ctx15_sintetico_00000000000"
IMAGEN_PROPIA = re.compile(r"vec-fase1-[0-9a-f]{12}\Z")
PREIMAGEN = re.compile(rb"PREIMAGEN ([1-9][0-9]*) (1) ([0-9a-f]{64})\n")
ESTADO_B = re.compile(rb"B_SQLSTATE ([0-9A-Z]{5})\n")


def psql(nombre, *opciones, entrada=None, timeout=20):
    return subprocess.run([
        "docker", "exec", "-i", "-u", "postgres", nombre, "psql", "-X", "-q", "-A", "-t",
        "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "postgres", *opciones,
    ], input=entrada, capture_output=True, timeout=timeout, check=True)


def comprobar_contenedor(nombre, etiqueta):
    if not IMAGEN_PROPIA.fullmatch(nombre):
        raise ValueError("El nombre no corresponde al contenedor de fase 1")
    inspeccion = subprocess.check_output(["docker", "container", "inspect", nombre], timeout=5)
    datos = json.loads(inspeccion)[0]
    if (datos["Config"].get("Labels", {}).get("vec.fase1") != etiqueta
            or datos["HostConfig"]["NetworkMode"] != "none"
            or datos["State"]["Running"] is not True):
        raise ValueError("El contenedor no pertenece a este ensayo aislado")


class Sesion:
    def __init__(self, nombre):
        self.proceso = subprocess.Popen([
            "docker", "exec", "-i", "-u", "postgres", nombre, "psql", "-X", "-q", "-A", "-t",
            "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "postgres",
        ], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, bufsize=0)
        self.salida = bytearray()
        self.errores = bytearray()

    def enviar(self, contenido):
        if self.proceso.poll() is not None:
            raise ValueError("La sesión SQL terminó antes de la orden")
        self.proceso.stdin.write(contenido.encode("utf-8"))
        self.proceso.stdin.flush()

    def esperar(self, patron, plazo=8):
        limite = time.monotonic() + plazo
        while time.monotonic() < limite:
            coincidencia = patron.search(self.errores)
            if coincidencia:
                return coincidencia
            abiertos = [tubo for tubo in (self.proceso.stdout, self.proceso.stderr) if not tubo.closed]
            if not abiertos:
                break
            listos, _, _ = select.select(abiertos, [], [], min(0.2, max(0, limite-time.monotonic())))
            for tubo in listos:
                bloque = os.read(tubo.fileno(), 4096)
                if bloque:
                    (self.salida if tubo is self.proceso.stdout else self.errores).extend(bloque)
                else:
                    tubo.close()
            if self.proceso.poll() is not None and all(t.closed for t in abiertos):
                break
        raise ValueError("La sesión SQL no llegó al punto de sincronización")

    def terminar(self):
        if self.proceso.poll() is None:
            try:
                self.enviar("ROLLBACK;\n")
            except (BrokenPipeError, ValueError):
                pass
            try:
                self.proceso.stdin.close()
            except OSError:
                pass
            try:
                self.proceso.wait(timeout=2)
            except subprocess.TimeoutExpired:
                self.proceso.terminate()
                try:
                    self.proceso.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    self.proceso.kill()
                    self.proceso.wait(timeout=2)
        for tubo in (self.proceso.stdin, self.proceso.stdout, self.proceso.stderr):
            if not tubo.closed:
                tubo.close()


INICIO = f"""BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='12s';
SET LOCAL statement_timeout='18s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL application_name='fase1_ctx15_{{lado}}';
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SELECT pg_backend_pid() AS pid,a.version::integer AS version,v.huella_sha256 AS huella
 FROM vec_contexto_actor_v1.contexto_externo_actual a
 JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version)
 WHERE a.provision_ref='{CASO}' AND a.version=1 AND v.snapshot->>'estado'='activo' \\gset
\\warn PREIMAGEN :pid :version :huella
"""

# Se obtiene el snapshot vigente real y se construye una sucesora legítima. El
# SQL del productor valida hash, versiones y CAS; el ensayo no escribe sus tablas.
PUBLICAR = f"""DO $publicar$
DECLARE anterior record; nuevo jsonb; clave text; huella text; publicado record;
BEGIN
 SELECT * INTO STRICT anterior FROM vec_contexto_actor_v1.contexto_externo_versiones
  WHERE provision_ref='{CASO}' AND version=1;
 nuevo:=anterior.snapshot;
 FOREACH clave IN ARRAY ARRAY['cuenta','persona','perfil','contexto'] LOOP
  nuevo:=jsonb_set(nuevo,ARRAY[clave],(nuevo->clave)||jsonb_build_object('version',2));
 END LOOP;
 huella:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(nuevo,2);
 SELECT * INTO STRICT publicado FROM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(
  nuevo,1,anterior.huella_sha256,huella);
 IF publicado.version<>2 OR publicado.huella_sha256<>huella THEN
  RAISE EXCEPTION 'recibo del publicador concurrente divergente';
 END IF;
END $publicar$;
"""

VERIFICAR = f"""BEGIN TRANSACTION READ ONLY;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $verificar$
DECLARE actual record; anterior record; caso1 record;
BEGIN
 SELECT v.* INTO STRICT actual FROM vec_contexto_actor_v1.contexto_externo_actual a
  JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version)
  WHERE a.provision_ref='{CASO}';
 SELECT * INTO STRICT anterior FROM vec_contexto_actor_v1.contexto_externo_versiones
  WHERE provision_ref='{CASO}' AND version=1;
 SELECT v.* INTO STRICT caso1 FROM vec_contexto_actor_v1.contexto_externo_actual a
  JOIN vec_contexto_actor_v1.contexto_externo_versiones v USING(provision_ref,version)
  WHERE a.provision_ref='pce_ctx15_sintetico_00000000000';
 IF actual.version<>2 OR actual.snapshot->>'estado'<>'activo'
  OR (actual.snapshot#>>'{{cuenta,version}}')::integer<>2
  OR actual.huella_sha256<>vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(actual.snapshot,2)
  OR anterior.snapshot->>'estado'<>'activo'
  OR anterior.huella_sha256<>vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(anterior.snapshot,1)
  OR caso1.version<>2 OR caso1.snapshot->>'estado'<>'revocado'
  OR (SELECT count(*) FROM vec_contexto_actor_v1.contexto_externo_versiones WHERE provision_ref='{CASO}')<>2
 THEN RAISE EXCEPTION 'historia o puntero de concurrencia divergente'; END IF;
END $verificar$;
COMMIT;
\\echo CONTEXTO-CAS-CONCURRENCIA-OK
"""


def bloqueado_por(nombre, pid_a, pid_b):
    consulta = ("SELECT EXISTS(SELECT 1 FROM pg_stat_activity b WHERE b.pid=" + str(pid_b)
                + " AND b.wait_event_type='Lock' AND " + str(pid_a)
                + "=ANY(pg_blocking_pids(b.pid)))")
    salida = psql(nombre, "-c", consulta, timeout=3).stdout.strip()
    return salida == b"t"


def esperar_bloqueo(nombre, pid_a, pid_b, sesion_b, plazo=7):
    limite = time.monotonic() + plazo
    while time.monotonic() < limite:
        if sesion_b.proceso.poll() is not None:
            raise ValueError("La segunda publicación terminó sin esperar al publicador")
        if bloqueado_por(nombre, pid_a, pid_b):
            return
        time.sleep(0.1)
    raise ValueError("No se observó el bloqueo real de la segunda publicación")


def ensayar(nombre, etiqueta, fixture):
    comprobar_contenedor(nombre, etiqueta)
    if not isinstance(fixture, bytes) or b"CTX15-PRUEBA-OK" not in fixture:
        raise ValueError("Falta el fixture versionado de contexto externo")
    preparado = psql(nombre, "-v", "ctx15_persistir_para_concurrencia=1", entrada=fixture, timeout=40)
    if b"CTX15-PRUEBA-OK" not in preparado.stdout:
        raise ValueError("La preparación de contexto externo no terminó")
    a = b = None
    try:
        a = Sesion(nombre)
        b = Sesion(nombre)
        a.enviar(INICIO.format(lado="a"))
        pre_a = a.esperar(PREIMAGEN)
        b.enviar(INICIO.format(lado="b"))
        pre_b = b.esperar(PREIMAGEN)
        if pre_a.group(3) != pre_b.group(3) or pre_a.group(1) == pre_b.group(1):
            raise ValueError("Las sesiones no parten de la misma preimagen")
        pid_a, pid_b = int(pre_a.group(1)), int(pre_b.group(1))
        a.enviar(PUBLICAR + "\\warn A_PUBLICADO\n")
        a.esperar(re.compile(rb"A_PUBLICADO\n"))
        b.enviar("\\set ON_ERROR_STOP off\n" + PUBLICAR + "\\warn B_SQLSTATE :SQLSTATE\nROLLBACK;\n")
        esperar_bloqueo(nombre, pid_a, pid_b, b)
        a.enviar("COMMIT;\n\\warn A_CONFIRMADO\n")
        a.esperar(re.compile(rb"A_CONFIRMADO\n"))
        if b.esperar(ESTADO_B).group(1) != b"40001":
            raise ValueError("La segunda publicación no terminó con SQLSTATE 40001")
        a.proceso.stdin.close()
        b.proceso.stdin.close()
        if a.proceso.wait(timeout=3) != 0 or b.proceso.wait(timeout=3) != 0:
            raise ValueError("Una sesión SQL terminó con error")
        comprobacion = psql(nombre, entrada=VERIFICAR.encode(), timeout=15)
        if b"CONTEXTO-CAS-CONCURRENCIA-OK" not in comprobacion.stdout:
            raise ValueError("El puntero y la historia no pasaron la comprobación")
    finally:
        for sesion in (a, b):
            if sesion is not None:
                sesion.terminar()
