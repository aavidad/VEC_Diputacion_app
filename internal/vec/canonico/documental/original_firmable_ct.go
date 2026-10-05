package documental

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	AccionReservaOriginalFirmable      = "documentos.original_firmable.reservar"
	AccionConfirmacionOriginalFirmable = "documentos.original_firmable.confirmar"
	FinalidadOriginalFirmable          = "custodiar_original_firmable"
	AudienciaOriginalFirmable          = "vec_documentos.operacion.v1"
	limitePDFOriginal                  = 1 << 20
)

var ErrOriginalFirmableInvalido = errors.New("documentos: original firmable invalido")

var (
	referenciaHashOriginal = regexp.MustCompile(`^ref:[0-9a-f]{64}$`)
	referenciaUUIDOriginal = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	moduloOriginal         = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)
	huellaOriginal         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func referenciaOpacaOriginal(s string) bool {
	return (referenciaHashOriginal.MatchString(s) && s != "ref:"+strings.Repeat("0", 64)) || referenciaUUIDOriginal.MatchString(s)
}

func referenciaObjetoOriginal(s string) bool {
	if len(s) < 3 || len(s) > 512 || strings.ContainsAny(s, " \t\r\n/\\\x00*?%") || strings.Contains(s, "..") {
		return false
	}
	for _, r := range s {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

func versionObjetoOriginal(s string) bool {
	if len(s) < 1 || len(s) > 512 || strings.ContainsAny(s, " \t\r\n/\\\x00*?%") || strings.Contains(s, "..") {
		return false
	}
	for _, r := range s {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

// DatosReservaOriginal es la preimagen neutral que compromete el PDF y su
// política antes de cualquier efecto en el almacén.
type DatosReservaOriginal struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME, HuellaSHA256                                      string
	Tamano                                                  int64
	PoliticaRef, HuellaPoliticaSHA256                       string
	VersionPolitica                                         uint64
	Proteccion, EstadoPolitica                              string
	ConservacionHasta                                       time.Time
}

func (d DatosReservaOriginal) Preimagen() ([]byte, error) {
	if !referenciaOpacaOriginal(d.ID) || !referenciaOpacaOriginal(d.ClaveIdempotencia) ||
		!moduloOriginal.MatchString(d.ModuloID) || !referenciaOpacaOriginal(d.ExpedienteRef) ||
		!referenciaOpacaOriginal(d.TipoRef) || d.Version == 0 || d.Version > 9007199254740991 ||
		d.MIME != "application/pdf" || !huellaOriginal.MatchString(d.HuellaSHA256) ||
		d.Tamano < 1 || d.Tamano > limitePDFOriginal || !referenciaOpacaOriginal(d.PoliticaRef) ||
		d.VersionPolitica == 0 || d.VersionPolitica > 9007199254740991 ||
		!huellaOriginal.MatchString(d.HuellaPoliticaSHA256) ||
		(d.Proteccion != "conservacion" && d.Proteccion != "bloqueo") ||
		(d.EstadoPolitica != "aprobada" && d.EstadoPolitica != "provisional") ||
		(d.EstadoPolitica == "provisional" && d.Proteccion != "conservacion") ||
		d.ConservacionHasta.IsZero() {
		return nil, ErrOriginalFirmableInvalido
	}
	return json.Marshal(struct {
		Accion            string `json:"accion"`
		ID                string `json:"id"`
		Clave             string `json:"clave_idempotencia"`
		Modulo            string `json:"modulo_id"`
		Expediente        string `json:"expediente_ref"`
		Tipo              string `json:"tipo_ref"`
		Version           uint64 `json:"version"`
		MIME              string `json:"mime"`
		Tamano            int64  `json:"tamano"`
		Huella            string `json:"huella_sha256"`
		Politica          string `json:"politica_ref"`
		VersionPolitica   uint64 `json:"version_politica"`
		HuellaPolitica    string `json:"huella_politica_sha256"`
		Proteccion        string `json:"proteccion"`
		ConservacionHasta string `json:"conservacion_hasta"`
		EstadoPolitica    string `json:"estado_politica"`
	}{AccionReservaOriginalFirmable, d.ID, d.ClaveIdempotencia, d.ModuloID,
		d.ExpedienteRef, d.TipoRef, d.Version, d.MIME, d.Tamano, d.HuellaSHA256,
		d.PoliticaRef, d.VersionPolitica, d.HuellaPoliticaSHA256, d.Proteccion,
		d.ConservacionHasta.UTC().Format(time.RFC3339Nano), d.EstadoPolitica})
}

type IntentoOriginalFirmable struct {
	ReservaRef, Estado, DocumentoID, HuellaSHA256, ClaveAlmacenRef string
	Numero                                                         uint64
}

func (i IntentoOriginalFirmable) ValidoContra(d DatosReservaOriginal) bool {
	return referenciaOpacaOriginal(i.ReservaRef) && i.DocumentoID == d.ID &&
		i.HuellaSHA256 == d.HuellaSHA256 && referenciaOpacaOriginal(i.ClaveAlmacenRef) &&
		i.Numero > 0 && i.Numero <= 9007199254740991 &&
		(i.Estado == "pendiente" || i.Estado == "confirmado")
}

type ObjetoOriginalFirmable struct {
	ClaveAlmacenRef          string     `json:"clave_almacen_ref"`
	ObjetoRef                string     `json:"objeto_ref"`
	ObjetoVersion            string     `json:"objeto_version"`
	ConectorRef              string     `json:"conector_ref"`
	ReciboObjetoRef          string     `json:"recibo_objeto_ref"`
	ReciboObjetoHuellaSHA256 string     `json:"recibo_objeto_huella_sha256"`
	RetenidoHasta            *time.Time `json:"retenido_hasta"`
	Inmovilizado             bool       `json:"inmovilizado"`
	MIME                     string     `json:"mime"`
	Tamano                   int64      `json:"tamano"`
	HuellaSHA256             string     `json:"huella_sha256"`
}

func (o ObjetoOriginalFirmable) Valido() bool {
	return referenciaOpacaOriginal(o.ClaveAlmacenRef) && referenciaObjetoOriginal(o.ObjetoRef) &&
		versionObjetoOriginal(o.ObjetoVersion) && moduloOriginal.MatchString(o.ConectorRef) &&
		referenciaObjetoOriginal(o.ReciboObjetoRef) && huellaOriginal.MatchString(o.ReciboObjetoHuellaSHA256) &&
		o.MIME == "application/pdf" && o.Tamano >= 1 && o.Tamano <= limitePDFOriginal &&
		huellaOriginal.MatchString(o.HuellaSHA256) &&
		(o.RetenidoHasta == nil || (!o.RetenidoHasta.IsZero() && o.RetenidoHasta.Location() == time.UTC))
}

type DatosConfirmacionOriginal struct {
	Intento IntentoOriginalFirmable
	Objeto  ObjetoOriginalFirmable
}

func (d DatosConfirmacionOriginal) Preimagen() ([]byte, error) {
	i := d.Intento
	if !referenciaOpacaOriginal(i.ReservaRef) || i.Estado != "pendiente" ||
		!referenciaOpacaOriginal(i.DocumentoID) || !huellaOriginal.MatchString(i.HuellaSHA256) ||
		i.Numero == 0 || i.Numero > 9007199254740991 || !referenciaOpacaOriginal(i.ClaveAlmacenRef) ||
		!d.Objeto.Valido() || d.Objeto.ClaveAlmacenRef != i.ClaveAlmacenRef ||
		d.Objeto.HuellaSHA256 != i.HuellaSHA256 {
		return nil, ErrOriginalFirmableInvalido
	}
	return json.Marshal(struct {
		Accion     string                 `json:"accion"`
		ReservaRef string                 `json:"reserva_ref"`
		IntentoNum uint64                 `json:"intento_num"`
		Clave      string                 `json:"clave_almacen_ref"`
		ID         string                 `json:"id"`
		Huella     string                 `json:"huella_sha256"`
		Objeto     ObjetoOriginalFirmable `json:"objeto"`
	}{AccionConfirmacionOriginalFirmable, i.ReservaRef, i.Numero,
		i.ClaveAlmacenRef, i.DocumentoID, i.HuellaSHA256, d.Objeto})
}

type DatosAutorizacionOriginal struct {
	MaterialValido               bool
	Accion, Finalidad            string
	RecursoRef, AmbitoRef        string
	PrincipalID, PerfilActivoRef string
	CorrelacionRef               string
	Audiencia, Operacion         string
	EfectoRef                    string
	EmitidaEn, ExpiraEn, Ahora   time.Time
}

func (d DatosAutorizacionOriginal) Valida() bool {
	return d.MaterialValido &&
		(d.Accion == AccionReservaOriginalFirmable || d.Accion == AccionConfirmacionOriginalFirmable) &&
		d.Finalidad == FinalidadOriginalFirmable && referenciaOpacaOriginal(d.RecursoRef) &&
		referenciaOpacaOriginal(d.AmbitoRef) && referenciaObjetoOriginal(d.PrincipalID) &&
		referenciaObjetoOriginal(d.PerfilActivoRef) && referenciaObjetoOriginal(d.CorrelacionRef) &&
		d.Audiencia == AudienciaOriginalFirmable && d.Operacion == d.Accion &&
		d.EfectoRef == d.RecursoRef && !d.Ahora.IsZero() && !d.EmitidaEn.IsZero() &&
		!d.ExpiraEn.IsZero() && !d.Ahora.Before(d.EmitidaEn) && d.Ahora.Before(d.ExpiraEn)
}

func HuellaReciboObjetoOriginal(recibo []byte) string {
	suma := sha256.Sum256(recibo)
	return hex.EncodeToString(suma[:])
}
