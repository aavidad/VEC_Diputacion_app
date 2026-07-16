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
	FinalidadSelloReservaBaremacion                       FinalidadSelloBaremacion = "reserva_baremacion_v1"
	FinalidadSelloConfirmacionBaremacion                  FinalidadSelloBaremacion = "confirmacion_baremacion_v1"
	FinalidadSelloSobreProbatorioConfirmacionBaremacionV2 FinalidadSelloBaremacion = "sobre_probatorio_confirmacion_baremacion_v2"
	FinalidadSelloManifiestoProbatorioBaremacionV2        FinalidadSelloBaremacion = "manifiesto_probatorio_baremacion_v2"
)

func (f FinalidadSelloBaremacion) valida() bool {
	return f == FinalidadSelloReservaBaremacion || f == FinalidadSelloConfirmacionBaremacion ||
		f == FinalidadSelloSobreProbatorioConfirmacionBaremacionV2 ||
		f == FinalidadSelloManifiestoProbatorioBaremacionV2
}

// SolicitudSellarSelloBaremacion obliga al productor a declarar el dominio
// criptografico que el verificador recibira despues. No debe sustituirse por un
// sellador generico: la finalidad permite seleccionar una clave y un llavero
// historico independientes sin interpretar bytes opacos en el conector.
type SolicitudSellarSelloBaremacion struct {
	Finalidad              FinalidadSelloBaremacion
	RepresentacionCanonica CargaProtegida
}

func (s SolicitudSellarSelloBaremacion) Validar() error {
	if !s.Finalidad.valida() || s.RepresentacionCanonica.Validar() != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

// MaterialCanonicoHMAC devuelve la preimagen contractual exacta:
//
//	HMAC(K_finalidad, finalidad || 0x00 || representacion_canonica)
//
// La finalidad cerrada no puede contener NUL. La representacion conserva
// ademas su dominio funcional para que el artefacto siga siendo autocontenido;
// esta doble declaracion es deliberada y queda congelada por vector de prueba.
func (s SolicitudSellarSelloBaremacion) MaterialCanonicoHMAC() (CargaProtegida, error) {
	if s.Validar() != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	representacion := s.RepresentacionCanonica.Revelar()
	defer func() {
		for indice := range representacion {
			representacion[indice] = 0
		}
	}()
	material := make([]byte, 0, len(s.Finalidad)+1+len(representacion))
	material = append(material, []byte(s.Finalidad)...)
	material = append(material, 0)
	material = append(material, representacion...)
	resultado, err := NuevaCargaProtegida(material)
	for indice := range material {
		material[indice] = 0
	}
	return resultado, err
}

// SelladorSellosBaremacion produce sellos con una finalidad explicita. La
// implementacion debe resolver la clave activa de esa finalidad y devolver el
// identificador de clave en el formato hmac-sha256:<clave>:<hex>.
type SelladorSellosBaremacion interface {
	SellarSelloBaremacion(context.Context, SolicitudSellarSelloBaremacion) (string, error)
}

// SelladorServicioBaremacion es el compuesto requerido por ServicioBaremacion.
// La fachada durable de firma conserva deliberadamente el puerto generico
// SelladorSolicitudBaremacion y no queda acoplada a este contrato transaccional.
type SelladorServicioBaremacion interface {
	SelladorSolicitudBaremacion
	SelladorSellosBaremacion
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

// MaterialCanonicoHMAC reutiliza sin variantes la misma preimagen que el
// productor. El sello solo autoriza la conversion; nunca entra en su propia
// preimagen.
func (s SolicitudVerificarSelloBaremacion) MaterialCanonicoHMAC() (CargaProtegida, error) {
	if s.Validar() != nil {
		return CargaProtegida{}, ErrSelloBaremacionNoAutentico
	}
	return (SolicitudSellarSelloBaremacion{
		Finalidad:              s.Finalidad,
		RepresentacionCanonica: s.RepresentacionCanonica,
	}).MaterialCanonicoHMAC()
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
	return representacionCanonicaConfirmacionBaremacion(
		s, FinalidadSelloConfirmacionBaremacion, nil,
	)
}

// RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2 liga el
// identificador opaco previo al COMMIT y el indice estable con todos los datos
// exactos del intento. Es material nominal del sobre probatorio, no el
// fingerprint semantico de DEC-045 ni una prueba de persistencia.
func RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(
	s IntentoNominalConfirmacionBaremacionV2,
) (CargaProtegida, error) {
	if s.ValidarForma() != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	referencia, indiceHMAC, err := s.IdentificadorOperacion.DatosReconciliacion()
	if err != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	return representacionCanonicaConfirmacionBaremacion(
		s.Confirmacion,
		FinalidadSelloSobreProbatorioConfirmacionBaremacionV2,
		[]string{referencia, indiceHMAC},
	)
}

func representacionCanonicaConfirmacionBaremacion(
	s SolicitudConfirmarCambioBaremacion,
	finalidad FinalidadSelloBaremacion,
	prefijo []string,
) (CargaProtegida, error) {
	if s.Validar() != nil || !finalidad.valida() ||
		(finalidad == FinalidadSelloConfirmacionBaremacion && len(prefijo) != 0) ||
		(finalidad == FinalidadSelloSobreProbatorioConfirmacionBaremacionV2 && len(prefijo) != 2) ||
		(finalidad != FinalidadSelloConfirmacionBaremacion &&
			finalidad != FinalidadSelloSobreProbatorioConfirmacionBaremacionV2) {
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
	partes = append(partes, string(finalidad))
	partes = append(partes, prefijo...)
	partes = append(partes,
		s.Token.Revelar(), string(s.Clase),
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
