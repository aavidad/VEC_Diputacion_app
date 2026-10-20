#!/usr/bin/env python3
"""Emite una operación del recorrido de contratación temporal y conserva su recuperación."""
import argparse
import fcntl
import hashlib
import json
import os
import stat
import sys
import uuid
from pathlib import Path

from preparar import Corte, ROOT, RUTAS, cargar, data_http, exigir_origen, guardar, mTLS


RUTAS_CLIENTE = {**RUTAS, "fiscalizacion": (
    "intervencion", "/api/vec/contratacion-temporal/fiscalizaciones/resultados")}


def ruta_privada(path, nombre):
    if (not path.is_absolute() or any(part.is_symlink() for part in (path, *path.parents))
            or ROOT == path.resolve() or ROOT in path.resolve().parents):
        raise Corte(f"FALTA: {nombre} exige ruta absoluta privada fuera de Git, sin enlace")
    return path


def entrada_privada(path):
    try:
        info = path.stat()
        if (path.is_symlink() or not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
                or stat.S_IMODE(info.st_mode) != 0o600 or not 0 < info.st_size <= 65536):
            raise ValueError("modo o tamaño")
        raw = path.read_bytes()
        value = json.loads(raw)
    except (OSError, ValueError) as exc:
        raise Corte("FALTA: entrada JSON privada 0600, regular y de hasta 64 KiB") from exc
    if not isinstance(value, dict):
        raise Corte("FALTA: la entrada debe ser un objeto JSON")
    return raw, value


def destino(args, actor):
    try:
        material_info = args.material.stat()
    except OSError as exc:
        raise Corte("FALTA: directorio de material no disponible") from exc
    if (not stat.S_ISDIR(material_info.st_mode) or material_info.st_uid != os.getuid()
            or stat.S_IMODE(material_info.st_mode) & 0o022):
        raise Corte("FALTA: directorio de material inseguro")
    cert, key = mTLS(args, actor)
    ca = args.material / "ca/ca.crt"
    if not ca.is_file():
        raise Corte("FALTA: CA privada para la conexión local")
    try:
        if any(path.is_symlink() or not path.is_file() for path in (cert, key, ca)):
            raise Corte("FALTA: material mTLS enlazado o inválido")
        key_info = key.stat()
        if (not stat.S_ISREG(key_info.st_mode) or key_info.st_uid != os.getuid()
                or stat.S_IMODE(key_info.st_mode) != 0o600):
            raise Corte("FALTA: clave mTLS privada insegura")
        if any(path.stat().st_size > 1048576 for path in (cert, ca)):
            raise Corte("FALTA: certificado o CA demasiado grande")
        huellas = [hashlib.sha256(path.read_bytes()).hexdigest() for path in (cert, ca)]
    except OSError as exc:
        raise Corte("FALTA: certificado o CA no legible") from exc
    return {"origen": args.origen, "material": str(args.material), "actor": actor,
            "certificado": str(cert), "clave": str(key),
            "certificado_sha256": huellas[0], "ca_sha256": huellas[1]}


def comprobar_recibo(name, body, receipt):
    def ref(value):
        return isinstance(value, str) and bool(value)

    if not isinstance(receipt, dict):
        raise Corte("FALTA: respuesta sin recibo verificable")
    if name == "alta":
        alta = receipt.get("recibo_alta")
        peticion = receipt.get("peticion")
        ok = (receipt.get("estado_entrega") == "confirmada" and isinstance(alta, dict)
              and isinstance(peticion, dict) and peticion.get("referencia") == body.get("peticion_ref")
              and ref(alta.get("recibo_ref")) and ref(alta.get("expediente_ref"))
              and alta.get("version") == 1)
    else:
        ok = ref(receipt.get("recibo_ref"))
        if name == "ratificar":
            ok = ok and receipt.get("peticion_ref") == body.get("peticion_ref")
        if name in ("analisis", "asignacion", "informe", "fiscalizacion"):
            version = body.get("version_esperada")
            ok = (ok and ref(body.get("expediente_ref"))
                  and receipt.get("expediente_ref") == body["expediente_ref"]
                  and type(version) is int and receipt.get("version_resultante") == version + 1)
        if name == "cobertura":
            version = body.get("version_esperada")
            ok = (ok and ref(body.get("expediente_ref"))
                  and receipt.get("esquema") == "vec.contratacion-temporal.recibo-cobertura.v1"
                  and receipt.get("estado") == "aplicada"
                  and ref(receipt.get("decision_cobertura_ref"))
                  and type(version) is int and receipt.get("version_resultante") == version + 1
                  and ref(receipt.get("confirmada_en")))
        if name == "fiscalizacion":
            ok = ok and receipt.get("resultado") == body.get("resultado")
        if name == "vinculo_rpt":
            vinculo = receipt.get("vinculo")
            revision = body.get("revision_esperada")
            ok = (ok and isinstance(vinculo, dict) and ref(body.get("expediente_ref"))
                  and receipt.get("expediente_ref") == body["expediente_ref"]
                  and type(revision) is int and receipt.get("revision") == revision + 1
                  and vinculo.get("recibo_ref") == receipt["recibo_ref"]
                  and vinculo.get("revision") == receipt["revision"]
                  and vinculo.get("categoria_id") == body.get("categoria_id")
                  and vinculo.get("catalogo_version") == body.get("catalogo_version")
                  and vinculo.get("catalogo_huella_sha256") == body.get("catalogo_huella_sha256")
                  and receipt.get("prospectivo") is True
                  and receipt.get("acredita_procedencia_historica") is False
                  and vinculo.get("prospectivo") is True
                  and vinculo.get("acredita_procedencia_historica") is False)
    if not ok:
        raise Corte("FALTA: recibo no corresponde a la intención; conservar estado privado")


def tramitar(args):
    actor, route = RUTAS_CLIENTE[args.operacion]
    raw, input_body = entrada_privada(args.entrada)
    input_hash = hashlib.sha256(raw).hexdigest()
    expected = destino(args, actor)
    args.estado.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    parent_info = args.estado.parent.stat()
    if (not stat.S_ISDIR(parent_info.st_mode) or parent_info.st_uid != os.getuid()
            or stat.S_IMODE(parent_info.st_mode) & 0o022):
        raise Corte("FALTA: directorio del estado privado inseguro")
    lock = args.estado.with_name(args.estado.name + ".lock")
    if args.estado.is_symlink() or lock.is_symlink():
        raise Corte("FALTA: estado o bloqueo privado enlazado")
    fd = os.open(lock, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        info = os.fstat(fd)
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600:
            raise Corte("FALTA: bloqueo privado inseguro")
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        if args.estado.exists():
            info = args.estado.stat()
            if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
                    or stat.S_IMODE(info.st_mode) != 0o600 or info.st_size > 1048576):
                raise Corte("FALTA: estado privado inseguro")
        state = cargar(args.estado)
        if (state["operaciones"] or state["valores"] or "destino" in state) and state.get("cliente_tramitar") != 1:
            raise Corte("FALTA: estado de otro preparador; use un estado nuevo para esta intención")
        if state.get("destino", expected) != expected:
            raise Corte("FALTA: estado de otro origen, actor o material")
        operations = state["operaciones"]
        if operations and set(operations) != {args.operacion}:
            raise Corte("FALTA: este estado pertenece a otra operación")
        old = operations.get(args.operacion)
        if old is None:
            body = dict(input_body)
            generated_key = None
            if args.operacion != "alta" and "clave_idempotencia" not in body:
                generated_key = str(uuid.uuid4())
                body["clave_idempotencia"] = generated_key
            if args.operacion != "alta" and (not isinstance(body.get("clave_idempotencia"), str)
                                             or not body["clave_idempotencia"]):
                raise Corte("FALTA: clave de idempotencia inválida")
            old = {"actor": actor, "ruta": route, "entrada_sha256": input_hash,
                   "cuerpo": body, "estado": "pendiente"}
            if generated_key is not None:
                old["clave_generada"] = generated_key
            state.update(cliente_tramitar=1, destino=expected)
            operations[args.operacion] = old
            guardar(args.estado, state)  # intención durable antes del POST
        elif (not isinstance(old, dict) or not isinstance(old.get("cuerpo"), dict)
              or old.get("actor") != actor or old.get("ruta") != route
              or old.get("entrada_sha256") != input_hash):
            raise Corte("FALTA: entrada o contrato distinto de la intención guardada")
        expected_body = dict(input_body)
        if "clave_generada" in old:
            if args.operacion == "alta" or "clave_idempotencia" in expected_body:
                raise Corte("FALTA: clave guardada incompatible con la entrada")
            expected_body["clave_idempotencia"] = old["clave_generada"]
        if old["cuerpo"] != expected_body:
            raise Corte("FALTA: cuerpo guardado distinto de la entrada original")
        if old.get("estado") == "confirmada":
            comprobar_recibo(args.operacion, old["cuerpo"], old.get("respuesta"))
            print("Operación ya confirmada; recibo conservado en el estado privado")
            return 0
        if old.get("estado") != "pendiente":
            raise Corte("FALTA: estado de operación desconocido")
        receipt = data_http(args, actor, "POST", route, old["cuerpo"], allowed=(200, 201))
        comprobar_recibo(args.operacion, old["cuerpo"], receipt)
        old.update(estado="confirmada", respuesta=receipt)
        guardar(args.estado, state)
        print("Operación confirmada; recibo conservado en el estado privado")
        return 0
    except BlockingIOError as exc:
        raise Corte("FALTA: otra ejecución usa este estado privado") from exc
    finally:
        os.close(fd)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__,
        epilog="Para activar, indique operación, origen local, material, estado nuevo y entrada JSON de esa actuación. Si el resultado es incierto, repita la misma orden con la misma entrada: se conserva la clave. El recibo queda en el estado privado. Las firmas se hacen en el navegador.")
    parser.add_argument("--operacion", choices=tuple(RUTAS_CLIENTE), required=True)
    parser.add_argument("--origen", required=True, help="origen HTTPS loopback con puerto")
    parser.add_argument("--material", type=Path, required=True, help="directorio privado de mTLS y CA")
    parser.add_argument("--estado", type=Path, required=True, help="JSON privado nuevo por intención")
    parser.add_argument("--entrada", type=Path, required=True, help="cuerpo JSON privado 0600 de la actuación")
    args = parser.parse_args(argv)
    os.umask(0o077)
    args.origen = exigir_origen(args.origen)
    args.material = ruta_privada(args.material, "--material").resolve()
    args.estado = ruta_privada(args.estado, "--estado")
    args.entrada = ruta_privada(args.entrada, "--entrada")
    return tramitar(args)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Corte as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(2)
