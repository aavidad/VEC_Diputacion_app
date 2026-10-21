# Etapa opcional de Dietas

Tarea: F4-DIETAS-COPIA. Trabajo en curso; no acredita una comisión desde la web.
La copia habitual conserva el recorrido de Bolsa y Contratación sin activar Dietas.

`copia_local.py crear --recorrido-completo --dietas` añade, al terminar la
preparación común, dos admisiones del catálogo y once conexiones privadas.
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
el motor de rutas, el arranque comprobado y la comisión desde el navegador con
recibo y reinicio. No se enciende el selector de Dietas.

## Dependencias comprobadas

- AUT107 se entrega por separado en #1420, con prueba SQL y ensayo. Aún necesita
  las dos revisiones independientes y la integración de dirección.
- El alta de la asignación de Personal para Dietas sigue cerrada: falta el
  catálogo de validadores competentes y su consumo dentro de la transacción.
  Dirección confirmó el 21 de octubre a las 09:58 que ese tramo debe detenerse.
- El manifiesto de comisiones selecciona un perfil de dos dimensiones. La
  consulta de asignación necesita cuatro; no se amplía ese perfil para cubrirla.
- El grupo técnico de asignación necesita la conexión a la base que acredita
  el procedimiento F4b. La etapa concede solo ese permiso de conexión dentro
  de la transacción de alta, conservando las guardas del aprovisionador común.
  No se ejecuta el preparador histórico.

Pruebas focales: veinte casos Python positivos sobre conexiones, catálogo y
conductor, incluidos el parser real de las fuentes y su vigencia. Los actos
usan dobles en esas pruebas: no acreditan publicación ni recibos PostgreSQL.
Las pruebas de publicación usan dobles: comprueban los contratos y que una
recuperación conserva el material sin otra propuesta. Un acto incierto se
detiene aunque la lectura encuentre el rol, hasta conciliar su operación original. No acreditan actos reales.
Semgrep local comprueba TLS, ejecución por shell y deserialización. El ensayo de esta etapa y el recorrido web siguen pendientes.
No se crea una copia para repetir el bloqueo ya identificado.

## Motor de rutas comprobado por separado

El 21 de octubre se consultó la ruta pública del README con OSRM 5.26.0,
imagen oficial fijada por `sha256:af5d4a83fb90086a43b1ae2ca22872e6768766ad5fcbb07a29ff90ec644ee409`.
El PBF local coincide con la huella publicada y el grafo existente se montó
solo para lectura. El contenedor propio, usuario 65534, memoria máxima 1 GiB,
sin capacidades y con puerto ligado a loopback, respondió `Ok`: 9.661 metros
y 923 segundos. Se retiró al terminar. La consulta no usó datos de personas.

Esto acredita el motor local y sus artefactos; no acredita cálculo autorizado,
tarifas, comisión ni recibo de Dietas. Su gestión desde la receta sigue pendiente.
