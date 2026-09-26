// Package constitucion convierte una lista definitiva autorizada, con el
// orden ya resuelto por su fuente, en una bolsa constituida.
package constitucion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	ErrDependenciasRequeridas = errors.New("bolsa constitucion: dependencias requeridas")
	ErrActaNoEncontrada       = errors.New("bolsa constitucion: acta no encontrada")
	ErrActaSinFilasAceptadas  = errors.New("bolsa constitucion: el acta no tiene filas aceptadas")
	ErrActaNoEsResumen        = errors.New("bolsa constitucion: solo se constituye desde el resumen de personas")
	ErrListaNoEncontrada      = errors.New("bolsa constitucion: lista definitiva no encontrada")
	ErrListaSinPosiciones     = errors.New("bolsa constitucion: lista definitiva sin posiciones")
)

const (
	// Entradas gobernadas de la situación inicial. Versión 1 del catálogo de
	// constitución; el llamamiento y las pausas (B7/B8) añaden situaciones.
	EstadoInicial = "disponible"
	CausaInicial  = "constitucion"
)

type Reloj func() time.Time

type Servicio struct {
	recuperador ports.RecuperadorListaDefinitiva
	repositorio ports.RepositorioConstitucion
	reloj       Reloj
}

func NuevoServicio(recuperador ports.RecuperadorListaDefinitiva, repositorio ports.RepositorioConstitucion, reloj Reloj) (*Servicio, error) {
	if recuperador == nil || repositorio == nil || reloj == nil {
		return nil, ErrDependenciasRequeridas
	}
	return &Servicio{recuperador: recuperador, repositorio: repositorio, reloj: reloj}, nil
}

type Solicitud struct {
	Fuente       ports.FuenteListaDefinitiva
	Referencia   string
	CategoriaRef string
	ActorRef     string
}

// Constituir usa exclusivamente las posiciones ya aprobadas por la fuente.
// El repositorio conserva la clave de replay de la lista y sus vínculos.
func (s *Servicio) Constituir(ctx context.Context, solicitud Solicitud) (ports.ReciboConstitucion, error) {
	if ctx == nil || s == nil {
		return ports.ReciboConstitucion{}, ErrDependenciasRequeridas
	}
	if solicitud.Referencia == "" || solicitud.CategoriaRef == "" || solicitud.ActorRef == "" ||
		(solicitud.Fuente != ports.FuenteImportacionConvoca && solicitud.Fuente != ports.FuenteSeleccionNativa) {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaInvalida
	}
	lista, existe, err := s.recuperador.RecuperarListaDefinitiva(ctx, ports.ConsultaListaDefinitiva{
		Fuente: solicitud.Fuente, Referencia: solicitud.Referencia, CategoriaRef: solicitud.CategoriaRef,
	})
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	if !existe {
		if solicitud.Fuente == ports.FuenteSeleccionNativa {
			return ports.ReciboConstitucion{}, ErrListaNoEncontrada
		}
		return ports.ReciboConstitucion{}, ErrActaNoEncontrada
	}
	if lista.Fuente != solicitud.Fuente || lista.CategoriaRef != solicitud.CategoriaRef ||
		(lista.Fuente == ports.FuenteSeleccionNativa && lista.Referencia != solicitud.Referencia) ||
		(lista.Fuente == ports.FuenteImportacionConvoca && lista.HuellaListadoSHA256 != solicitud.Referencia) {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaInvalida
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	constitucion, vinculos, err := construirConstitucion(lista, solicitud.ActorRef, ahora)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	recibo, err := s.repositorio.Constituir(ctx, constitucion)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	if len(vinculos) > 0 {
		recibo.Vinculos, err = s.repositorio.RegistrarVinculos(ctx, recibo.ActaRef, vinculos, ahora)
		if err != nil {
			return ports.ReciboConstitucion{}, err
		}
	}
	return recibo, nil
}

func construirConstitucion(lista ports.ListaDefinitivaAutorizada, actorRef string, ahora time.Time) (ports.Constitucion, []ports.VinculoCandidato, error) {
	if len(lista.Posiciones) == 0 {
		if lista.Fuente == ports.FuenteSeleccionNativa {
			return ports.Constitucion{}, nil, ErrListaSinPosiciones
		}
		return ports.Constitucion{}, nil, ErrActaSinFilasAceptadas
	}
	if lista.Referencia == "" || lista.Version == 0 || lista.ConvocatoriaRef == "" || lista.CategoriaRef == "" ||
		(lista.Fuente != ports.FuenteImportacionConvoca && lista.Fuente != ports.FuenteSeleccionNativa) ||
		lista.AutorizacionPublicacionRef == "" || !huellaValida(lista.HuellaListadoSHA256) ||
		!huellaValida(lista.HuellaAutorizacionSHA256) {
		return ports.Constitucion{}, nil, ports.ErrConstitucionBolsaInvalida
	}
	sufijoActa := sufijoOpaco(lista.Referencia)
	bolsaRef := lista.BolsaRef
	if bolsaRef == "" {
		bolsaRef = "bolsa:" + claveCategoria(lista.CategoriaRef) + ":" + sufijoActa
	}
	resolucionRef := "resolucion:constitucion:" + sufijoActa
	huellaResolucion := huellaHex(lista.Referencia, actorRef, ahora.Format(time.RFC3339Nano))
	// Las referencias del acta llevan huellas hexadecimales; el dominio rechaza
	// referencias que parezcan un documento de identidad (ocho dígitos y una
	// letra), así que se derivan formas opacas sin dígitos. El acta real queda
	// enlazada en la propia constitución.
	bolsa := dominio.BolsaConstituida{
		BolsaRef:                  bolsaRef,
		Version:                   1,
		ProcesoRef:                lista.ConvocatoriaRef,
		CategoriaRef:              lista.CategoriaRef,
		ListadoDefinitivoRef:      "acta:importacion-convoca:" + sufijoActa,
		VersionListado:            lista.Version,
		HuellaListadoSHA256:       lista.HuellaListadoSHA256,
		ResolucionConstitucionRef: resolucionRef,
		HuellaResolucionSHA256:    huellaResolucion,
		ConstituidaEn:             ahora,
		VigenteDesde:              ahora,
	}
	if lista.Fuente == ports.FuenteSeleccionNativa {
		bolsa.ListadoDefinitivoRef = lista.Referencia
	}
	if err := bolsa.Validar(); err != nil {
		return ports.Constitucion{}, nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
	}
	entradas := make([]dominio.EntradaOrdenBolsa, 0, len(lista.Posiciones))
	vinculos := make([]ports.EntradaConstitucion, 0, len(lista.Posiciones))
	candidatos := make([]ports.VinculoCandidato, 0, len(lista.Posiciones))
	huellaEstado := huellaHex(EstadoInicial, "1")
	huellaCausa := huellaHex(CausaInicial, "1")
	personas := make(map[string]bool, len(lista.Posiciones))
	for indice, fila := range lista.Posiciones {
		orden := fila.Posicion
		if orden != uint64(indice+1) || fila.PersonaRef == "" || personas[fila.PersonaRef] || fila.Puntuacion == "" ||
			(!strings.HasPrefix(fila.PersonaRef, "persona:") && !ReferenciaCandidatoValida(fila.PersonaRef)) ||
			(fila.CandidatoRef != "" && !ReferenciaCandidatoValida(fila.CandidatoRef)) ||
			(lista.Fuente == ports.FuenteImportacionConvoca && (fila.FilaOrigenNumero <= 0 || !ReferenciaCandidatoValida(fila.CandidatoRef) || fila.SujetoRef == "" || fila.ParticipacionRef == "")) ||
			(lista.Fuente == ports.FuenteSeleccionNativa && (fila.FilaOrigenNumero != 0 || fila.SujetoRef != "" || fila.ParticipacionRef != "")) {
			return ports.Constitucion{}, nil, ports.ErrConstitucionBolsaInvalida
		}
		personas[fila.PersonaRef] = true
		participacionRef := fila.ParticipacionRef
		sujetoRef := fila.SujetoRef
		if lista.Fuente == ports.FuenteSeleccionNativa {
			participacionRef = "participacion:" + sufijoOpaco(bolsaRef+"|"+fila.PersonaRef+"|"+strconv.FormatUint(orden, 10))
			sujetoRef = "sujeto:seleccion:" + sufijoOpaco(fila.PersonaRef)
		}
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
		entradas = append(entradas, dominio.EntradaOrdenBolsa{Orden: orden, Participacion: participacion})
		vinculos = append(vinculos, ports.EntradaConstitucion{Orden: orden, ParticipacionRef: participacionRef, FilaNumero: fila.FilaOrigenNumero})
		if fila.CandidatoRef != "" {
			candidatos = append(candidatos, ports.VinculoCandidato{CandidatoRef: fila.CandidatoRef, ParticipacionRef: participacionRef})
		}
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
		Fuente:                     lista.Fuente,
		ActaRef:                    lista.Referencia,
		ActorRef:                   actorRef,
		CategoriaRef:               lista.CategoriaRef,
		AutorizacionPublicacionRef: lista.AutorizacionPublicacionRef,
		HuellaAutorizacionSHA256:   lista.HuellaAutorizacionSHA256,
		Bolsa:                      bolsa,
		Instantanea:                instantanea,
		Entradas:                   vinculos,
		ConfirmadaEn:               ahora,
	}, candidatos, nil
}

func huellaValida(huella string) bool {
	if len(huella) != 64 {
		return false
	}
	_, err := hex.DecodeString(huella)
	return err == nil
}

// puntuacion conserva la equivalencia histórica de CONVOCA: un total no
// numérico cuenta como cero. Solo su adaptador emplea esta conversión.
func puntuacion(total string) float64 {
	valor, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(total), ",", "."), 64)
	if err != nil {
		return 0
	}
	return valor
}

func PuntuacionConvoca(total string) float64 { return puntuacion(total) }

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

// SufijoReferenciaOpaca conserva las referencias históricas de constitución
// al traducir actas CONVOCA en su adaptador.
func SufijoReferenciaOpaca(semilla string) string { return sufijoOpaco(semilla) }

func huellaHex(partes ...string) string {
	suma := sha256.Sum256([]byte(strings.Join(partes, "\x1f")))
	return hex.EncodeToString(suma[:])
}
