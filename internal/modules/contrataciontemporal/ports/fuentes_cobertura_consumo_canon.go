package ports

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	dominioCanonAtestacionConsumoCobertura   = "VEC-CT-CONSUMO-COBERTURA-ATESTACION-V1"
	dominioCanonConfirmacionConsumoCobertura = "VEC-CT-CONSUMO-COBERTURA-" +
		"CONFIRMACION-TCB-V1"
	dominioCanonCatalogoConsumoCobertura    = "VEC-CT-CONSUMO-COBERTURA-CATALOGO-V1"
	dominioCanonVerificadorConsumoCobertura = "VEC-CT-CONSUMO-COBERTURA-" +
		"VERIFICADOR-V1"
	dominioCanonResumenConsumoCobertura       = "VEC-CT-CONSUMO-COBERTURA-RESUMEN-V1"
	redaccionPruebasCanonicasConsumoCobertura = "[PRUEBAS-CANONICAS-" +
		"CONSUMO-COBERTURA-REDACTADAS]"
)

var ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida = errors.New(
	"contratacion temporal: serializacion de pruebas canonicas prohibida",
)

type bloqueoSerializacionPruebasCanonicasConsumoCobertura struct{}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) MarshalJSON() (
	[]byte,
	error,
) {
	return nil, ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (*bloqueoSerializacionPruebasCanonicasConsumoCobertura) UnmarshalJSON(
	[]byte,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (*bloqueoSerializacionPruebasCanonicasConsumoCobertura) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) MarshalText() (
	[]byte,
	error,
) {
	return nil, ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (*bloqueoSerializacionPruebasCanonicasConsumoCobertura) UnmarshalText(
	[]byte,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) MarshalBinary() (
	[]byte,
	error,
) {
	return nil, ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (*bloqueoSerializacionPruebasCanonicasConsumoCobertura) UnmarshalBinary(
	[]byte,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) GobEncode() (
	[]byte,
	error,
) {
	return nil, ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (*bloqueoSerializacionPruebasCanonicasConsumoCobertura) GobDecode(
	[]byte,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) MarshalCBOR() (
	[]byte,
	error,
) {
	return nil, ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (*bloqueoSerializacionPruebasCanonicasConsumoCobertura) UnmarshalCBOR(
	[]byte,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) MarshalYAML() (
	any,
	error,
) {
	return nil, ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (*bloqueoSerializacionPruebasCanonicasConsumoCobertura) UnmarshalYAML(
	func(any) error,
) error {
	return ErrSerializacionPruebasCanonicasConsumoCoberturaProhibida
}

func (bloqueoSerializacionPruebasCanonicasConsumoCobertura) String() string {
	return redaccionPruebasCanonicasConsumoCobertura
}

func (b bloqueoSerializacionPruebasCanonicasConsumoCobertura) GoString() string {
	return b.String()
}

func (b bloqueoSerializacionPruebasCanonicasConsumoCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, b.String())
}

func (b bloqueoSerializacionPruebasCanonicasConsumoCobertura) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

// DatosPruebasCanonicasOrdenConsumoCobertura contiene copias de los siete
// materiales exactos que la persistencia C1 conserva y resume. No es un codec
// ni una orden: solo puede obtenerse desde una orden ya validada.
type DatosPruebasCanonicasOrdenConsumoCobertura struct {
	bloqueoSerializacionPruebasCanonicasConsumoCobertura
	Peticion        []byte
	Resultado       []byte
	Atestacion      []byte
	ConfirmacionTCB []byte
	Catalogo        []byte
	Verificador     []byte
	Resumen         []byte
}

// PruebasCanonicasOrdenConsumoCobertura es una proyección nominal, opaca y
// defensiva. Impide que un adaptador reconstruya pruebas con JSON o con una
// representación distinta de la que validó el núcleo.
type PruebasCanonicasOrdenConsumoCobertura struct {
	bloqueoSerializacionPruebasCanonicasConsumoCobertura
	datos *DatosPruebasCanonicasOrdenConsumoCobertura
}

// PruebasCanonicas deriva las siete pruebas únicamente de los valores privados
// de la orden. La validez temporal operativa se vuelve a comprobar al crear el
// fragmento TCB para el instante del efecto.
func (o OrdenConsumoCobertura) PruebasCanonicas() (
	PruebasCanonicasOrdenConsumoCobertura,
	error,
) {
	datos, errDatos := o.Datos()
	datosConfirmacion, errConfirmacion := datos.ConfirmacionRespuesta.Datos()
	resumen, errResumen := o.ResumenPendienteEn(
		datosConfirmacion.VerificadaEn,
	)
	if errDatos != nil || errConfirmacion != nil || errResumen != nil {
		return PruebasCanonicasOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}

	peticion, errPeticion := canonPeticionCobertura(o.solicitud)
	resultado, errResultado := o.resultado.preimagen.Bytes()
	atestacion, errAtestacion := canonAtestacionConsumoCobertura(
		datos.Atestacion,
	)
	confirmacion, errConfirmacionCanon :=
		canonConfirmacionConsumoCobertura(datos.ConfirmacionRespuesta)
	catalogo, errCatalogo := canonCatalogoConsumoCobertura(
		datos.ConfirmacionCatalogo,
	)
	verificador, errVerificador := canonVerificadorConsumoCobertura(
		datos.IdentidadVerificador,
		datos.EvidenciaVerificador,
	)
	resumenCanon, errResumenCanon := canonResumenConsumoCobertura(resumen)
	pruebas := DatosPruebasCanonicasOrdenConsumoCobertura{
		Peticion: peticion, Resultado: resultado, Atestacion: atestacion,
		ConfirmacionTCB: confirmacion, Catalogo: catalogo,
		Verificador: verificador, Resumen: resumenCanon,
	}
	if errPeticion != nil || errResultado != nil || errAtestacion != nil ||
		errConfirmacionCanon != nil || errCatalogo != nil ||
		errVerificador != nil || errResumenCanon != nil ||
		validarDatosPruebasCanonicasConsumoCobertura(pruebas) != nil {
		return PruebasCanonicasOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	copia := clonarDatosPruebasCanonicasConsumoCobertura(pruebas)
	return PruebasCanonicasOrdenConsumoCobertura{datos: &copia}, nil
}

// Datos entrega una copia nueva; modificarla no altera la orden ni una
// proyección posterior.
func (p PruebasCanonicasOrdenConsumoCobertura) Datos() (
	DatosPruebasCanonicasOrdenConsumoCobertura,
	error,
) {
	if p.datos == nil ||
		validarDatosPruebasCanonicasConsumoCobertura(*p.datos) != nil {
		return DatosPruebasCanonicasOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return clonarDatosPruebasCanonicasConsumoCobertura(*p.datos), nil
}

func validarDatosPruebasCanonicasConsumoCobertura(
	datos DatosPruebasCanonicasOrdenConsumoCobertura,
) error {
	for _, prueba := range [][]byte{
		datos.Peticion, datos.Resultado, datos.Atestacion,
		datos.ConfirmacionTCB, datos.Catalogo, datos.Verificador, datos.Resumen,
	} {
		if len(prueba) == 0 || len(prueba) > 64*1024 {
			return ErrResultadoFuenteCoberturaNoConfiable
		}
	}
	return nil
}

func clonarDatosPruebasCanonicasConsumoCobertura(
	datos DatosPruebasCanonicasOrdenConsumoCobertura,
) DatosPruebasCanonicasOrdenConsumoCobertura {
	return DatosPruebasCanonicasOrdenConsumoCobertura{
		Peticion:        append([]byte(nil), datos.Peticion...),
		Resultado:       append([]byte(nil), datos.Resultado...),
		Atestacion:      append([]byte(nil), datos.Atestacion...),
		ConfirmacionTCB: append([]byte(nil), datos.ConfirmacionTCB...),
		Catalogo:        append([]byte(nil), datos.Catalogo...),
		Verificador:     append([]byte(nil), datos.Verificador...),
		Resumen:         append([]byte(nil), datos.Resumen...),
	}
}

func canonAtestacionConsumoCobertura(
	atestacion AtestacionRespuestaCobertura,
) ([]byte, error) {
	if atestacion.Validar() != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioCanonAtestacionConsumoCobertura)
	escritor.texto(atestacion.Metadatos.AutoridadRef)
	escritor.entero64(uint64(atestacion.Metadatos.Generacion))
	escritor.texto(atestacion.Metadatos.ReciboRef)
	escritor.instante(atestacion.Metadatos.EmitidaEn)
	escritor.instante(atestacion.Metadatos.ValidaHasta)
	escritor.texto(atestacion.SelloHMAC)
	return resultadoCanonConsumoCobertura(escritor)
}

func canonConfirmacionConsumoCobertura(
	confirmacion ConfirmacionRespuestaCobertura,
) ([]byte, error) {
	datos, err := confirmacion.Datos()
	if err != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	firma := append([]byte(nil), datos.FirmaEd25519...)
	datos.FirmaEd25519 = nil
	preimagen, err := canonConfirmacionRespuestaCobertura(datos)
	if err != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioCanonConfirmacionConsumoCobertura)
	escritor.texto(string(preimagen))
	escritor.texto(string(firma))
	return resultadoCanonConsumoCobertura(escritor)
}

func canonCatalogoConsumoCobertura(
	confirmacion ConfirmacionPublicacionCobertura,
) ([]byte, error) {
	datos, err := confirmacion.Datos()
	catalogo, errCatalogo := domain.RestaurarCatalogoViasCobertura(
		datos.Publicacion,
	)
	if err != nil || errCatalogo != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	publicacion := catalogo.Publicacion()
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioCanonCatalogoConsumoCobertura)
	escritor.texto(datos.PublicadorRef)
	escritor.instante(datos.VerificadaEn)
	escritor.texto(publicacion.Canon.Dominio)
	escritor.entero16(publicacion.Canon.VersionEsquema)
	escritor.texto(publicacion.Canon.Algoritmo)
	escritor.texto(publicacion.Referencia)
	escritor.entero64(publicacion.Version)
	escritor.texto(publicacion.HuellaSHA256)
	escritor.instante(publicacion.PublicadoEn)
	escritor.instante(publicacion.Vigencia.Desde)
	escritor.booleano(!publicacion.Vigencia.Hasta.IsZero())
	if !publicacion.Vigencia.Hasta.IsZero() {
		escritor.instante(publicacion.Vigencia.Hasta)
	}
	escritor.texto(publicacion.ProcedenciaRef)
	escritor.entero64(uint64(len(publicacion.Vias)))
	for _, via := range publicacion.Vias {
		escritor.texto(string(via.Clave))
		escritor.entero16(via.Orden)
		escritor.entero64(uint64(len(via.Comprobaciones)))
		for _, comprobacion := range via.Comprobaciones {
			escritor.texto(string(comprobacion.Clave))
			escritor.entero16(comprobacion.Orden)
			escritor.booleano(comprobacion.Obligatoria)
			escritor.texto(string(comprobacion.Procedencia.Clave))
			escritor.texto(comprobacion.Procedencia.DefinicionFuenteRef)
		}
	}
	return resultadoCanonConsumoCobertura(escritor)
}

func canonVerificadorConsumoCobertura(
	identidad IdentidadAutoridadFuenteAnalisis,
	evidencia EvidenciaPublicaAutoridadFuenteAnalisis,
) ([]byte, error) {
	datos, err := evidencia.DatosPublicos()
	if err != nil || identidad.Rol() != RolVerificadorCobertura ||
		identidad.AutoridadRef() != datos.CredencialDatos.AutoridadRef ||
		identidad.BackendRef() != datos.CredencialDatos.BackendRef ||
		identidad.Rol() != datos.Rol {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioCanonVerificadorConsumoCobertura)
	escritor.texto(identidad.AutoridadRef())
	escritor.texto(identidad.BackendRef())
	escritor.texto(string(identidad.ClavePruebaEd25519()))
	escritor.texto(string(identidad.Rol()))
	escribirCredencialCanonConsumoCobertura(escritor, datos.CredencialDatos)
	escritor.texto(string(datos.FirmaInstitucional))
	escritor.texto(string(datos.PruebaPosesion))
	escritor.texto(string(datos.Desafio))
	escritor.texto(string(datos.Rol))
	escritor.instante(datos.ComprobadaEn)
	return resultadoCanonConsumoCobertura(escritor)
}

func escribirCredencialCanonConsumoCobertura(
	escritor *escritorCanonFuenteAnalisis,
	datos DatosCredencialAutoridadFuenteAnalisis,
) {
	escritor.texto(datos.RaizClaveID)
	escritor.texto(datos.AutoridadRef)
	escritor.texto(datos.BackendRef)
	escritor.texto(datos.OrganizacionRef)
	escritor.texto(datos.Audiencia)
	escritor.texto(string(datos.Rol))
	escritor.entero64(datos.Serie)
	escritor.entero64(uint64(datos.Generacion))
	escritor.texto(string(datos.ClavePruebaEd25519))
	escritor.instante(datos.EmitidaEn)
	escritor.instante(datos.ValidaHasta)
}

func canonResumenConsumoCobertura(
	resumen ResumenOrdenConsumoCobertura,
) ([]byte, error) {
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioCanonResumenConsumoCobertura)
	escritor.texto(resumen.PeticionRef)
	escritor.texto(resumen.OrganizacionRef)
	escritor.texto(resumen.ExpedienteRef)
	escritor.entero64(resumen.VersionExpediente)
	escritor.texto(resumen.Catalogo.Referencia)
	escritor.entero64(resumen.Catalogo.Version)
	escritor.texto(resumen.Catalogo.HuellaSHA256)
	escritor.texto(string(resumen.ViaClave))
	escritor.texto(string(resumen.Comprobacion.Clave))
	escritor.texto(string(resumen.Comprobacion.Resultado))
	escritor.entero16(resumen.OrdenComprobacion)
	escritor.booleano(resumen.ComprobacionObligatoria)
	escritor.texto(string(resumen.ProcedenciaClave))
	escritor.texto(resumen.DefinicionFuenteRef)
	escritor.texto(resumen.CategoriaRef)
	escritor.instante(resumen.Periodo.Inicio)
	escritor.instante(resumen.Periodo.Fin)
	escritor.instante(resumen.SolicitadaEn)
	escritor.instante(resumen.EmitidaEn)
	escritor.instante(resumen.ValidaHasta)
	escritor.texto(resumen.HuellaPeticionSHA256)
	escritor.texto(resumen.HuellaResultadoSHA256)
	escritor.texto(resumen.HuellaRespuestaSHA256)
	escritor.texto(resumen.AutoridadRef)
	escritor.entero64(uint64(resumen.Generacion))
	escritor.texto(resumen.ReciboRespuestaRef)
	escritor.texto(resumen.VerificadorRef)
	escritor.texto(resumen.PublicadorCatalogoRef)
	return resultadoCanonConsumoCobertura(escritor)
}

func resultadoCanonConsumoCobertura(
	escritor *escritorCanonFuenteAnalisis,
) ([]byte, error) {
	contenido, err := escritor.resultado()
	if err != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	return contenido, nil
}
