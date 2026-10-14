# Dietas: resultado del recorrido local

Estado a 14 de octubre de 2026: **pendiente de completar**.

## Base restaurada

PostgreSQL 18.4, copia propia de HZ37. Restauración de globales, base y permisos
de tipos y base terminada sin error. Huella SHA-256 del respaldo de datos:
`7ae0613fa174aebc8aa36c079a6c4b268b771df41c9310129ee9dfae9650cc45`.

El inventario encontró las funciones de Dietas 000001–000008, ninguna
comisión y ninguna liquidación. Las versiones de tarifa y su regla nacional
son provisionales. La asignación R1D vigente está retirada y sus ocho cuentas
no permiten iniciar sesión; las tres cuentas adicionales de F4b no existían.

## Activación

- El incremental existente omitió correctamente las piezas instaladas y se
  detuvo en AD75. Toda esa transacción se revirtió: no instaló Dietas
  000009–000011.
- La primera función cotejada conserva el cuerpo esperado por AD75, pero su
  entorno instalado incluye `pg_temp` en `search_path`. Se prepara una
  sucesora prospectiva; no se cambia la guarda ni la función a mano.
- AD268 y el incremental posterior terminaron sin error. El incremental
  omitió AD75 y las piezas ya instaladas; Dietas 000009–000011 quedaron juntas
  en una transacción. Dos revisiones independientes dieron GO a AD268 en
  `58cdfe06f`, con huellas, permisos y rechazos de listas de campos cotejados.
- AD81 se ensayó y se instaló una vez. La comprobación posterior confirmó
  `USAGE` para el ejecutor Dietas, sin opción de concederlo a otros.
- F4b se ensayó sin confirmar. AUT22 rechaza revivir la asignación retirada.
  La solución será preparar una asignación nueva. Dirección autorizó después
  la creación directa de datos ficticios en la copia propia; no se espera al
  kit de Administración para continuar el ensayo local.
- La conexión local con TLS verificado y SCRAM funciona; sin TLS se rechaza.

## Código comprobado

La prueba focal de arranque del shell conserva la sesión común con respuesta
200 y cierra las rutas de Dietas con 503 al faltar su configuración, al activar
un selector incompleto y al configurar parcialmente la cartografía. Esta
prueba no acredita el arranque completo con PostgreSQL de Contratación.

La prueba focal de la vista comprueba recuperación de competencias, un solo
GET por reintento, ausencia de consultas de bandeja antes de acreditar etapa,
denegación y cancelación al desmontar. Pasaron ocho casos de la vista, dos de
textos y diez de montaje y renovación de caché.

## Pendiente

Arranque completo con y sin configuración, comisión desde Chrome con personas
distintas, justificante custodiado, decisiones, liquidación económica, PDF y
Word, repetición de operaciones y recuperación tras reiniciar aplicación y
PostgreSQL. Las comprobaciones anteriores no sustituyen este recorrido.
