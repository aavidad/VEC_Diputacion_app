# Censo de los cargos de firma

F1-118 corrige la admisión del catálogo después de publicar un cargo con
`vec-publicar-cargos-firma`. El documento SQL del cargo omite `retirada_en`;
la representación canónica de Go incluye `0001-01-01T00:00:00Z`. El cotejo
exacto de AUT58 y AUT61 los trataba como documentos distintos.

AUT106 reconstruye únicamente
`vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(jsonb)`, desde la
definición instalada en la copia de main. Completa ese instante solo para un
rol con estado `publicada` que no contiene la clave. Un instante explícito,
incluso nulo, se conserva. Los roles retirados y los documentos sin estado no
reciben ese campo. El normalizador sigue quitando únicamente las listas
vacías de campos y obligaciones que ya quitaba.

La corrección protege el cotejo completo: no elimina concesiones, campos
extraños, retiradas ni controles. La prueba SQL contrasta las versiones de rol
almacenadas con la normalización anterior y comprueba los casos negativos.
No cambia el publicador de cargos, las versiones o huellas guardadas, los
permisos ni las asignaciones de las personas.

## Activación

CONFIG NUEVA: ninguna. No hay selectores ni ficheros de arranque nuevos.

1. Crear una copia propia con la orden completa de
   `LEEME_copia_local.md`, incluidos `--modulos`, `--rrhh-original` y los
   preparadores privados propios. Instalar la lista única de main en su orden.
2. Aplicar AUT106 una sola vez, al final de
   `lista_sql_main_tras_hz37.txt`. Su guarda exige PostgreSQL 18 y la huella,
   el propietario y los privilegios exactos de la función instalada.
3. Ejecutar la prueba
   `deploy/postgresql/autorizacion/pruebas_sql/000106_canon_retirada_cargos_firma.sql`.
4. Publicar un cargo ficticio mediante la herramienta existente y preparar
   el censo completo desde Go. Presentar su plan aprobado por el circuito
   técnico de copia, con un LOGIN exclusivo y la configuración protegida.
   Conservar el plan y el recibo; repetir la misma operación debe recuperar
   el original. Ningún dato de copia acredita una aprobación personal real.
5. Reiniciar la aplicación y PostgreSQL propios. Consultar la admisión exacta
   y repetir la misma operación; cotejar recibo, fecha y ausencia de duplicados.
   Comprobar también el arranque sin AUT106: el portal sigue atendiendo y la
   admisión del cargo reproduce el rechazo de censo previo.

No reaplicar una migración instalada ni ejecutar DOWN. Ante una guarda
incompatible, conservar el registro y revisar la definición real. No
reescribir los roles para sortear el cotejo.

## Comprobación de esta candidata

Pendiente de ensayo y de las dos revisiones independientes. Esta guía no
acredita todavía una admisión, una asignación ni una firma desde la ficha.
El consumidor web sigue siendo la firma del expediente de Contratación
Temporal; la corrección permite su preparación administrativa.
