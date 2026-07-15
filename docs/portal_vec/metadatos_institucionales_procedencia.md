# Metadatos institucionales de procedencia documental

Fecha de referencia normativa: 15-07-2026.

## Finalidad y alcance

El Portal VEC puede incorporar a una representacion documental metadatos que
permitan reconocer que fue creada por la Diputacion y vincularla con su
evidencia de integridad. Esta capacidad no es una huella del destinatario ni un
mecanismo de vigilancia. Su finalidad cerrada es procedencia, autenticidad,
interoperabilidad y cotejo.

El contenido permitido se limita a referencias institucionales gobernadas,
UUID v4 generado criptograficamente, perfil y version de formato, instante UTC,
referencia opaca preasignada del manifiesto y, solo cuando la politica lo
autorice, URI institucional de verificacion. No se incorporan DNI, nombre,
usuario, destinatario, IP, ruta, servidor, credencial, version vulnerable ni
secreto.

Un UUID aleatorio no describe por si mismo a una persona. Sin embargo, pasa a
ser dato personal indirecto si el responsable puede relacionarlo con una
persona o si el servicio de cotejo revela la existencia o contenido de su
expediente. Por prudencia, el UUID, el manifiesto y las consultas se protegen
como referencias seudonimas: no se indexan publicamente, no se registran junto
con datos personales en logs generales y la respuesta del cotejo aplica la
misma politica de acceso y minimizacion que el documento.

## Encaje normativo

- El articulo 5 del RGPD exige licitud, limitacion de finalidad, minimizacion,
  exactitud, limitacion del plazo, integridad, confidencialidad y responsabilidad
  proactiva. Si una referencia resulta enlazable con una persona, se trata como
  dato personal aunque no contenga su nombre.
- La base juridica de una Administracion se determina por obligacion legal o
  mision de interes publico, segun el caso; no se usa el consentimiento para
  legitimar un control necesario de autenticidad o seguridad.
- Los articulos 24 y 25 del RGPD exigen medidas desde el diseno y por defecto;
  el articulo 32 exige medidas adecuadas al riesgo. La evaluacion de impacto del
  articulo 35 se analiza con el DPD cuando la funcion se combine con observacion
  sistematica, perfiles o tratamiento de alto riesgo.
- El articulo 24 del ENS permite y exige trazabilidad proporcionada, con la
  informacion estrictamente necesaria y respetando derechos y privacidad. Los
  articulos 21 y 40 obligan a gobernar activos y considerar autenticidad,
  integridad, confidencialidad y trazabilidad.
- El ENI y la NTI de Documento Electronico tratan contenido, firma y metadatos
  como componentes del documento administrativo. El metadato de procedencia se
  integra en ese modelo y no sustituye los metadatos ENI obligatorios.
- La Ley 39/2015 y el Real Decreto 203/2021 exigen autenticidad e integridad de
  documentos y copias y permiten sistemas de CSV/cotejo. Una marca interna no
  sustituye la firma o sello electronico ni el servicio oficial de cotejo.

Fuentes oficiales:

- [Reglamento (UE) 2016/679](https://eur-lex.europa.eu/legal-content/ES/TXT/?uri=celex%3A32016R0679)
- [Real Decreto 311/2022, Esquema Nacional de Seguridad](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191)
- [Real Decreto 4/2010, Esquema Nacional de Interoperabilidad](https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331)
- [NTI de Documento Electronico](https://www.boe.es/diario_boe/txt.php?id=BOE-A-2011-13169&lang=es)
- [Ley 39/2015](https://www.boe.es/eli/es/l/2015/10/01/39)
- [Real Decreto 203/2021](https://www.boe.es/buscar/act.php?id=BOE-A-2021-5032)
- [Ley Organica 3/2018](https://www.boe.es/buscar/act.php?id=BOE-A-2018-16673)

La politica concreta se aprueba antes de produccion por responsables funcional,
juridico y de seguridad, con intervencion del DPD. Este documento es una
especificacion de proyecto y no sustituye el informe juridico del tratamiento.

## Orden criptografico obligatorio

```text
renderizar borrador
        |
        v
incorporar metadato institucional estandar
        |
        v
extraer y comparar el metadato con verificador independiente
        |
        v
comprobar equivalencia semantica
        |
        v
calcular SHA-256 de los bytes finales
        |
        v
crear y firmar manifiesto de integridad
        |
        v
firma/sello del documento + sello de tiempo + custodia
```

Nunca se modifica el original despues de firmarlo. Si se necesita otra marca o
correccion, se crea una representacion derivada con identidad, huella, relacion
y firma propias.

La marca embebida no contiene el SHA-256 de los mismos bytes que la contienen
ni el digest de un manifiesto que a su vez comprometa ese SHA-256. La referencia
opaca del manifiesto se reserva antes; el hash final se calcula despues y se
vincula externamente. Esto evita una autorreferencia circular imposible.

## Estrategia por formato

| Familia | Procedencia | Regla de seguridad |
|---|---|---|
| PDF | Metadatos XMP u otro perfil estandar homologado | Incorporar antes de firma y validar estructura, firma y perfil. |
| ODT/DOCX | Propiedades estandar del paquete, sin macros ni relaciones externas | Validar ZIP/XML, expansion, recursos y equivalencia. |
| JSON/XML | Bloque explicito definido por esquema versionado | No aceptar campos libres; canonicalizacion y esquema exactos. |
| CSV/TXT | Manifiesto lateral firmado | No usar espacios, comentarios o caracteres invisibles. |
| Formato nuevo | Estrategia declarada por perfil y conector homologado | Ausencia de marcador/extractor/validador exactos deniega. |

Los metadatos pueden perderse al convertir o sanear un fichero. Por ello solo
son una ayuda de procedencia. La prueba fuerte es la combinacion de firma o
sello, hash, manifiesto firmado, CSV/cotejo, custodia y auditoria.

## Gobierno y arquitectura

1. La identidad del formato no fija MIME, extension ni capacidades.
2. Un perfil inmutable y versionado fija MIME, extension, charset, conformidad,
   limites y estrategia de metadatos.
3. El estado operativo actual se conserva fuera del perfil. Una revocacion
   bloquea nuevas operaciones sin reescribir documentos o evidencias historicos.
4. Cada operacion resuelve un componente atestado para su rol: renderizador,
   marcador, extractor y verificador. El componente no acredita su propio
   binario; lo hace un broker o supervisor independiente.
5. Marcador, extractor y verificador usan identidades, artefactos y dominios de
   confianza segregados. Un componente no se certifica a si mismo.
6. La entidad, el organo y el endpoint se obtienen de un catalogo institucional
   positivo. El cliente no aporta nombres, plantillas de URI ni hosts libres.
7. La URI se construye desde una base HTTPS homologada, sin credenciales,
   consulta, fragmento, IP, `localhost` o host interno. Puede omitirse para
   documentos privados.
8. La salida se extrae y compara campo a campo. Que cambien los bytes o que el
   marcador declare una huella no demuestra que exista un metadato estandar.
9. Todo valor desconocido, ambiguo, retirado, revocado, no atestado o fuera de
   lista positiva produce denegacion.

## Controles de privacidad y seguridad

- Registro de actividades con finalidad, categorias, destinatarios, plazo y
  controles de acceso, cuando las referencias sean enlazables.
- Retencion coordinada entre documento, manifiesto, indice de cotejo, claves y
  auditoria; el cotejo no prolonga por defecto la vida del contenido.
- Separacion entre servicio publico de cotejo y repositorio privado. Una
  respuesta publica minima puede limitarse a autenticidad/estado sin mostrar
  titular, expediente ni contenido.
- Rate limiting, deteccion de enumeracion, identificadores de alta entropia y
  respuestas uniformes para documentos inexistentes o no autorizados.
- Logs sin URI completa, UUID, manifiesto o contenido salvo la referencia
  seudonimizada estrictamente necesaria en una traza protegida.
- Acceso interno por permiso positivo, finalidad, expediente y ambito; ningun
  rol administrador universal.
- Pruebas de eliminacion de metadatos, conversion, firma alterada, UUID/URI
  hostiles, PII en catalogos, revocacion concurrente, replay y conectores que
  mutan o retienen buffers.

## Criterios de aceptacion

- Un documento privado puede marcarse sin URI publica.
- Un UUID que no sea v4 generado por el servicio se rechaza.
- Entidad, organo o endpoint no publicados se rechazan antes del conector.
- El marcador solo consume un borrador pre-firma atestado; bytes arbitrarios o
  un documento ya firmado no se aceptan.
- Un extractor independiente recupera exactamente la marca esperada de la
  salida. Comentarios, espacios o zero-width no satisfacen el contrato.
- El verificador recibe copias desechables y no puede modificar los bytes
  autoritativos ni retenerlos para alterarlos despues.
- La situacion operativa se relee inmediatamente antes de cada efecto; una
  revision historica no autoriza una ejecucion nueva.
- El SHA-256 final y la firma se calculan despues del marcado; el original
  firmado permanece inmutable.
- La ausencia de metadatos en una copia no demuestra que sea ajena; el cotejo
  autoritativo usa el artefacto canonico y su manifiesto.
