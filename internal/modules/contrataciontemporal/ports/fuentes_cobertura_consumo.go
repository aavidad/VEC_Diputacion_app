package ports

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const dominioResultadoDurableCobertura = "VEC-CT-EFECTO-COBERTURA-V1"

// DatosOrdenConsumoCobertura liga el efecto durable a la respuesta completa,
// a su verificación independiente y al catálogo gobernado usado para decidir.
type DatosOrdenConsumoCobertura struct {
	PeticionRef           string
	OrganizacionRef       string
	ExpedienteRef         string
	VersionExpediente     uint64
	HuellaPeticionSHA256  string
	HuellaResultadoSHA256 string
	AutoridadRef          string
	Generacion            uint32
	ReciboRespuestaRef    string
	HuellaRespuestaSHA256 string
	Atestacion            AtestacionRespuestaCobertura
	ConfirmacionRespuesta ConfirmacionRespuestaCobertura
	ConfirmacionCatalogo  ConfirmacionPublicacionCobertura
	IdentidadVerificador  IdentidadAutoridadFuenteAnalisis
	EvidenciaVerificador  EvidenciaPublicaAutoridadFuenteAnalisis
}

// ResumenOrdenConsumoCobertura expone únicamente las coordenadas y huellas
// necesarias para ligar una decisión posterior a la evidencia comprobada.
// No contiene atestaciones, claves, credenciales ni texto libre del proveedor.
type ResumenOrdenConsumoCobertura struct {
	PeticionRef, OrganizacionRef, ExpedienteRef               string
	VersionExpediente                                         uint64
	Catalogo                                                  domain.IdentidadCatalogoViasCobertura
	ViaClave                                                  domain.ClaveCatalogo
	Comprobacion                                              domain.ComprobacionCobertura
	OrdenComprobacion                                         uint16
	ComprobacionObligatoria                                   bool
	ProcedenciaClave                                          domain.ClaveCatalogo
	DefinicionFuenteRef, CategoriaRef                         string
	Periodo                                                   domain.PeriodoPrevisto
	SolicitadaEn, EmitidaEn, ValidaHasta                      time.Time
	HuellaPeticionSHA256, HuellaResultadoSHA256               string
	HuellaRespuestaSHA256, AutoridadRef                       string
	Generacion                                                uint32
	ReciboRespuestaRef, VerificadorRef, PublicadorCatalogoRef string
}

type OrdenConsumoCobertura struct {
	datos     *DatosOrdenConsumoCobertura
	solicitud SolicitudConsultarCobertura
	resultado ResultadoConsultaCobertura
}

func NuevaOrdenConsumoCobertura(
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
	confirmacion ConfirmacionRespuestaCobertura,
	confirmacionCatalogo ConfirmacionPublicacionCobertura,
	identidadVerificador IdentidadAutoridadFuenteAnalisis,
	evidenciaVerificador EvidenciaPublicaAutoridadFuenteAnalisis,
) (OrdenConsumoCobertura, error) {
	if resultado.ValidarPara(solicitud) != nil {
		return OrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	huellaRespuesta, errHuellaRespuesta :=
		resultado.preimagen.huellaSHA256()
	huellaPeticion, errHuellaPeticion :=
		huellaPeticionCobertura(solicitud)
	huellaResultado, errHuellaResultado :=
		huellaResultadoDurableCobertura(solicitud, resultado)
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	datos := DatosOrdenConsumoCobertura{
		PeticionRef:           solicitud.PeticionRef,
		OrganizacionRef:       solicitud.OrganizacionRef,
		ExpedienteRef:         solicitud.ExpedienteRef,
		VersionExpediente:     solicitud.VersionExpediente,
		HuellaPeticionSHA256:  huellaPeticion,
		HuellaResultadoSHA256: huellaResultado,
		AutoridadRef:          resultado.atestacion.Metadatos.AutoridadRef,
		Generacion:            resultado.atestacion.Metadatos.Generacion,
		ReciboRespuestaRef:    resultado.atestacion.Metadatos.ReciboRef,
		HuellaRespuestaSHA256: huellaRespuesta,
		Atestacion:            resultado.atestacion,
		ConfirmacionRespuesta: confirmacion,
		ConfirmacionCatalogo:  confirmacionCatalogo,
		IdentidadVerificador:  identidadVerificador,
		EvidenciaVerificador:  evidenciaVerificador,
	}
	if errHuellaRespuesta != nil || errHuellaPeticion != nil ||
		errHuellaResultado != nil || errConfirmacion != nil ||
		datosConfirmacion.HuellaMaterialSHA256 != huellaRespuesta ||
		validarOrdenConsumoCobertura(
			datos,
			solicitud,
			resultado,
			datosConfirmacion.VerificadaEn,
		) != nil {
		return OrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return OrdenConsumoCobertura{
		datos:     &datos,
		solicitud: solicitud,
		resultado: resultado,
	}, nil
}

func huellaResultadoDurableCobertura(
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
) (string, error) {
	datos, err := resultado.Datos()
	huellaPeticion, errHuella := huellaPeticionCobertura(solicitud)
	if err != nil || errHuella != nil ||
		resultado.ValidarPara(solicitud) != nil {
		return "", ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioResultadoDurableCobertura)
	escritor.texto(huellaPeticion)
	escritor.texto(string(datos.Comprobacion.Clave))
	escritor.texto(string(datos.Comprobacion.Resultado))
	escritor.texto(datos.DefinicionFuenteRef)
	contenido, err := escritor.resultado()
	if err != nil {
		return "", ErrResultadoFuenteCoberturaNoConfiable
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func validarOrdenConsumoCobertura(
	datos DatosOrdenConsumoCobertura,
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
	comprobadaEn time.Time,
) error {
	huellaPeticion, errHuellaPeticion :=
		huellaPeticionCobertura(solicitud)
	huellaResultado, errHuellaResultado :=
		huellaResultadoDurableCobertura(solicitud, resultado)
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil ||
		errHuellaPeticion != nil || errHuellaResultado != nil ||
		!domain.ReferenciaOpacaValida(datos.PeticionRef) ||
		datos.PeticionRef != solicitud.PeticionRef ||
		datos.OrganizacionRef != solicitud.OrganizacionRef ||
		datos.ExpedienteRef != solicitud.ExpedienteRef ||
		datos.VersionExpediente != solicitud.VersionExpediente ||
		datos.VersionExpediente > maximoEnteroSeguroFuenteAnalisis ||
		datos.HuellaPeticionSHA256 != huellaPeticion ||
		datos.HuellaResultadoSHA256 != huellaResultado ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaPeticionSHA256) ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaResultadoSHA256) ||
		datos.AutoridadRef != resultado.atestacion.Metadatos.AutoridadRef ||
		datos.Generacion != resultado.atestacion.Metadatos.Generacion ||
		datos.ReciboRespuestaRef !=
			resultado.atestacion.Metadatos.ReciboRef ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaRespuestaSHA256) ||
		datos.IdentidadVerificador.Rol() != RolVerificadorCobertura ||
		datos.IdentidadVerificador.AutoridadRef() == "" ||
		len(datos.IdentidadVerificador.ClavePruebaEd25519()) !=
			ed25519.PublicKeySize ||
		datos.Atestacion.Validar() != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	datosResultado, errDatosResultado := resultado.Datos()
	solicitudVerificacion, err := nuevaSolicitudVerificarRespuestaCobertura(
		datosResultado.HuellaPeticionSHA256,
		resultado.preimagen,
		datos.Atestacion,
	)
	if errDatosResultado != nil || err != nil ||
		datosConfirmacionVerificadorNoCoincide(
			datos.ConfirmacionRespuesta,
			datos.IdentidadVerificador,
		) ||
		evidenciaVerificadorNoPrecedeConfirmacion(
			datos.EvidenciaVerificador,
			datos.ConfirmacionRespuesta,
		) ||
		datos.ConfirmacionRespuesta.ValidarPara(
			solicitudVerificacion,
			comprobadaEn,
			datos.IdentidadVerificador.ClavePruebaEd25519(),
		) != nil ||
		datos.ConfirmacionCatalogo.ValidarPara(
			solicitud,
			comprobadaEn,
		) != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func datosConfirmacionVerificadorNoCoincide(
	confirmacion ConfirmacionRespuestaCobertura,
	identidad IdentidadAutoridadFuenteAnalisis,
) bool {
	datos, err := confirmacion.Datos()
	return err != nil ||
		datos.VerificadorRef != identidad.AutoridadRef() ||
		identidad.Rol() != RolVerificadorCobertura
}

func evidenciaVerificadorNoPrecedeConfirmacion(
	evidencia EvidenciaPublicaAutoridadFuenteAnalisis,
	confirmacion ConfirmacionRespuestaCobertura,
) bool {
	_, _, rol, comprobadaEn, errEvidencia := evidencia.Datos()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	return errEvidencia != nil || errConfirmacion != nil ||
		rol != RolVerificadorCobertura ||
		comprobadaEn.After(datosConfirmacion.VerificadaEn)
}

func (o OrdenConsumoCobertura) Datos() (
	DatosOrdenConsumoCobertura,
	error,
) {
	if o.datos == nil {
		return DatosOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos := *o.datos
	identidad, errIdentidad := NuevaIdentidadAutoridadFuenteAnalisis(
		o.datos.IdentidadVerificador.AutoridadRef(),
		o.datos.IdentidadVerificador.BackendRef(),
		o.datos.IdentidadVerificador.ClavePruebaEd25519(),
		o.datos.IdentidadVerificador.Rol(),
	)
	presentacion, desafio, rol, comprobadaEn, errEvidencia :=
		o.datos.EvidenciaVerificador.Datos()
	evidencia, errNuevaEvidencia :=
		NuevaEvidenciaPublicaAutoridadFuenteAnalisis(
			desafio,
			presentacion,
			rol,
			comprobadaEn,
		)
	if errIdentidad != nil || errEvidencia != nil ||
		errNuevaEvidencia != nil {
		return DatosOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos.IdentidadVerificador = identidad
	datos.EvidenciaVerificador = evidencia
	return datos, nil
}

// ResumenPendienteEn revalida la orden en el instante indicado sin consumirla
// ni producir un recibo durable. La capa transaccional final deberá volver a
// validarla y consumirla de forma atómica con el efecto jurídico.
func (o OrdenConsumoCobertura) ResumenPendienteEn(
	comprobadaEn time.Time,
) (ResumenOrdenConsumoCobertura, error) {
	datos, errDatos := o.Datos()
	datosResultado, errResultado := o.resultado.Datos()
	datosConfirmacion, errConfirmacion := datos.ConfirmacionRespuesta.Datos()
	datosCatalogo, errCatalogo := datos.ConfirmacionCatalogo.Datos()
	if errDatos != nil || errResultado != nil ||
		errConfirmacion != nil || errCatalogo != nil ||
		validarOrdenConsumoCobertura(
			datos,
			o.solicitud,
			o.resultado,
			comprobadaEn,
		) != nil {
		return ResumenOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	comprobacion := datosResultado.Comprobacion
	comprobacion.Detalle = ""
	resumen := ResumenOrdenConsumoCobertura{
		PeticionRef:             datos.PeticionRef,
		OrganizacionRef:         datos.OrganizacionRef,
		ExpedienteRef:           datos.ExpedienteRef,
		VersionExpediente:       datos.VersionExpediente,
		Catalogo:                o.solicitud.Catalogo,
		ViaClave:                o.solicitud.ViaClave,
		Comprobacion:            comprobacion,
		OrdenComprobacion:       o.solicitud.Comprobacion.Orden,
		ComprobacionObligatoria: o.solicitud.Comprobacion.Obligatoria,
		ProcedenciaClave:        o.solicitud.Comprobacion.Procedencia.Clave,
		DefinicionFuenteRef: o.solicitud.Comprobacion.
			Procedencia.DefinicionFuenteRef,
		CategoriaRef:          o.solicitud.CategoriaRef,
		Periodo:               o.solicitud.Periodo,
		SolicitadaEn:          o.solicitud.SolicitadaEn,
		EmitidaEn:             datos.Atestacion.Metadatos.EmitidaEn,
		ValidaHasta:           datos.Atestacion.Metadatos.ValidaHasta,
		HuellaPeticionSHA256:  datos.HuellaPeticionSHA256,
		HuellaResultadoSHA256: datos.HuellaResultadoSHA256,
		HuellaRespuestaSHA256: datos.HuellaRespuestaSHA256,
		AutoridadRef:          datos.AutoridadRef,
		Generacion:            datos.Generacion,
		ReciboRespuestaRef:    datos.ReciboRespuestaRef,
		VerificadorRef:        datosConfirmacion.VerificadorRef,
		PublicadorCatalogoRef: datosCatalogo.PublicadorRef,
	}
	return resumen, nil
}

func (OrdenConsumoCobertura) String() string {
	return "[ORDEN-CONSUMO-COBERTURA-REDACTADA]"
}

func (o OrdenConsumoCobertura) GoString() string { return o.String() }
func (o OrdenConsumoCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, o.String())
}
func (o OrdenConsumoCobertura) LogValue() slog.Value {
	return slog.StringValue(o.String())
}

type ReciboConsumoCobertura struct {
	ConsumoRef                   string
	PeticionRef                  string
	OrganizacionRef              string
	HuellaPeticionSHA256         string
	HuellaResultadoSHA256        string
	AutoridadRef                 string
	Generacion                   uint32
	ReciboRespuestaRef           string
	HuellaRespuestaSHA256        string
	ConsumidaEn                  time.Time
	SolicitudOriginal            SolicitudConsultarCobertura
	ResultadoOriginal            ResultadoConsultaCobertura
	ConfirmacionOriginal         ConfirmacionRespuestaCobertura
	IdentidadVerificadorOriginal IdentidadAutoridadFuenteAnalisis
	EvidenciaVerificadorOriginal EvidenciaPublicaAutoridadFuenteAnalisis
}

func NuevoReciboConsumoCobertura(
	orden OrdenConsumoCobertura,
	consumoRef string,
	consumidaEn time.Time,
) (ReciboConsumoCobertura, error) {
	datos, err := orden.Datos()
	recibo := ReciboConsumoCobertura{
		ConsumoRef:                   consumoRef,
		PeticionRef:                  datos.PeticionRef,
		OrganizacionRef:              datos.OrganizacionRef,
		HuellaPeticionSHA256:         datos.HuellaPeticionSHA256,
		HuellaResultadoSHA256:        datos.HuellaResultadoSHA256,
		AutoridadRef:                 datos.AutoridadRef,
		Generacion:                   datos.Generacion,
		ReciboRespuestaRef:           datos.ReciboRespuestaRef,
		HuellaRespuestaSHA256:        datos.HuellaRespuestaSHA256,
		ConsumidaEn:                  consumidaEn,
		SolicitudOriginal:            orden.solicitud,
		ResultadoOriginal:            orden.resultado,
		ConfirmacionOriginal:         datos.ConfirmacionRespuesta,
		IdentidadVerificadorOriginal: datos.IdentidadVerificador,
		EvidenciaVerificadorOriginal: datos.EvidenciaVerificador,
	}
	if err != nil || recibo.ValidarPara(orden) != nil {
		return ReciboConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return recibo, nil
}

func (r ReciboConsumoCobertura) ValidarPara(
	orden OrdenConsumoCobertura,
) error {
	datos, err := orden.Datos()
	confirmacionActual, errActual := datos.ConfirmacionRespuesta.Datos()
	confirmacionOriginal, errOriginal := r.ConfirmacionOriginal.Datos()
	datosResultadoOriginal, errResultadoOriginal :=
		r.ResultadoOriginal.Datos()
	atestacionOriginal, errAtestacionOriginal :=
		r.ResultadoOriginal.Atestacion()
	solicitudVerificacionOriginal, errSolicitudOriginal :=
		r.ResultadoOriginal.SolicitudVerificacion()
	_, _, rolEvidenciaOriginal, comprobadaEnEvidenciaOriginal,
		errEvidenciaOriginal := r.EvidenciaVerificadorOriginal.Datos()
	huellaPeticionOriginal, errHuellaPeticionOriginal :=
		huellaPeticionCobertura(r.SolicitudOriginal)
	huellaResultadoOriginal, errHuellaResultadoOriginal :=
		huellaResultadoDurableCobertura(
			r.SolicitudOriginal,
			r.ResultadoOriginal,
		)
	huellaRespuestaOriginal, errHuellaRespuestaOriginal :=
		r.ResultadoOriginal.preimagen.huellaSHA256()
	if err != nil ||
		errActual != nil || errOriginal != nil ||
		errResultadoOriginal != nil || errAtestacionOriginal != nil ||
		errEvidenciaOriginal != nil ||
		errSolicitudOriginal != nil || errHuellaPeticionOriginal != nil ||
		errHuellaResultadoOriginal != nil ||
		errHuellaRespuestaOriginal != nil ||
		!domain.ReferenciaOpacaValida(r.ConsumoRef) ||
		r.PeticionRef != datos.PeticionRef ||
		r.OrganizacionRef != datos.OrganizacionRef ||
		r.HuellaPeticionSHA256 != datos.HuellaPeticionSHA256 ||
		r.HuellaResultadoSHA256 != datos.HuellaResultadoSHA256 ||
		r.HuellaPeticionSHA256 != huellaPeticionOriginal ||
		r.HuellaResultadoSHA256 != huellaResultadoOriginal ||
		r.AutoridadRef != atestacionOriginal.Metadatos.AutoridadRef ||
		r.Generacion != atestacionOriginal.Metadatos.Generacion ||
		r.ReciboRespuestaRef != atestacionOriginal.Metadatos.ReciboRef ||
		r.HuellaRespuestaSHA256 != huellaRespuestaOriginal ||
		!instanteFuenteAnalisisCanonico(r.ConsumidaEn) ||
		r.ConsumidaEn.Before(atestacionOriginal.Metadatos.EmitidaEn) ||
		!r.ConsumidaEn.Before(atestacionOriginal.Metadatos.ValidaHasta) ||
		confirmacionOriginal.VerificadorRef !=
			confirmacionActual.VerificadorRef ||
		confirmacionOriginal.VerificadorRef !=
			r.IdentidadVerificadorOriginal.AutoridadRef() ||
		r.IdentidadVerificadorOriginal.Rol() !=
			RolVerificadorCobertura ||
		rolEvidenciaOriginal != RolVerificadorCobertura ||
		comprobadaEnEvidenciaOriginal.After(
			confirmacionOriginal.VerificadaEn,
		) ||
		confirmacionOriginal.HuellaMaterialSHA256 !=
			r.HuellaRespuestaSHA256 ||
		confirmacionOriginal.HuellaPeticionSHA256 !=
			r.HuellaPeticionSHA256 ||
		datosResultadoOriginal.HuellaPeticionSHA256 !=
			r.HuellaPeticionSHA256 ||
		r.ConfirmacionOriginal.ValidarPara(
			solicitudVerificacionOriginal,
			r.ConsumidaEn,
			r.IdentidadVerificadorOriginal.ClavePruebaEd25519(),
		) != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (r ReciboConsumoCobertura) EvidenciaPublicaVerificador() (
	EvidenciaPublicaAutoridadFuenteAnalisis,
	error,
) {
	presentacion, desafio, rol, comprobadaEn, err :=
		r.EvidenciaVerificadorOriginal.Datos()
	if err != nil {
		return EvidenciaPublicaAutoridadFuenteAnalisis{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return NuevaEvidenciaPublicaAutoridadFuenteAnalisis(
		desafio,
		presentacion,
		rol,
		comprobadaEn,
	)
}

func (r ReciboConsumoCobertura) ValidarIdentidadVerificadorOriginal(
	identidad IdentidadAutoridadFuenteAnalisis,
) error {
	if !IdentidadesAutoridadFuenteAnalisisIguales(
		r.IdentidadVerificadorOriginal,
		identidad,
	) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (ReciboConsumoCobertura) String() string {
	return "[RECIBO-CONSUMO-COBERTURA-REDACTADO]"
}

func (r ReciboConsumoCobertura) GoString() string { return r.String() }
func (r ReciboConsumoCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r ReciboConsumoCobertura) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

// ConsumidorCobertura debe imponer unicidad durable por
// (organización, petición). La misma petición y resultado semántico devuelve
// el recibo original aunque se renueven recibos o confirmaciones probatorias.
// Otra semántica o resultado para esa petición devuelve
// ErrRespuestaCoberturaYaConsumida. Además, una evidencia
// (autoridad, generación, recibo) no puede ligarse a otra respuesta o petición.
// El puerto no afirma que exista un adaptador productivo.
type ConsumidorCobertura interface {
	ConsumirCobertura(
		context.Context,
		OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error)
}
