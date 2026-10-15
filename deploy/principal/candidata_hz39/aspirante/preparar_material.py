#!/usr/bin/env python3
"""Copia el material nominal ya aprobado a un clon local de HZ39."""

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import stat
import sys
import tempfile
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

sys.path.insert(0, str(Path(__file__).resolve().parents[4]))
from scripts.recorridos.clon_material import cert_hash, reference


ORIGEN_INTERNO = "material/desarrollo-fiscalizacion-20260904"
ORIGEN_INCORPORACION = "incorporacion-servidor-20260910/config"
ORIGEN_EXTERNO = "incorporacion-servidor-20260910/config/portal-externo"
PERFIL_TECNICO = "prf_5d140bb82db5f22e5ea49d54c9eb75ba"
DSN_RRHH = (
    "dsn_registro_identidad", "dsn_revalidacion_identidad", "dsn_contexto",
    "dsn_fuente_autorizacion", "dsn_registro_autorizacion", "dsn_motivos",
)
CONEXIONES_BASE = (
    "VEC_CT_DATABASE_URL", "VEC_CT_GOBIERNO_DATABASE_URL",
    "VEC_CT_CONFIRMADOR_DATABASE_URL", "VEC_CT_LECTOR_RESULTADO_DATABASE_URL",
    "VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL", "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL",
    "VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL", "VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL",
    "VEC_CALENDARIOS_DATABASE_URL", "VEC_CT_CONSULTAS_RRHH_DATABASE_URL",
    "VEC_CT_MOTIVOS_RRHH_DATABASE_URL", "VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL",
    "VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL", "VEC_CT_CONTEXTO_ACTOR_DATABASE_URL",
    "VEC_BOLSA_RELEVO_NO_INCORPORACION_DATABASE_URL",
    "VEC_AUDITORIA_SELLADO_DATABASE_URL", "VEC_BOLSA_CESE_CT_DATABASE_URL",
)
CONEXIONES_EXTERNAS = {
    "VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL": "preflight.dsn",
    "VEC_EXTERNO_BOLSA_DATABASE_URL": "bolsa.dsn",
    "VEC_EXTERNO_BOLSA_FRONTERA_DATABASE_URL": "bolsa_frontera.dsn",
    "VEC_EXTERNO_BOLSA_INSCRIPCIONES_LECTOR_DATABASE_URL": "inscripciones_lector.dsn",
    "VEC_EXTERNO_CALENDARIOS_DATABASE_URL": "calendarios.dsn",
    "VEC_EXTERNO_AUTORIZACION_FUENTE_DATABASE_URL": "autorizacion_fuente.dsn",
    "VEC_EXTERNO_AUTORIZACION_REGISTRO_DATABASE_URL": "autorizacion_registro.dsn",
    "VEC_EXTERNO_AUTORIZACION_MOTIVOS_DATABASE_URL": "autorizacion_motivos.dsn",
    "VEC_EXTERNO_IDENTIDAD_REGISTRO_DATABASE_URL": "identidad_registro.dsn",
    "VEC_EXTERNO_IDENTIDAD_REVALIDACION_DATABASE_URL": "identidad_revalidacion.dsn",
    "VEC_EXTERNO_CONTEXTO_DATABASE_URL": "contexto.dsn",
}
CONEXIONES_MERITOS = {
    "VEC_EXTERNO_MERITOS_OPERACION_DATABASE_URL": "operacion.dsn",
    "VEC_EXTERNO_MERITOS_CARTERA_DATABASE_URL": "cartera.dsn",
    "VEC_EXTERNO_MERITOS_FRONTERA_DATABASE_URL": "frontera.dsn",
    "VEC_EXTERNO_MERITOS_FUENTE_AUTORIZACION_DATABASE_URL": "fuente_autorizacion.dsn",
    "VEC_EXTERNO_MERITOS_REGISTRO_AUTORIZACION_DATABASE_URL": "registro_autorizacion.dsn",
    "VEC_EXTERNO_MERITOS_MOTIVOS_AUTORIZACION_DATABASE_URL": "motivos_autorizacion.dsn",
    "VEC_EXTERNO_MERITOS_CONTEXTO_DATABASE_URL": "contexto.dsn",
}


def exigir(condicion, clave):
    if not condicion:
        raise ValueError(clave)


def comprobar_ruta(ruta, directorio):
    dato = ruta.lstat()
    exigir(dato.st_uid == os.getuid(), "propietario_invalido")
    exigir(not ruta.is_symlink(), "enlace_no_admitido")
    exigir(stat.S_ISDIR(dato.st_mode) if directorio else stat.S_ISREG(dato.st_mode),
           "tipo_de_ruta_invalido")


def comprobar_privado(ruta, directorio):
    comprobar_ruta(ruta, directorio)
    exigir(not ruta.stat().st_mode & 0o077, "material_no_privado")


def leer_json(ruta):
    comprobar_ruta(ruta, False)
    def claves_unicas(pares):
        salida = {}
        for clave, valor in pares:
            exigir(clave not in salida, "clave_json_repetida")
            salida[clave] = valor
        return salida
    return json.loads(ruta.read_text(), object_pairs_hook=claves_unicas)


def escribir_json(ruta, valor):
    ruta.write_text(json.dumps(valor, ensure_ascii=False, separators=(",", ":")) + "\n")
    ruta.chmod(0o600)


def normalizar_dsn(dsn):
    dsn = dsn.replace("${vec_local_pg_puerto}", "5432").replace("$vec_local_pg_puerto", "5432")
    dsn = dsn.replace("${vec_local_ca}", "/vec-pg-ca.crt").replace("$vec_local_ca", "/vec-pg-ca.crt")
    exigir("$" not in dsn, "dsn_con_sustitucion_pendiente")
    partes = urlsplit(dsn)
    exigir(partes.scheme == "postgresql" and partes.hostname == "localhost"
           and partes.port == 5432 and partes.path == "/postgres", "dsn_origen_inesperada")
    exigir("@" in partes.netloc, "dsn_sin_login")
    opciones = parse_qsl(partes.query, keep_blank_values=True)
    exigir([k for k, _ in opciones].count("sslmode") == 1
           and [k for k, _ in opciones].count("sslrootcert") == 1,
           "dsn_tls_incompleta")
    exigir(dict(opciones)["sslmode"] == "verify-full", "dsn_tls_no_verificado")
    opciones = [(k, "/vec-pg-ca.crt" if k == "sslrootcert" else v) for k, v in opciones]
    credenciales = partes.netloc.rsplit("@", 1)[0]
    return urlunsplit((partes.scheme, credenciales + "@localhost:5432",
                       partes.path, urlencode(opciones), partes.fragment))


def adaptar_json(valor):
    if isinstance(valor, dict):
        return {k: adaptar_json(v) for k, v in valor.items()}
    if isinstance(valor, list):
        return [adaptar_json(v) for v in valor]
    if isinstance(valor, str) and valor.startswith("postgresql://"):
        return normalizar_dsn(valor)
    return valor


def copiar_arbol(origen, destino, excluir_externo=False):
    comprobar_ruta(origen, True)
    for ruta in origen.rglob("*"):
        comprobar_ruta(ruta, ruta.is_dir())
    with tempfile.TemporaryDirectory(prefix=".vec-hz39-", dir=destino.parent) as temporal:
        etapa = Path(temporal) / "arbol"
        shutil.copytree(origen, etapa, ignore=(lambda _, nombres: {"portal-externo"}
                        if excluir_externo and "portal-externo" in nombres else set()))
        etapa.chmod(0o700)
        for ruta in etapa.rglob("*"):
            if ruta.is_dir():
                ruta.chmod(0o700)
            elif ruta.is_file():
                ruta.chmod(0o600)
                if ruta.suffix == ".dsn":
                    ruta.write_text(normalizar_dsn(ruta.read_text().strip()) + "\n")
                elif ruta.suffix == ".json":
                    dato = leer_json(ruta)
                    nuevo = adaptar_json(dato)
                    if nuevo != dato:
                        escribir_json(ruta, nuevo)
            else:
                raise ValueError("tipo_de_material_invalido")
        etapa.rename(destino)


def preparar_rrhh(interno, aplicar):
    identidad = leer_json(interno / "identidad/identidad.json")
    huella = cert_hash(interno / "mtls/cliente.crt")
    perfil = reference("prf_", identidad["subject"] + "\0" + huella + "\0perfil-fijo-alta-perfil")
    exigir(perfil == PERFIL_TECNICO, "perfil_tecnico_no_coincide")
    ruta = interno / "identidad/inscripcion-bolsa-rrhh.json"
    inscripcion = leer_json(ruta)
    exigir(inscripcion.get("version") == 1 and inscripcion.get("autoridad") == "no_autoritativo",
           "inscripcion_rrhh_invalida")
    cuentas = [c for c in inscripcion["cuentas"]
               if c["sujeto"] == identidad["subject"] and c["certificado_sha256"] == huella]
    exigir(len(cuentas) == 1 and len(inscripcion["cuentas"]) == 1, "cuenta_rrhh_ambigua")
    exigir(all(isinstance(inscripcion.get(k), str) and inscripcion[k].startswith("postgresql://")
               for k in DSN_RRHH), "faltan_seis_dsn_rrhh")
    if cuentas[0]["perfil_ref"] != perfil:
        exigir(aplicar, "perfil_tecnico_pendiente")
        cuentas[0]["perfil_ref"] = perfil
        escribir_json(ruta, inscripcion)
    manifiesto = {"version": 1, "autoridad": "no_autoritativo", "entradas": [{
        "certificate": "mtls/cliente.crt", "identity": "identidad/identidad.json",
        "subject": identidad["subject"], "perfil_ref": perfil,
        "organizacion_ref": "organizacion:desarrollo:dipgra",
        "clase_ambito": "organizacion", "ambito_ref": "organizacion:desarrollo:dipgra",
    }]}
    consultas = interno / "identidad/consultas-rrhh.json"
    if consultas.exists():
        comprobar_privado(consultas, False)
        exigir(leer_json(consultas) == manifiesto, "lector_rrhh_divergente")
    else:
        exigir(aplicar, "lector_rrhh_pendiente")
        escribir_json(consultas, manifiesto)


def comprobar_externo(interno, externo):
    propio = leer_json(interno / "identidad/bolsa-candidato.json")
    separado = leer_json(externo / "identidad/bolsa-candidato.json")
    exigir(all(propio[k] == separado[k] for k in
               ("persona_ref", "candidato_ref", "sujeto", "certificado", "identidad")),
           "identidad_candidata_divergente")
    exigir(all(propio[k] != separado[k] for k in ("perfil_ref", "cuenta_ref")),
           "perfil_o_cuenta_externa_reutilizada")
    exigir(leer_json(externo / "portal-proceso.json") == {"version": 1, "portal": "externo"},
           "portal_externo_invalido")
    exigir(not (externo / "mtls/candidato.key").exists(), "clave_candidata_en_servidor")


def preparar_arranque(respaldo, incorporacion, aplicar):
    fuente = respaldo / "material/arrancar-local.sh"
    comprobar_ruta(fuente, False)
    patron = re.compile(r'^\s*export (VEC_[A-Z_]*DATABASE_URL)="(postgresql://[^"]+)"$')
    conexiones = {}
    for linea in fuente.read_text().splitlines():
        coincidencia = patron.fullmatch(linea)
        if coincidencia:
            nombre, dsn = coincidencia.groups()
            exigir(nombre not in conexiones, "conexion_base_repetida")
            conexiones[nombre] = normalizar_dsn(dsn)
    exigir(set(conexiones) == set(CONEXIONES_BASE), "conexiones_base_incompletas")
    destino_conexiones = incorporacion / "conexiones_referencia.sh"
    contenido = "".join(f'export {nombre}="{conexiones[nombre]}"\n' for nombre in CONEXIONES_BASE)
    if destino_conexiones.exists():
        comprobar_privado(destino_conexiones, False)
        exigir(destino_conexiones.read_text() == contenido, "conexiones_base_divergentes")
    else:
        exigir(aplicar, "conexiones_base_pendientes")
        destino_conexiones.write_text(contenido)
        destino_conexiones.chmod(0o600)

    original = respaldo / "arrancar_app.sh"
    comprobar_ruta(original, False)
    texto = original.read_text()
    exigir(texto.count("/vec-conexiones-referencia.sh") == 2 and
           texto.count('exec /usr/local/bin/vec-server "$@"') == 1,
           "arranque_original_no_reconocido")
    texto = texto.replace("/vec-conexiones-referencia.sh", "/vec-incorporacion/conexiones_referencia.sh")
    adicional = ['unset VEC_PORTAL_PROCESO',
                 'export VEC_BOLSA_INSCRIPCIONES_ENABLED=true',
                 'export VEC_MATERIAL_PORTAL_EXTERNO_DIR=/vec-material-externo',
                 'export VEC_BOLSA_INSCRIPCIONES_RRHH_LECTOR_DATABASE_URL="$(< /vec-incorporacion/inscripcion_rrhh_lector.dsn)"']
    for nombre, fichero in CONEXIONES_EXTERNAS.items():
        adicional.append(f'export {nombre}="$(< /vec-incorporacion/externo-dsn/{fichero})"')
    requisitos_meritos = ['/vec-material-externo/identidad/meritos-cartera-externa.json']
    requisitos_meritos.extend('/vec-incorporacion/externo-dsn/meritos/' + fichero
                              for fichero in CONEXIONES_MERITOS.values())
    adicional.append('if ' + ' && '.join('[ -r ' + ruta + ' ]'
                                        for ruta in requisitos_meritos) + '; then')
    for nombre, fichero in CONEXIONES_MERITOS.items():
        adicional.append(f'  export {nombre}="$(< /vec-incorporacion/externo-dsn/meritos/{fichero})"')
    adicional.append('fi')
    texto = texto.replace('exec /usr/local/bin/vec-server "$@"',
                          "\n".join(adicional) + '\nexec /usr/local/bin/vec-server "$@"')
    destino_arranque = incorporacion / "arrancar_combinado.sh"
    if destino_arranque.exists():
        comprobar_privado(destino_arranque, False)
        exigir(destino_arranque.read_text() == texto, "arranque_combinado_divergente")
    else:
        exigir(aplicar, "arranque_combinado_pendiente")
        destino_arranque.write_text(texto)
        destino_arranque.chmod(0o700)


def preparar_fuentes_admin(incorporacion, aplicar):
    repo = Path(__file__).resolve().parents[4]
    fuentes = {
        "inscripcion_bolsa_actos_v1.json": repo / "deploy/principal/inscripcion_recorrido_20261013/inscripcion_bolsa_actos_v1.json",
        "meritos_cartera_propia_v1.json": repo / "deploy/principal/fuentes/meritos_cartera_propia_v1.json",
    }
    destino = incorporacion / "actos"
    if aplicar:
        destino.mkdir(mode=0o700, exist_ok=True)
    comprobar_privado(destino, True)
    for nombre, fuente in fuentes.items():
        comprobar_ruta(fuente, False)
        dato = fuente.read_bytes()
        archivo = destino / nombre
        if archivo.exists():
            comprobar_privado(archivo, False)
            exigir(archivo.read_bytes() == dato, "fuente_admin_divergente")
        else:
            exigir(aplicar, "fuente_admin_pendiente")
            with archivo.open("xb") as salida:
                salida.write(dato)
            archivo.chmod(0o600)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--respaldo", required=True, type=Path)
    parser.add_argument("--destino", required=True, type=Path)
    parser.add_argument("--aplicar", action="store_true")
    args = parser.parse_args()
    respaldo, destino = args.respaldo, args.destino
    exigir(respaldo.is_absolute() and destino.is_absolute() and respaldo != destino,
           "rutas_absolutas_separadas_requeridas")
    repo = Path(__file__).resolve().parents[4]
    exigir(repo not in destino.parents and destino != repo and
           destino / "material" not in respaldo.parents and
           destino / "incorporacion" not in respaldo.parents and
           destino / "externo" not in respaldo.parents,
           "material_en_ruta_no_admitida")
    comprobar_ruta(respaldo, True)
    comprobar_privado(destino, True)
    fuentes = [(respaldo / ORIGEN_INTERNO, destino / "material"),
               (respaldo / ORIGEN_INCORPORACION, destino / "incorporacion"),
               (respaldo / ORIGEN_EXTERNO, destino / "externo")]
    for origen, final in fuentes:
        comprobar_ruta(origen, True)
        if args.aplicar and not final.exists():
            copiar_arbol(origen, final, excluir_externo=final.name == "incorporacion")
        comprobar_privado(final, True)
    preparar_rrhh(destino / "material", args.aplicar)
    preparar_arranque(respaldo, destino / "incorporacion", args.aplicar)
    preparar_fuentes_admin(destino / "incorporacion", args.aplicar)
    comprobar_externo(destino / "material", destino / "externo")
    for raiz in (destino / "material", destino / "incorporacion", destino / "externo"):
        for fichero in raiz.rglob("*.dsn"):
            if fichero.name == "publicador_confianza.dsn":
                continue  # Conexión administrativa del host: no pertenece al proceso web.
            if "actos" in fichero.relative_to(raiz).parts:
                continue  # Conexiones de gobierno separadas del servidor web.
            if "meritos" in fichero.relative_to(raiz).parts:
                continue  # DSN nuevas segregadas; las valida su provisionador nominal.
            exigir(normalizar_dsn(fichero.read_text().strip()) == fichero.read_text().strip(),
                   "dsn_clon_invalida")
    print("material_hz39=preparado" if args.aplicar else "material_hz39=comprobado")
    print("confianza_externa=" + ("publicada" if (destino / "externo/externo/v3/inventario.json").exists()
                                  else "pendiente_publicacion_admin"))


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
        raise SystemExit("material_hz39=error clave=" + str(error).split(":", 1)[0]) from None
