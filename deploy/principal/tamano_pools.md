# Tamaño de los pools de PostgreSQL

Cada pool de conexiones tiene un máximo explícito. El despliegue lo cambia
sin tocar código, añadiendo `pool_max_conns` al DSN de esa conexión, por
ejemplo `...?sslmode=verify-full&pool_max_conns=12`. Si el DSN no lo trae,
se usa el valor por defecto de esta tabla, nunca el de pgx (que depende de
los núcleos de la máquina).

La suma de los máximos de todos los procesos que comparten una base debe
quedar por debajo de su `max_connections` (100 en la principal), dejando
margen para administración y copias.

| Pool | Proceso | Por defecto |
|---|---|---|
| Proyección pública de Bolsa (`VEC_BOLSA_PUBLICA_DATABASE_URL` y su gemela del externo) | vec-publico, portal externo | 6 |
| Mi bolsa del portal externo | portal externo | 8 |
| Comprobación previa V3 del portal externo | portal externo | 4 |
| Preferencias de usuario | interno y externo | 4 |
| Consultas RRHH de Contratación temporal | interno | 4 |
| Resolución de motivos RRHH de Contratación temporal | interno | 4 |
| Acreditación de cobertura O4-05 de Contratación temporal | interno | 4 |

Con más conexiones solo se gana si PostgreSQL tiene núcleos libres: en el
laboratorio de carga (`docs/estudio_requisitos/rendimiento_carga_20261006.md`)
la proyección pública con 4 núcleos ya estaba saturada con 6 conexiones.
