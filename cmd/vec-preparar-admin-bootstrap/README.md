# Preparar el plan privado de ADMIN

Este comando prepara o coteja un plan finito de bootstrap. Reutiliza el
formato 2 gobernado: dos cuentas administrativas nominativas distintas y
sus referencias, certificados admitidos, roles, ámbitos, vigencia, fuentes
y huellas. La asignación de Sistemas, cuando figure en el plan, conserva
un perfil separado; no recibe autoridad para administrar usuarios.

La fuente y el plan deben estar fuera de Git, en directorios propios `0700`
y archivos propios `0600`, sin enlaces. Se rechazan campos desconocidos,
duplicados, incompletos o permisos de archivo abiertos. La salida contiene
solo la huella del plan. Repetir el mismo contenido lo coteja; un contenido
distinto no sobrescribe el archivo anterior.

```sh
go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json

go run ./cmd/vec-preparar-admin-bootstrap \
  -fuente /ruta/privada/fuente-admin.json \
  -plan /ruta/privada/plan-admin.json -cotejar
```

`-aplicar` exige cotejo, conexión y aprobación privadas y el destino del
recibo. En este corte valida esos datos y termina con
`provision_no_confirmada`: falta componer el proveedor durable. No abre una
conexión, no asigna perfiles ni genera un recibo de éxito. La preparación o
la aprobación local no acreditan instalación, autorización ni publicación.

La conexión futura debe usar socket local o TLS con verificación del
servidor también en todos sus destinos alternativos. El proveedor deberá
cotejar las fuentes y la huella aprobada y unir CAS, asignaciones, historia,
auditoría y recibo en una transacción. El proceso ADMIN no recibe este canal
privado de operador.
