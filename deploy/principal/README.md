# D6 + B3 — paquete de réplica principal

Este paquete actualiza exclusivamente la instancia sintética principal. Parte de
su estado conocido: Contratación `000108`, Bolsa llamamientos `000003…000008`
(más C21) y AD3 hasta `000043`. No ejecuta `DOWN`, no toca otra base y no incluye
secretos.

## Diferencia que aplica

En este orden causal y dentro de una sola transacción de migraciones:

1. rol NOLOGIN `vec_bolsa_llamamientos_registrador_frontera` y LOGIN nominal
   `vec_b2_auditoria_frontera_desarrollo` (`01_roles.sql`); el nombre evita
   deliberadamente el prefijo reservado a los roles internos del módulo;
2. AD3 `000044`, consumidor V3 de crear/consultar borrador B-BACK;
3. Bolsa llamamientos `000011`, agregado durable y bitácora de frontera B-BACK;
4. AD3 `000045`, consumidor V3 de cambio de situación B2;
5. Bolsa llamamientos `000012`, historia append-only de las siete situaciones B2.
6. AD3 `000046`, consumidor V3 nominal para registrar contactos B3;
7. Bolsa llamamientos `000013`, tabla append-only y lectura paginada de contactos B3.

El inventario completo no encuentra otra migración de `main@48e64f84` posterior
al estado indicado: Bolsa `000009/000010` no existen en `main` y CT termina en
`000108`. El árbol versionado salta de AD3 `000038` a `000044`; la principal
conserva AD3-43 por la cadena instalada declarada por Dirección. El paquete no
la reaplica. `02_migraciones.sql` conserva los `REVOKE ... FROM PUBLIC` y
`GRANT ... TO <rol>` de las seis fuentes y las ejecuta como una unidad.

`03_entorno.md` enumera las seis conexiones ausentes, el contador 18 y el
material privado `bolsa-bback.json` v2. La configuración privada debe prepararse
antes del comando; no puede derivarse ni guardarse en Git.

Decisión de Dirección de 21/09: AD3 `000044/000045` acreditan la preimagen por
estructura y firmas sobre AD3-32/43, sin autohuella SHA-256 del núcleo V3.
AD3 `000046` conserva esa decisión D6 y añade únicamente el perfil nominal B3.

## Un comando

Requisitos: sesión `openclaw`, checkout del repositorio, `git`, `psql`, Go con
`GOTOOLCHAIN=auto`, `podman`, `rsync`, `curl`, `jq`; 18 conexiones ya incorporadas
al arranque privado y las variables de administración/verificación cargadas en
la sesión sin imprimir sus valores.

```bash
./deploy/principal/desplegar.sh
```

El script avanza `main` solo por `pull --ff-only`, para la aplicación, ensaya
roles y migraciones con `ROLLBACK`, los aplica, compila, respalda y sustituye el
binario, sincroniza `web/`, arranca, comprueba ambos contenedores, muestra 20
líneas de log y ejecuta `verificar.sh`. Si falla tras parar la aplicación,
intenta arrancar de nuevo el contenedor conservado. El respaldo queda junto al
artefacto, en `respaldo-d6-<UTC>`.

`verificar.sh` exige por variables la URL, CA/certificado/clave cliente y las
referencias sintéticas. Comprueba HTTP 200 en incorporación, ficha GINPIX,
recuperación de anotación, preparación de cierre, seguimiento, B12, B5 y B10;
no contiene ni imprime secretos.

## Prueba local

Validada el 21/09/2026 en `postgres:18.4-bookworm` desechable, reproduciendo CT
`000108`, AD3-32/43 y Bolsa `000003…000008` antes del paquete. `01_roles.sql` y
`02_migraciones.sql` pasaron primero con `ROLLBACK` y después con `COMMIT`; CT88
y una llamada CT108 conservaron su resultado. `vec-server` arrancó por TLS con
las 18 conexiones: B5 devolvió HTTP 200 y un POST B2 sintético devolvió HTTP 201
con recibo durable. No se usó el servidor real ni material personal.

B3 se comprobó el 22/09/2026 sobre la misma réplica: `000013` final completó
su ciclo `DOWN/UP` y el paquete conjunto ya había superado `ROLLBACK/COMMIT` en
PostgreSQL 18.4. El servidor arrancó con la migración final; el alta devolvió
HTTP 201, su repetición HTTP 200 con el mismo recibo y una sola fila, la lectura
paginada HTTP 200, el comando divergente HTTP 409 y una bolsa fuera del ámbito
nominal HTTP 403. La composición conservó el rol B2 v2 y publicó B3 como v3
desde esa preimagen exacta. Son identidad, datos y autoridad sintéticos de desarrollo.
