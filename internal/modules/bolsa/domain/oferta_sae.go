package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// La oferta al SAE es selección manual externa a la bolsa constituida. Una
// oferta_publicada (B28) siempre pertenece a una bolsa y no representa este trámite.
const (
	EstadoSAEPreparada           = "preparada"
	EstadoSAEEnviada             = "enviada"
	EstadoSAECandidatosRecibidos = "candidatos_recibidos"
	EstadoSAEEnSeleccion         = "en_seleccion"
	EstadoSAEResuelta            = "resuelta"
	EstadoSAEDesierta            = "desierta"

	AccionSAEEnviar             = "registrar_envio"
	AccionSAERegistrarCandidato = "registrar_candidato"
	AccionSAEConciliarPersona   = "conciliar_persona"
	AccionSAERecibirCandidatos  = "confirmar_recepcion"
	AccionSAEIniciarSeleccion   = "iniciar_seleccion"
	AccionSAEValorar            = "registrar_valoracion"
	AccionSAEResolver           = "resolver"
	AccionSAEDeclararDesierta   = "declarar_desierta"
)

const (
	DocumentoSAESolicitud     = "solicitud_sae"
	DocumentoSAEActa          = "acta_seleccion"
	DocumentoSAEResolucion    = "resolucion"
	ConciliacionSAEPendiente  = "pendiente_conciliacion"
	ConciliacionSAEAcreditada = "acreditada"
)

var (
	ErrOfertaSAEInvalida              = errors.New("bolsa: oferta SAE invalida")
	ErrOfertaSAEVersion               = errors.New("bolsa: version de oferta SAE distinta")
	ErrOfertaSAETransicion            = errors.New("bolsa: transicion de oferta SAE no admitida")
	ErrOfertaSAEClaveReutilizada      = errors.New("bolsa: clave de oferta SAE reutilizada con otro contenido")
	ErrOfertaSAEConciliacionPendiente = errors.New("bolsa: identidad de candidato SAE pendiente de conciliacion")
)

// CatalogoOfertaSAE inmoviliza la versión usada por una oferta. Las opciones
// permiten valoración humana; no fijan un baremo ni calculan ganadores.
type CatalogoOfertaSAE struct {
	Referencia, HuellaSHA256 string
	Version                  int64
	Modalidades              []string
	CriteriosValoracion      []string
	ResultadosValoracion     []string
	ResultadosEntrevista     []string
	Plantillas               map[string]string // tipo documental -> referencia de plantilla
}

func (c CatalogoOfertaSAE) Validar() error {
	if !referenciaPrefijoSAE(c.Referencia, "catalogo:") || c.Version < 1 || len(c.HuellaSHA256) != 64 ||
		!hexadecimalSAE(c.HuellaSHA256) || !listaCatalogoSAE(c.Modalidades) ||
		!listaCatalogoSAE(c.CriteriosValoracion) || !listaCatalogoSAE(c.ResultadosValoracion) ||
		!listaCatalogoSAE(c.ResultadosEntrevista) || len(c.Plantillas) != 3 {
		return ErrOfertaSAEInvalida
	}
	for _, tipo := range []string{DocumentoSAESolicitud, DocumentoSAEActa, DocumentoSAEResolucion} {
		if !referenciaPrefijoSAE(c.Plantillas[tipo], "plantilla:") {
			return ErrOfertaSAEInvalida
		}
	}
	return nil
}

type DatosOfertaSAE struct {
	CategoriaRef, PuestoRef, Modalidad, Duracion, Requisitos string
	NumeroPlazas                                             int
	ExpedienteCTRef                                          string // referencia opaca opcional; Bolsa no consulta CT
}

func (d DatosOfertaSAE) Validar(c CatalogoOfertaSAE) error {
	if c.Validar() != nil || !referenciaPrefijoSAE(d.CategoriaRef, "categoria:") || !referenciaPrefijoSAE(d.PuestoRef, "puesto:") ||
		d.NumeroPlazas < 1 || d.NumeroPlazas > 1000 || !contieneSAE(c.Modalidades, d.Modalidad) ||
		!textoSAE(d.Duracion, 500) || !textoSAE(d.Requisitos, 4000) ||
		(d.ExpedienteCTRef != "" && !referenciaPrefijoSAE(d.ExpedienteCTRef, "expediente:")) {
		return ErrOfertaSAEInvalida
	}
	return nil
}

// El nombre, documento y contacto se guardan mediante referencias a material
// protegido. Una persona externa no necesita ser empleada; su vínculo con la
// Persona canónica se obtiene después por una autoridad, nunca del formulario.
type CandidatoOfertaSAE struct {
	Referencia, PersonaRef, NombreProtegidoRef, DocumentoProtegidoRef, ContactoProtegidoRef string
	EstadoConciliacion                                                                      string
	Acreditacion                                                                            *AcreditacionPersonaSAE
}

func (c CandidatoOfertaSAE) Validar() error {
	if !referenciaPrefijoSAE(c.Referencia, "candidato:") || !referenciaPrefijoSAE(c.NombreProtegidoRef, "dato:") ||
		!referenciaPrefijoSAE(c.DocumentoProtegidoRef, "dato:") || !referenciaPrefijoSAE(c.ContactoProtegidoRef, "dato:") {
		return ErrOfertaSAEInvalida
	}
	switch c.EstadoConciliacion {
	case ConciliacionSAEPendiente:
		if c.PersonaRef != "" || c.Acreditacion != nil {
			return ErrOfertaSAEInvalida
		}
	case ConciliacionSAEAcreditada:
		if c.Acreditacion == nil || c.Acreditacion.PersonaRef != c.PersonaRef || c.Acreditacion.Validar() != nil {
			return ErrOfertaSAEInvalida
		}
	default:
		return ErrOfertaSAEInvalida
	}
	return nil
}

// AcreditacionPersonaSAE es evidencia emitida por la autoridad canónica de
// identidad. Su forma válida no acredita su origen: la aplicación debe recibirla
// solo por el puerto confiable y el repositorio revalidarla antes del efecto.
type AcreditacionPersonaSAE struct {
	PersonaRef, EvidenciaRef  string
	PersonaVersion            int64
	VerificadaEn, ValidaHasta time.Time
}

func (a AcreditacionPersonaSAE) Validar() error {
	if !referenciaPrefijoSAE(a.PersonaRef, "per_") || !referenciaPrefijoSAE(a.EvidenciaRef, "evidencia:") ||
		a.PersonaVersion < 1 || a.VerificadaEn.IsZero() || !a.ValidaHasta.After(a.VerificadaEn) ||
		a.VerificadaEn.Nanosecond()%1000 != 0 || a.ValidaHasta.Nanosecond()%1000 != 0 {
		return ErrOfertaSAEConciliacionPendiente
	}
	return nil
}
func (a AcreditacionPersonaSAE) VigenteEn(instante time.Time) bool {
	return a.Validar() == nil && !instante.Before(a.VerificadaEn) && instante.Before(a.ValidaHasta)
}

type ResultadoCriterioSAE struct{ Criterio, Resultado string }

type ValoracionOfertaSAE struct {
	CandidatoRef, EntrevistaResultado string
	Criterios                         []ResultadoCriterioSAE
	Orden                             int
}

func (v ValoracionOfertaSAE) Validar(c CatalogoOfertaSAE) error {
	if !referenciaSAE(v.CandidatoRef) || v.Orden < 1 || len(v.Criterios) == 0 ||
		!contieneSAE(c.ResultadosEntrevista, v.EntrevistaResultado) {
		return ErrOfertaSAEInvalida
	}
	vistos := make(map[string]bool, len(v.Criterios))
	for _, r := range v.Criterios {
		if vistos[r.Criterio] || !contieneSAE(c.CriteriosValoracion, r.Criterio) ||
			!contieneSAE(c.ResultadosValoracion, r.Resultado) {
			return ErrOfertaSAEInvalida
		}
		vistos[r.Criterio] = true
	}
	return nil
}

type ActuacionOfertaSAE struct {
	Clave, HuellaSHA256, ReciboRef, ActorRef, Accion string
	Version                                          int64
	Instante                                         time.Time
}

type OfertaSAE struct {
	Referencia, Estado, NumeroSAE, FechaEnvio, CandidatoSeleccionadoRef string
	Version                                                             int64
	Datos                                                               DatosOfertaSAE
	Catalogo                                                            CatalogoOfertaSAE
	Candidatos                                                          []CandidatoOfertaSAE
	Valoraciones                                                        []ValoracionOfertaSAE
	Actuaciones                                                         []ActuacionOfertaSAE
}

type CambioOfertaSAE struct {
	Accion, Clave, ReciboRef, ActorRef, NumeroSAE, FechaEnvio string
	CandidatoElegidoRef                                       string
	VersionEsperada                                           int64
	Instante                                                  time.Time
	Candidato                                                 *CandidatoOfertaSAE
	Valoracion                                                *ValoracionOfertaSAE
	Acreditacion                                              *AcreditacionPersonaSAE
}

func NuevaOfertaSAE(ref string, datos DatosOfertaSAE, catalogo CatalogoOfertaSAE) (OfertaSAE, error) {
	if !referenciaPrefijoSAE(ref, "oferta-sae:") || datos.Validar(catalogo) != nil {
		return OfertaSAE{}, ErrOfertaSAEInvalida
	}
	catalogo.Modalidades = append([]string(nil), catalogo.Modalidades...)
	catalogo.CriteriosValoracion = append([]string(nil), catalogo.CriteriosValoracion...)
	catalogo.ResultadosValoracion = append([]string(nil), catalogo.ResultadosValoracion...)
	catalogo.ResultadosEntrevista = append([]string(nil), catalogo.ResultadosEntrevista...)
	plantillas := make(map[string]string, len(catalogo.Plantillas))
	for k, v := range catalogo.Plantillas {
		plantillas[k] = v
	}
	catalogo.Plantillas = plantillas
	return OfertaSAE{Referencia: ref, Estado: EstadoSAEPreparada, Version: 1, Datos: datos, Catalogo: catalogo}, nil
}

// Aplicar devuelve una copia y el recibo. Repetir la misma clave y contenido
// devuelve el recibo anterior incluso si la versión ya avanzó; otro contenido
// con esa clave se rechaza. El repositorio deberá repetirlo transaccionalmente.
func (o OfertaSAE) Aplicar(c CambioOfertaSAE) (OfertaSAE, string, bool, error) {
	if !referenciaPrefijoSAE(o.Referencia, "oferta-sae:") || o.Version < 1 ||
		o.Datos.Validar(o.Catalogo) != nil || !estadoSAEValido(o.Estado) || !o.candidatosConsistentes() || !referenciaSAE(c.Clave) ||
		!referenciaSAE(c.ReciboRef) || !referenciaSAE(c.ActorRef) || c.Instante.IsZero() ||
		c.Instante.Nanosecond()%1000 != 0 || !contenidoCambioSAEValido(c) {
		return OfertaSAE{}, "", false, ErrOfertaSAEInvalida
	}
	huella, err := huellaCambioSAE(c)
	if err != nil {
		return OfertaSAE{}, "", false, ErrOfertaSAEInvalida
	}
	for _, anterior := range o.Actuaciones {
		if anterior.Clave == c.Clave {
			if anterior.HuellaSHA256 != huella || anterior.ActorRef != c.ActorRef || anterior.ReciboRef != c.ReciboRef {
				return OfertaSAE{}, "", false, ErrOfertaSAEClaveReutilizada
			}
			return o.clonar(), anterior.ReciboRef, true, nil
		}
	}
	if c.VersionEsperada != o.Version {
		return OfertaSAE{}, "", false, ErrOfertaSAEVersion
	}
	siguiente := o.clonar()
	if err := siguiente.aplicarAccion(c); err != nil {
		return OfertaSAE{}, "", false, err
	}
	siguiente.Version++
	siguiente.Actuaciones = append(siguiente.Actuaciones, ActuacionOfertaSAE{
		Clave: c.Clave, HuellaSHA256: huella, ReciboRef: c.ReciboRef, ActorRef: c.ActorRef,
		Accion: c.Accion, Version: siguiente.Version, Instante: c.Instante.UTC(),
	})
	return siguiente, c.ReciboRef, false, nil
}

func (o OfertaSAE) candidatosConsistentes() bool {
	referencias, documentos, personas := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, candidato := range o.Candidatos {
		if candidato.Validar() != nil || referencias[candidato.Referencia] || documentos[candidato.DocumentoProtegidoRef] ||
			(candidato.PersonaRef != "" && personas[candidato.PersonaRef]) {
			return false
		}
		referencias[candidato.Referencia], documentos[candidato.DocumentoProtegidoRef] = true, true
		if candidato.PersonaRef != "" {
			personas[candidato.PersonaRef] = true
		}
	}
	return true
}

func (o OfertaSAE) clonar() OfertaSAE {
	copia := o
	copia.Candidatos = append([]CandidatoOfertaSAE(nil), o.Candidatos...)
	for i := range copia.Candidatos {
		if o.Candidatos[i].Acreditacion != nil {
			vinculo := *o.Candidatos[i].Acreditacion
			copia.Candidatos[i].Acreditacion = &vinculo
		}
	}
	copia.Valoraciones = append([]ValoracionOfertaSAE(nil), o.Valoraciones...)
	for i := range copia.Valoraciones {
		copia.Valoraciones[i].Criterios = append([]ResultadoCriterioSAE(nil), o.Valoraciones[i].Criterios...)
	}
	copia.Actuaciones = append([]ActuacionOfertaSAE(nil), o.Actuaciones...)
	copia.Catalogo.Modalidades = append([]string(nil), o.Catalogo.Modalidades...)
	copia.Catalogo.CriteriosValoracion = append([]string(nil), o.Catalogo.CriteriosValoracion...)
	copia.Catalogo.ResultadosValoracion = append([]string(nil), o.Catalogo.ResultadosValoracion...)
	copia.Catalogo.ResultadosEntrevista = append([]string(nil), o.Catalogo.ResultadosEntrevista...)
	copia.Catalogo.Plantillas = make(map[string]string, len(o.Catalogo.Plantillas))
	for k, v := range o.Catalogo.Plantillas {
		copia.Catalogo.Plantillas[k] = v
	}
	return copia
}

func contenidoCambioSAEValido(c CambioOfertaSAE) bool {
	switch c.Accion {
	case AccionSAEEnviar:
		return c.Candidato == nil && c.Valoracion == nil && c.Acreditacion == nil && c.CandidatoElegidoRef == ""
	case AccionSAERegistrarCandidato:
		return c.Candidato != nil && c.Valoracion == nil && c.Acreditacion == nil && c.CandidatoElegidoRef == "" && c.NumeroSAE == "" && c.FechaEnvio == ""
	case AccionSAEConciliarPersona, AccionSAEResolver:
		return c.Candidato == nil && c.Valoracion == nil && referenciaPrefijoSAE(c.CandidatoElegidoRef, "candidato:") && c.NumeroSAE == "" && c.FechaEnvio == ""
	case AccionSAEValorar:
		return c.Candidato == nil && c.Valoracion != nil && c.Acreditacion == nil && c.CandidatoElegidoRef == "" && c.NumeroSAE == "" && c.FechaEnvio == ""
	case AccionSAERecibirCandidatos, AccionSAEIniciarSeleccion, AccionSAEDeclararDesierta:
		return c.Candidato == nil && c.Valoracion == nil && c.Acreditacion == nil && c.CandidatoElegidoRef == "" && c.NumeroSAE == "" && c.FechaEnvio == ""
	}
	return false
}

func (o *OfertaSAE) aplicarAccion(c CambioOfertaSAE) error {
	switch c.Accion {
	case AccionSAEEnviar:
		if o.Estado != EstadoSAEPreparada || !numeroSAEValido(c.NumeroSAE) || !fechaSAE(c.FechaEnvio) {
			break
		}
		o.Estado, o.NumeroSAE, o.FechaEnvio = EstadoSAEEnviada, c.NumeroSAE, c.FechaEnvio
		return nil
	case AccionSAERegistrarCandidato:
		if (o.Estado != EstadoSAEEnviada && o.Estado != EstadoSAECandidatosRecibidos) || c.Candidato == nil ||
			c.Candidato.PersonaRef != "" || c.Candidato.Acreditacion != nil ||
			(c.Candidato.EstadoConciliacion != "" && c.Candidato.EstadoConciliacion != ConciliacionSAEPendiente) {
			break
		}
		candidatoNuevo := *c.Candidato
		candidatoNuevo.EstadoConciliacion = ConciliacionSAEPendiente
		if candidatoNuevo.Validar() != nil {
			break
		}
		for _, candidato := range o.Candidatos {
			if candidato.Referencia == candidatoNuevo.Referencia || candidato.DocumentoProtegidoRef == candidatoNuevo.DocumentoProtegidoRef {
				return ErrOfertaSAEInvalida
			}
		}
		o.Candidatos = append(o.Candidatos, candidatoNuevo)
		return nil
	case AccionSAEConciliarPersona:
		if o.Estado != EstadoSAEEnviada && o.Estado != EstadoSAECandidatosRecibidos && o.Estado != EstadoSAEEnSeleccion {
			break
		}
		if c.Acreditacion == nil || !c.Acreditacion.VigenteEn(c.Instante) {
			return ErrOfertaSAEConciliacionPendiente
		}
		for _, candidato := range o.Candidatos {
			if candidato.Referencia != c.CandidatoElegidoRef && candidato.PersonaRef == c.Acreditacion.PersonaRef {
				return ErrOfertaSAEInvalida
			}
		}
		for i := range o.Candidatos {
			if o.Candidatos[i].Referencia != c.CandidatoElegidoRef {
				continue
			}
			if o.Candidatos[i].PersonaRef != "" && o.Candidatos[i].PersonaRef != c.Acreditacion.PersonaRef {
				return ErrOfertaSAEInvalida
			}
			o.Candidatos[i].PersonaRef = c.Acreditacion.PersonaRef
			o.Candidatos[i].EstadoConciliacion = ConciliacionSAEAcreditada
			vinculo := *c.Acreditacion
			o.Candidatos[i].Acreditacion = &vinculo
			return nil
		}
		return ErrOfertaSAEInvalida
	case AccionSAERecibirCandidatos:
		if o.Estado != EstadoSAEEnviada || len(o.Candidatos) == 0 {
			break
		}
		o.Estado = EstadoSAECandidatosRecibidos
		return nil
	case AccionSAEIniciarSeleccion:
		if o.Estado != EstadoSAECandidatosRecibidos {
			break
		}
		o.Estado = EstadoSAEEnSeleccion
		return nil
	case AccionSAEValorar:
		if o.Estado != EstadoSAEEnSeleccion || c.Valoracion == nil || c.Valoracion.Validar(o.Catalogo) != nil {
			break
		}
		existe := false
		for _, candidato := range o.Candidatos {
			existe = existe || candidato.Referencia == c.Valoracion.CandidatoRef
		}
		if !existe {
			break
		}
		for _, v := range o.Valoraciones {
			if v.CandidatoRef == c.Valoracion.CandidatoRef || v.Orden == c.Valoracion.Orden {
				return ErrOfertaSAEInvalida
			}
		}
		valoracion := *c.Valoracion
		valoracion.Criterios = append([]ResultadoCriterioSAE(nil), c.Valoracion.Criterios...)
		o.Valoraciones = append(o.Valoraciones, valoracion)
		return nil
	case AccionSAEResolver:
		if o.Estado != EstadoSAEEnSeleccion || len(o.Valoraciones) == 0 {
			break
		}
		if c.Acreditacion == nil || !c.Acreditacion.VigenteEn(c.Instante) {
			return ErrOfertaSAEConciliacionPendiente
		}
		for _, v := range o.Valoraciones {
			if v.CandidatoRef != c.CandidatoElegidoRef {
				continue
			}
			for _, candidato := range o.Candidatos {
				if candidato.Referencia != v.CandidatoRef {
					continue
				}
				if candidato.EstadoConciliacion != ConciliacionSAEAcreditada || candidato.Acreditacion == nil ||
					candidato.PersonaRef != c.Acreditacion.PersonaRef {
					return ErrOfertaSAEConciliacionPendiente
				}
				o.Estado, o.CandidatoSeleccionadoRef = EstadoSAEResuelta, candidato.Referencia
				return nil
			}
		}
		return ErrOfertaSAEConciliacionPendiente
	case AccionSAEDeclararDesierta:
		if o.Estado != EstadoSAEEnSeleccion || len(o.Valoraciones) == 0 {
			break
		}
		o.Estado = EstadoSAEDesierta
		return nil
	}
	return ErrOfertaSAETransicion
}

func huellaCambioSAE(c CambioOfertaSAE) (string, error) {
	// Clave, versión e instante son metadatos de operación; la huella vincula
	// acción y contenido para que un replay no introduzca otros datos. La
	// evidencia renovable acredita el efecto nuevo, pero no cambia la identidad
	// semántica de una actuación ya aceptada.
	var persona *struct{ PersonaRef string }
	if c.Acreditacion != nil {
		persona = &struct{ PersonaRef string }{c.Acreditacion.PersonaRef}
	}
	material := struct {
		Accion, Numero, Fecha, CandidatoElegidoRef string
		Candidato                                  *CandidatoOfertaSAE
		Valoracion                                 *ValoracionOfertaSAE
		Acreditacion                               *struct{ PersonaRef string }
	}{c.Accion, c.NumeroSAE, c.FechaEnvio, c.CandidatoElegidoRef, c.Candidato, c.Valoracion, persona}
	b, err := json.Marshal(material)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func referenciaSAE(s string) bool {
	return s != "" && len(s) <= 256 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func referenciaPrefijoSAE(s, prefijo string) bool {
	return strings.HasPrefix(s, prefijo) && len(s) > len(prefijo) && referenciaSAE(s)
}
func textoSAE(s string, max int) bool {
	return s != "" && len(s) <= max && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00")
}
func numeroSAEValido(s string) bool { return textoSAE(s, 200) && !strings.ContainsAny(s, "\r\n\t") }
func estadoSAEValido(s string) bool {
	switch s {
	case EstadoSAEPreparada, EstadoSAEEnviada, EstadoSAECandidatosRecibidos, EstadoSAEEnSeleccion, EstadoSAEResuelta, EstadoSAEDesierta:
		return true
	}
	return false
}
func fechaSAE(s string) bool {
	t, err := time.Parse("2006-01-02", s)
	return err == nil && t.Format("2006-01-02") == s
}
func contieneSAE(lista []string, valor string) bool {
	for _, v := range lista {
		if v == valor {
			return true
		}
	}
	return false
}
func hexadecimalSAE(s string) bool { _, err := hex.DecodeString(s); return err == nil }
func listaCatalogoSAE(lista []string) bool {
	if len(lista) == 0 || len(lista) > 100 {
		return false
	}
	vistos := map[string]bool{}
	for _, v := range lista {
		if !referenciaSAE(v) || vistos[v] {
			return false
		}
		vistos[v] = true
	}
	return true
}
