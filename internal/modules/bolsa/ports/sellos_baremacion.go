package ports

import (
	"bytes"
	"context"
	"encoding/binary"
	"strconv"
	"time"
)

// FinalidadSelloBaremacion separa criptograficamente reservas y
// confirmaciones. Un sello autentico de una finalidad no vale para la otra.
type FinalidadSelloBaremacion string

const (
	FinalidadSelloReservaBaremacion      FinalidadSelloBaremacion = "reserva_baremacion_v1"
	FinalidadSelloConfirmacionBaremacion FinalidadSelloBaremacion = "confirmacion_baremacion_v1"
)

func (f FinalidadSelloBaremacion) valida() bool {
	return f == FinalidadSelloReservaBaremacion || f == FinalidadSelloConfirmacionBaremacion
}

// SolicitudVerificarSelloBaremacion entrega al componente criptografico la
// representacion canonica completa. El repositorio nunca recibe ni conserva
// material de clave.
type SolicitudVerificarSelloBaremacion struct {
	Finalidad              FinalidadSelloBaremacion
	RepresentacionCanonica CargaProtegida
	SelloHMAC              string
}

func (s SolicitudVerificarSelloBaremacion) Validar() error {
	if !s.Finalidad.valida() || s.RepresentacionCanonica.Validar() != nil ||
		!huellaHMACSHA256Valida(s.SelloHMAC) {
		return ErrSelloBaremacionNoAutentico
	}
	return nil
}

// VerificadorSellosBaremacion debe comparar en tiempo constante y devolver
// error ante clave desconocida, indisponibilidad o sello no autentico.
type VerificadorSellosBaremacion interface {
	VerificarSelloBaremacion(context.Context, SolicitudVerificarSelloBaremacion) error
}

// RepresentacionCanonicaReservaBaremacion cubre todos los datos que fijan la
// reserva, salvo el propio sello. Usa longitudes binarias para que no existan
// concatenaciones ambiguas.
func RepresentacionCanonicaReservaBaremacion(s SolicitudReservarCambioBaremacion) (CargaProtegida, error) {
	if s.Validar() != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	versionPresente, versionRef, versionNumero, versionHuella := "no", "", "0", ""
	if s.VersionEsperada != nil {
		versionPresente = "si"
		versionRef = s.VersionEsperada.BaremacionMeritoRef
		versionNumero = strconv.FormatUint(s.VersionEsperada.Numero, 10)
		versionHuella = s.VersionEsperada.HuellaEstadoSHA256
	}
	partes, err := partesCanonicasAutorizacion(s.Contexto)
	if err != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	partes = append(partes,
		string(FinalidadSelloReservaBaremacion), string(s.Clase), s.ClaveIdempotencia,
		s.BaremacionMeritoRef, versionPresente, versionRef, versionNumero, versionHuella,
		s.SolicitadaEn.UTC().Format(time.RFC3339Nano), s.ExpiraEn.UTC().Format(time.RFC3339Nano),
	)
	return cargaPartesCanonicas(partes)
}

// RepresentacionCanonicaConfirmacionBaremacion cubre token, agregado exacto,
// version, trazabilidad, instante y autorizacion. Por ello alterar un solo dato
// exige un sello nuevo y autentico.
func RepresentacionCanonicaConfirmacionBaremacion(s SolicitudConfirmarCambioBaremacion) (CargaProtegida, error) {
	if s.Validar() != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	huellaAgregado, err := s.Agregado.HuellaEstadoSHA256()
	if err != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	versionPresente, versionRef, versionNumero, versionHuella := "no", "", "0", ""
	if s.VersionEsperada != nil {
		versionPresente = "si"
		versionRef = s.VersionEsperada.BaremacionMeritoRef
		versionNumero = strconv.FormatUint(s.VersionEsperada.Numero, 10)
		versionHuella = s.VersionEsperada.HuellaEstadoSHA256
	}
	manifiestoPresente, manifiestoRef, manifiestoHuella, manifiestoSello := "no", "", "", ""
	if s.Manifiesto != nil {
		manifiestoPresente = "si"
		manifiestoRef = s.Manifiesto.Referencia
		manifiestoHuella = s.Manifiesto.HuellaManifiestoSHA256
		manifiestoSello = s.Manifiesto.SelloManifiestoHMACSHA256
	}
	partes, err := partesCanonicasAutorizacion(s.Contexto)
	if err != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	partes = append(partes,
		string(FinalidadSelloConfirmacionBaremacion), s.Token.Revelar(), string(s.Clase),
		versionPresente, versionRef, versionNumero, versionHuella, huellaAgregado,
		manifiestoPresente, manifiestoRef, manifiestoHuella, manifiestoSello,
		s.Trazabilidad.MotivoClave, s.Trazabilidad.Motivo,
		s.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
	)
	return cargaPartesCanonicas(partes)
}

func partesCanonicasAutorizacion(c ContextoOperacionBaremacion) ([]string, error) {
	if c.Validar() != nil {
		return nil, ErrAutorizacionBaremacionInvalida
	}
	datosEvidencia, err := c.datos.evidencia.Datos()
	if err != nil {
		return nil, ErrAutorizacionBaremacionInvalida
	}
	p := c.Proyeccion()
	partes := []string{
		p.PrincipalRef, p.SujetoRef, p.PerfilActorClave, string(p.MetodoAutenticacion),
		string(p.NivelAutenticacion), string(p.GarantiaMinima), p.AutenticacionRef, p.SesionRef,
		p.AutorizacionRef, datosEvidencia.EsquemaHuella, datosEvidencia.HuellaDecisionSHA256,
		datosEvidencia.VerificadaEn.UTC().Format(time.RFC3339Nano), string(p.Accion),
		string(p.ClaseRecurso), p.RecursoRef, p.FinalidadClave, p.CorrelacionRef,
		p.SesionEmitidaEn.UTC().Format(time.RFC3339Nano), p.SesionValidaHasta.UTC().Format(time.RFC3339Nano),
		p.EmitidaEn.UTC().Format(time.RFC3339Nano), p.ValidaHasta.UTC().Format(time.RFC3339Nano),
		strconv.Itoa(len(p.CamposPermitidos)),
	}
	return append(partes, p.CamposPermitidos...), nil
}

func cargaPartesCanonicas(partes []string) (CargaProtegida, error) {
	var contenido bytes.Buffer
	for _, parte := range partes {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(parte)))
		_, _ = contenido.Write(longitud[:])
		_, _ = contenido.WriteString(parte)
	}
	return NuevaCargaProtegida(contenido.Bytes())
}
