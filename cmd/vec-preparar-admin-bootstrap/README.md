# Preparar y aplicar el alta de los dos primeros administradores

El comando existente prepara un plan privado y muestra su huella SHA-256. La
opción `-aplicar` usa el proveedor nominal de AUT24 por un canal de operador
separado. No existe una ruta HTTP de bootstrap ni se presta el permiso del
operador a la aplicación ADMIN.

El formato 2 conserva las dos personas de F y añade `gobierno`. El plan reúne
las referencias, versiones y huellas de cuentas, personas, certificados, CA,
procedencia y rol administrativo publicado. Contiene también la revisión de
continuidad esperada y la caducidad. Las personas deben ser distintas y estar
ordenadas por referencia. No incluye nombres, DNI, PEM ni claves.

`gobierno` contiene la audiencia administrativa admitida, la referencia y
huella de la política de certificado, y una lista finita de roles publicados.
Cada entrada fija referencia y huella del rol, clase de control, necesidad de
unidad, ámbitos permitidos, vigencia y duración de las propuestas. No admite
comodines ni duplicados; ADMIN debe tener ámbitos explícitos. Un descriptor
que requiere unidad puede dejar vacía su lista de ámbitos fijos: al asignarlo,
la autoridad deberá comprobar la unidad y que pertenece al ámbito del
administrador. Esa posibilidad no concede un alcance global.

Todos esos valores forman parte de la huella aprobada. No hay valores de
negocio por defecto. Un plan del formato 1 no se convierte automáticamente ni
permite aplicar altas: debe prepararse el formato 2 con su configuración
expresa. Los roles y huellas se cotejan con sus fuentes actuales en SQL; el
archivo no acredita por sí mismo una cuenta, certificado o permiso.

Fuente, plan, conexión, aprobación y recibo viven fuera de Git, en directorios
propios `0700` y ficheros `0600`, sin enlaces. Se exigen campos exactos y únicos.

```sh
go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json

go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json -cotejar
```

El operador coteja las fuentes y aprueba la huella exacta por su canal privado.
La aprobación es un JSON con `huella_plan_sha256`. La conexión privada contiene
`dsn` y `timeout_segundos`; el plazo de conexión debe estar indicado, entre 1 y
60 segundos. Se admite socket local o TLS con verificación del servidor. Nunca
se pasan credenciales por argumentos ni se muestran errores de PostgreSQL.

```sh
go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json -cotejar -aplicar \
  -conexion /ruta/privada/conexion.json \
  -aprobacion /ruta/privada/aprobacion.json \
  -recibo /ruta/privada/recibo-admin.json
```

El LOGIN del operador pertenece únicamente a
`vec_admin_perfiles_bootstrap_ejecutor`, con herencia, sin `SET ROLE` ni
administración de miembros. Carece de propiedad y acceso directo a los datos;
solo ejecuta la función nominal de bootstrap. Es distinto del LOGIN de la
aplicación. La función debe cotejar la aprobación y las dos identidades, hacer
el CAS y guardar perfiles, configuración, historia, auditoría y recibo en una
transacción SERIALIZABLE. El comando valida el recibo antes del COMMIT.

Repetir el mismo plan aprobado recupera el mismo recibo según el contrato SQL.
Otro contenido no reescribe el plan ni el recibo. Si falla la escritura del
recibo local después del COMMIT, se informa `recibo_no_guardado`: no se afirma
que el alta haya fallado. Se recupera con el mismo plan y aprobación.

La implementación Go está preparada contra la función AUT24 en borrador.
La aplicación real exige instalar y ensayar esa fuente por el canal autorizado,
con sus dependencias de identidad, contexto y autorización. Las pruebas con
dobles verifican el transporte y el COMMIT; no acreditan altas en PostgreSQL.
