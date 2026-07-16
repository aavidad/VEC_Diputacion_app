# Fachada de uso de autorización

`FachadaUsoDecisionAutorizacion` es la entrada reutilizable para que un módulo
obtenga `ports.EvidenciaUsoDecisionAutorizacion` sin implementar otro PEP. La
fachada reutiliza la validación vinculada del núcleo y nunca devuelve la
`domain.DecisionAutorizacion` mutable.

## Composición

La aplicación crea una única fachada con el `ports.Autorizador` y el
`ports.Reloj` confiables:

```go
fachada, err := application.NuevaFachadaUsoDecisionAutorizacion(
    autorizador,
    reloj,
)
```

Estas dependencias pertenecen a la composición del servidor. No se eligen ni
se reconstruyen desde HTTP, CLI, MCP u otra frontera de entrada.

## Política del consumidor

Cada caso de uso declara una política cerrada antes de atender peticiones:

```go
politica, err := application.NuevaPoliticaUsoDecisionAutorizacion(
    "bolsa.convocatoria.publicar",
    "bolsa",
    "convocatoria_bolsa",
    "gobernar_convocatoria",
    []string{"convocatoria.estado", "convocatoria.fecha_publicacion"},
    application.PerfilProteccionUsoAutorizacionInternoAlto,
)
```

La política fija de forma inseparable la acción, módulo, tipo de recurso,
finalidad, campos y perfil de protección. Esos valores no llegan como
parámetros libres de cada petición y no pueden sustituirse desde un adaptador
de entrada. La referencia y el contexto concreto del recurso siguen siendo
resueltos por el servidor para cada operación.

La lista de campos es un conjunto exacto y no depende del orden. Una lista
vacía indica que la operación es atómica y no admite ningún campo restringido.
Se rechazan campos ausentes, adicionales, repetidos, no canónicos o con
comodín. Toda obligación se deniega mientras no exista una capacidad tipada
que demuestre su cumplimiento.

Los perfiles disponibles son:

- `PerfilProteccionUsoAutorizacionOrdinario`: permite superficie personal
  externa y exige la garantía fijada por el PDP.
- `PerfilProteccionUsoAutorizacionInternoAlto`: solo admite superficie
  corporativa o de administración privilegiada, sesión de garantía alta y una
  decisión cuya garantía mínima también sea alta.

La política es configuración interna nominal, no una capacidad criptográfica:
se reconstruye deliberadamente mediante su constructor. Hace copia defensiva
de los campos y bloquea serialización y reconstrucción por JSON, XML, texto,
binario y gob. No debe formar parte de un DTO.

## Uso por un módulo

```go
var exigidor application.ExigidorEvidenciaUsoDecisionAutorizacion = fachada

evidencia, err := exigidor.ExigirEvidencia(
    ctx,
    contextoActor,              // resuelto por la frontera confiable
    vinculoAutenticacionActor,  // revalidado y opaco
    recursoResueltoPorServidor,
    correlacionRef,
    motivo,
    politica,
)
```

El módulo y tipo del recurso deben coincidir exactamente con la política antes
de consultar al PDP. La referencia, ámbitos y atributos del recurso, la
correlación y el motivo deben llegar ya canónicos desde fuentes confiables; la
fachada no corrige ni completa entradas. El recurso se copia antes de entregarlo
al autorizador.

Después del PDP se vuelven a comprobar la cancelación y la correspondencia
exacta de principal, perfil, acción, referencia, módulo, tipo, huella de
contexto, finalidad, correlación y vínculo. En el instante final del reloj del
servidor se revalidan otra vez el documento de actor, su vínculo y la vigencia
de la decisión. Los campos han de ser exactamente los declarados, la garantía
de un perfil interno debe seguir siendo alta y cualquier obligación no
consumida provoca denegación.

La evidencia resultante tampoco autoriza por sí sola un efecto. El adaptador
duradero debe revalidarla y consumir la decisión de forma atómica en la misma
transacción que el cambio de negocio, su auditoría y el evento de salida.
