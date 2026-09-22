# B4 parte 2 + D3-B7 — paquete incremental de la réplica principal

Este paquete conecta los datos de contacto cifrados B4 y añade «Nuevo
llamamiento» B7 a la instancia sintética principal. Exige como preimagen AD3
`000046` y Bolsa llamamientos `000015`; no ejecuta `DOWN`, no toca otra base y
no contiene secretos ni datos personales reales.

`02_migraciones.sql` aplica dentro de una sola transacción y en orden causal:

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

El script rechaza un checkout sucio, ensaya roles y migraciones con
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
