# Continuidad del producto real — 21 de julio de 2026

## Decisión de dirección

Dirección comunica que la propuesta ha sido aceptada y ordena continuar el
desarrollo de la aplicación real. Desde este corte, la presentación deja de
ser el objetivo de entrega: se conserva únicamente como referencia visual y
como entorno de datos sintéticos, físicamente excluible del artefacto
productivo.

Esta decisión autoriza el desarrollo, las migraciones, los conectores y las
pruebas técnicas necesarios para completar el producto. No equivale a una
aceptación funcional formal de RRHH, a una autorización de puesta en
producción ni a una autorización para cargar datos personales reales.

## Reglas de continuidad

1. Los casos de uso, contratos HTTP y componentes visuales válidos se
   conservan. Los adaptadores de presentación se sustituyen por conectores
   durables a través de sus puertos; no se reescribe la web para cada cambio
   de infraestructura.
2. Ninguna identidad, perfil, rol, persona, DNI o permiso procede del
   navegador. La frontera autenticada entrega referencias opacas; el servidor
   revalida la sesión, resuelve el contexto de actor y consulta el PDP antes de
   cada efecto.
3. El código de demostración permanece marcado, sin validez administrativa y
   fuera del manifiesto productivo. Sus expedientes no se migran.
4. Se mantiene el mínimo privilegio: una capacidad no concedida expresamente
   se deniega. Cada adaptador PostgreSQL usa una credencial nominal distinta y
   acreditada al arrancar.
5. La composición productiva sigue bloqueada hasta disponer de los proveedores
   corporativos y de las conformidades de Sistemas, DPD y RRHH que correspondan.
   T12 (durabilidad probatoria) y T13 (registro de accesos con finalidad)
   continúan siendo condiciones previas al piloto con datos reales.

## Orden inmediato

1. Cerrar la vertical de borradores de convocatorias: identidad, contexto de
   actor, autorización, PostgreSQL, KMS, recibo y web interna.
2. Ejecutar una prueba E2E técnica con PostgreSQL 18 y datos exclusivamente
   sintéticos, incluyendo denegaciones y fallos de dependencias.
3. Completar T12 y T13 antes de cualquier piloto con información personal.
4. Incorporar de forma gobernada los datos de Convoca y continuar por bases,
   baremación, revisión técnica y llamamientos.

## Evidencia y estados

La matriz operativa seguirá separando cinco hechos: componente probado, E2E
técnico, recorrido manual, UAT de RRHH y producción. La aceptación de la
propuesta solo cambia la continuidad del proyecto; no adelanta ninguna de las
otras cuatro columnas.
