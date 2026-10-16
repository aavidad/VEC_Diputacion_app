"""Alta ADMIN sintética en una copia nueva, mediante el kit privado vigente.

Los planes, contraseñas y acuses permanecen en la raíz privada de la copia.
Una interrupción después de aplicar exige recuperar el mismo plan con el kit.
"""

from __future__ import annotations

import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys


CONTRATO = "contrato-copia-local-admin-v1.json"
SCRIPTS = (
    "guardar_preimagen_admin.py", "preparar_fixture_hz37.py", "preparar_copia.py",
    "arranque_principal_h10.py", "registrar_encargo_local.py", "preparar_gobierno.py",
    "preparar_runtime.py", "preparar_vec_admin.py",
)
BINARIOS = ("bin/h-root-seed.test", "bin/vec-gobierno-usuarios-admin")
GOCACHE = "/dev/shm/vec/codexl-go-build"
FASES = ("fuentes", "unidad", "bootstrap")
GRUPOS = (
    "vec_admin_fuentes_iniciales_ejecutor", "vec_personal_unidad_inicial_ejecutor",
    "vec_admin_bootstrap_central_v3_ejecutor",
    "vec_autorizacion_atestada_v3_lector_gobierno",
    "vec_gobierno_capacidades_admin_operador", "vec_auditoria_encadenador",
)


def _privado(ruta: Path, directorio: bool = False) -> None:
    dato = ruta.lstat()
    if (ruta.resolve(strict=True) != ruta or dato.st_uid != os.getuid()
            or dato.st_mode & 0o077 or (not directorio and dato.st_nlink != 1)
            or not (stat.S_ISDIR(dato.st_mode) if directorio else stat.S_ISREG(dato.st_mode))):
        raise ValueError("Ruta privada ausente, enlazada o con permisos inseguros")


def _json_privado(ruta: Path) -> dict:
    _privado(ruta)
    dato = json.loads(ruta.read_text())
    if not isinstance(dato, dict):
        raise ValueError("Documento privado inválido")
    return dato


def _huella(ruta: Path) -> str:
    with ruta.open("rb") as entrada:
        return hashlib.file_digest(entrada, "sha256").hexdigest()


def _comprobar_kit(copia, kit: Path, estado: dict) -> dict:
    raiz = Path(copia.raiz)
    if (not kit.is_absolute() or kit.is_relative_to(raiz)
            or kit.is_relative_to(Path(__file__).resolve().parents[2])):
        raise ValueError("El kit ADMIN debe ser absoluto y estar fuera de Git y de la copia")
    _privado(kit, directorio=True)
    contrato = _json_privado(kit / CONTRATO)
    if (contrato.get("version") != 1 or contrato.get("vigencia_dias") != 89
            or contrato.get("certificados_dias") != 90
            or contrato.get("conjunto_capacidades") != 5
            or contrato.get("gocache") != GOCACHE):
        raise ValueError("Kit ADMIN antiguo: preparar gobierno de 89 días, certificados de 90 y conjunto 5")
    _privado(Path(GOCACHE), directorio=True)
    for clave, nombres in (("scripts_sha256", SCRIPTS), ("bin_sha256", BINARIOS)):
        huellas = contrato.get(clave)
        if not isinstance(huellas, dict) or set(huellas) != set(nombres):
            raise ValueError("Contrato de archivos del kit incompleto")
        for nombre in nombres:
            ruta = kit / nombre
            _privado(ruta)
            if not re.fullmatch(r"[0-9a-f]{64}", huellas[nombre]) or _huella(ruta) != huellas[nombre]:
                raise ValueError("Archivo del kit ADMIN divergente")
    if not re.fullmatch(r"[a-z0-9_]{3,80}", str(contrato.get("catalogo_entrada_unidad", ""))):
        raise ValueError("Falta la entrada de unidad de la copia")
    if (not isinstance(contrato.get("puerto_admin"), int)
            or not 1024 <= contrato["puerto_admin"] <= 65535
            or contrato["puerto_admin"] in (estado["puerto_pg"], estado["puerto_app"])):
        raise ValueError("Puerto ADMIN inválido")
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._:-]{7,127}", str(contrato.get("encargo_ref", ""))):
        raise ValueError("Falta la referencia del encargo local autorizado")
    # Estas comprobaciones detectan las dos incompatibilidades concretas del kit H.
    if GOCACHE not in (kit / "preparar_copia.py").read_text():
        raise ValueError("El generador del kit sigue usando caché Go antigua")
    if "gocache_fuera_de_disco_privado" in (kit / "arranque_principal_h10.py").read_text():
        raise ValueError("El ejecutor del kit aún prohíbe la caché en memoria")
    gobierno = (kit / "preparar_gobierno.py").read_text()
    if '"conjunto_capacidades": 5' not in gobierno and "'conjunto_capacidades': 5" not in gobierno:
        raise ValueError("El kit no prepara el conjunto 5")
    return contrato


def _ejecutar(kit: Path, nombre: str, *args: str, entorno: dict | None = None) -> None:
    comando = [sys.executable, str(kit / nombre), *map(str, args)]
    resultado = subprocess.run(comando, text=True, capture_output=True, timeout=180,
                              env=entorno, check=False)
    if resultado.returncode:
        _guardar_error(kit / "errores-admin.log", nombre, resultado)
        # El kit puede mostrar rutas y DSN en errores inesperados. No propagarlos.
        raise ValueError(f"Etapa ADMIN {nombre} falló ({resultado.returncode}); consulte la carpeta privada")


def _guardar_error(ruta: Path, fase: str, resultado) -> None:
    """Guarda la salida íntegra solo en el kit privado, sin llevarla al terminal."""
    if ruta.exists() or ruta.is_symlink():
        _privado(ruta)
    descriptor = os.open(ruta, os.O_WRONLY | os.O_APPEND | os.O_CREAT | os.O_NOFOLLOW, 0o600)
    os.fchmod(descriptor, 0o600)
    with os.fdopen(descriptor, "a", encoding="utf-8") as salida:
        salida.write(f"\n[{dt.datetime.now(dt.timezone.utc).isoformat()}] {fase}: {resultado.returncode}\n")
        salida.write(resultado.stdout or "")
        salida.write(resultado.stderr or "")


def _sql(copia, ruta: Path, log: str) -> None:
    _privado(ruta)
    copia.sql_bytes(ruta.read_bytes(), log)


def _grupos_disponibles(copia) -> None:
    nombres = ",".join("'" + nombre + "'" for nombre in GRUPOS)
    consulta = ("DO $v$ BEGIN IF (SELECT count(*) FROM pg_catalog.pg_roles "
                f"WHERE rolname IN ({nombres}) AND NOT rolcanlogin) <> {len(GRUPOS)} "
                "THEN RAISE EXCEPTION 'grupos ADMIN ausentes o LOGIN'; END IF; END $v$;")
    copia.sql_bytes(consulta.encode(), "admin-preflight.log")


def _fase_h10(copia, kit: Path, run: Path, fase: str, encargo: str) -> None:
    config = run / "config.json"
    _ejecutar(kit, "arranque_principal_h10.py", "--config", config,
             "--etapa", "preparar_" + fase)
    _ejecutar(kit, "registrar_encargo_local.py", "--run", run,
             "--fase", fase, "--encargo-ref", encargo)
    _ejecutar(kit, "arranque_principal_h10.py", "--config", config,
             "--etapa", "configurar_" + fase)
    _sql(copia, run / "dba" / (fase + "-password.sql"), "admin-h10.log")
    try:
        _ejecutar(kit, "arranque_principal_h10.py", "--config", config,
                 "--etapa", "aplicar_" + fase)
    except ValueError as error:
        raise ValueError(f"{error}; si COMMIT es incierto use recuperar_{fase} con el mismo plan") from None
    acuse = _json_privado(run / fase / "acuse-primero.json")
    if (acuse.get("estado") != "permitido" or not isinstance(acuse.get("recibo"), dict)
            or not acuse["recibo"].get("auditoria_ref")):
        raise ValueError("Falta acuse ADMIN permitido y auditado")


def _semilla_y_gobierno(copia, kit: Path, run: Path, sufijo: str, contrato: dict, ca: Path) -> str:
    material = Path(copia.material)
    raiz = run / "raiz"
    raiz.mkdir(mode=0o700)
    semilla = raiz / "semilla-root-dev.bin"
    entorno = {"PATH": "/usr/bin:/bin", "LANG": "C", "HOME": str(run),
               "VEC_H_MATERIAL_DIR": str(material),
               "VEC_H_HMAC_CONFIG": str(material / "idempotencia/configuracion.json"),
               "VEC_H_ROOT_OUT": str(semilla)}
    resultado = subprocess.run([str(kit / "bin/h-root-seed.test"), "-test.run",
                             "^TestDerivarSemillaRaizHPrivada$", "-test.v"],
                             env=entorno, text=True, capture_output=True, timeout=90)
    if resultado.returncode:
        _guardar_error(kit / "errores-admin.log", "derivar-raiz", resultado)
        raise ValueError("No se pudo derivar la raíz de esta copia")
    _privado(semilla)
    huellas = re.findall(r"spki_h=([0-9a-f]{64})", resultado.stdout)
    if len(huellas) != 1:
        raise ValueError("No se pudo cotejar la raíz derivada")
    # El puntero publicado de esta base debe referir exactamente esa raíz.
    cotejo = ("DO $v$ BEGIN IF (SELECT count(*) FROM "
             "vec_autorizacion_atestada_v3.puntero_configuracion_actual p "
             "JOIN vec_autorizacion_atestada_v3.configuracion_raiz c "
             "ON c.configuracion_revision=p.configuracion_revision "
             "JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r "
             "ON (r.clave_id,r.version)=(c.raiz_clave_id,c.raiz_version) "
             "WHERE p.orden=(SELECT max(orden) FROM "
             "vec_autorizacion_atestada_v3.puntero_configuracion_actual) "
             f"AND r.huella_spki_sha256='{huellas[0]}' "
             "AND r.valida_desde<=clock_timestamp() AND r.valida_hasta>clock_timestamp())<>1 "
             "THEN RAISE EXCEPTION 'raíz de copia divergente'; END IF; END $v$;")
    copia.sql_bytes(cotejo.encode(), "admin-raiz.log")
    dia = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d") + hashlib.sha256(sufijo.encode()).hexdigest()[:6]
    comunes = ("--run", run, "--instancia", sufijo, "--dia", dia,
               "--contenedor", copia.nombre, "--semilla-raiz", semilla,
               "--ca-pg", ca, "--puerto-pg", copia.leer()["puerto_pg"])
    _ejecutar(kit, "preparar_gobierno.py", "--fase", "entrada", *comunes)
    config = run / ("config-gobierno-" + dia + ".json")
    datos = _json_privado(config)
    if datos.get("horas_validez_claves") != 2136 or datos.get("conjunto_capacidades") != 5:
        raise ValueError("El gobierno no ofrece 89 días y conjunto 5")
    _sql(copia, run / "dba" / ("roles-gobierno-" + dia + ".sql"), "admin-gobierno.log")
    cli = kit / "bin/vec-gobierno-usuarios-admin"
    textos = kit / "source/web/static/textos/es/admin-gobierno-usuarios.json"
    for fase in ("preparar",):
        _cli_gobierno(cli, fase, config, textos)
    _ejecutar(kit, "preparar_gobierno.py", "--fase", "configurar", *comunes)
    _sql(copia, run / "dba" / ("config-gobierno-" + dia + ".sql"), "admin-gobierno.log")
    _cli_gobierno(cli, "aplicar", config, textos, "-acuse", "acuse-aplicar.json")
    acuse = _json_privado(run / "gobierno" / dia / "acuse-aplicar.json")
    if (acuse.get("estado") != "permitido" or not isinstance(acuse.get("recibo"), dict)
            or not acuse["recibo"].get("auditoria_ref")):
        raise ValueError("Gobierno ADMIN sin recibo confirmado")
    _cli_gobierno(cli, "verificar", config, textos)
    return dia


def _cli_gobierno(cli: Path, fase: str, config: Path, textos: Path, *extra: str) -> None:
    resultado = subprocess.run([str(cli), "-fase", fase, "-config", str(config),
                              "-textos", str(textos), "-timeout", "30s", *extra],
                              text=True, capture_output=True, timeout=90,
                              env={"PATH": "/usr/bin:/bin", "LANG": "C",
                                   "VEC_EXECUTION_PROFILE": "desarrollo",
                                   "VEC_AUTH_MODE": "desarrollo",
                                   "VEC_DEVELOPMENT_GUARD": "ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO"})
    if resultado.returncode:
        _guardar_error(config.parent / "errores-admin.log", "gobierno-" + fase, resultado)
        raise ValueError(f"Gobierno ADMIN {fase} falló; consulte su salida privada")


def preparar(copia, kit_path) -> dict[str, str]:
    """Ejecuta el alta en una instancia nueva. Devuelve solo rutas y estado público."""
    raiz, material, kit = Path(copia.raiz), Path(copia.material), Path(kit_path)
    estado = copia.leer()
    nombre = copia.nombre
    if (not re.fullmatch(r"vec-copia-[a-z]-[a-z0-9][a-z0-9-]{0,40}", nombre)
            or estado.get("nombre") != nombre or estado.get("raiz") != str(raiz)
            or not estado.get("revision") or "admin_preparada" in estado
            or not isinstance(estado.get("puerto_pg"), int)
            or not 1024 <= estado["puerto_pg"] <= 65535
            or not isinstance(estado.get("puerto_app"), int)):
        raise ValueError("La copia ADMIN debe ser propia y no estar ya preparada para ADMIN")
    _privado(raiz, directorio=True)
    _privado(material, directorio=True)
    ca = material / "ca/ca.crt"
    _privado(ca)
    contrato = _comprobar_kit(copia, kit, estado)
    origen = Path(estado["origen"])
    volcado = origen / "principal-hz37-20261014.dump"
    if not origen.is_absolute() or origen.is_symlink() or not volcado.is_file() or volcado.is_symlink():
        raise ValueError("Falta el respaldo HZ37 declarado en la copia")
    admin = raiz / "admin"
    if admin.exists() or admin.is_symlink():
        raise ValueError("ADMIN ya iniciado en esta copia; no se repite la preparación")
    _grupos_disponibles(copia)
    sufijo = (nombre.removeprefix("vec-copia-").replace("-", "_")[:10]
              + hashlib.sha256(nombre.encode()).hexdigest()[:6])
    admin.mkdir(mode=0o700)
    preimagen = admin / "preimagen-admin.dump"
    _ejecutar(kit, "guardar_preimagen_admin.py", "--contenedor", nombre,
             "--salida", preimagen)
    _ejecutar(kit, "preparar_fixture_hz37.py", "--contenedor", nombre,
             "--respaldo-hz37", volcado, "--preimagen-admin", preimagen,
             "--salida", admin)
    _sql(copia, admin / "fixture-admin-hz37.sql", "admin-fixture.log")
    run = admin / "h10"
    _ejecutar(kit, "preparar_copia.py", "--instancia", sufijo,
             "--contenedor", nombre, "--puerto-pg", estado["puerto_pg"],
             "--backup-admin", preimagen, "--material", material,
             "--ca-pg", ca, "--catalogo-entrada-unidad",
             contrato["catalogo_entrada_unidad"], "--salida", run)
    config = _json_privado(run / "config.json")
    psql = config.get("psql_comando")
    if (Path(config.get("privado", "")) != run
            or not isinstance(psql, list) or len(psql) < 6
            or psql[:5] != ["/usr/bin/docker", "exec", "-i", "-u", "postgres"]
            or psql[5] != nombre
            or config.get("kit", {}).get("gocache_cache") != contrato["gocache"]
            or config.get("kit", {}).get("source_snapshot") != str(kit / "source")):
        raise ValueError("El kit preparó una configuración para otra copia o caché")
    _ejecutar(kit, "arranque_principal_h10.py", "--config", run / "config.json",
             "--etapa", "comprobar")
    for fase in FASES:
        _fase_h10(copia, kit, run, fase, contrato["encargo_ref"])
    _ejecutar(kit, "arranque_principal_h10.py", "--config", run / "config.json",
             "--etapa", "verificar")
    dia = _semilla_y_gobierno(copia, kit, run, sufijo, contrato, ca)
    _ejecutar(kit, "preparar_runtime.py", "--run", run, "--instancia", sufijo,
             "--dia-gobierno", dia, "--semilla-raiz", run / "raiz/semilla-root-dev.bin",
             "--ca-pg", ca, "--puerto-pg", estado["puerto_pg"],
             "--puerto-admin", contrato["puerto_admin"])
    for fase in ("tls", "activos", "pools"):
        _ejecutar(kit, "preparar_vec_admin.py", "--entrada", run / "entrada-vec-admin.json",
                 "--fase", fase, "--dia", dia)
    _sql(copia, run / "vec-admin/dba-logins.sql", "admin-runtime.log")
    for fase in ("claves", "base", "overlay", "runtime", "entorno"):
        _ejecutar(kit, "preparar_vec_admin.py", "--entrada", run / "entrada-vec-admin.json",
                 "--fase", fase, "--dia", dia)
    return {"VEC_ADMIN_ENTORNO_FILE": str(run / "vec-admin/config" / f"entorno-{dia}.env"),
            "VEC_ADMIN_RUN": str(run), "VEC_ADMIN_ESTADO": "preparado_con_acuses"}
