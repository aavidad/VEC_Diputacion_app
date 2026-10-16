# Modalidad de inscripción en la versión certificada de la convocatoria

F1-40 incorpora `modalidad_inscripcion` al contenido sellado de la convocatoria.
La lista y la ficha de la aspirante muestran si corresponde una solicitud por
categoría o una conjunta. Una convocatoria conjunta no ofrece el formulario
individual; el servidor devuelve `409 inscripcion_conjunta_pendiente` antes del
primer INSERT si alguien intenta ese acto. La inscripción conjunta completa
corresponde a F1-41.

La modalidad de ensayo entra **solo como dato de la copia** mediante la carga
de una versión certificada preparada desde su fuente JSON. No hay elección ni
guardado productivo de RRHH en Preparación de bases en este corte. Adaptar esa
función al núcleo vigente es una tarea posterior a AD288. F1-40 permanece
abierta: `ConfirmarPublicacion` sigue sin adaptador productivo. Esta carga no
acredita aprobación o publicación de bases.

## Activación en la copia

No hay variable, ruta ni autoridad nuevas. BC12 reconstruye tres funciones
existentes después de BC9 y B96, con las prehuellas y poshuellas declaradas en
su migración. `lista_sql_codexb_f1_40_20261015.txt` fija el orden. Antes de
instalar, comprobar las dependencias y la definición instalada en PostgreSQL 18;
no reaplicar una migración presente ni usar DOWN. La prueba focal SQL revierte
sus filas: legado sin campo, modalidad separada, conjunta visible, y valores
desconocido o nulo rechazados. Cotejar OID, propietario, ACL, configuración,
volatilidad y paralelismo de las tres funciones.

La copia de ensayo debe contener el prefijo completo de autorización externa.
Si incluye las 28 audiencias de #1147, preparar el material con su publicador
antes de arrancar esta aplicación. En el ensayo se usa el productor congelado
`da8d6de32`, SHA-256
`db10f97ac2b453b096c154c0c7b398d3a15b6c8f6156a1ad42deba144a2cc59c`,
solo para `preparar-portal-externo --consumidores mi_bolsa,inscripcion_bolsa`
y con material propio de la copia. #1147 es una dependencia de activación.
La preparación del material y la aplicación de este corte se prueban juntas.

Para ese recorrido, seleccionar el proceso externo con `VEC_PORTAL_PROCESO=externo`,
su material privado y sus conexiones nominales con TLS verificado; activar
`VEC_BOLSA_INSCRIPCIONES_ENABLED`. Mantener vivo el sellador nativo con su
conexión separada en el proceso interno de la misma copia. Para comprobar la
ausencia de configuración, apagar inscripción y retirar sus conexiones:
la página y `/livez` deben seguir respondiendo 200 y la ruta quedar cerrada.

Hasta la corrección de vigencias de F1-64, dirección autoriza este material
solo para el ensayo local, con responsable B y retirada al terminar, en un
máximo de 24 horas. Las nuevas asignaciones y credenciales de la copia duran
89 días. La excepción del material se registra por separado.

Para preparar el dato, reutilizar
`deploy/principal/inscripcion_recorrido_20261013/version_certificada_v2/generar.py`
con `fuente_v2.json`. Generar SQL y manifiesto fuera de Git, comprobar las
preimágenes y aplicar el resultado solo a la copia de ensayo. Es una carga de
datos ficticios; no sustituye el circuito de gobierno de las bases. No instalar
AD153 ni BC8 históricos sobre el núcleo actual para activar esta pantalla.

Lista y detalle de convocatoria añaden `modalidad_inscripcion` a su JSON. El
filtro de lectura exige todas las hojas concedidas: una concesión anterior sin
ellas se deniega, sin recortar la respuesta. La ampliación gobernada del rol
externo no forma parte de F1-40. En la copia, el dato ficticio de acceso se
prepara por separado con el guion privado ya revisado; no se atribuye a ese
dato una aprobación, un cierre o un recibo de dos administradores.

## Comprobación y límites

- Abrir en Chrome la lista y la ficha de la aspirante en castellano e inglés:
  ambas muestran la modalidad. La conjunta explica el límite antes de mostrar
  cualquier formulario y no emite una solicitud individual.
- Comprobar que la API devuelve 409 para una presentación individual conjunta
  y que no se crea solicitud ni recibo parcial. Verificar el caso separado
  conservado y la lectura del legado sin alterar bytes ni huellas.
- Recuperar las lecturas de la misma convocatoria después de reiniciar
  aplicación y PostgreSQL. Arrancar también sin la carga opcional: la función
  ausente falla localmente y la aplicación sigue disponible.

La preparación de bases ya existente conserva su lectura anterior; esta PR no
añade su guardado ni prueba un recibo de preparación. El ensayo de la candidata
completa y su cabeza exacta se consignan en la PR. Esta guía no atribuye
pruebas de runtime que aún no se hayan realizado.
