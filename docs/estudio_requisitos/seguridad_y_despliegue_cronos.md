# Seguridad y despliegue del módulo Cronos

Estado: **dictamen técnico y jurídico preliminar para validar con Sistemas, Seguridad,
RRHH, DPD, Asesoría y representación del personal**.

Fecha de corte: 14 de julio de 2026.

## 1. Decisión

Cronos se desplegará como un enclave de seguridad exclusivamente interno en la red
Mulhacén. Podrá compartir repositorio, contratos, identidad visual y referencias de persona
con el resto del portal, pero no compartirá con la superficie pública:

- proceso ni artefacto ejecutable;
- clúster, nodos o plano de control;
- NGINX Gateway ni rutas;
- base de datos, almacenamiento o copias;
- secretos, claves, sesión o audiencia de tokens;
- servicios cartográficos;
- bot, MCP, notificaciones o telemetría externos.

Un contenedor o una red `internal` de Docker no bastan como frontera. La solución es **zona
interna segmentada más máquinas endurecidas más contenedores restringidos más autorización
por dato**.

## 2. Qué significa «no puede salir ningún dato»

El requisito se convertirá en controles verificables:

1. Cronos no tiene DNS, IP, ruta, puerto, activo web ni API publicados en Internet.
2. El clúster y sus cargas no tienen salida general a Internet por IPv4 ni IPv6.
3. Navegador, servidor, mapas, fuentes, scripts, analítica, logs, copias, soporte y
   actualizaciones usan exclusivamente servicios internos autorizados.
4. El portal público no puede alcanzar ninguna dependencia de Cronos.
5. Exportaciones, impresión, portapapeles y descargas se limitan según sensibilidad y rol.
6. Los puestos son corporativos y gestionados; los accesos sensibles pueden usar VDI/DLP.
7. Todo acceso, incluso de lectura, se autoriza y audita.

La red no puede impedir que una persona autorizada fotografíe una pantalla. Por tanto, la
garantía máxima combina prevención técnica, minimización, detección, puestos gestionados,
formación y responsabilidad. No se formulará una promesa absoluta imposible de demostrar.

El acceso remoto estará deshabilitado por defecto. Una VPN cifra y controla el canal, pero
mostrar datos en un domicilio significa que la información abandona físicamente las
instalaciones. Si la Diputación quisiera habilitarlo, deberá redefinir formalmente el
perímetro, limitarlo a equipo gestionado y repetir el análisis de riesgos.

## 3. Topología recomendada

```text
Internet / portal público / clúster DMZ
                       X
             sin ruta y sin confianza
                       X
Red Mulhacén de usuarios gestionados
       |
       +-- NAC / EDR / DLP
       |
       v
WAF o proxy interno + IdP/AD + MFA
       |
       v
NGINX Gateway interno
       |
       v
Enclave Cronos
       +-- UI/API Cronos
       +-- autorización RBAC + atributos + relaciones
       +-- trabajadores internos
       +-- PostgreSQL y objeto exclusivos
       +-- teselas, geocodificador y OSRM internos
       +-- auditoría -> SIEM/WORM interno
       +-- copias cifradas -> segundo emplazamiento interno

Vehículo
       +-- GNSS + certificado propio + SIM M2M
       +-- APN privada / transporte cifrado
       v
DMZ telemática restringida
       +-- valida dispositivo, firma, secuencia y fecha
       +-- no consulta Cronos ni su base
       v
relé/broker autorizado -> consumidor Cronos interno

Internet de actualización
       v
zona de preparación: procedencia + firma + hash + SBOM + antivirus
       v
promoción controlada al repositorio interno
```

El ENS exige documentar y autorizar interconexiones, segmentar redes y considerar prohibido
todo flujo que no esté expresamente permitido. También permite segregar subsistemas cuando
requieren medidas diferentes: [Real Decreto 311/2022](https://www.boe.es/eli/es/rd/2022/05/03/311/con).

## 4. Contenedores y red

Defensa mínima:

- clúster o grupo de VM distinto del portal público;
- VLAN/VRF propias para acceso, aplicación, datos, telemetría y gestión;
- cortafuegos L3/L4 y de host con denegación predeterminada;
- políticas Kubernetes de entrada y salida;
- ningún pod o contenedor conectado simultáneamente a una red pública e interna;
- contenedores sin root, sin privilegios, sin capacidades y con raíz de solo lectura;
- `no-new-privileges`, `seccomp` y AppArmor/SELinux;
- sin socket de contenedores ni montajes generales del anfitrión;
- imágenes internas por digest, firmadas, analizadas y con SBOM;
- DNS, NTP, PKI, registro, Git, secretos, antivirus y observabilidad internos;
- administración desde bastión, MFA y PAM;
- alertas ante cualquier intento de salida.

La configuración de una red Docker `internal: true` puede ayudar en laboratorio, pero el
anfitrión sigue accediendo a ella y una segunda interfaz puede reabrir la salida. No se
considerará control suficiente para producción.

## 5. Identidad y autorización

Active Directory o el IdP corporativo autenticarán al personal. La presencia en Mulhacén o
la pertenencia a un grupo AD no concederán por sí solas acceso funcional.

El proxy eliminará cabeceras de identidad aportadas por el cliente y emitirá o validará un
token corto con emisor y audiencia exclusivos de Cronos. Se exigirá MFA a responsables,
RRHH, flota, administradores y operaciones sensibles.

La autorización combinará:

- **RBAC:** función general;
- **ABAC:** unidad, fecha, finalidad, tipo de dato y nivel de autenticación;
- **relaciones:** jefatura o delegación vigente sobre la persona concreta;
- **filtro por campo:** ocultar motivos y documentos que el rol no necesita.

Matriz inicial:

| Perfil | Vista necesaria | Datos que no ve por defecto |
| --- | --- | --- |
| Empleado | Sus fichajes, calendario, saldos, solicitudes y resoluciones | Datos de otra persona |
| Jefe/responsable | Disponibilidad, fecha/duración y solicitudes de subordinados vigentes | Diagnósticos, justificantes íntimos y detalle sindical |
| Suplente/delegado | Mismo alcance solo durante delegación formal | Otras unidades o periodos |
| RRHH de tiempo | Expediente necesario según unidad y competencia | Unidades no asignadas |
| Relaciones laborales | Información sindical estrictamente necesaria | Resto del expediente sin competencia |
| Gestor de flota | Vehículo, misión y posición autorizada | Fichajes, salud o permisos personales |
| Auditor/Seguridad | Evidencias y alertas justificadas | Contenido ordinario no necesario |
| Administrador técnico | Infraestructura, salud y configuración técnica | Datos funcionales ordinarios |

Un permiso `cronos.leer` no basta. Cada petición comprobará actor, perfil activo, acción,
objeto, relación jerárquica, unidad, vigencia, finalidad, campos y nivel de autenticación.

La API interna futura no recibirá del navegador el identificador de empleado como fuente
de autoridad. El servidor resolverá la persona empleada vinculada a la identidad y el PDP
devolverá una concesión positiva e inmutable para el recurso exacto. El contrato distinguirá,
como mínimo, estas operaciones; ninguna implicará otra:

| Operación | Recurso y ámbito obligatorio |
| --- | --- |
| consultar fichajes propios | persona propia resuelta por servidor y periodo exacto |
| registrar fichaje propio | persona propia, terminal/canal autorizado y fecha exacta |
| consultar permisos propios | persona propia y periodo exacto |
| solicitar permiso propio | persona propia, política vigente y expediente exacto |
| consultar fichajes de subordinado | relación de jefatura/delegación vigente, persona y periodo exactos |
| resolver permiso de subordinado | relación vigente, solicitud exacta, finalidad y campos permitidos |
| gestionar por RRHH | competencia, unidad, persona, operación y periodo expresos |
| consultar GPS | vehículo/comisión exactos; posición actual e historial como permisos distintos |
| ver justificante sensible | documento y campos exactos, finalidad y nivel reforzado |

Una colección de personas autorizadas será siempre una lista positiva no vacía calculada
por el servidor. Una lista vacía, un identificador ausente, una relación no encontrada, un
campo desconocido o un fallo del resolver significan denegación; nunca significan «todas
las personas».

Las horas sindicales pueden revelar afiliación y los justificantes pueden contener salud,
ambos datos de especial protección conforme al artículo 9 del RGPD. La vista del jefe usará
categorías neutras como «ausencia autorizada» y separará los documentos:
<https://eur-lex.europa.eu/eli/reg/2016/679/oj?locale=es>.

Las delegaciones y suplencias no se crearán solo en configuración técnica: deben reflejar
competencia o delegación válida conforme a la Ley 40/2015.

## 6. GPS de vehículos

### 6.1 Límite físico

Un receptor GNSS calcula la posición en el vehículo. Para verla en directo hay que
transmitirla desde fuera de las instalaciones. Existen dos alternativas:

1. **Tiempo real contenido:** SIM M2M y APN privada del operador, sin Internet general,
   túnel/enlace privado y cifrado extremo a extremo hasta la Diputación.
2. **Cero tránsito exterior literal:** el dispositivo cifra y guarda localmente y descarga
   al volver a Mulhacén. No ofrece posición en directo.

RRHH, Sistemas, Seguridad y DPD deben elegir expresamente qué interpretación se aplica. Si
se usa operador, se analizará su papel, contrato, subencargados, metadatos y respuesta a
incidentes.

### 6.2 Dispositivo y canal

- certificado y clave únicos por vehículo/dispositivo;
- mTLS, firma del mensaje y cifrado de coordenadas hasta el consumidor interno;
- firmware firmado, inventario y baja/revocación remota;
- sin credenciales predeterminadas ni nube del fabricante;
- secuencia, sello de tiempo y nonce contra repetición;
- límite de frecuencia, tamaño y área razonable;
- pasarela que valida y publica, pero no consulta Cronos;
- relé de sentido único cuando el análisis lo exija;
- posición asociada primero a vehículo y comisión, no a persona permanente.

El GPS estará activo solo durante el servicio o finalidad aprobada. No se seguirá fuera de
jornada ni durante uso privado autorizado. Posición actual e historial serán permisos
diferentes.

### 6.3 Finalidad y proporcionalidad

La geolocalización laboral requiere información previa expresa, clara e inequívoca a las
personas y, en su caso, representantes: [artículo 90 de la
LOPDGDD](https://www.boe.es/eli/es/lo/2018/12/05/3/con).

No se usará para observación permanente, puntuación automática, control genérico «por si
acaso» ni fichaje continuo. No se obligará a instalarla en un teléfono personal. La guía
laboral de la AEPD insiste en finalidad, necesidad, proporcionalidad y limitación temporal:
<https://www.aepd.es/guias/la-proteccion-de-datos-en-las-relaciones-laborales.pdf>.

## 7. OpenStreetMap sin salida

Se utilizarán los datos y licencia de OpenStreetMap, no sus servicios públicos:

- Leaflet u otra biblioteca servida localmente;
- teselas ráster o vectoriales internas;
- estilos, iconos y fuentes locales;
- Nominatim u otro geocodificador interno;
- OSRM interno para rutas;
- sin CDN, analítica, fuentes o respaldo externo;
- Content Security Policy limitada a orígenes internos;
- atribución visible a OpenStreetMap y cumplimiento ODbL.

La política oficial de teselas recomienda autoalojamiento para uso sin conexión y advierte
contra el envío de información personal o confidencial:
<https://operations.osmfoundation.org/policies/tiles/>. La instancia pública de Nominatim
tampoco se usará para seguimiento ni datos confidenciales:
<https://operations.osmfoundation.org/policies/nominatim/>.

El prototipo actual descarga recursos de `unpkg.com`, solicita teselas públicas y construye
enlaces externos con coordenadas en `web/static/index.html` y `web/static/app.js`. Son
bloqueantes absolutos para Cronos y deberán eliminarse del artefacto interno. Ya existe una
base útil de OSRM local en `deploy/osrm-granada`, pero no es todavía un despliegue
productivo.

Las actualizaciones OSM se descargarán en una zona de preparación, se verificarán,
analizarán y empaquetarán con versión. Se transferirán al repositorio interno y se construirá
el nuevo grafo en paralelo. Tras pruebas de rutas y salud se hará un cambio atómico; Cronos
no saldrá a Internet.

## 8. Navegador, puesto y prevención de fuga

- solo equipos corporativos gestionados;
- CSP estricta para scripts, estilos, imágenes, fuentes, conexiones y marcos;
- `Cache-Control: no-store` para vistas sensibles;
- sin modo sin conexión ni `service worker` para datos Cronos;
- deshabilitar exportación masiva por defecto;
- permisos separados para descargar, imprimir y ver posición histórica;
- marca de agua y justificación en exportaciones sensibles;
- DLP/EDR y, cuando el riesgo lo justifique, VDI sin portapapeles, impresión o descarga;
- tiempo de sesión y bloqueo de pantalla corporativos;
- no incluir datos sensibles en URL, historial, caché, notificaciones de escritorio o
  informes de error.

Correo, Telegram, bot público, MCP público, TTS externo, analítica y modelos externos no
recibirán información de Cronos. Un posible asistente interno tendrá que ejecutarse dentro
de Mulhacén y heredar exactamente la autorización del usuario.

## 9. Datos, conservación y copias

- base, cuentas, esquemas y almacenamiento documental exclusivos;
- cifrado en tránsito y reposo; claves en KMS/HSM o gestor aprobado;
- documentos médicos/sindicales en depósito segregado;
- relación conductor–vehículo separada de la serie de coordenadas y con referencias opacas;
- coordenadas ausentes de logs generales;
- copias cifradas, inmutables y en segundo emplazamiento interno;
- claves de copia separadas del dominio comprometible de producción;
- restauración periódica con conciliación.

No se fijará un único periodo de conservación. Se aprobarán plazos distintos para:

- registro de jornada según régimen laboral o funcionarial;
- solicitudes, resoluciones y expediente;
- posición actual y búfer operativo;
- trayecto detallado;
- evidencia de incidente o comisión;
- auditoría de accesos;
- copias y bloqueos por litigio.

El plazo del registro de jornada del personal laboral no debe copiarse automáticamente a
funcionarios ni a GPS. RRHH, Archivo, Asesoría y DPD definirán el calendario documental por
colectivo y finalidad.

## 10. Auditoría

Se registrarán, al menos:

- lecturas permitidas y denegadas;
- empleado, unidad, vehículo o periodo consultado;
- finalidad y relación que autorizó el acceso;
- aprobaciones, denegaciones, motivos y delegación usada;
- correcciones con valor anterior/posterior necesario;
- acceso y descarga de justificantes;
- posición actual e historial consultado;
- exportaciones, impresión y acceso excepcional;
- cambios de roles, organigrama, reglas, retención y configuración;
- intentos de entrada/salida de red y de desactivar controles.

La auditoría de negocio será append-only, encadenada o firmada, se remitirá a un almacén
interno resistente a alteración y se separará del log técnico. El acceso de auditoría será
también auditable.

## 11. Privacidad, negociación y autoridad de control

La EIPD se tratará como obligatoria antes de contratar, pilotar o producir: concurren
monitorización sistemática, geolocalización, personas empleadas, cruce de fuentes y posibles
categorías especiales. Los criterios oficiales de la AEPD consideran normalmente exigible
la evaluación cuando concurren dos o más criterios:
<https://www.aepd.es/documento/listas-dpia-es-35-4.pdf>.

Para una Diputación andaluza, la autoridad de control competente es ordinariamente el
Consejo de Transparencia y Protección de Datos de Andalucía. Su Plan de Control e
Inspección 2026-2027 incluye diputaciones, DPD, EIPD, tecnologías intensivas y contratación:
<https://www.juntadeandalucia.es/boja/2026/118/23>.

La representación del personal participará en el método de fichaje, organización del
registro, finalidades y horario del GPS, dispositivos, información visible para superiores,
conservación y posible uso disciplinario. No se incorporará biometría en la primera versión.

## 12. Integración con el resto del portal

Cronos podrá consumir referencias canónicas de persona, puesto, unidad, jerarquía y
situación mediante API o eventos internos mínimos. No escribirá directamente en tablas de
otro módulo ni permitirá que la DMZ consulte sus tablas.

Su manifiesto declarará como mínimo:

```yaml
modulo: cronos
zona_datos: mulhacen_restringida
superficies_permitidas:
  - portal_empleado_interno
  - backoffice_rrhh_interno
proyeccion_publica: ninguna
salida_internet: prohibida
escala_a_cero: prohibida
mcp_publico: prohibido
notificacion_externa: prohibida
conectores:
  - identidad_corporativa
  - persona_rpt_interna
  - nomina_interna
  - mapas_internos
  - auditoria_interna
```

Si una integración necesita cruzar una frontera, tendrá contrato mínimo, identidad propia,
mTLS, lista de campos, finalidad, reintento, auditoría y aprobación de la interconexión.

## 13. Requisitos verificables

- **CRN-RED-001:** desde Internet no existe DNS, ruta, puerto ni endpoint de Cronos.
- **CRN-RED-002:** desde cada carga fallan salidas exteriores por IPv4, IPv6, DNS, DoH,
  HTTP/HTTPS, QUIC e ICMP.
- **CRN-RED-003:** el clúster público no alcanza API, datos, broker, mapas ni copias de
  Cronos.
- **CRN-DES-001:** el artefacto público no incluye código, rutas o manifiesto Cronos.
- **CRN-IAM-001:** cabeceras de identidad del cliente se eliminan y una identidad falsa
  obtiene denegación.
- **CRN-IAM-002:** tokens exigen emisor y audiencia internos y MFA según operación.
- **CRN-AUT-001:** cambiar un identificador no permite consultar a otra persona.
- **CRN-AUT-002:** una jefatura solo ve relaciones vigentes y pierde acceso al cesar.
- **CRN-AUT-003:** salud, actividad sindical y justificantes se filtran por campo.
- **CRN-GPS-001:** dispositivo sin certificado, mensaje repetido o fuera de misión se
  rechaza.
- **CRN-GPS-002:** fuera del servicio no se recogen posiciones salvo finalidad excepcional
  aprobada.
- **CRN-MAP-001:** una captura de red del mapa solo contiene destinos internos.
- **CRN-AUD-001:** toda lectura, denegación, descarga y exportación deja evidencia íntegra.
- **CRN-CON-001:** contenedores no root, sin privilegios, sin socket y por digest firmado.
- **CRN-BCK-001:** copia y restauración cumplen RPO/RTO aprobados.
- **CRN-ACT-001:** una actualización solo entra tras procedencia, firma, SBOM y análisis.
- **CRN-FAL-001:** si falla IdP, autorización o mapa, el sistema falla cerrado y no recurre
  a Internet.
- **CRN-DLP-001:** no existe conector externo habilitado para datos, avisos, telemetría o IA
  de Cronos.

## 14. Bloqueantes actuales del prototipo

El ejecutable actual no debe recibir datos Cronos reales porque, entre otros:

- todos los módulos se componen en el mismo proceso y API;
- autenticación e identidades de demostración no son productivas;
- la autorización actual no acredita relación jerárquica por objeto y fecha;
- rutas y respuestas permiten representaciones demasiado amplias;
- el `docker-compose.yml` general publica el servicio sin esta separación;
- el frontend usa recursos y teselas externas;
- OSRM usa una imagen mutable `latest`;
- no existen aún clúster interno, CSP cerrada, DLP, EIPD, APN ni mapas íntegramente locales.

Estos hallazgos no se corregirán ocultando menús. Cronos se extraerá de la composición
pública y solo se habilitará cuando supere todas las pruebas de frontera.

### 14.1 Contención aplicada al HTTP heredado

Como medida inmediata, no como implementación productiva de autorización:

- `/api/vec/workspace`, `/api/vec/cronos/timecards` y
  `/api/vec/cronos/leave-requests` están cerrados incluso para una identidad que posea el
  permiso grueso heredado;
- una petición autenticada sin el permiso positivo preliminar recibe `403`; quien sí lo
  posee recibe `503` y ningún dato, porque falta el resolver servidor de persona, relación,
  ámbito, finalidad y campos;
- los `POST` no decodifican ni persisten el `employee_id` aportado por el cliente;
- las respuestas cerradas usan `Cache-Control: no-store`;
- la carcasa HTTP ya no construye ni conserva un generador de datos Cronos de
  demostracion;
- el puerto de jornadas exige una fecha y una lista positiva no vacía de empleados; los
  puertos de saldos y solicitudes exigen empleado y ejercicio exactos. Vacíos, duplicados,
  entradas no canónicas y `*` producen error;
- el antiguo `workspaceSnapshot` sintetico e inalcanzable fue eliminado para
  impedir que una refactorizacion futura pudiera publicarlo por accidente.

La puerta solo podrá abrirse en el artefacto interno separado después de incorporar el PDP,
el resolver corporativo de persona/organigrama/delegaciones, consultas acotadas en el puerto
de persistencia, filtrado positivo por campo, auditoría de lecturas y denegaciones, y pruebas
de cambio de identificador, cese de jefatura, suplencia vencida y fallo de backend. No se
aceptarán cabeceras del cliente, convenciones sobre el nombre del usuario ni un rol
«administrador» como sustituto de esas evidencias.

## 15. Decisiones pendientes

1. Interpretación exacta de «no salir»: transporte cifrado por APN o GPS diferido.
2. Colectivos laborales/funcionarios y reglas de jornada aplicables.
3. Finalidades concretas y proporcionalidad del GPS.
4. EIPD, riesgo residual e información a personas y representantes.
5. Organigrama maestro, delegaciones, suplencias y baja automática.
6. Matriz de datos/campos por perfil.
7. Plazos de conservación por serie y finalidad.
8. WAF, clúster, base, S3, SIEM, copias y DLP internos.
9. APN, dispositivo, PKI y contrato de telecomunicaciones.
10. Política de acceso remoto, inicialmente prohibido.
11. Procedimiento de actualizaciones cartográficas y software sin salida.
12. RTO/RPO, segundo emplazamiento y simulacros.

Estas decisiones forman parte del
[Manual 00 para Sistemas](manual_sistemas_preparacion_plataforma.md) y de su futuro anexo
«Cronos, APN y cero salida».
