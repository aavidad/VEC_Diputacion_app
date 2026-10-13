# Registrar a Carmen como lectora técnica de RRHH

Carmen utiliza el perfil fijo de Alta, `tecnico_rrhh_desarrollo`, ya provisto
por el circuito existente. El cargador selecciona su discriminador
`perfil-fijo-alta-perfil` únicamente cuando reconoce el mismo certificado y
sujeto técnicos. Los lectores ordinarios conservan el discriminador genérico.
El montaje reutiliza el contexto del perfil Alta registrado, con la misma
identidad; no prepara otro perfil ni suma concesiones. El PDP continúa
comprobando los permisos vigentes.

SQL NO. CONFIG NUEVA: preparar `identidad/consultas-rrhh.json` y usar el mismo
perfil técnico en la cuenta de `identidad/inscripcion-bolsa-rrhh.json`.
Perfil indicado por dirección para cidonia:
`prf_5d140bb82db5f22e5ea49d54c9eb75ba`. Preferencias conserva su perfil propio.

Después de desplegar el cambio, ejecutar desde el repositorio con el material
privado de ese entorno. El comando reutiliza el cálculo y la lectura de
certificado existentes. Si el perfil fijo no coincide, no escribe ningún
fichero y muestra la clave y las dos referencias opacas para cotejarlas con
ContextoActor. No publica perfiles ni modifica PostgreSQL.

```sh
python3 - <<'LECTOR'
from pathlib import Path
import json, os, tempfile
from scripts.recorridos.clon_material import reference, cert_hash
r = Path(os.environ["VEC_DEVELOPMENT_MATERIAL_DIR"])
i = json.loads((r / "identidad/identidad.json").read_text())
h = cert_hash(r / "mtls/cliente.crt")
perfil = reference("prf_", i["subject"] + "\0" + h + "\0perfil-fijo-alta-perfil")
esperado = "prf_5d140bb82db5f22e5ea49d54c9eb75ba"
if perfil != esperado:
    raise SystemExit("perfil_tecnico_alta esperado=" + esperado + " actual=" + perfil)
f = r / "identidad/inscripcion-bolsa-rrhh.json"
c = json.loads(f.read_text())
cuentas = [x for x in c["cuentas"] if x["sujeto"] == i["subject"] and x["certificado_sha256"] == h]
if len(cuentas) != 1:
    raise SystemExit(f"cuenta_tecnica_rrhh esperado=1 actual={len(cuentas)}")
m = {"version": 1, "autoridad": "no_autoritativo", "entradas": [{
    "certificate": "mtls/cliente.crt", "identity": "identidad/identidad.json", "subject": i["subject"],
    "perfil_ref": perfil, "organizacion_ref": "organizacion:desarrollo:dipgra",
    "clase_ambito": "organizacion", "ambito_ref": "organizacion:desarrollo:dipgra"}]}
destino = r / "identidad/consultas-rrhh.json"
if destino.exists() and json.loads(destino.read_text()) != m:
    raise SystemExit("manifiesto_rrhh esperado=identico actual=diferente")
cuentas[0]["perfil_ref"] = perfil
def escribir_privado(p, dato):
    fd, nombre = tempfile.mkstemp(prefix=".lector-rrhh-", dir=p.parent)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "w") as out:
            json.dump(dato, out, ensure_ascii=False)
            out.write("\n")
            out.flush()
            os.fsync(out.fileno())
        os.replace(nombre, p)
    finally:
        if os.path.exists(nombre):
            os.unlink(nombre)
if not destino.exists():
    escribir_privado(destino, m)
escribir_privado(f, c)
print("Lector técnico preparado. Reiniciar y comprobar Solicitudes y admisión.")
LECTOR
```

El cargador mantiene CA, certificado, sujeto, referencia exacta, ámbito y
unicidad. Si el perfil fijo no aparece una sola vez en el soporte de Alta,
el montaje se deniega. Identificar al lector no concede las operaciones de
inscripción: dirección debe comprobar su asignación técnica y sus permisos
vigentes antes de atribuir un recorrido terminado.
