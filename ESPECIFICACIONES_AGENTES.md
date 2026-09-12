# Especificaciones obligatorias para agentes VEC

Documento solicitado expresamente por Alberto el 12 de septiembre de 2026.
Obliga a dirección, programadores, revisores, documentación e integración,
incluidos todos los subagentes locales y remotos. Es un índice operativo de
requisitos existentes; no sustituye sus especificaciones detalladas ni acredita
que estén implementados. Su creación está autorizada expresamente pese a la
moratoria anterior de documentos de gobernanza.

## E01. Fuentes y precedencia

Leer este fichero y las instrucciones aplicables antes de trabajar. Las órdenes
vigentes del operador y `INSTRUCCIONES_DESATASCO.md` prevalecen sobre decisiones
históricas contradictorias. Comprobar la ubicación y rama activas en el seguimiento
vigente; los bloques antiguos no autorizan reiniciar entornos retirados.

Consultar los apartados afectados de estas fuentes, sin sustituirlos por este resumen:

- [Flujo de RRHH](docs/portal_vec/expediente_contratacion_temporal_rrhh.md).
- [Arquitectura](docs/portal_vec/arquitectura_tecnica.md).
- [Roles y ámbitos](docs/portal_vec/matriz_roles_y_ambitos.md).
- [Cumplimiento, seguridad y auditoría](docs/portal_vec/cumplimiento_y_seguridad.md).
- [Acceso interno y administración](docs/estudio_requisitos/acceso_interno_tecnicos_administracion.md).
- [Persistencia PostgreSQL](docs/portal_vec/seguridad_persistencia_postgresql.md).
- [Matriz normativa CT](docs/portal_vec/matriz_normativa_contratacion_temporal_2026-07-23.md).
- [Instrucciones del repositorio](AGENTS.md), incluidos sus contratos específicos.

No editar el Word original de RRHH. Ante una ambigüedad funcional o jurídica,
registrarla en el seguimiento existente y continuar las partes independientes;
no convertir una suposición o una prueba sintética en aprobación del operador.

## E02. Fidelidad a RRHH y propiedad de los datos

Implementar el recorrido completo definido por RRHH: solicitud del centro,
análisis y RC, cobertura, unidad, informe, fiscalización, llamamiento,
formalización y firma, incorporación, GINPIX, seguimiento y cierre, conservando
las dependencias de la definición vigente. No equiparar pantallas con fases ni
alterar la ambigüedad documentada de ocho fases sin validación.

Contratación coordina por puertos, referencias opacas y eventos. Bolsa conserva
orden, disponibilidad y llamamientos; Personal conserva relaciones jurídicas,
ocupaciones e incorporación. Ningún módulo consulta ni escribe tablas de otro.
Los catálogos de modalidades, causas, rutas, unidades, documentos y reglas son
gobernados y versionados. El expediente conserva su definición, versión, huella
e historia. Ofrecer las modalidades y los seis documentos exigidos por la fuente.

Un borrador no es documento firmado; autenticación no es firma; aviso no es
entrega acreditada; declaración no es resolución; referencia de correo no es
custodia. No inventar plazos, nombramiento eficaz ni incorporación.

## E03. Arquitectura hexagonal

`domain` no importa aplicación, puertos, adaptadores, HTTP, SQL ni proveedores.
`application` coordina dominio y contratos neutrales. `ports` contiene interfaces
y DTO mínimos, nunca implementación de validación o serialización de negocio.
Los adaptadores traducen transportes y proveedores; la composición los conecta.
Las reglas son comunes a web, CLI, escritorio y otros canales.

Reutilizar las autoridades existentes de identidad, autorización, auditoría,
i18n y tema. No crear autoridades paralelas. Configuración canónica y adaptadores
intercambiables; no conectar dobles de prueba como infraestructura real.

## E04. Roles, ámbitos y autorización

Denegar por defecto. Exigir concesión central positiva, exacta y vigente para
actor, acción, recurso, ámbito, finalidad, perfil, campos y obligaciones.
RBAC es necesario y ABAC solo restringe. No inferir permiso de menú, ruta,
titularidad, prefijo de identificador o datos enviados por el cliente. No hay
administrador universal, suma implícita de perfiles ni comodines positivos.
Revalidar antes del efecto y conservar separación de funciones y unidades.

El sistema completo de roles sigue siendo requisito de producto. Una ruta con
un permiso correcto no lo da por terminado ni autoriza posponerlo por defecto.

## E05. Administración privada y fronteras

ADMIN en producción solo desde intranet, con DNIe o certificado digital y
autorización administrativa explícita. Por orden posterior del operador del
12 de septiembre de 2026, ADMIN de pruebas se publicará en
`admin.cidonia.cloud` por Internet: exclusivamente DNIe o certificado FNMT válido,
con comprobación de revocación y permiso únicamente para el DNI autorizado en
configuración privada externa a Git. Un certificado válido no concede permiso
por sí solo. No incluir ese DNI en fuentes, pruebas, documentación ni logs.
La revisión RRHH usará `vec.cidonia.cloud` con acceso autenticado; `app.cidonia.cloud`
solo es alternativa temporal. Ninguna publicación permite omitir las fronteras
de identidad o exponer directamente un perfil de desarrollo local.
SSH es acceso por terminal al servidor: no es segundo factor del portal.
Mantener la política específica del portal interno ordinario.

La administración de sistema tiene superficie segregada del portal RRHH y del
público; ocultar un menú o devolver 403 no acredita esa segregación. Identidades,
audiencias, emisores, red y rutas deben cumplir su fuente. Verificar canal,
origen, vigencia y revocación; no confiar en cabeceras libres de identidad.

## E06. Auditoría y efectos duraderos

Usar la autoridad de auditoría existente. Registrar accesos correctos y fallidos,
lecturas y cambios administrativos conforme al contrato, con minimización y
destino segregado que el administrador funcional no pueda alterar.
Conservar actor, acción, recurso, resultado, instante, correlación y versión
cuando correspondan; no registrar secretos ni documentos personales completos.

Todo efecto consume la autorización en la misma transacción que estado,
auditoría y outbox. Mantener idempotencia semántica, versión optimista e historia
de solo adición. Una tabla local de cambios aceptados no sustituye la auditoría
completa. Un replay o reinicio no debe duplicar efectos, recibos ni comunicaciones.

## E07. Privacidad, secretos y almacenamiento

Solo datos sintéticos hasta autorización expresa y requisitos formales de la
fuente. Minimizar información y usar referencias opacas. Nada de cookies,
localStorage, sessionStorage, IndexedDB ni credenciales persistidas por la web;
no emitir Set-Cookie ni introducir tokens en URL. Respetar la frontera de canal
establecida para cada cliente.

No subir claves, certificados privados, tokens, DSN, rutas privadas o datos reales
a Git ni exponerlos en logs, errores o fixtures. Cifrar tránsito y almacenamiento
según la política; usar gestión de claves existente, rotación y límites de tamaño,
tiempo y cardinalidad. Un fallo o dependencia caída nunca concede autorización.
No declarar cumplimiento normativo únicamente por superar pruebas técnicas.

## E08. i18n, accesibilidad y presentación

Castellano coherente. Todo texto visible, validación, estado, ayuda, documento y
notificación usa el catálogo i18n común. Localizar fechas, moneda, números y
plurales; probar con el traductor real y escapar datos al renderizar.

Aspecto limpio y profesional, datos agrupados por tarea y jerarquía clara.
Respetar las referencias visuales de RRHH y el tema común: no inventar otra
paleta ni duplicar CSS estructural. No dispersar datos ni hacer de identificadores
técnicos la presentación principal. Comprobar escritorio y móvil, teclado, foco,
contraste, zoom, lector de pantalla y estados vacío/carga/error/denegación.

## E09. Correo corporativo

El correo del destinatario procede del alta propia de VEC, donde es obligatorio;
no de una fuente alternativa inventada. Configuración SMTP en ADMIN privada:
IP interna de Diputación, puerto, TLS y cuenta emisora. IP y credenciales reales
pendientes no se sustituyen por valores inventados ni se dan por configuradas.

Secreto de solo escritura en UI, cifrado y redactado; permisos y auditoría reales
al consultar/cambiar configuración. Autenticación SMTP solo por canal protegido
según política. Reintentos y reservas deben impedir duplicar despachos. Código de
transporte probado no demuestra entrega ni integración del alta con Bolsa.

## E10. SQL, criptografía y revisión sensible

Dos revisiones independientes para identidad/autorización, criptografía, SQL y
fronteras de datos personales. Revisar la versión exacta final; un cambio posterior
reabre la parte afectada. Fuera de esas zonas, revisión proporcionada al riesgo.

Separar roles técnicos y mínimos privilegios. Probar PostgreSQL real cuando se
afirmen garantías de ACL, transacción, concurrencia y recuperación. No reaplicar
migraciones instaladas ni ejecutar DOWN sobre historia conservada. Inventariar
preimagen y dependencias; corregir con migración nueva cuando corresponda.
Un P2 se resuelve con parche acotado. No regenerar identidades o claves al arrancar.

## E11. Dirección y paralelización

Dividir trabajo útil por dependencias y archivos de propiedad exclusiva. Local y
remoto mantienen responsabilidades acordadas y un integrador de publicación.
No tocar WIP ni worktrees ajenos, ni programar en la raíz histórica. Comprobar
ramas y cambios antes de integrar; revisar y conservar trabajo válido.

Transmitir estas reglas en cada delegación, también recursiva. Al terminar una
tarea, asignar otra independiente si aporta valor; abrir una sesión nueva cuando
convenga ahorrar contexto. Contar agentes activos, no sesiones históricas.
Usar modelos económicos para mecánica, documentación y programación ordinaria;
reservar mayor capacidad para revisión sensible y xhigh para problemas difíciles
acotados. La concurrencia depende de tareas independientes y recursos disponibles.
Revisar cuota remota cada cinco minutos y entregar/preservar trabajo de forma
ordenada al quedar 5%; no matar procesos perdiendo cambios.

## E12. Aceptación y publicación

Probar lo afectado y los controles relevantes. Go: formato, pruebas focales,
`go test ./...` y `go vet ./...` antes de integración según alcance; race,
mutación y revisión adversarial proporcionadas al riesgo sensible. Web: contratos,
i18n y revisión visual. No repetir campañas cerradas sin cambio o fallo nuevo.

Una capacidad recorrible requiere navegador → API → autorización → aplicación →
PostgreSQL → recibo, y recuperación tras reinicio cuando se afirma persistencia.
Distinguir código escrito, revisado, probado, instalado, recorrible y publicado.
Informar omisiones y defectos; pruebas verdes no sustituyen conformidad funcional.
Commits seleccionados por archivos, autor aavidad, sin atribución de IA. Verificar
hash publicado, estado de ramas y revisión antes del push. No usar git add -A
sobre trabajo compartido. Actualizar el seguimiento único, no crear otro tablero.

## E13. Orden mínima que debe emitir cada director

En cada asignación, cambio de alcance y revisión previa a integrar, especificar:

```text
Objetivo funcional y paso de RRHH al que aporta:
Archivos propios; dependencias; archivos que pertenecen a otros agentes:
Fuentes y apartados aplicables; requisitos E01–E12 afectados:
Invariantes concretas que este cambio debe conservar:
Pruebas y recorrido necesarios; revisiones sensibles si corresponden:
Entrega: archivos/commit, resultados reales, límites y siguiente dependencia:
```

En sesiones prolongadas repasar las instrucciones de agentes activos al menos
cada 30 minutos y transmitir cambios pertinentes. No basta pegar una lista de
principios: explicar su efecto en la tarea. La obligación se aplica por dirección
y revisión; este Markdown no es un mecanismo automático de cumplimiento.
