# H10: orden SQL y ensayo del arranque de Administración

La copia sintética H9 se actualizó y completó el circuito de fuentes,
titularidad cuenta–Persona, unidad y arranque de dos administradores de
Aplicación y un perfil separado de Sistemas. Las tres CLI recuperaron los
mismos recibos después de reiniciar PostgreSQL. No se modificó la principal.

`sql_main.txt` enumera, en el orden ensayado, las 28 migraciones integradas
entre `39ec9858d` y `86f97d14c`. Cada UP terminó con salida 0 una sola vez.
`manifiesto.json` fija el commit y SHA256 de cada archivo. No contiene
configuración privada, identidades, certificados ni claves.

Después de esas 28 se ensayaron cuatro candidatas, también una sola vez:

| Orden | Pieza | Commit | Estado de integración al ensayar |
| --- | --- | --- | --- |
| 29 | AUT39, fuentes iniciales | `6d029e2627c72d9a8307327315420f8ded02bb5d` | PR #566 abierta |
| 30 | AUT40, arranque auditado | `97118f00e7079aa37df7ff2419b8bd626428ea89` | PR #584 abierta |
| 31 | IS16, vínculo de sesión | `57f63cc889ce4dd5b4dce60a70857b161e075f6c` | Rama de trabajo, sin PR |
| 32 | CA36, contexto ADMIN | `ffafcf7f67dfec7b2bd8d4f1eecfb8bd525d4cc0` | Rama de trabajo, sin PR |

La CLI de fuentes pertenece a la PR #565 (`fe05cda840`); la de arranque,
a #584. Para el ensayo se compilaron sus carpetas exactas sobre main
`86f97d14c`, junto con los preparadores de unidad y bootstrap. Esto no
convierte esas candidatas en código integrado. El manifiesto distingue
ambos grupos; **la lista de main por sí sola no completa H10**.

## Resultados y límites

Las 6.240 auditorías H9 conservaron todos sus campos y la cabeza anterior
tras los UP. Se añadieron 20 columnas, nulas en esas filas históricas;
comparar el JSON de todas las columnas habría cambiado la huella por esa
ampliación de esquema. El cotejo de contenido usa los campos H9 originales.
Después de los efectos y reintentos hay 6.253 auditorías; el prefijo H9
permanece idéntico. Los reintentos añaden su propia auditoría, conservan el
recibo y no duplican fuentes, unidad ni perfiles.

Fuentes y bootstrap tuvieron primera confirmación y recuperación con salida
0. Una aprobación divergente fue denegada y auditada con COMMIT en ambos
casos, sin recibo favorable. La unidad tuvo confirmación y recuperación tras
reinicio con salida 0. La consulta de control comprobó dos administradores
efectivos de Aplicación y una asignación vigente de Sistemas.

Los originales nuevos se guardan fuera de Git, en archivos 0600 y carpetas
0700. Las HMAC proceden del proveedor DEV existente, con configuración
explícita reutilizable por el runtime; no se reconstruyeron identidades
anteriores. Los certificados son de desarrollo. Las asignaciones del ensayo
vencen el **5 de octubre de 2026 a las 00:46:49 UTC**: antes de continuar hay
que comprobar su vigencia por la autoridad, sin ampliar fechas a mano.

El frío post-arranque queda en la bitácora privada de K con SHA256
`f7c080048ec19825da2bd244de8ad1df28438dc08bcd64a19efc4bf2ac94cf4b`.
El frío se restauró en otro clon de sólo lectura: se recuperaron las 6.253
auditorías, su cabeza, el prefijo H9 y los registros únicos de fuentes y unidad.
Este corte no acredita garantía alta, PDP favorable, montaje ADMIN,
recorrido HTTP ni despliegue. IS16 y CA36 sólo tienen ensayo estructural y
ACL; todavía no se crearon vínculos de sesión o contexto en este circuito.

## Antes de la subida

Dirección debe integrar las dependencias, fijar el ensamblaje y volver a
cotejar las versiones del destino y las huellas del manifiesto. Aplicar
únicamente SQL ausente y respetar el orden; nunca ejecutar DOWN ni repetir
una instalada. El frío H9 y las copias anteriores se conservan. La configuración
de LOGIN, material HMAC, aprobaciones, reparto y originales viaja por el canal
privado. Este directorio documenta el ensayo; no es un instalador ni una
aprobación para actuar sobre la principal.
