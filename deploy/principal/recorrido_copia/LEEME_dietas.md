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
los dos actos comunes de la copia ni publican por sí solas un rol. Las cuentas
usan TLS verificado, una sola membresía directa y contraseñas distintas. Una
respuesta SQL incierta impide repetir su alta: se conserva la operación original.

La etapa deja `activada: false` en su resultado. Quedan pendientes la
publicación y asignación de los perfiles por Administración, los manifiestos,
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
  el procedimiento F4b. No se ejecuta el preparador histórico ni se sustituyen
  sus precondiciones por permisos improvisados.

Pruebas focales: quince casos Python positivos sobre conexiones, catálogo y
conductor, incluidos el parser real de las fuentes y su vigencia. Los actos
usan dobles en esas pruebas: no acreditan publicación ni recibos PostgreSQL.
Semgrep local comprueba TLS, ejecución por shell y deserialización en cuatro
archivos Python. El ensayo de esta etapa y el recorrido web siguen pendientes.
No se crea una copia para repetir el bloqueo ya identificado.
