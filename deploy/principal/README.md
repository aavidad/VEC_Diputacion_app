# D3-B7 — paquete incremental de la réplica principal

Este paquete añade exclusivamente «Nuevo llamamiento» B7 a la instancia
sintética principal. Exige como preimagen AD3 `000047` y Bolsa llamamientos
`000016`; no los reaplica, no ejecuta `DOWN`, no toca otra base y no contiene
secretos ni datos personales reales.

`02_migraciones.sql` aplica dentro de una sola transacción y en orden causal:

1. AD3 `000048`, consumidor nominal V3 `llamamiento.emitir.v1`;
2. Bolsa `000017`, emisión append-only, recuperación idempotente, resultados
   B3 `enviado/no_enviado` y contador B12.

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

En PostgreSQL 18.4, Bolsa `000016` completó `ROLLBACK` y `COMMIT`; el par
exacto AD3 `000048` + Bolsa `000017` completó conjuntamente `ROLLBACK`.
El `COMMIT`, el recorrido HTTP/SMTP, el replay y la recuperación tras reinicio
se acreditan únicamente cuando Dirección publique y prepare AD3-47/B4 parte 2
en la réplica D6.
