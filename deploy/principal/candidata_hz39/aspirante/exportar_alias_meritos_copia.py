#!/usr/bin/env python3
"""Exporta alias ID7 con la clave real del portal externo de la copia."""

import argparse
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import tempfile


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def privado(ruta, directorio=False):
    dato = ruta.lstat()
    exigir(dato.st_uid == os.getuid() and not ruta.is_symlink() and not dato.st_mode & 0o077,
           "ruta_no_privada")
    exigir(stat.S_ISDIR(dato.st_mode) if directorio else stat.S_ISREG(dato.st_mode),
           "tipo_de_ruta_invalido")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ensayo", type=Path, required=True)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    exigir(args.ensayo.is_absolute() and
           Path(__file__).resolve().parents[4] not in args.ensayo.parents,
           "rutas_no_admitidas")
    privado(args.ensayo, True)
    privado(args.ensayo / "externo", True)
    servidor = args.ensayo / "scratch/vec-server"
    exigir(servidor.is_file() and not servidor.is_symlink() and
           servidor.stat().st_uid == os.getuid() and os.access(servidor, os.X_OK),
           "servidor_ausente")
    actos = args.ensayo / "incorporacion/actos"
    privado(actos, True)
    usuarios = json.loads((actos / "usuarios-preferencias-externa.preparado.json").read_text())
    meritos = json.loads((actos / "meritos-cuentas.preparado.json").read_text())
    configuraciones = {"usuarios": usuarios}
    config_meritos = json.loads(json.dumps(usuarios))
    config_meritos["cuentas"][0]["cuenta_ref"] = meritos["cuentas"][0]["cuenta_ref"]
    config_meritos["cuentas"][0]["perfil_ref"] = meritos["cuentas"][0]["perfil_ref"]
    configuraciones["meritos"] = config_meritos
    for etiqueta, configuracion in configuraciones.items():
        destino = actos / ("alias-" + etiqueta + "-externo.json")
        if destino.exists():
            privado(destino)
            dato = json.loads(destino.read_text())
        else:
            exigir(args.aplicar, "alias_pendiente")
            with tempfile.TemporaryDirectory(prefix=".alias-externo-", dir=args.ensayo) as temporal:
                copia = Path(temporal) / "material"
                shutil.copytree(args.ensayo / "externo", copia)
                copia.chmod(0o700)
                for ruta in copia.rglob("*"):
                    ruta.chmod(0o700 if ruta.is_dir() else 0o600)
                archivo = copia / "identidad/usuarios-preferencias-externa.json"
                archivo.write_text(json.dumps(configuracion, ensure_ascii=False, separators=(",", ":")) + "\n")
                archivo.chmod(0o600)
                entorno = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"),
                           "VEC_PORTAL_PROCESO": "externo", "VEC_EXECUTION_PROFILE": "desarrollo",
                           "VEC_AUTH_MODE": "desarrollo",
                           "VEC_DEVELOPMENT_GUARD": "ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO",
                           "VEC_DEVELOPMENT_MATERIAL_DIR": str(copia)}
                # Ruta fija del ensayo, ejecutable propio comprobado y argumentos cerrados; sin shell.
                resultado = subprocess.run([str(servidor), "exportar-seudonimos-portal-externo"],  # nosemgrep: python.lang.security.audit.dangerous-subprocess-use-tainted-env-args.dangerous-subprocess-use-tainted-env-args
                                           env=entorno, capture_output=True, timeout=25, check=False)
                exigir(resultado.returncode == 0, "exportacion_alias_rechazada")
                dato = json.loads(resultado.stdout)
                with destino.open("xb") as salida:
                    salida.write(resultado.stdout)
                destino.chmod(0o600)
        cuenta = configuracion["cuentas"][0]["cuenta_ref"]
        exigir(dato.get("version") == 1 and
               sum(c.get("cuenta_ref") == cuenta for c in dato.get("cuentas", [])) == 1,
               "alias_cuenta_no_unico")
    print("alias_usuarios_meritos=preparados")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError,
            subprocess.TimeoutExpired) as error:
        raise SystemExit("alias_usuarios_meritos=error clave=" + str(error).split(":", 1)[0]) from None
