# Material privado de Organización histórica

El GET integrado conserva su publicación y su autorización nominal. Este
preparador deriva únicamente la clave de OH, comprueba el gobierno y el motivo
existentes, y crea una salida nueva. No publica claves, perfiles ni permisos.

1. Publicar OH con el `vec-server` existente y su selector
   `VEC_ORGANIZACION_HISTORICA_GOBIERNO_ENABLED=true`, por el circuito aprobado.
   No necesita el inventario OH: el selector sólo añade el descriptor propio
   al catálogo de material de `vec-server`, que publica la audiencia antes
   del montaje de `vec-interno`. Conservar su material de idempotencia y el
   resto de su entorno aprobado. Arrancar o reiniciar ese publicador, comprobar
   que la fila OH y su puntero están vigentes y preparar después la salida del
   paso 3. `vec-interno` conserva OH sin montar mientras no exista el inventario.
2. Proporcionar `organizacion_historica_v3.json` privado con versión 1, catálogo,
   motivo y contextos admitidos. Cada cuenta conserva perfil, versión,
   organismo y unidad. La capacidad de entrada sólo necesita su ruta local;
   se sustituye por las coordenadas reales de la fila publicada.
3. Como el usuario del servicio, preparar en un directorio nuevo fuera de Git:

```bash
vec-preparar-material-interno \
  -inventario-ct "$MATERIAL_INTERNO/ct_v3.json" \
  -material-idempotencia "$MATERIAL_VEC_SERVER/idempotencia" \
  -organizacion-historica-config "$ENTRADA_OH/organizacion_historica_v3.json" \
  -dsn-archivo "$DSN_GOBIERNO_ARCHIVO" \
  -salida "$SALIDA_NUEVA"
```

El fichero DSN y las entradas son privados (0600); los directorios son 0700.
La salida contiene sólo `organizacion_historica_v3.json` y
`organizacion_historica_consulta.hmac`. La clave, su versión, su revisión,
huellas, emisor, ventana y puntero se cotejan con el gobierno vigente en
una instantánea de solo lectura. La referencia de motivo se valida por su
lector nominal existente. El inventario se comprueba con
`CargarMaterialOrganizacionHistorica` y los constructores reales antes del rename.

Los contextos se conservan exactamente como selecciones privadas admitidas.
Se valida su estructura; el preparador no acredita una resolución F1 viva,
no registra contextos y no convierte una fuente externa en propia. El GET
revalida identidad, perfil, versión, organismo y unidad mediante F1, PDP y
el consumo transaccional de Personal en cada acceso. Una salida preparada
no acredita permiso ni una consulta recorrida.

Revisar la salida e instalar sus dos archivos por el procedimiento privado
existente, con el inventario al final. Conservar el resto del material CT y
la clave OH anterior. No hay renovación automática ni cambio de interfaz.
Tras instalar, comprobar el GET real y su evidencia con una identidad y una
fuente admitidas; una fixture sintética no acredita datos corporativos.
