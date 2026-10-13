#!/usr/bin/env python3
"""Prepara Alegaciones usando los publicadores de motivos y material existentes."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import tempfile


def parar(clave, esperado, actual):
    raise ValueError(f"{clave} esperado={esperado} actual={actual}")


def sin_duplicadas(pares):
    dato = {}
    for clave, valor in pares:
        if clave in dato:
            parar("json_clave", "unica", "duplicada")
        dato[clave] = valor
    return dato


def leer_json(ruta, privado=False):
    modo = ruta.lstat()
    if not stat.S_ISREG(modo.st_mode) or modo.st_size > 262144:
        parar("material_json", "regular_acotado", "invalido")
    if privado and modo.st_mode & 0o077:
        parar("material_permisos", "0600", oct(stat.S_IMODE(modo.st_mode)))
    return json.loads(ruta.read_text(), object_pairs_hook=sin_duplicadas)


def sql_texto(valor):
    return "'" + str(valor).replace("'", "''") + "'"


def psql(sql):
    resultado = subprocess.run(
        ["psql", "-X", "-w", "-qAt", "-v", "ON_ERROR_STOP=1", "-v", "VERBOSITY=sqlstate"],
        input=sql, text=True, capture_output=True, check=False, shell=False,
    )
    if resultado.returncode:
        # No publicar parámetros de conexión ni el contenido de errores SQL.
        estado = re.search(r"ERROR:\s+([0-9A-Z]{5})", resultado.stderr)
        causa = estado.group(1) if estado else "conexion_o_cliente"
        parar("publicador_v2", "codigo_0", f"codigo_{resultado.returncode}_causa_{causa}")
    return resultado.stdout.strip()


def preparar(args):
    aqui = Path(__file__).resolve().parent
    interno = Path(os.environ["VEC_DEVELOPMENT_MATERIAL_DIR"])
    externo = Path(args.material_externo).resolve(strict=True)
    servidor = Path(args.servidor).resolve(strict=True)
    if not interno.is_absolute() or not externo.is_dir() or not os.access(servidor, os.X_OK):
        parar("rutas_material", "absolutas_y_servidor_ejecutable", "invalido")
    interno = interno.resolve(strict=True)
    if interno == externo or interno in externo.parents or externo in interno.parents:
        parar("material_externo", "separado_del_interno", "solapado")
    if not os.environ.get("PGSERVICE") or os.environ.get("PGPASSWORD"):
        parar("conexion", "PGSERVICE_privado_sin_PGPASSWORD", "invalida")
    catalogo = leer_json(aqui / "motivos_alegaciones_bolsa_v1.json")
    motivo = leer_json(aqui / "bolsa-alegaciones.ejemplo.json")
    referencia = motivo["motivo_presentacion"]
    canon = {k: catalogo[k] for k in ("catalogo_id", "catalogo_version", "entradas")}
    huella = hashlib.sha256(json.dumps(canon, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()
    if huella != referencia["catalogo_huella_sha256"] or huella != catalogo["catalogo_huella_publicada_sha256"]:
        parar("catalogo_huella", referencia["catalogo_huella_sha256"], huella)
    if (catalogo["catalogo_id"] != referencia["catalogo_id"] or
            catalogo["catalogo_version"] != referencia["catalogo_version"] or
            len(catalogo["entradas"]) != 1 or
            catalogo["entradas"][0]["clave"] != referencia["entrada_clave"]):
        parar("catalogo_referencia", "coincidente", "divergente")
    destino = externo / "identidad/bolsa-alegaciones.json"
    if not destino.parent.is_dir() or destino.parent.is_symlink() or not os.access(destino.parent, os.W_OK):
        parar("directorio_identidad", "existente_y_escribible", "invalido")
    if destino.exists() or destino.is_symlink():
        if leer_json(destino, privado=True) != motivo:
            parar("motivo_privado", "identico", "divergente")
    inventario = leer_json(externo / "externo/v3/inventario.json", privado=True)
    claves = inventario["consumidores"].get("inscripcion_bolsa", [])
    audiencias = [c["audiencia"] for c in claves]
    obligatoria = "vec_bolsa_llamamientos.inscripcion.presentar.v1"
    adicional = "vec_bolsa_llamamientos.alegacion.presentar.v1"
    if audiencias not in ([obligatoria], [obligatoria, adicional]):
        parar("audiencias_inscripcion", "inscripcion_o_inscripcion_y_alegacion", "divergentes")
    comprobacion = subprocess.run(
        [str(servidor), "comprobar-separacion-portales", "--material-interno", str(interno),
         "--material-externo", str(externo)], check=False, capture_output=True, shell=False,
    )
    if comprobacion.returncode:
        parar("separacion_portales", "codigo_0", f"codigo_{comprobacion.returncode}")
    cat, h = sql_texto(catalogo["catalogo_id"]), sql_texto(huella)
    version = int(catalogo["catalogo_version"])
    entradas = sql_texto(json.dumps(catalogo["entradas"], separators=(",", ":")))
    desde = sql_texto(catalogo["entradas"][0]["vigente_desde"])
    guarda = f"""
DO $pre$ DECLARE controles integer; BEGIN
 IF pg_catalog.to_regprocedure('vec_autorizacion.publicar_motivos_autorizacion_v2(text,bigint,text,text,integer,text,timestamp with time zone,jsonb)') IS NULL
 THEN RAISE EXCEPTION 'publicador_v2 esperado=presente actual=ausente'; END IF;
 SELECT count(*) INTO controles FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id=true;
 IF controles<>1 THEN RAISE EXCEPTION 'checkpoint_motivos esperado=1 actual=%',controles; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_catalogo_publicado
   WHERE catalogo_id={cat} AND catalogo_version={version}) THEN
  IF (SELECT catalogo_huella_publicada_sha256 FROM vec_autorizacion.motivo_v2_catalogo_publicado
    WHERE catalogo_id={cat} AND catalogo_version={version}) IS DISTINCT FROM {h}
   OR (SELECT jsonb_agg(jsonb_build_object('clave',entrada_clave,
    'vigente_desde',to_char(vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'),
    'vigente_hasta',CASE WHEN vigente_hasta IS NULL THEN NULL ELSE
       to_char(vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"') END) ORDER BY entrada_clave)
    FROM vec_autorizacion.motivo_v2_entrada WHERE catalogo_id={cat} AND catalogo_version={version}) IS DISTINCT FROM {entradas}::jsonb
   OR EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada WHERE catalogo_id={cat} AND catalogo_version={version})
  THEN RAISE EXCEPTION 'catalogo_alegaciones esperado=identico_vigente actual=divergente'; END IF;
 END IF;
END $pre$;
"""
    # Toda discrepancia de configuración o catálogo se rechaza antes de efectos.
    psql("BEGIN; SET LOCAL search_path=pg_catalog; SET LOCAL statement_timeout='30s'; " + guarda + " ROLLBACK;")
    if not args.aplicar:
        print("preparacion_alegaciones=validada aplicar=false")
        return
    subprocess.run(
        [str(servidor), "preparar-portal-externo", "--material-externo", str(externo),
         "--consumidores", "inscripcion_bolsa"], check=True, shell=False,
    )
    actualizado = leer_json(externo / "externo/v3/inventario.json", privado=True)
    audiencias_finales = [c["audiencia"] for c in actualizado["consumidores"].get("inscripcion_bolsa", [])]
    if audiencias_finales != [obligatoria, adicional]:
        parar("audiencias_preparadas", "inscripcion_y_alegacion", f"cantidad_{len(audiencias_finales)}")
    publicacion = f"""
DO $publicar$
DECLARE secuencia bigint; evento_huella text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_catalogo_publicado
   WHERE catalogo_id={cat} AND catalogo_version={version}) THEN
  SELECT ultima_secuencia+1 INTO STRICT secuencia
   FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id=true FOR UPDATE;
  evento_huella:=encode(sha256(convert_to({h}||':'||secuencia::text,'UTF8')),'hex');
  IF vec_autorizacion.publicar_motivos_autorizacion_v2('evento_'||substr(evento_huella,1,32),
    secuencia,evento_huella,{cat},{version},{h},{desde}::timestamptz,{entradas}::jsonb) IS NOT TRUE
  THEN RAISE EXCEPTION 'publicacion_v2 esperado=true actual=false'; END IF;
 END IF;
END $publicar$;
"""
    psql("BEGIN; SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; "
         "SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='30s'; "
         "SELECT pg_advisory_xact_lock(hashtextextended('vec:alegaciones:motivos:v1',0)); "
         + guarda + publicacion + guarda + " COMMIT;")
    if not destino.exists():
        fd, temporal = tempfile.mkstemp(prefix=".alegaciones-", dir=destino.parent)
        try:
            os.fchmod(fd, 0o600)
            with os.fdopen(fd, "w") as salida:
                json.dump(motivo, salida, ensure_ascii=False)
                salida.write("\n")
                salida.flush()
                os.fsync(salida.fileno())
            os.replace(temporal, destino)
        finally:
            if os.path.exists(temporal):
                os.unlink(temporal)
    print("preparacion_alegaciones=correcta consumidores=inscripcion_bolsa")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--material-externo", required=True)
    parser.add_argument("--servidor", required=True)
    parser.add_argument("--aplicar", action="store_true")
    try:
        preparar(parser.parse_args())
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        if isinstance(error, ValueError):
            parser.exit(1, str(error) + "\n")
        parser.exit(1, "preparacion_alegaciones esperado=correcta actual=" + type(error).__name__ + "\n")
