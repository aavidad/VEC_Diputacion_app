# Candidata HZ39

Versión candidata en preparación. Se ensaya entera en una copia
de cidonia y, si sale bien, se instala en cidonia exactamente igual (regla 1 de
`AGENTS.md`, «Entrega y subida»). Caso de aceptación: `docs/producto/FLUJO_APP.md`,
apartado 4.0.

- **Base de partida:** lo instalado en cidonia (HZ37 + web HZ38, código
  `44218865a`; catálogo de acciones v6).
- **Código de la candidata:** el commit de `origin/main` que se fije al congelar
  (se anota aquí antes de instalar).
- **Nada se instala desde fuera de esta carpeta**: si falta algo, se añade aquí
  en una PR y se vuelve a ensayar.

## Orden de instalación

1. **SQL**: `lista_sql.txt`, en ese orden, cada una una vez y con
   `ON_ERROR_STOP`. Comprobar el SHA-256 de cada fichero antes de aplicarlo.
   La lista incluye AD266, después de AD265; AUT235 ya forma parte de HZ37.
   Después se aplican las ocho SQL de `personal/lista_sql.txt`, en su orden.
   Al restaurar la copia, aplicar también las ACL de tipos y las ACL de la
   base entregadas por dirección. El dump de datos no conserva las ACL de
   la base: sin ellas faltan los permisos nominales de conexión y vuelve el
   permiso temporal de PUBLIC. No se reconstruyen a mano.
2. **Datos del recorrido**: `datos/` (guion repetible de K: centro 520, puesto
   9/01, plaza 2026/0033, bolsa de Administrativo).
3. **Entrada del aspirante**: Mi Bolsa interna (HZ37) para el recorrido
   principal, con Antonio disponible en la bolsa vigente. Los pasos e1–e5 son
   opcionales y se enseñan aparte. La inscripción RRHH queda apagada
   (`VEC_BOLSA_INSCRIPCIONES_ENABLED=false`), como en HZ37; su lector nominal
   se resolverá después del ensayo principal. La confianza externa (#1120, `deploy/principal/confianza_externa_hz38.md`) solo si su
   ensayo valida el acceso y el recorrido; decisión de corte el 13/10 a las
   17:00 y se anota aquí la vía elegida.
4. **Actos de administración**, por la vía de gobierno existente y con dos
   administradores distintos:
   - inscripción RRHH (lectora técnica Elena Ruiz Martín) cuando la vía del paso 3 lo
     permita, con reinicio vigilado;
   - plan de firma v2 de cinco pasos (`scripts/operacion/firma_ct_tres_pasos/`):
     crear y publicar; después `identidad/firma-vec.json` con los valores
     resultantes y reinicio.
5. **Arranque vigilado** con vuelta atrás. La firma nominal necesita también
   `VEC_CT_CIRCUITO_RRHH_SOURCE_PATH`: copiar el catálogo existente
   `config/contratacion_temporal_circuito_rrhh_v2.json` al material de
   incorporación y configurar su ruta de lectura. No confundirlo con el
   catálogo de reglas de firma. Conservar ambos durante el arranque.
6. **Recorrido** con el guion de P (`deploy/principal/ensayo_candidata/`):
   los pasos principales del caso, reinicio de app y PostgreSQL y repetición
   de lecturas; mismos recibos y sin duplicados.

## Estado del ensayo

| Pieza | Estado |
| --- | --- |
| SQL (20) | Once SQL iniciales, AD266 y ocho SQL de Personal aplicadas una vez, sin error; ACL de tipos y base restauradas |
| Datos del recorrido | Recetas #1135 aplicadas: plantilla existente, Antonio disponible y primer turno; manifiesto con la nueva referencia de candidato |
| Vía del aspirante | Mi Bolsa interna; inscripción RRHH opcional apagada por dirección |
| Plan de firma y cinco firmas | Plan de cinco pasos creado y publicado por dos administradores, con recibos reales. Catorce actos nominales confirmados. Las cinco firmas siguen pendientes |
| Arranque | Mi Bolsa activa e inscripción RRHH apagada. La firma nominal encuentra un rechazo del lector del plan; vuelta atrás comprobada con respuesta de salud 200 |
| Recorrido completo y reinicio | Pasos 1–7 con respuestas y recibos reales; informe preparado, firma 7b pendiente. Falta el recorrido final en copia limpia |
| Administración propia | Fuentes, unidad y arranque con tres recibos propios. Gobierno confirmado; comprobación de cadena pendiente (`manifiesto_checkpoint_limite`) |
| Mi Bolsa | Chrome muestra Administrativo, Disponible y orden inicial 1 de 12; lectura e historial responden 200. Preferencias devuelve 404 y se ha comunicado a su dueño |

Lo que no se haya ensayado en la copia no se instala en cidonia.

El ensayo se retomó el 14/10 desde el respaldo HZ37, en una copia nueva.
La restauración terminó antes de aplicar la lista; un intento previo iniciado
antes de terminar la restauración se descartó entero. Se conservaron las
conexiones separadas, TLS y las comprobaciones de permisos. Las conexiones y
la escucha se trasladaron a los puertos de la copia, accesibles solo desde
loopback, y se preparó el directorio privado del almacén documental.

Chrome del sistema abre la pantalla del centro sin errores de JavaScript ni
de red. El transporte HTTPS conserva el certificado del actor, verifica la
CA y transmite la respuesta real sin añadir la cabecera `Connection` que la
frontera rechaza. La búsqueda espera
el código RPT `9`; `9/01` es el código del puesto individual. El formulario
también exige plaza y datos presupuestarios, ya incluidos en el guion. El
centro presenta sin retención de crédito y RRHH la recoge en el análisis,
por decisión de dirección del 14/10. La petición presentada y ratificada originó
un único expediente, con referencia MOAD `2026/39014`. Se registraron análisis,
vía Bolsa, asignación de unidad e informe jurídico mediante la pantalla y sus
confirmaciones. El informe necesita su firma antes del tramo siguiente.

La copia incorpora el código de GINPIX de la principal y el guion conserva
el orden 17, 18, 16a y 16b aprobado por dirección. La versión definitiva
se fijará al congelar. La receta `mi_bolsa/` corrige
el ámbito ficticio de Antonio exclusivamente en la copia P; no se instala en
cidonia. El ensayo final aplicará los datos antes de comenzar los actos del
recorrido y conservará los recibos que produzca.

La configuración nominal, las cuatro admisiones, el plan y la ampliación de
conexiones de auditoría tienen revisión independiente. El verificador de la
copia responde por HTTPS con su CA y token propios. El arranque nominal
localizó un requisito ausente del circuito RRHH y, después de añadirlo, un
rechazo del lector del plan: su detector de teléfonos coincide dentro de
una referencia opaca de cargo. Se ha comunicado a H; los actos y recibos
publicados se conservan. La app vuelve a arrancar sin la firma nominal.

El plan nominal ya se ha publicado. Siguen pendientes la activación de firma,
la congelación del código, el tramo Personal/cese y la recuperación completa
tras reiniciar. El arranque y
las lecturas iniciales no acreditan esos pasos.
Durante la preparación se incorporan correcciones y se continúa. La prueba
que cuenta se ejecuta una vez al final, sobre una copia restaurada limpia,
con el commit congelado, el recorrido entero y el reinicio.
