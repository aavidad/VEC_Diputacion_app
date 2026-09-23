# B4 parte 2 + D3-B7 — paquete incremental de la réplica principal

Este paquete conecta los datos de contacto cifrados B4 y añade «Nuevo
llamamiento» B7 a la instancia sintética principal. Exige como preimagen AD3
`000046` y Bolsa llamamientos `000015`; no ejecuta `DOWN`, no toca otra base y
no contiene secretos ni datos personales reales.

`02_migraciones.sh` ensambla las cuatro migraciones canónicas en cada ejecución
y las aplica dentro de una sola transacción y en orden causal:

1. AD3 `000047`, consumidor nominal V3 para registrar datos de contacto B4;
2. Bolsa `000016`, versiones cifradas de correo y teléfonos;
3. AD3 `000048`, consumidor nominal V3 `llamamiento.emitir.v1`;
4. Bolsa `000017`, reserva append-only previa a SMTP, finalización ligada a
   una capacidad efímera privada, resultados B3 `enviado/no_enviado`
   deterministas, recuperación idempotente, bitácora de accesos B4/B7 y
   contador B12.

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
