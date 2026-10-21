# Etapa opcional de Dietas

Tarea: F4-DIETAS-COPIA. Preparación comprobada; la comisión desde la web sigue pendiente.
La copia habitual conserva el recorrido de Bolsa y Contratación sin activar Dietas.

`copia_local.py crear --recorrido-completo --dietas` añade, al terminar la
preparación común, dos admisiones del catálogo y diez conexiones privadas.
Reutiliza el publicador del catálogo, sus comprobaciones de copia, las huellas,
los recibos y su recuperación. Las cinco acciones ya declaradas por Dietas se
reparten entre una fuente de Personal y otra de Dietas; sus ámbitos y campos
son los de AUT107. El material nuevo tiene una vigencia de 90 días.

Las dos admisiones conservan sus recibos y su consulta cotejada. No sustituyen
los dos actos comunes de la copia. La etapa publica después los dos roles por
propuesta y aprobación de dos administradores, usando el ejecutor existente.
Antes de cualquier efecto exige los cuerpos exactos de AUT107. Las cuentas
usan TLS verificado, una sola membresía directa y contraseñas distintas. Una
respuesta SQL incierta impide repetir su alta: se conserva la operación original.

La etapa deja `activada: false` en su resultado. Quedan pendientes la
asignación de los perfiles por Administración, los manifiestos,
el motor de rutas integrado y la comisión desde el navegador con recibo y reinicio.
La aplicación arranca y reinicia con esta preparación y Dietas desactivado. No se enciende el selector de Dietas.

## Dependencias comprobadas

- AUT107 (#1420) y el adaptador de roles compuestos (#1429) están integrados.
  La receta requiere ambos; no instala migraciones adicionales.
- El alta de la asignación de Personal para Dietas sigue cerrada: falta el
  catálogo de validadores competentes y su consumo dentro de la transacción.
  Dirección confirmó el 21 de octubre a las 09:58 que ese tramo debe detenerse.
- El manifiesto de comisiones selecciona un perfil de dos dimensiones. La
  consulta de asignación necesita cuatro; no se amplía ese perfil para cubrirla.
- La conexión del ejecutor de asignación de Personal y su permiso de conexión
  a la base se aplazan con ese tramo, por la revisión de dirección del 21 de
  octubre a las 15:05. No se ejecuta el preparador histórico.
- Un error SQL confirmado por psql (salida 3) informa de la reversión de la
  transacción y remite a `dietas-conexiones.log` antes de conciliar. Una pérdida
  de conexión conserva el resultado como incierto; nunca afirma una reversión.

## Primer ensayo y corrección de revisión

El 21 de octubre, la cabeza `0477a5ab8`, con main `3796cf010` incorporada,
completó la versión anterior de `crear --recorrido-completo --dietas` en
**275,82 segundos**, con
salida 0 y material nuevo. No se reutilizó la copia del intento anterior.
El ejecutable GrxFirma se declaró con su huella mediante las opciones de la
[receta común](LEEME.md); no se descargó software ni se cambió el instalado.

- Dos admisiones nominales, con recibos reales, repetición con el mismo recibo
  y consulta cotejada del catálogo.
- Dos roles publicados por propuesta y aprobación de administradores distintos,
  con sus contratos de tres y dos concesiones, respectivamente.
- Once cuentas separadas: las once conexiones respondieron por TLS con
  `verify-full`; `pg_stat_ssl` confirmó el cifrado de cada conexión.
- `copia_local.py reiniciar` y `comprobar` terminaron con salida 0. Tras el
  reinicio de aplicación y PostgreSQL se repitieron los cotejos de roles y las
  once conexiones, conservando el resultado y sin nuevas propuestas ni altas.
- La copia se borró con la fábrica, salida 0. Los recibos y registros quedan
  en la evidencia privada del equipo; no se incorporan conexiones a Git.

Las pruebas Python focales de Dietas, catálogo y conductor pasan.
La revisión posterior aplaza la conexión de asignación de Personal: quedan
diez conexiones. Su nuevo ensayo está pendiente.
Cubren conexiones, recuperación, contratos y la parada de actos inciertos,
aunque una lectura encuentre el rol. Semgrep local: tres reglas sobre los
cinco archivos Python modificados, sin hallazgos. La revisión focal de
seguridad conserva separación de cuentas, TLS y actos de dos administradores;
no se han ampliado permisos ni abierto el catálogo de validadores de Personal.

Este ensayo acredita la preparación y su persistencia. Dietas conserva
`activada: false`; no acredita manifiestos, asignaciones, comisión, cálculo
autorizado ni recibo de comisión desde Chrome. Ese recorrido depende del
tramo de Personal detenido por dirección.

## Motor de rutas comprobado por separado

El 21 de octubre se consultó la ruta pública del README con OSRM 5.26.0,
imagen oficial fijada por `sha256:af5d4a83fb90086a43b1ae2ca22872e6768766ad5fcbb07a29ff90ec644ee409`.
El PBF local coincide con la huella publicada y el grafo existente se montó
solo para lectura. El contenedor propio, usuario 65534, memoria máxima 1 GiB,
sin capacidades y con puerto ligado a loopback, respondió `Ok`: 9.661 metros
y 923 segundos. Se retiró al terminar. La consulta no usó datos de personas.

Esto acredita el motor local y sus artefactos; no acredita cálculo autorizado,
tarifas, comisión ni recibo de Dietas. Su gestión desde la receta sigue pendiente.
