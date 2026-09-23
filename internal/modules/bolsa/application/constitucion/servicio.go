// Package constitucion convierte un acta de importación Convoca, confirmada
// por RRHH, en una bolsa constituida con su orden (B1→B2 de la ficha de
// adaptación). Lee las filas aceptadas del staging protegido, construye los
// agregados del dominio y los persiste con vínculo al acta; es idempotente
// por acta.
package constitucion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	ErrDependenciasRequeridas = errors.New("bolsa constitucion: dependencias requeridas")
	ErrActaNoEncontrada       = errors.New("bolsa constitucion: acta no encontrada")
	ErrActaSinFilasAceptadas  = errors.New("bolsa constitucion: el acta no tiene filas aceptadas")
	ErrActaNoEsResumen        = errors.New("bolsa constitucion: solo se constituye desde el resumen de personas")
)

const (
	// Entradas gobernadas de la situación inicial. Versión 1 del catálogo de
	// constitución; el llamamiento y las pausas (B7/B8) añaden situaciones.
	EstadoInicial = "disponible"
	CausaInicial  = "constitucion"
)

// Recuperador es la parte del repositorio de importación que devuelve el lote
// aceptado de un acta (descifrando el staging con el protector).
type Recuperador interface {
	RecuperarLote(context.Context, string, string) (importacion.LoteValidado, importacionapp.EstadoImportacion, bool, error)
}

type Reloj func() time.Time

type Servicio struct {
	recuperador Recuperador
	repositorio ports.RepositorioConstitucion
	derivador   DerivadorCandidato
	reloj       Reloj
}

func NuevoServicio(recuperador Recuperador, repositorio ports.RepositorioConstitucion, derivador DerivadorCandidato, reloj Reloj) (*Servicio, error) {
	if recuperador == nil || repositorio == nil || derivador == nil || reloj == nil {
		return nil, ErrDependenciasRequeridas
	}
	return &Servicio{recuperador: recuperador, repositorio: repositorio, derivador: derivador, reloj: reloj}, nil
}

type Solicitud struct {
	HuellaFicheroSHA256 string
	CategoriaRef        string
	ActorRef            string
}

// FilaVinculoCandidato transporta solo el número del staging acreditado y su
// referencia opaca. La participación se resuelve contra el acta en Bolsa SQL.
type FilaVinculoCandidato struct {
	FilaNumero   int    `json:"fila_numero"`
	CandidatoRef string `json:"candidato_ref"`
}

// DerivarFilasVinculo reutiliza el derivador vigente sin recrear la
// constitución ni leer datos de otro módulo. El lote procede del recuperador
// que descifra y verifica la atestación del staging de CONVOCA.
func DerivarFilasVinculo(lote importacion.LoteValidado, derivador DerivadorCandidato) ([]FilaVinculoCandidato, error) {
	if derivador == nil || lote.Validar() != nil || lote.Acta.Esquema != importacion.EsquemaResumenPersona {
		return nil, ports.ErrConstitucionBolsaInvalida
	}
	filas := make([]FilaVinculoCandidato, 0, len(lote.Aceptadas))
	for _, fila := range lote.Aceptadas {
		if fila.Resumen == nil || fila.Numero <= 0 {
			return nil, ports.ErrConstitucionBolsaInvalida
		}
		ref, err := derivador.CandidatoRef(fila.Identidad)
		if err != nil {
			return nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
		}
		filas = append(filas, FilaVinculoCandidato{FilaNumero: fila.Numero, CandidatoRef: ref})
	}
	if len(filas) == 0 {
		return nil, ErrActaSinFilasAceptadas
	}
	return filas, nil
}

// Constituir construye la bolsa y su instantánea desde las filas aceptadas del
// acta (orden: Total descendente, empate por apellidos y nombre), la persiste
// y registra el vínculo `can_* → participación` de cada fila. Como la
// constitución es idempotente por acta y las referencias de participación son
// deterministas, una nueva llamada sobre un acta ya constituida completa los
// vínculos que falten.
func (s *Servicio) Constituir(ctx context.Context, solicitud Solicitud) (ports.ReciboConstitucion, error) {
	if ctx == nil || s == nil {
		return ports.ReciboConstitucion{}, ErrDependenciasRequeridas
	}
	if solicitud.HuellaFicheroSHA256 == "" || solicitud.CategoriaRef == "" || solicitud.ActorRef == "" {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaInvalida
	}
	lote, _, existe, err := s.recuperador.RecuperarLote(ctx, solicitud.HuellaFicheroSHA256, solicitud.CategoriaRef)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	if !existe {
		return ports.ReciboConstitucion{}, ErrActaNoEncontrada
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	constitucion, vinculos, err := construirConstitucion(lote, solicitud.ActorRef, ahora, s.derivador)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	recibo, err := s.repositorio.Constituir(ctx, constitucion)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	recibo.Vinculos, err = s.repositorio.RegistrarVinculos(ctx, recibo.ActaRef, vinculos, ahora)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	return recibo, nil
}

func construirConstitucion(lote importacion.LoteValidado, actorRef string, ahora time.Time, derivador DerivadorCandidato) (ports.Constitucion, []ports.VinculoCandidato, error) {
	acta := lote.Acta
	if acta.Esquema != importacion.EsquemaResumenPersona {
		return ports.Constitucion{}, nil, ErrActaNoEsResumen
	}
	filas := make([]importacion.FilaAceptada, 0, len(lote.Aceptadas))
	for _, fila := range lote.Aceptadas {
		if fila.Resumen != nil {
			filas = append(filas, fila)
		}
	}
	if len(filas) == 0 {
		return ports.Constitucion{}, nil, ErrActaSinFilasAceptadas
	}
	sort.SliceStable(filas, func(i, j int) bool {
		a, b := filas[i], filas[j]
		ta, tb := puntuacion(a.Resumen.Total), puntuacion(b.Resumen.Total)
		if ta != tb {
			return ta > tb
		}
		ka := strings.ToLower(a.Identidad.PrimerApellido + " " + a.Identidad.SegundoApellido + " " + a.Identidad.Nombre)
		kb := strings.ToLower(b.Identidad.PrimerApellido + " " + b.Identidad.SegundoApellido + " " + b.Identidad.Nombre)
		if ka != kb {
			return ka < kb
		}
		return a.Numero < b.Numero
	})
	sufijoActa := sufijoOpaco(acta.ActaRef)
	bolsaRef := acta.BolsaRef
	if bolsaRef == "" {
		bolsaRef = "bolsa:" + claveCategoria(acta.CategoriaRef) + ":" + sufijoActa
	}
	resolucionRef := "resolucion:constitucion:" + sufijoActa
	huellaResolucion := huellaHex(acta.ActaRef, actorRef, ahora.Format(time.RFC3339Nano))
	// Las referencias del acta llevan huellas hexadecimales; el dominio rechaza
	// referencias que parezcan un documento de identidad (ocho dígitos y una
	// letra), así que se derivan formas opacas sin dígitos. El acta real queda
	// enlazada en la propia constitución.
	bolsa := dominio.BolsaConstituida{
		BolsaRef:                  bolsaRef,
		Version:                   1,
		ProcesoRef:                "importacion:convoca:" + sufijoOpaco(acta.ImportacionRef),
		CategoriaRef:              acta.CategoriaRef,
		ListadoDefinitivoRef:      "acta:importacion-convoca:" + sufijoActa,
		VersionListado:            1,
		HuellaListadoSHA256:       acta.HuellaFicheroSHA256,
		ResolucionConstitucionRef: resolucionRef,
		HuellaResolucionSHA256:    huellaResolucion,
		ConstituidaEn:             ahora,
		VigenteDesde:              ahora,
	}
	if err := bolsa.Validar(); err != nil {
		return ports.Constitucion{}, nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
	}
	entradas := make([]dominio.EntradaOrdenBolsa, 0, len(filas))
	vinculos := make([]ports.EntradaConstitucion, 0, len(filas))
	candidatos := make([]ports.VinculoCandidato, 0, len(filas))
	huellaEstado := huellaHex(EstadoInicial, "1")
	huellaCausa := huellaHex(CausaInicial, "1")
	for indice, fila := range filas {
		orden := uint64(indice + 1)
		participacionRef := "participacion:" + sufijoOpaco(bolsaRef+"|"+fila.Identidad.Documento+"|"+strconv.FormatUint(orden, 10))
		sujetoRef := "sujeto:convoca:" + sufijoOpaco(fila.Identidad.Documento+"|"+fila.Identidad.PrimerApellido+"|"+fila.Identidad.SegundoApellido+"|"+fila.Identidad.Nombre)
		decisionRef := resolucionRef + ":" + strconv.FormatUint(orden, 10)
		participacion := dominio.ParticipacionBolsa{
			ParticipacionRef: participacionRef,
			BolsaRef:         bolsaRef,
			SujetoRef:        sujetoRef,
			Version:          1,
			AltaEn:           ahora,
			Situaciones: []dominio.SituacionParticipacionBolsa{{
				Secuencia:            1,
				EstadoClave:          EstadoInicial,
				EstadoVersion:        1,
				HuellaEstadoSHA256:   huellaEstado,
				CausaClave:           CausaInicial,
				CausaVersion:         1,
				HuellaCausaSHA256:    huellaCausa,
				DecisionRef:          decisionRef,
				HuellaDecisionSHA256: huellaHex(decisionRef, huellaResolucion),
				Desde:                ahora,
			}},
		}
		candidatoRef, err := derivador.CandidatoRef(fila.Identidad)
		if err != nil {
			return ports.Constitucion{}, nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
		}
		entradas = append(entradas, dominio.EntradaOrdenBolsa{Orden: orden, Participacion: participacion})
		vinculos = append(vinculos, ports.EntradaConstitucion{Orden: orden, ParticipacionRef: participacionRef, FilaNumero: fila.Numero})
		candidatos = append(candidatos, ports.VinculoCandidato{CandidatoRef: candidatoRef, ParticipacionRef: participacionRef})
	}
	instantanea, err := dominio.NuevaInstantaneaOrdenBolsa(dominio.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:constitucion:" + sufijoActa,
		Version:        1,
		Bolsa:          bolsa,
		ReferidaEn:     ahora,
		GeneradaEn:     ahora,
		Entradas:       entradas,
	})
	if err != nil {
		return ports.Constitucion{}, nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
	}
	return ports.Constitucion{
		ActaRef:      acta.ActaRef,
		ActorRef:     actorRef,
		CategoriaRef: acta.CategoriaRef,
		Bolsa:        bolsa,
		Instantanea:  instantanea,
		Entradas:     vinculos,
		ConfirmadaEn: ahora,
	}, candidatos, nil
}

// puntuacion interpreta el Total del resumen (decimal con punto) para ordenar;
// un valor no numérico cuenta como cero y queda al final entre iguales.
func puntuacion(total string) float64 {
	valor, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(total), ",", "."), 64)
	if err != nil {
		return 0
	}
	return valor
}

func claveCategoria(categoriaRef string) string {
	if indice := strings.LastIndex(categoriaRef, ":"); indice >= 0 {
		return categoriaRef[indice+1:]
	}
	return categoriaRef
}

// sufijoOpaco deriva un identificador de 32 caracteres sin dígitos (el dominio
// rechaza referencias que parezcan un documento de identidad).
func sufijoOpaco(semilla string) string {
	suma := sha256.Sum256([]byte(semilla))
	hexadecimal := hex.EncodeToString(suma[:])[:32]
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return 'g' + (r - '0')
		}
		return r
	}, hexadecimal)
}

func huellaHex(partes ...string) string {
	suma := sha256.Sum256([]byte(strings.Join(partes, "\x1f")))
	return hex.EncodeToString(suma[:])
}
