# Composición candidata O2-07 del alta de contratación temporal

Última actualización: 24 de julio de 2026.

Estado: **candidata implementada y verificada; pendiente de revisión
independiente**. No autoriza producción ni levanta por sí sola el `NO-GO`
histórico de O2-06.

## Resultado

La composición aislada
`internal/app/composicion/contrataciontemporal` construye la primera vertical
interna de alta con código real:

```text
POST /api/interno/v1/contratacion-temporal/solicitudes
  → autoridad del canal interno
  → contexto de actor
  → flujo gobernado
  → HMAC de huella y ámbito de idempotencia
  → motivo y referencias opacas
  → PDP V3 común de VEC
  → candidatura técnica durable O2-06
  → confirmación PostgreSQL atómica O2-05
  → expediente, actuación, auditoría, outbox y recibo
```

La ruta se registra mediante una lista positiva exacta. Las variantes de ruta
no se aproximan ni se redirigen. El servidor interno permite entregar
`/api/interno` al manejador inyectado; el servidor público continúa
rechazándolo.

## Dependencias obligatorias

`DependenciasAlta` exige doce capacidades ya construidas por sus conectores
propietarios:

1. autoridad del canal;
2. resolución del contexto de actor;
3. resolución del flujo;
4. derivación HMAC de la huella;
5. sellado HMAC del ámbito de idempotencia;
6. catálogo gobernado de motivos;
7. generación de correlaciones;
8. generación de referencias del expediente;
9. autorización ligada PDP V3;
10. material de confirmación VEC-AD-3;
11. reloj;
12. pool PostgreSQL exclusivo de altas.

Una ausencia, incluida una interfaz con puntero nulo, impide construir la
superficie. El error público de composición está redactado; el inventario
tipado de dependencias se conserva únicamente para diagnóstico interno.

La composición no admite DSN, claves HMAC, material criptográfico, cookies ni
cabeceras HTTP de identidad. Tampoco crea implementaciones alternativas de
identidad, autorización o persistencia.

## Acreditación del pool PostgreSQL

Antes de registrar el manejador se consulta la identidad efectiva de la
conexión real. La composición solo acepta una cuenta que:

- coincide con la identidad de sesión y puede iniciar sesión;
- no es superusuario ni posee `BYPASSRLS`, creación de roles, creación de
  bases o replicación;
- pertenece al rol ejecutor y no pertenece a los roles migrador o
  propietario;
- puede usar el esquema, pero no crearlo ni modificarlo;
- puede ejecutar exclusivamente el resolver de candidatura O2-06 y la función
  de confirmación V2;
- no puede ejecutar la confirmación V1, la preparación histórica ni el
  reconciliador interno;
- no puede leer, insertar, actualizar o borrar directamente las tablas de
  identidad, expediente o candidatura técnica.

Una consulta fallida o una sola comprobación negativa produce cierre seguro.
La cancelación del contexto se conserva sin revelar el error de PostgreSQL.

La acreditación no reemplaza la configuración de red, TLS, secretos,
observabilidad ni rotación de credenciales. Demuestra el contrato nominal del
pool que recibe esta vertical.

## Neutralidad de clientes

El caso de uso y la composición no dependen de navegador, cookies ni
almacenamiento web. Web, escritorio, CLI y MCP podrán consumir el mismo
contrato mediante el canal autorizado correspondiente. La autoridad nunca
procede de campos libres enviados por uno de esos clientes.

## Evidencia ejecutada

- unitarias del adaptador PostgreSQL y de composición: 100 repeticiones;
- las mismas pruebas con detector de carreras: 10 repeticiones;
- `go vet` focal y `git diff --check`;
- PostgreSQL 18 efímero, imagen fijada por digest y sin red: 3 de 3;
- rol de ejecución nominal aceptado y cuenta administradora rechazada;
- acceso directo a tablas, funciones históricas y reconciliador: rechazado;
- replay tras pool/proceso nuevos, conflicto de huella, concurrencia,
  cancelación previa a `COMMIT`, respuesta perdida, segunda ambigüedad y
  recibo manipulado: verificados.

El ejecutor reproducible es:

```bash
./deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_07.sh
```

## Límite de este corte

Este corte compone el manejador real, pero deliberadamente no crea valores
ficticios para las dependencias corporativas ni conecta aún
`cmd/vec-interno`. La raíz de proceso definitiva necesita los conectores
homologados de identidad, PDP, HMAC/KMS, flujo y secretos que suministrará la
infraestructura autorizada.

O2-07 tampoco cierra O2-08, O2-09 ni O2-10. Para ello faltan:

1. revisión independiente de la corrección O2-06 y de esta composición;
2. integrar la composición en la raíz interna definitiva sin ampliar el grafo
   del proceso público;
3. conectar el cliente web final al contrato real;
4. ejecutar navegador → API → autorización → PostgreSQL → recibo con reinicio,
   concurrencia y fallos;
5. registrar la aceptación de RRHH y las conformidades operativas.
