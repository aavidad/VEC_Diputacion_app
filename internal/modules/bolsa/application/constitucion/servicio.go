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
// que descifra y verifica la atestación del staging de CONVOCA. Las filas que
// comparten referencia con otra fila del acta no se devuelven para vincular:
// salen como pendientes de revisión, porque fundirlas uniría a dos personas.
func DerivarFilasVinculo(lote importacion.LoteValidado, derivador DerivadorCandidato) ([]FilaVinculoCandidato, []ports.FilaPendienteRevision, error) {
	if derivador == nil || lote.Validar() != nil || lote.Acta.Esquema != importacion.EsquemaResumenPersona {
		return nil, nil, ports.ErrConstitucionBolsaInvalida
	}
	if len(lote.Aceptadas) == 0 {
		return nil, nil, ErrActaSinFilasAceptadas
	}
	for _, fila := range lote.Aceptadas {
		if fila.Resumen == nil || fila.Numero <= 0 {
			return nil, nil, ports.ErrConstitucionBolsaInvalida
		}
	}
	referencias, pendientes, err := referenciasCandidato(lote.Aceptadas, derivador)
	if err != nil {
		return nil, nil, err
	}
	filas := make([]FilaVinculoCandidato, 0, len(referencias))
	for _, fila := range lote.Aceptadas {
		if ref, vinculable := referencias[fila.Numero]; vinculable {
			filas = append(filas, FilaVinculoCandidato{FilaNumero: fila.Numero, CandidatoRef: ref})
		}
	}
	return filas, pendientes, nil
}

// referenciasCandidato deriva la referencia `can_*` de cada fila y aparta las
// que coinciden con la de otra fila del mismo acta. Esas filas no se vinculan
// a nadie: con cuatro dígitos del documento y el nombre no se puede saber si
// son dos personas o la misma, y el vínculo no admite inventar la respuesta.
// Devuelve las referencias vinculables por número de fila y las pendientes
// ordenadas por número de fila.
func referenciasCandidato(filas []importacion.FilaAceptada, derivador DerivadorCandidato) (map[int]string, []ports.FilaPendienteRevision, error) {
	referencias := make(map[int]string, len(filas))
	filasPorReferencia := make(map[string][]int, len(filas))
	for _, fila := range filas {
		if _, repetida := referencias[fila.Numero]; repetida {
			return nil, nil, ports.ErrConstitucionBolsaInvalida
		}
		ref, err := derivador.CandidatoRef(fila.Identidad)
		if err != nil {
			return nil, nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
		}
		referencias[fila.Numero] = ref
		filasPorReferencia[ref] = append(filasPorReferencia[ref], fila.Numero)
	}
	pendientes := make([]ports.FilaPendienteRevision, 0)
	for _, numeros := range filasPorReferencia {
		if len(numeros) < 2 {
			continue
		}
		for _, numero := range numeros {
			delete(referencias, numero)
			pendientes = append(pendientes, ports.FilaPendienteRevision{FilaNumero: numero, Motivo: ports.MotivoRevisionIdentidadAmbigua})
		}
	}
	sort.Slice(pendientes, func(i, j int) bool { return pendientes[i].FilaNumero < pendientes[j].FilaNumero })
	return referencias, pendientes, nil
}

// Constituir construye la bolsa y su instantánea desde las filas aceptadas del
// acta (orden: Total descendente, empate por apellidos y nombre), la persiste
// y registra el vínculo `can_* → participación` de cada fila que no esté
// pendiente de revisión. Todas las filas conservan su puesto; el recibo lista
// las pendientes. Como la constitución es idempotente por acta y las
// referencias de participación son deterministas, una nueva llamada sobre un
// acta ya constituida completa los vínculos que falten.
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
	constitucion, vinculos, pendientes, err := construirConstitucion(lote, solicitud.ActorRef, ahora, s.derivador)
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
	recibo.PendientesRevision = pendientes
	return recibo, nil
}

// OrdenarFilasConstitucion ordena las filas del resumen como quedarán en la
// bolsa: Total descendente y, a igual Total, apellidos y nombre; por último,
// número de fila. La vista previa de la carga usa el mismo orden.
func OrdenarFilasConstitucion(filas []importacion.FilaAceptada) {
	sort.SliceStable(filas, func(i, j int) bool {
		a, b := filas[i], filas[j]
		ta, tb := puntuacion(totalFila(a)), puntuacion(totalFila(b))
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
}

func totalFila(fila importacion.FilaAceptada) string {
	if fila.Resumen == nil {
		return ""
	}
	return fila.Resumen.Total
}

func construirConstitucion(lote importacion.LoteValidado, actorRef string, ahora time.Time, derivador DerivadorCandidato) (ports.Constitucion, []ports.VinculoCandidato, []ports.FilaPendienteRevision, error) {
	acta := lote.Acta
	if acta.Esquema != importacion.EsquemaResumenPersona {
		return ports.Constitucion{}, nil, nil, ErrActaNoEsResumen
	}
	filas := make([]importacion.FilaAceptada, 0, len(lote.Aceptadas))
	for _, fila := range lote.Aceptadas {
		if fila.Resumen != nil {
			filas = append(filas, fila)
		}
	}
	if len(filas) == 0 {
		return ports.Constitucion{}, nil, nil, ErrActaSinFilasAceptadas
	}
	referencias, pendientes, err := referenciasCandidato(filas, derivador)
	if err != nil {
		return ports.Constitucion{}, nil, nil, err
	}
	sujetos := semillasSujeto(filas)
	OrdenarFilasConstitucion(filas)
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
		return ports.Constitucion{}, nil, nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
	}
	entradas := make([]dominio.EntradaOrdenBolsa, 0, len(filas))
	vinculos := make([]ports.EntradaConstitucion, 0, len(filas))
	candidatos := make([]ports.VinculoCandidato, 0, len(filas))
	huellaEstado := huellaHex(EstadoInicial, "1")
	huellaCausa := huellaHex(CausaInicial, "1")
	for indice, fila := range filas {
		orden := uint64(indice + 1)
		participacionRef := "participacion:" + sufijoOpaco(bolsaRef+"|"+fila.Identidad.Documento+"|"+strconv.FormatUint(orden, 10))
		sujetoRef := "sujeto:convoca:" + sufijoOpaco(sujetos[fila.Numero])
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
		vinculos = append(vinculos, ports.EntradaConstitucion{Orden: orden, ParticipacionRef: participacionRef, FilaNumero: fila.Numero})
		if candidatoRef, vinculable := referencias[fila.Numero]; vinculable {
			candidatos = append(candidatos, ports.VinculoCandidato{CandidatoRef: candidatoRef, ParticipacionRef: participacionRef})
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
		return ports.Constitucion{}, nil, nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
	}
	return ports.Constitucion{
		ActaRef:      acta.ActaRef,
		ActorRef:     actorRef,
		CategoriaRef: acta.CategoriaRef,
		Bolsa:        bolsa,
		Instantanea:  instantanea,
		Entradas:     vinculos,
		ConfirmadaEn: ahora,
	}, candidatos, pendientes, nil
}

// semillasSujeto devuelve la semilla de la referencia de sujeto de cada fila:
// documento y nombre tal como vienen en el acta. Si dos filas traen
// exactamente el mismo texto, esa semilla lleva además el número de fila (tras
// un separador de control que la importación nunca acepta en un nombre), de
// modo que cada puesto de la lista conserve una entrada propia en la
// instantánea sin afirmar quién es cada persona (sus vínculos quedan
// pendientes de revisión). Las filas sin repetición conservan la semilla
// histórica y, con ella, las referencias ya emitidas.
func semillasSujeto(filas []importacion.FilaAceptada) map[int]string {
	semillas := make(map[int]string, len(filas))
	repeticiones := make(map[string]int, len(filas))
	for _, fila := range filas {
		semilla := fila.Identidad.Documento + "|" + fila.Identidad.PrimerApellido + "|" + fila.Identidad.SegundoApellido + "|" + fila.Identidad.Nombre
		semillas[fila.Numero] = semilla
		repeticiones[semilla]++
	}
	for numero, semilla := range semillas {
		if repeticiones[semilla] > 1 {
			semillas[numero] = semilla + "\x1ffila:" + strconv.Itoa(numero)
		}
	}
	return semillas
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
