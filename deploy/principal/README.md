# B4 parte 2 + D3-B7 + D3-B11-D — paquete incremental de la réplica principal

Este paquete conecta los datos de contacto cifrados B4 y añade «Nuevo
llamamiento» B7 a la instancia sintética principal. Exige como preimagen AD3
`000046` y Bolsa llamamientos `000015`; no ejecuta `DOWN`, no toca otra base y
no contiene secretos ni datos personales reales.

`02_migraciones.sh` ensambla las cinco migraciones canónicas en cada ejecución
y las aplica dentro de una sola transacción y en orden causal:

1. AD3 `000047`, consumidor nominal V3 para registrar datos de contacto B4;
2. Bolsa `000016`, versiones cifradas de correo y teléfonos;
3. AD3 `000048`, consumidor nominal V3 `llamamiento.emitir.v1`;
4. Bolsa `000017`, reserva append-only previa a SMTP, finalización ligada a
   una capacidad efímera privada, resultados B3 `enviado/no_enviado`
   deterministas, recuperación idempotente, bitácora de accesos B4/B7 y
   contador B12.
5. Bolsa `000021`, función propietaria para vincular actas anteriores a 000008
   con referencias `can_*` derivadas por el recuperador protegido. Solo la
   conexión administrativa puede invocarla como propietario. Su
   instalación no ejecuta el relleno: el procedimiento está en `03_entorno.md`.

No se añaden roles, conexiones ni variables `*_DATABASE_URL`. B7 reutiliza
las conexiones Bolsa/AD3, el KMS de desarrollo, los datos de contacto B4 y el
relay SMTP ya configurado. El plazo visible es provisional y queda rotulado
«pendiente de RRHH, dudas 1–3».

## Aplicación

La configuración privada y los certificados se preparan fuera de Git. Con la
réplica detenible y la preimagen comprobada:

```bash
./deploy/principal/desplegar.sh
```

El script rechaza un checkout sucio, ensambla desde las fuentes para impedir
copias desfasadas, ensaya roles y migraciones con
`ROLLBACK`, las aplica con `COMMIT`, compila, conserva un respaldo, sincroniza
`web/`, reinicia y ejecuta `verificar.sh`. No publica ni despliega por sí
solo fuera de la instancia indicada por sus variables.

## Evidencia local

En PostgreSQL 18.4, AD3 `000047` completó `ROLLBACK` y `COMMIT`; la secuencia
AD3 `000047` → Bolsa `000016` → AD3 `000048` → Bolsa `000017` completó con
`COMMIT`; Bolsa `000017` completó además un ensayo íntegro con `ROLLBACK`. El
recorrido HTTP/SMTP, concurrencia, replay y recuperación tras reinicio se
acreditan por separado sobre la réplica D6 preparada para este corte. Una
reserva sin resultados se recupera como `emision_reservada_resultado_pendiente`
y nunca provoca un reenvío automático ambiguo. Los contactos manuales B3 no
alteran el resultado SMTP ni el contador del llamamiento.
# Incremento D3-B6

El paquete aplica además Bolsa `000018`: política de orden versionada y rotulada
como provisional, lectura derivada para B5/B7 y registro append-only de la
reposición al volver de `trabajando`. `desplegar.sh` ensaya primero el `UP` con
`ROLLBACK`; `verificar.sh` exige la política y los campos de orden calculado.
No existe AD3 `000049`: B6 reutiliza la lectura autorizada B5 y la reposición se
produce dentro del cambio B2 ya autorizado, auditado e idempotente.

# Incremento D3-AV

Bolsa `000020` añade exclusivamente la función de lectura que deriva los avisos
de salto de orden y de tres años desde B7, B6 y el histórico B2. El despliegue
ensaya el `UP` con `ROLLBACK`, lo aplica después de `000018` y comprueba el GET
`/api/vec/bolsa/avisos`; no añade consumidor AD3, tablas ni escritura de negocio.

# Incremento D3-B8

Bolsa `000019` registra pausar, reactivar y excluir en la situación B2 y en una
fila append-only con motivo, referencia y huella del justificante, actor y
validador. El paquete ensaya el `UP` con `ROLLBACK` y lo aplica tras `000018`.
Reutiliza la autorización B2; no incorpora documentos ni otro consumidor AD3.
La exigencia de validador distinto al registrar una exclusión es provisional
hasta resolver la duda 6 de RRHH. La pausa conserva la posición de acta y B6
la recupera al reactivar con la política de orden vigente.

# Incremento B2 — registro de empleado de Personal

Migraciones nuevas, en este orden y cada una ensayada antes con `ROLLBACK`:
ContextoActor `000008`; Personal `000017`; AD3 `000054`, `000055`, `000056`
(sus números son huecos reservados: se instalan después de AD3 59/61/70/80,
ya presentes en la principal); Personal `000018`, `000019`, `000020` y
`000021`. Las AD3 toman el consultivo común del núcleo y rechazan una segunda
aplicación. El material privado V3 de Personal B2 usa el formato 3 (ocho
capacidades, con la lista de empleados del organismo).

`personal_altas_sinteticas.py` da de alta empleados sintéticos por la API
interna real con el certificado mTLS de una persona de RRHH: cada alta es el
acto V3 de Personal `000019` y publica la proyección persona→empleado
(`000016`) que consumen Cronos y Dietas. Lee un plan privado 0600 fuera del
repositorio (solo referencias opacas), publica antes las entradas de catálogo
que el plan declara y es idempotente: la clave incluye el SHA-256 del cuerpo,
así que repetirlo devuelve los mismos recibos; un 409 (entrada o persona ya
registradas con otro contenido o por otra vía) se informa como divergencia y
detiene el plan con código 1, igual que la primera caída.
`--comprobar` valida el plan sin enviar nada.

# Incremento Dietas D5/D6 — otros gastos, corregir y reenviar

Ninguna de estas migraciones tiene historia en una base productiva. El orden de
activación, verificado contra las precondiciones SQL de cada una, es:

1. **SQL** — `04_dietas_migraciones.sh --incremental` ensambla en una sola
   transacción, por este orden, Dietas `000005`, AD3 `000059`, Personal
   `000012`/`000013`, Dietas `000006`, `000007`, AD3 `000080`, Dietas `000008`,
   AD3 `000075`, Dietas `000009`, `000010` y `000011`. Cada migración va tras
   una marca de catálogo y se omite (`\echo OMITIDA`) si ya está instalada: el
   paquete nunca reaplica y puede repetirse. Exige R1 (Dietas
   `000001`–`000004`), AD3 `000051`/`000052`, Personal `000010`/`000011` y
   Composición `000010a`; si falta alguna, su precondición aborta todo.
   Ensayar con `ROLLBACK` y confirmar con `COMMIT` sustituyendo `:finalizar;`:

   ```bash
   bash deploy/principal/04_dietas_migraciones.sh --incremental \
     | sed 's/^:finalizar;$/ROLLBACK;/' | psql -X -v ON_ERROR_STOP=1 ...
   ```

   AD3 `000075` no tiene `DOWN` (como AD3 `000054`/`000056`) y no depende de
   Dietas `000009`–`000011` ni estas de ella; se instala antes, de modo que la
   lista con devolución sigue cerrada en Dietas hasta `000011`. Dietas `000010`
   y `000011` se instalan juntas, sin tráfico entre ambas, y nunca se ejecuta el
   `DOWN` de `000011` dejando `000010`: reabriría la devolución al revisor sin
   cotejar la decisión V3.
2. **Política V3** — `politica_dietas_d5_d6.py` da las concesiones exactas
   (editar, borrar, enviar y consultar el documento propio para la titular;
   `dietas.circuito.documento.consultar` para quien revisa), con
   `campos_permitidos` en orden de bytes y copiadas de AD3 `000075`
   (`test_politica_dietas_d5_d6.py` las compara con la migración y con Dietas
   `000011`). Solo emite datos: la nueva versión del rol y de la asignación se
   publica con el procedimiento revisado de Dirección, sin reutilizar el
   preparador histórico R1D.
3. **Binario** — por último, la aplicación con la web de esta tanda.

Evidencia local: PostgreSQL 18.4 desechable sobre una base sintética con núcleo
AD3: paquete incremental con `ROLLBACK` sin rastro, `COMMIT`, repetición con
las doce migraciones omitidas y sin cambios, y `DOWN` de `000011` seguido del
paquete, que reinstala solo `000011` al estado idéntico.
