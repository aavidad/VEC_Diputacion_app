# Acceso interno de tecnicos, RRHH y administracion

Estado: decision arquitectonica de seguridad; adaptadores corporativos y
despliegue pendientes de validar con Sistemas y Seguridad.

Fecha de corte: 14 de julio de 2026.

## Decision

El acceso ciudadano y el acceso del personal no compartiran una unica puerta
de entrada que cambie de aspecto segun el rol. Podran reutilizar nucleo y casos
de uso, pero tendran superficies tecnicas separadas.

| Superficie | Red y entrada | Identidad | Capacidades |
| --- | --- | --- | --- |
| Portal ciudadano | Internet/DMZ, WAF y proxy publico | Cl@ve, DNIe, certificado u otros proveedores admitidos | Solo datos y tramites propios, y consultas publicas minimizadas. |
| Aplicacion interna corporativa de escritorio | Solo Mulhacen o VPN corporativa autorizada, pasarela interna | Cuenta corporativa AD por Kerberos/SPNEGO, mTLS del dispositivo y autenticacion fuerte basada en certificado | Autoservicio del empleado y funciones de gestion, revision o baremacion conforme al unico perfil activo y sus ambitos. |
| Administracion de seguridad/sistema | Segmento de gestion, bastion o puesto privilegiado | Cuenta administrativa separada, autenticacion fuerte y elevacion temporal | Configuracion sensible, claves/referencias, roles, auditoria y operacion. |

Ninguna ruta interna se publicara en el proxy de Internet. Un control de menu o
un `403` no sustituye el aislamiento de red.

El portal ciudadano se divide a su vez en zona anonima, limitada a informacion
expresamente publicable, y area personal autenticada del titular o su
representante acreditado. Una persona empleada no obtiene capacidades internas
desde esa superficie. La linea base de perfiles y alcances esta en
[`../portal_vec/matriz_roles_y_ambitos.md`](../portal_vec/matriz_roles_y_ambitos.md).

## Kerberos y certificado

Kerberos aporta identidad corporativa, ciclo de alta/baja, pertenencia al
dominio y SSO desde un puesto administrado. El certificado aporta una prueba
criptografica adicional y se reutiliza, cuando proceda, para firmar decisiones
o documentos. Se aplican con estas condiciones:

1. La pasarela o proveedor de identidad interno negocia Kerberos/SPNEGO con la
   aplicacion de escritorio. El cliente nunca envia usuario, grupos o cabeceras
   de identidad libres a la aplicacion.
2. Las cabeceras de identidad entrantes de Internet se eliminan. Solo se acepta
   una asercion firmada y de vida corta del proveedor confiable, por un canal
   autenticado entre proxy y servicio. La frontera actual ignora tanto el PEM
   reenviado como una cabecera `SUCCESS`: una CIDR no acredita esa afirmacion.
   Solo se habilitara la terminacion en proxy cuando exista una asercion
   criptografica breve, con audiencia, enlace al canal y antirrepeticion; hasta
   entonces el certificado solo cuenta en un handshake TLS directo verificado.
3. La identidad Kerberos se enlaza con el certificado mediante un identificador
   corporativo estable; una coincidencia de nombre visible o correo no basta.
4. El certificado de alta garantia debe residir en tarjeta, token o elemento
   protegido equivalente y requerir PIN o biometria. Un `.p12` exportable en el
   mismo equipo no se considerara control equivalente sin un analisis expreso.
5. Si el inicio Kerberos ya se obtuvo mediante la misma tarjeta y PKINIT, no se
   afirmara automaticamente que una segunda comprobacion del mismo certificado
   sea un factor independiente. La matriz ENS y el analisis de riesgos decidiran
   si hace falta otro factor o un reto de presencia/usuario separado.
6. Para validar, rechazar o rectificar un merito se exige reautenticacion
   reciente y firma de la decision. En cada peticion, incluidas las lecturas
   ordinarias, el cliente presenta una credencial o asercion breve, con
   audiencia, vigencia y enlace al canal; no existe una sesion ambiental del
   navegador.

Cuando Go termine mTLS directamente, solo se acepta el certificado de cliente
que figure en la cadena verificada del `handshake`, coincida byte a byte con el
certificado par y admita autenticacion de cliente. `PeerCertificates` sin
`VerifiedChains`, una estructura TLS fabricada o una cadena no verificada se
tratan como ausencia de identidad.

La autenticacion y la firma son actos distintos: estar autenticado no firma una
decision, y poseer una firma valida no concede por si solo autorizacion para el
recurso.

## Perfiles y cuentas separadas

- Una persona que tambien sea candidata tendra un perfil ciudadano distinto de
  su perfil de empleado.
- Un administrador usara una cuenta ordinaria para su trabajo habitual y una
  cuenta privilegiada nominativa para administrar. No se permiten cuentas
  compartidas como `admin_rrhh`.
- Cada peticion identifica un solo perfil activo junto a la credencial. El
  servidor resuelve y valida su asignacion; no se suman permisos de varios
  perfiles ni se confia en roles declarados por el cliente.
- Los grupos de Active Directory son fuente de asignacion o propuesta de
  perfil, no autorizaciones aceptadas ciegamente. El motor RBAC+ABAC conserva
  version, ambito, vigencia, emisor y revocacion.
- El cese, cambio de unidad o retirada de funcion desactiva la asignacion y
  revoca las credenciales o aserciones vigentes dentro del plazo fijado.
- Las operaciones criticas pueden exigir concurrencia de dos personas distintas
  y nunca se satisfacen con dos perfiles de la misma persona.

## Separacion tecnica minima

- DNS, certificado TLS, virtual host y cliente del proveedor de identidad
  distintos para portal publico e interno.
- Ninguna superficie usa cookies de sesion. La credencial o asercion se envia
  de forma explicita en cada peticion y no se guarda en `localStorage` ni
  `sessionStorage`; la aplicacion de escritorio solo podra custodiarla en
  memoria o en el almacen seguro del sistema operativo conforme a la politica
  aprobada.
- Claves de firma de credenciales, audiencias, emisores, politicas de origen y
  CORS y limites de tasa separados. Los clientes web autorizados presentan
  `Authorization` de forma explicita y usan `credentials: "omit"`; bajo ese
  contrato no hay una credencial ambiental que el navegador pueda adjuntar a
  una peticion entre sitios. CORS y la validacion de origen se mantienen como
  defensa en profundidad, y XSS sigue tratandose como amenaza porque podria
  ejecutar operaciones o sustraer una credencial en memoria. Si en el futuro
  se habilita un navegador que negocie Kerberos/SPNEGO o presente mTLS de forma
  automatica, ambos pueden actuar como credenciales ambientales y sera
  obligatorio reevaluar CSRF y origen antes de abrir esa superficie.
- API interna con audiencia propia y sin rutas en el gateway publico.
- Listeners o despliegues separados cuando la frontera de red lo requiera,
  aunque ambos compilen desde los mismos puertos/casos de uso.
- Administracion de sistema fuera incluso del portal RRHH ordinario, mediante
  segmento de gestion, puesto autorizado y elevacion just-in-time cuando la
  infraestructura lo permita.
- Registro de accesos correctos y fallidos, ultimo acceso mostrado al usuario,
  reautenticacion en puntos criticos, expiracion breve y revocacion de
  credenciales.

## Flujo interno recomendado

1. La aplicacion de escritorio del puesto administrado accede al endpoint
   interno desde Mulhacen/VPN mediante mTLS.
2. La pasarela interna comprueba red, dispositivo y Kerberos/SPNEGO.
3. El proveedor exige certificado protegido por PIN/biometria o el segundo
   factor aprobado para el nivel resultante.
4. Emite una asercion corta con sujeto opaco, cuenta vinculada, autenticadores
   vinculados al mismo sujeto, instante y nivel de garantia; no incluye roles
   ni permisos finales inventados por el cliente o el proxy.
5. La aplicacion de escritorio presenta esa asercion de forma explicita en cada
   peticion, junto al perfil activo seleccionado; nunca la persiste en
   almacenamiento web.
6. El servidor valida audiencia, vigencia, canal y antirrepeticion, consulta la
   asignacion versionada y solicita autorizacion RBAC+ABAC sobre recurso, ambito
   y finalidad.
7. Una actuacion probatoria prepara bytes canonicos, solicita firma, valida la
   firma y confirma estado, auditoria y outbox de forma atomica.

## Administradores

Ademas de lo anterior:

- acceso desde puesto de administracion privilegiada o bastion;
- cuenta administrativa separada y nominativa;
- sin correo, navegacion general ni uso ciudadano con esa cuenta;
- privilegio temporal y aprobado para tareas de alto impacto;
- doble control para cambios de roles, politicas, claves, retencion, borrado,
  publicacion o rectificacion masiva;
- credenciales mas breves, reautenticacion por accion sensible y auditoria
  enviada a un destino que el propio administrador no pueda alterar.

## Referencias oficiales y lectura

El [Real Decreto 311/2022, ENS, texto consolidado](https://www.boe.es/eli/es/rd/2022/05/03/311/con)
exige identificadores singulares para perfiles diferentes, minimo privilegio,
segregacion de funciones, mecanismos de autenticacion adecuados, registro de
accesos y, para personal de la organizacion en nivel alto, refuerzos de
reautenticacion, suspension y doble factor en zonas no controladas. Sus
refuerzos de certificado requieren PIN o biometria, y contemplan soporte fisico
para la opcion de mayor garantia.

La documentacion oficial de Microsoft explica que el inicio con tarjeta
inteligente puede usar [PKINIT para obtener credenciales Kerberos](https://learn.microsoft.com/en-us/windows/security/identity-protection/smart-cards/smart-card-certificate-requirements-and-enumeration).
De ahi se infiere que hay que revisar la independencia real de los factores: no
basta contar dos veces la misma tarjeta o clave.

La categoria y los refuerzos definitivos no se presumiran. Se fijaran tras la
categorizacion, el analisis de riesgos y la validacion de Sistemas/Seguridad.

## Pruebas de aceptacion futuras

- Ninguna IP externa alcanza el listener o rutas internas.
- Una cabecera de identidad falsificada por el cliente no produce un contexto
  autenticado.
- Un certificado presentado sin cadena TLS verificada no produce un contexto
  autenticado.
- Kerberos sin el segundo mecanismo requerido no abre la API interna.
- Certificado valido con identidad distinta de la cuenta Kerberos falla
  cerrado y genera alerta sin exponer datos.
- Cuenta ciudadana, ordinaria y administrativa de una misma persona producen
  sujetos/perfiles y trazas inequívocos.
- Una credencial o asercion de RRHH no sirve en el portal ciudadano ni
  viceversa.
- La retirada de grupo/asignacion revoca nuevas autorizaciones y credenciales
  segun el objetivo temporal aprobado.
- Una accion critica exige reautenticacion y, si la politica lo marca, segundo
  aprobador distinto.
