# Auditoría de la lectura de inscripción del aspirante externo

Dependencia de F1-40 (#1177). Al leer su inscripción, el aspirante del portal
externo recibía un 503: la función que deja constancia de la lectura
(`registrar_lectura_inscripcion_v1`, AD228) buscaba el contexto del actor solo
en el almacén interno (`registros_contexto`), y desde CA15 el portal externo lo
guarda en `registros_contexto_externo_v2`. PostgreSQL respondía P0002.

## Qué cambia

- CA47 añade `cotejar_contexto_historico_portal_externo_v1`. Si el recibo
  está en el almacén externo, lo reconstruye desde su provisión versionada y
  exige los mismos bytes, huellas y vigencias en el instante original. Si está
  en el interno, solo lo acepta cuando la cuenta no tiene población externa
  segregada, y entonces delega en el cotejo de siempre (CA26/CA27). Un recibo
  en los dos almacenes se rechaza. Solo puede ejecutarla el propietario de AD3.
- AD289 reconstruye únicamente `registrar_lectura_inscripcion_v1` desde su
  cuerpo instalado. Con canal `externa_personal` usa el cotejo nuevo; con
  `interna_corporativa` (RRHH) sigue con el anterior. Conserva propietario,
  permisos, configuración y firma. No toca el núcleo ni otras lecturas.

## Activación en una copia

1. Comprobar que la copia tiene CA15, CA17, CA27 y AD228 instaladas.
2. Aplicar como superusuario, en este orden y una sola vez, la lista
   `lista_sql_claude_auditoria_inscripcion_externa_20261016.txt`. Cada
   migración se detiene con un error claro si la preimagen no coincide o si ya
   está aplicada.
3. Ensayo SQL (todo se revierte al final):

   ```sh
   psql -X -v AD289_DISPOSABLE_CLONE=1 -f deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad289_lectura_inscripcion_externa_pg18.sql
   ```

   Debe terminar en `AD289-PRUEBA-OK`. Con `-v AD289_REPRODUCIR=1` antes de
   aplicar la lista muestra el P0002 original.
4. No hace falta reiniciar la aplicación ni cambiar configuración. Después,
   con el aspirante sintético, abrir en el portal externo la consulta de sus inscripciones:
   debe cargar y quedar un asiento `lectura_inscripcion` con canal
   `externa_personal`.

No hay retirada automática. Si hiciera falta volver atrás, dirección decide
una migración nueva; no se aplica ningún DOWN sobre historia.
