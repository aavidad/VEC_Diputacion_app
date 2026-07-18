package almacen

import (
	"strings"
	"time"
)

// DatosEvidenciaOperacionAlmacen es la proyeccion escalar de una evidencia
// tecnica. El puerto conserva el tipo de paso y aporta su representacion.
type DatosEvidenciaOperacionAlmacen struct {
	Referencia             string
	ConectorID             string
	EsquemaContexto        string
	EsquemaEsperado        string
	AccionNegocio          string
	Accion                 string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	HuellaPasoSHA256       string
	PasoRef                string
	HuellaDecisionSHA256   string
	Objeto                 ReferenciaObjetoAlmacen
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	RealizadaEn            time.Time
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	HuellaSolicitudHMAC    string
	FundamentoRef          string
	ReintentoIdempotente   bool
}

func ValidarEvidenciaOperacion(datos DatosEvidenciaOperacionAlmacen) error {
	if !ReferenciaOpacaValida(datos.Referencia, 512) || !ReferenciaOpacaValida(datos.ConectorID, 128) ||
		datos.EsquemaContexto != datos.EsquemaEsperado || !ReferenciaOpacaValida(datos.AccionNegocio, 256) ||
		!ReferenciaOpacaValida(datos.Accion, 128) || !ReferenciaOpacaValida(datos.EfectoRef, 512) ||
		!SHA256HexadecimalValido(datos.HuellaPlanEfectoSHA256) || datos.PasoRef == "" ||
		!SHA256HexadecimalValido(datos.HuellaDecisionSHA256) ||
		!ReferenciaOpacaValida(datos.OperacionRef, 512) || !ReferenciaOpacaValida(datos.CorrelacionRef, 512) ||
		!ReferenciaOpacaValida(datos.AutorizacionRef, 512) || !ReferenciaOpacaValida(datos.Finalidad, 1024) ||
		!ReferenciaOpacaValida(datos.Clasificacion, 256) || datos.RealizadaEn.IsZero() ||
		!AccionOperacionValida(datos.Accion) || !ReferenciaOpacaValida(datos.CargaRef, 512) ||
		!HMACSHA256Valido(datos.SujetoSeudonimoHMAC) || !ReferenciaOpacaValida(datos.RecursoRef, 512) ||
		!ReferenciaOpacaValida(datos.ModuloID, 128) || !HMACSHA256Valido(datos.HuellaSolicitudHMAC) ||
		(datos.FundamentoRef != "" && !ReferenciaOpacaValida(datos.FundamentoRef, 512)) ||
		(datos.ReintentoIdempotente && !AccionIdempotente(datos.Accion)) ||
		contieneComodin(datos.AccionNegocio, datos.Accion, datos.EfectoRef, datos.PasoRef) {
		return ErrSolicitudAlmacenInvalida
	}
	tieneHuellaDocumental := datos.HuellaManifiestoSHA256 != "" || datos.HuellaPasoSHA256 != ""
	if tieneHuellaDocumental &&
		(!SHA256HexadecimalValido(datos.HuellaManifiestoSHA256) || !SHA256HexadecimalValido(datos.HuellaPasoSHA256)) {
		return ErrSolicitudAlmacenInvalida
	}
	return datos.Objeto.Validar()
}

// DatosResultadoOperacionObjeto agrupa el estado material y su evidencia sin
// introducir una dependencia desde el paquete canónico hacia los puertos.
type DatosResultadoOperacionObjeto struct {
	Objeto    ObjetoAlmacenado
	Evidencia DatosEvidenciaOperacionAlmacen
}

func ValidarResultadoOperacion(datos DatosResultadoOperacionObjeto) error {
	if err := datos.Objeto.Validar(); err != nil {
		return err
	}
	if err := ValidarEvidenciaOperacion(datos.Evidencia); err != nil {
		return err
	}
	if datos.Objeto.Eliminado || !AccionResultadoValida(datos.Evidencia.Accion) ||
		datos.Objeto.ConectorID != datos.Evidencia.ConectorID ||
		datos.Objeto.Objeto != datos.Evidencia.Objeto ||
		datos.Evidencia.RealizadaEn.Before(datos.Objeto.AlmacenadoEn) {
		return ErrSolicitudAlmacenInvalida
	}
	creacion := AccionCreaObjeto(datos.Evidencia.Accion) && !datos.Evidencia.ReintentoIdempotente
	if creacion {
		if datos.Objeto.EvidenciaCreacionRef != datos.Evidencia.Referencia ||
			!datos.Objeto.AlmacenadoEn.Equal(datos.Evidencia.RealizadaEn) {
			return ErrSolicitudAlmacenInvalida
		}
	} else if datos.Objeto.EvidenciaCreacionRef == datos.Evidencia.Referencia {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

func ValidarResultadoMutacion(
	resultado DatosResultadoOperacionObjeto,
	anterior ObjetoAlmacenado,
	accion, fundamentoRef string,
	evidenciaLigada bool,
) error {
	if anterior.Validar() != nil || anterior.Eliminado || ValidarResultadoOperacion(resultado) != nil ||
		resultado.Evidencia.Accion != accion || resultado.Evidencia.ReintentoIdempotente ||
		resultado.Evidencia.FundamentoRef != fundamentoRef || !evidenciaLigada ||
		resultado.Objeto.Objeto != anterior.Objeto || resultado.Objeto.ConectorID != anterior.ConectorID ||
		resultado.Objeto.Zona != anterior.Zona || resultado.Objeto.MIME != anterior.MIME ||
		resultado.Objeto.Tamano != anterior.Tamano || resultado.Objeto.HuellaSHA256 != anterior.HuellaSHA256 ||
		resultado.Objeto.EvidenciaCreacionRef != anterior.EvidenciaCreacionRef ||
		!resultado.Objeto.AlmacenadoEn.Equal(anterior.AlmacenadoEn) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

func contieneComodin(valores ...string) bool {
	for _, valor := range valores {
		if strings.ContainsRune(valor, '*') {
			return true
		}
	}
	return false
}
