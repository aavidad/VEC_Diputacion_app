# Incorporación B2

Kit para Claude. Se prepara sobre main con #334 y #335 integradas. No instala
migraciones ni cambia permisos. La preparación produce una configuración nueva;
el servicio vuelve a comprobar permisos, motivos y gobierno al arrancar.

1. Ensayar en un clon desechable de main. Conservar el acta, el hash de fuente,
   las versiones publicadas y las comprobaciones antes y después del reinicio.
   Las 72 dependencias H6 ya instaladas se conservan; no repetirlas ni hacer DOWN.
2. Obtener del circuito privado existente la configuración B2 pura aprobada:
   tres DSN raíz, cinco DSN B2 de LOGIN distintos, las referencias nominales y
   la correspondencia explícita de los 22 motivos. El motivo de detalle debe
   coincidir con el resolutor RRHH. La candidata de D con huellas `a…a`, claves
   locales y versiones provisionales sirve de inventario, no de entrada de activación.
3. Arrancar el `vec-server` de main por su lanzador existente con esa configuración
   y `VEC_PERSONAL_B2_GOBIERNO_ENABLED=true`. La selección comprueba el contrato
   antes de publicar; no exige todavía ficheros de capacidad activables. El
   publicador existente publica las nueve audiencias de incorporación y las ocho
   de Personal. Un fallo posterior de montaje puede dejar claves publicadas;
   comprobar el gobierno completo antes de seguir. El lote no es una transacción única.
4. Preparar en un directorio nuevo fuera de Git, como el usuario del servicio:

```bash
bash preparar.sh "$FUENTE" "$MATERIAL_INTERNO/ct_v3.json" \
  "$MATERIAL_VEC_SERVER/idempotencia" "$CONFIG_B2_APROBADA" \
  "$DSN_GOBIERNO_ARCHIVO" "$SALIDA_NUEVA"
```

El DSN de gobierno y los ficheros de entrada tienen modo 0600. La herramienta
reutiliza la derivación del publicador y coteja 15 audiencias para 22 operaciones.
Toma versión y revisión de las filas vigentes; comprueba huellas, emisor, ventana,
puntero, revocación y raíz en una instantánea de gobierno `REPEATABLE READ READ ONLY`.
Los motivos se consultan por su autoridad nominal. La resolución exacta de detalle
reutiliza la transacción `SERIALIZABLE READ WRITE` del resolutor RRHH, con sus
bloqueos y reacreditación; no publica motivos, perfiles ni concesiones.

La salida contiene `servidor.json`, ocho DSN privados y 22 capacidades. Se valida
con el cargador y los constructores de runtime antes de activar el directorio
con un único rename. Conserva las referencias y motivos originales. No sobrescribe
la entrada ni el material actual del servicio.

5. Revisar la salida y preparar el entorno del lanzador:

```bash
VEC_CT_INCORPORACION_V2_FILE="$SALIDA_NUEVA/servidor.json"
VEC_CT_INCORPORACION_ACREDITADA_ENABLED=true
VEC_PERSONAL_B2_GOBIERNO_ENABLED=true
```

Mantener las otras variables privadas y el selector Bolsa que exige el circuito
existente. Si el arranque exige provisión, usar sus aprobaciones y preimágenes
reales mediante el CAS existente; retirar esas variables tras provisionar.
Un perfil revocado se conserva revocado. No fabricar una preimagen ni un permiso.

6. Comprobar el montaje por el contexto RRHH existente, el recibo de la operación
   sintética y su recuperación tras reiniciar aplicación y PostgreSQL. Conservar
   los mismos recibos, fechas, versiones e historia, sin otra alta. Comprobar
   también rechazo sin permiso y motivo distinto. Publicar en principal únicamente
   tras las dos revisiones independientes y el ensayo completo.

La preparación local de material y la publicación de claves no acreditan por sí
solas una incorporación, firma legal, entrega ni puesta en producción. La ejecución
en principal corresponde a Claude; este kit no conecta con cidonia.
