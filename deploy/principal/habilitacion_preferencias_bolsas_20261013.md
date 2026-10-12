# Activar Preferencias entre bolsas en Mi Bolsa

Dirección ejecuta este guion. Reutiliza #1041, #1048 y el kit local
`~/.cache/vec-claude-t/b103-encender`: admisión de catálogo, dos LOGIN
segregados, gobierno ADMIN y propuesta/cierre por dos personas distintas.
Esta entrega añade la fuente; no instala SQL ni cambia cidonia.

## 1. Admitir la fuente conservando el catálogo actual

Añadir `fuentes/bolsa_preferencias_bolsas_propias_v1.json` al paquete del
catálogo vigente mediante el mismo acto de admisión del kit B103. Conservar
las fuentes anteriores y el censo completo actualizado, incluidos los roles
RRHH y externo. El kit fija B103/C3 en `FUENTES_NUEVAS`: incorporar esta
fuente a la preparación y recalcular paquete, catálogo, plan y aprobación;
no ejecutar el paquete B103 anterior esperando que añada preferencias.

- Fuente: `fuente:bolsa:preferencias_bolsas_propias`, versión `1`.
- SHA-256 de `entradas_canon`: `4ef4d31641136feda9d9d29cb97b133ca3dc1312fa093e9439d31b2b68c01788`.
- SHA-256 del archivo: `1b3047c1c11b611b666a2dffa705aaef1249589ca94b3e7509664e8c109c33d8`.
- Vigencia: `2026-10-01T00:00:00Z` a `2027-10-01T00:00:00Z`.

Las dos entradas usan recurso `participaciones_candidato`, ámbito
`candidato_ref`, garantía `alto` y finalidad
`gestion_preferencias_bolsas_propias`. La lectura tiene los ocho campos de
`camposPreferenciasBolsaExterna`, en su orden, y obligación `auditar`.
Ordenar no tiene campos ni obligaciones. La huella de una entrada se obtiene
con `EntradaAccionAdministracionV1.HuellaSHA256()`; **no es la huella de la
fuente**. Si cambia la vigencia, preparar otra versión de la fuente.

Comprobar las listas ya existentes
`lista_sql_codexi_preferencias_bolsas_20261012.txt` y
`lista_sql_codexi_perfil_externo_admin_20261012.txt`. No reaplicar sus SQL.
La nueva fuente se admite por el acto de catálogo, no por SQL directo.

## 2. LOGIN propios y origen AD172

Con el patrón `pools` del kit, preparar archivos privados `0600`:

| Archivo en `vec-admin/pools/` | LOGIN elegido | Única membresía |
| --- | --- | --- |
| `version-bolsa-externa-ejecutor.json` | `vec_adm_externa_version_20261013` | `vec_admin_version_rol_bolsa_externa_ejecutor` |
| `catalogo-acciones-lector-externa.json` | `vec_adm_externa_catalogo_20261013` | `vec_admin_catalogo_acciones_lector` |

Usar contraseñas nuevas fuera de Git, TLS verificado y el patrón SCRAM/HBA
del kit; no copiar DSN ni claves B103. El ejecutor hereda su grupo sin
opciones ADMIN ni SET; ninguno recibe otros grupos. El DBA registra estas
dos filas en `vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1`:

| login_nombre | audiencia_consumo | operacion | proceso | canal_permitido |
| --- | --- | --- | --- | --- |
| `vec_adm_externa_version_20261013` | `vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1` | `administracion.perfiles.version_bolsa.proponer` | `vec-admin` | `administracion_privilegiada` |
| `vec_adm_externa_version_20261013` | `vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1` | `administracion.perfiles.version_bolsa.aprobar` | `vec-admin` | `administracion_privilegiada` |

Las operaciones mantienen `version_bolsa`, sin sufijo `externa`. Si las
filas ya existen, cotejar sus cinco valores y conservarlas. El lector no
necesita filas de consumo. Estas filas no conceden permisos al aspirante.

## 3. Gobierno conjunto 6 y overlay externo

En una carpeta nueva de la renovación ADMIN, mantener la configuración
vigente y cambiar `conjunto_capacidades` a `6`. Añadir este bloque, con
`motivos` tomados del catálogo publicado para cada audiencia:

```json
{
  "conjunto_capacidades": 6,
  "version_bolsa_externa": {
    "pool_gobierno": "/home/openclaw/.local/state/vec-admin-principal-20261005/vec-admin/pools/version-bolsa-externa-ejecutor.json",
    "pool_catalogo": "/home/openclaw/.local/state/vec-admin-principal-20261005/vec-admin/pools/catalogo-acciones-lector-externa.json",
    "motivos": {
      "vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1": {
        "catalogo_id": "<catálogo de motivos vigente>",
        "catalogo_version": 1,
        "catalogo_huella_sha256": "<huella vigente>",
        "entrada_clave": "<motivo publicado para proponer>"
      },
      "vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1": {
        "catalogo_id": "<catálogo de motivos vigente>",
        "catalogo_version": 1,
        "catalogo_huella_sha256": "<huella vigente>",
        "entrada_clave": "<motivo publicado para aprobar>"
      }
    }
  }
}
```

El bloque se integra en la configuración completa; sustituir también las
versiones de motivos por las vigentes. Con `CFG` apuntando a ese archivo y
`TEXTOS` al catálogo de la herramienta, ejecutar las fases existentes:

```sh
vec-gobierno-usuarios-admin -fase preparar -config "$CFG" -textos "$TEXTOS" -timeout 2m
vec-gobierno-usuarios-admin -fase aplicar -config "$CFG" -textos "$TEXTOS" -timeout 2m -acuse acuse-aplicar.json
vec-gobierno-usuarios-admin -fase verificar -config "$CFG" -textos "$TEXTOS" -timeout 2m
vec-gobierno-usuarios-admin -fase version-bolsa-externa -config "$CFG" -textos "$TEXTOS" -timeout 2m -acuse acuse-aplicar.json
```

La última fase escribe `version-bolsa-externa.json` y sus dos materiales
propios en `salida`. Añadir al entorno ADMIN del día, conservando B1/B103:
`VEC_ADMIN_VERSION_BOLSA_EXTERNA_CONFIG_FILE=<salida>/version-bolsa-externa.json`.
Regenerar los overlays que dependan del gobierno renovado y conservar este
bloque en la renovación diaria. Arrancar `vec-admin` por el launcher del kit.
No convertir un acuse incierto en una preparación nueva: recuperar la misma
operación por el circuito existente.

## 4. Propuesta y cierre externos

RolID: `candidato_bolsa_portal_historial_propio_desarrollo`. La base exigida
es `rol:candidato_bolsa_portal_historial_propio_desarrollo:v1`, publicada y
habilitada, con sus siete concesiones originales. El objetivo es `v2`, con
nueve concesiones. Leer catálogo, cabeza, control y asignaciones externos
vigentes siguiendo las lecturas del kit B103, cambiando el RolID consultado.
Cada asignación conserva su documento completo y exactamente un
`candidato_ref`; seleccionar entre una y dieciséis asignaciones.

Preparar el plan con `domain.PrepararPlanVersionarRolBolsaExterna` y obtener
`huella_plan_esperada` mediante `plan.HuellaSHA256()`. El kit B103 prepara
cuerpos sin ese campo: su cuerpo no sirve directamente para este handler.
Guardar el plan en `plan.json` y su huella en `huella-plan.txt`, fuera de Git.
Conservar en `lecturas.json` los campos `base_huella_sha256` y
`control_huella_sha256` de la misma lectura B103 adaptada al rol externo;
el documento del plan no contiene esas dos huellas como campos JSON.
Así se reutiliza la validación del dominio y no se calcula la huella sobre
un JSON reordenado por `jq`.

Este comando obtiene el **cuerpo completo de la propuesta** del plan ya
preparado, copiando las asignaciones y selecciones CAS sin reconstruirlas:

```sh
umask 077
jq --arg operacion "propuesta_admin:$(openssl rand -hex 16)" \
   --arg huella "$(cat huella-plan.txt)" --slurpfile lecturas lecturas.json '
  {operacion_ref:$operacion,
   catalogo_ref:.catalogo_ref, catalogo_version:.catalogo_version,
   catalogo_huella_sha256:.catalogo_huella_sha256,
   base_ref:("rol:"+.base.rol.rol_id+":v"+(.base.rol.version|tostring)),
   base_huella_sha256:$lecturas[0].base_huella_sha256,
   control_revision:.base.control_vigencia.revision,
   control_huella_sha256:$lecturas[0].control_huella_sha256,
   selecciones:.selecciones, asignaciones:.asignaciones, motivo:.motivo,
   huella_plan_esperada:$huella}
  + (if (.referencia_acto // "") == "" then {} else {referencia_acto:.referencia_acto} end)
' plan.json > propuesta.json
```

Antes de enviar, `selecciones` debe contener, en este orden:
`accion:bolsa.preferencias_bolsas.consultar_propias` y
`accion:bolsa.preferencias_bolsas.ordenar_propias`, cada una con
`entrada_ref`, `entrada_version=1` y su propia `entrada_huella_sha256`.
La primera persona ADMIN envía `propuesta.json` por POST a
`/api/admin/perfiles/v1/gobierno-version-bolsa-externa/propuestas` y guarda
la respuesta en `respuesta-propuesta.json`. Se espera `201`; la recuperación
con el mismo cuerpo devuelve `200` y la misma propuesta.

La segunda persona ADMIN usa el motivo publicado de aprobación,
guardado en `motivo-cierre.json`, para obtener el **cuerpo completo del cierre**:

```sh
jq --arg operacion "cierre_admin:$(openssl rand -hex 16)" \
   --slurpfile motivo motivo-cierre.json '
  {operacion_ref:$operacion, propuesta_ref:.propuesta_ref,
   propuesta_huella_sha256:.huella_sha256, motivo:$motivo[0]}
' respuesta-propuesta.json > cierre.json
```

Enviar por POST a `/api/admin/perfiles/v1/gobierno-version-bolsa-externa/cierres`.
Se espera `200`, objetivo `v2` y `recibo_ref`. Guardar ambos cuerpos y respuestas;
repetir el mismo archivo ante un resultado incierto, sin generar otra clave.
Si el rol ya es `v2`, recuperar el cierre existente en vez de proponer otra versión.

## 5. Material del aspirante y comprobación en pantalla

Con el entorno **interno** vigente y `MATERIAL_EXTERNO` apuntando al material
externo ya separado, ejecutar:

```sh
vec-server preparar-portal-externo --material-externo "$MATERIAL_EXTERNO" --consumidores preferencias_bolsas
```

Preparar conserva los demás consumidores cuando la raíz es la misma. Si se
renovó la raíz, pedir todos los consumidores activos juntos. Reiniciar el
proceso externo por su launcher, abrir Mi Bolsa con el aspirante aprovisionado,
consultar Preferencias entre bolsas, cambiar el orden y guardar. Comprobar
el recibo, volver a abrir y conservar el orden. La lectura usa sesión y
auditoría; únicamente guardar consume V3. La preferencia guardada no adjudica
una plaza ni altera por sí sola un llamamiento.

CONFIG NUEVA: `VEC_ADMIN_VERSION_BOLSA_EXTERNA_CONFIG_FILE=/home/openclaw/.local/state/vec-admin-principal-20261005/gobierno/20261013/version-bolsa-externa.json`
si `salida` es esa carpeta; en la renovación diaria usar la carpeta vigente.
No añade selectores ni configuración nueva a `vec-server`: añade el consumidor
`preferencias_bolsas` al material externo existente.
