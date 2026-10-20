# Lectura puntual de la cola de sellado (F1-105)

El cuadro de Contratación con resumen puede fallar al abrirse en varias
sesiones a la vez. La revalidación lee una secuencia de la cola de sellado,
pero PostgreSQL elige recorrer toda la tabla cuando es pequeña. Esa lectura
abarca entradas de otras peticiones y provoca conflictos `40001`.

La cola ya tiene una clave primaria B-tree por `secuencia`, creada en AD207.
AD303 fija `enable_seqscan=off` solo en
`revalidar_evidencia_consumo_consulta_rrhh_v3_interna`, para preferir ese
índice. No crea uno duplicado ni modifica el cuerpo de la función, las
comprobaciones de identidad y permisos o la auditoría. La configuración de
la sesión se recupera al salir de la función, también con error.

## Instalación

CONFIG NUEVA: ninguna. Instalar una vez
`autorizacion_atestada_v3/migraciones/000303_revalidacion_lectura_indice_sellado.up.sql`,
al final de `deploy/principal/lista_sql_main_tras_hz37.txt`. Solo UP, sin DOWN
ni reaplicar AD207. La guarda exige PostgreSQL 18, la función con su dueño,
permisos y configuración exactos y la clave primaria válida de la cola.

La huella previa es `ad81322f33bbd1640230f33567700376154eb595072a72d74da2fc8239fe7580`;
la final, `266950eb649d62b73efaa124bb72a717ddbb810e2aae761b664dfc1358f074a1`.
No requiere cambiar el entorno ni reiniciar para que PostgreSQL use la
configuración; el ensayo incluye el reinicio para comprobar que persiste.
Si una guarda falla, no instalar otra vez ni quitarla: cotejar la versión
instalada y preparar una migración prospectiva.

## Comprobación

1. Crear una copia propia con la orden completa de `LEEME_copia_local.md`,
   módulos, identidad original de RRHH y lista de la candidata.
2. Ejecutar `pruebas_sql/ad303_revalidacion_indice_sellado.sql`: huella,
   permisos, configuración, rechazo de un perfil ajeno, restauración de la
   configuración tras ese rechazo y plan con la clave primaria de la cola.
3. Abrir el cuadro de Contratación en Chrome y capturar su consulta POST
   con `resumen: true` y límite 100. Usar el cliente existente de
   `scripts/carga/cliente` con CA y certificado de la copia, tres
   calentamientos, diez clientes y cincuenta muestras por cliente.
   Conservar recuento de respuestas y errores `40001`; medir la latencia sin
   confundir ausencia de errores con cumplimiento del objetivo de 300 ms.
4. Reiniciar aplicación y PostgreSQL. Comprobar que la función conserva su
   huella y recuperar el cuadro desde Chrome.

El caso de rendimiento consume la pantalla existente. No añade rutas,
concesiones ni controles nuevos de negocio. La prueba de plan protege la
lectura puntual; la de configuración comprueba que el ajuste no se extiende
al resto de la sesión. No arregla el cambio de perfil entre el cuadro y el
detalle, que corresponde a F1-114.
