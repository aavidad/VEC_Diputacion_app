# Ensayo de las migraciones del aspirante

La versión final de las tres migraciones se instaló una vez en una copia nueva
de PostgreSQL 18.4, restaurada desde el volcado de la principal del 13 de
octubre. La copia y los registros permanecen fuera del repositorio.

Antes se instalaron AD263 y AD265, ausentes en ese respaldo. Para el ensayo se
usaron las dos ampliaciones privadas de autorización documental y lectores
externos descritas en `contrato_nucleo.md`. Esas ampliaciones no forman parte
de esta PR: C debe incluirlas en la consolidada y ensayar el conjunto definitivo.

| Orden | Migración | SHA-256 ensayado |
| --- | --- | --- |
| 1 | AUT95 | `f5ceff6d900ba5fb4110195faefa315840a53103cb2742a7e977a35041923a1d` |
| 2 | Documentos20 | `d30d7ea1d565874172535662927034617fde81ce2797923f3c0e5ef6871943df` |
| 3 | Méritos7 | `c5e30eb398a34ab4db2e253df92299ba35a99bff7127e22fd8829e0bb1a39d97` |

Las tres instalaciones terminaron con `COMMIT`. Las tres pruebas de
`pruebas_sql/` pasaron. La prueba de Documentos comprueba permisos nominales,
políticas de filas dirigidas al propietario y recuperación con historia,
salida o huella de estado incoherentes. Sus datos se revierten con `ROLLBACK`;
no simula una capacidad ni devuelve un recibo de operación.

Dos revisiones independientes del contenido final dieron su visto bueno a
estas migraciones. AUT95 y Méritos7 contienen las definiciones completas de
las funciones, con las guardas de su versión anterior. Documentos20 conserva
el perfil original al recuperar y verifica las piezas de cada versión.

Este ensayo acredita instalación y comprobaciones SQL. No acredita todavía
los pasos e1–e5 en Chrome, la recuperación del recorrido después de reiniciar
ni la candidata consolidada de C. La PR sigue en borrador hasta completar
esas comprobaciones y la calidad.
