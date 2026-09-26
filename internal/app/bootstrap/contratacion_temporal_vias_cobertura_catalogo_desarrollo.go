package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Vías de cobertura gobernadas por el catálogo de reglas (entradas
// «c17.via_cobertura.<clave>», duda 7). Cada entrada es una vía: su valor es la
// lista ordenada de comprobaciones, «procedencia» dice qué fuente las responde
// y «prioridad» ordena las vías. Sin ninguna entrada rigen las tres de siempre.
const (
	prefijoViaCoberturaCT        = reglas.CTPrefijoViaCobertura
	atributoPrioridadViaCT       = "prioridad"
	atributoProcedenciaViaCT     = "procedencia"
	maximoViasCoberturaCT        = 64
	maximoComprobacionesViaCT    = 32
	maximaPrioridadViaCT         = 65535
	versionGobiernoCatalogoCT    = 3
	maximoSondeosSecuenciaCT     = 4096
	maximoReintentosSerieCT      = 3
	expedienteSondeoGobiernoCT   = "expediente:ct:desarrollo:gobierno-cobertura"
	dominioEventoGobiernoCatalog = "vec.ct.desarrollo.gobierno-cobertura.evento.v1"
)

var (
	errViasCoberturaNoValidas = errors.New(
		"bootstrap: vías de cobertura del catálogo de reglas no válidas",
	)
	errGobiernoCoberturaCatalogoNoPublicado = errors.New(
		"bootstrap: gobierno de cobertura del catálogo no publicado",
	)
)

// viaCoberturaCT es una vía de cobertura normalizada: las comprobaciones en su
// orden y una sola procedencia por vía. La etiqueta no forma parte del
// gobierno publicado ni de su huella.
type viaCoberturaCT struct {
	Clave          domain.ClaveCatalogo   `json:"clave"`
	Procedencia    domain.ClaveCatalogo   `json:"procedencia"`
	Comprobaciones []domain.ClaveCatalogo `json:"comprobaciones"`
	Etiqueta       string                 `json:"-"`
}

// viasCoberturaPredeterminadasCT son las vías anteriores al catálogo, las de
// la publicación v2 del gobierno de cobertura.
func viasCoberturaPredeterminadasCT() []viaCoberturaCT {
	return []viaCoberturaCT{
		{Clave: "bolsa_vigente", Procedencia: "bolsa", Etiqueta: "Bolsa vigente",
			Comprobaciones: []domain.ClaveCatalogo{"existe_bolsa_vigente", "hay_candidaturas_disponibles"}},
		{Clave: "oferta_sae", Procedencia: "sae", Etiqueta: "Oferta SAE",
			Comprobaciones: []domain.ClaveCatalogo{"oferta_sae_disponible"}},
		{Clave: "nueva_convocatoria_bolsa", Procedencia: "bolsa", Etiqueta: "Nueva convocatoria de Bolsa",
			Comprobaciones: []domain.ClaveCatalogo{"requiere_nueva_convocatoria"}},
	}
}

// viasCoberturaDesdeReglasCT lee las vías del catálogo. Devuelve nil si no hay
// ninguna; un catálogo con una vía mal formada impide arrancar.
func viasCoberturaDesdeReglasCT(vigentes []reglas.Regla) ([]viaCoberturaCT, error) {
	type viaOrdenada struct {
		via       viaCoberturaCT
		prioridad int
	}
	var ordenadas []viaOrdenada
	for _, regla := range vigentes {
		if !strings.HasPrefix(regla.Clave, prefijoViaCoberturaCT) {
			continue
		}
		clave, err := claveOpcionDesdeReglaCT(regla, prefijoViaCoberturaCT)
		if err != nil || regla.Unidad != reglas.UnidadLista {
			return nil, errViasCoberturaNoValidas
		}
		texto := regla.Atributos[atributoPrioridadViaCT]
		prioridad, err := strconv.Atoi(texto)
		procedencia := domain.ClaveCatalogo(regla.Atributos[atributoProcedenciaViaCT])
		if err != nil || strconv.Itoa(prioridad) != texto || prioridad < 1 ||
			prioridad > maximaPrioridadViaCT || !procedencia.Valida() {
			return nil, errViasCoberturaNoValidas
		}
		via := viaCoberturaCT{Clave: clave, Procedencia: procedencia, Etiqueta: regla.Etiqueta}
		for _, elemento := range regla.Elementos() {
			via.Comprobaciones = append(via.Comprobaciones, domain.ClaveCatalogo(elemento))
		}
		ordenadas = append(ordenadas, viaOrdenada{via: via, prioridad: prioridad})
	}
	if len(ordenadas) == 0 {
		return nil, nil
	}
	slices.SortStableFunc(ordenadas, func(a, b viaOrdenada) int { return a.prioridad - b.prioridad })
	vias := make([]viaCoberturaCT, 0, len(ordenadas))
	for indice, ordenada := range ordenadas {
		if indice > 0 && ordenadas[indice-1].prioridad == ordenada.prioridad {
			return nil, errViasCoberturaNoValidas
		}
		vias = append(vias, ordenada.via)
	}
	if !viasCoberturaCoherentesCT(vias) {
		return nil, errViasCoberturaNoValidas
	}
	return vias, nil
}

// viasCoberturaCoherentesCT comprueba lo que exige la publicación durable:
// claves válidas y únicas, límites, y una misma comprobación con la misma
// procedencia en todas las vías que la usen.
func viasCoberturaCoherentesCT(vias []viaCoberturaCT) bool {
	if len(vias) == 0 || len(vias) > maximoViasCoberturaCT {
		return false
	}
	vistas := make(map[domain.ClaveCatalogo]bool, len(vias))
	procedencias := make(map[domain.ClaveCatalogo]domain.ClaveCatalogo)
	for _, via := range vias {
		if !via.Clave.Valida() || vistas[via.Clave] || !via.Procedencia.Valida() ||
			len(via.Comprobaciones) == 0 || len(via.Comprobaciones) > maximoComprobacionesViaCT {
			return false
		}
		vistas[via.Clave] = true
		propias := make(map[domain.ClaveCatalogo]bool, len(via.Comprobaciones))
		for _, comprobacion := range via.Comprobaciones {
			anterior, usada := procedencias[comprobacion]
			if !comprobacion.Valida() || propias[comprobacion] || (usada && anterior != via.Procedencia) {
				return false
			}
			propias[comprobacion] = true
			procedencias[comprobacion] = via.Procedencia
		}
	}
	return true
}

// huellaViasCoberturaCT resume el contenido gobernado (claves, orden,
// procedencias y comprobaciones); la etiqueta no cuenta.
func huellaViasCoberturaCT(vias []viaCoberturaCT) string {
	material, _ := json.Marshal(vias)
	suma := sha256.Sum256(append([]byte("vec.ct.vias-cobertura.v1\n"), material...))
	return hex.EncodeToString(suma[:])
}

// viasCoberturaVigentes son las vías del catálogo o, sin él, las de siempre.
func (o *opcionesAnalisisCTDesarrollo) viasCoberturaVigentes() []viaCoberturaCT {
	if o == nil || len(o.viasCobertura) == 0 {
		return viasCoberturaPredeterminadasCT()
	}
	return o.viasCobertura
}

// plantillaCoberturaCT es una terna vía/comprobación/procedencia con su orden
// dentro de la vía (desde 1).
type plantillaCoberturaCT struct {
	via, comprobacion, procedencia domain.ClaveCatalogo
	orden                          uint16
}

func plantillasViasCoberturaCT(vias []viaCoberturaCT) []plantillaCoberturaCT {
	var plantillas []plantillaCoberturaCT
	for _, via := range vias {
		for indice, comprobacion := range via.Comprobaciones {
			plantillas = append(plantillas, plantillaCoberturaCT{
				via: via.Clave, comprobacion: comprobacion, procedencia: via.Procedencia,
				orden: uint16(indice + 1),
			})
		}
	}
	return plantillas
}

// gobiernoCoberturaDeseadoCT es el contenido que debe estar vigente: catálogo,
// política y una actuación por acción (decidir y rectificar).
type gobiernoCoberturaDeseadoCT struct {
	catalogo    domain.PublicacionCatalogoViasCobertura
	politica    domain.PublicacionPoliticaDecisionCobertura
	actuaciones []cobertura.PublicacionPoliticaActuacionCobertura
}

// nuevoGobiernoCoberturaParaViasCT construye catálogo, política y actuaciones
// para unas vías con un instante fijo: el mismo contenido da siempre los
// mismos bytes y la misma huella, en cualquier arranque.
func nuevoGobiernoCoberturaParaViasCT(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	vias []viaCoberturaCT,
	sufijo string,
	version uint64,
) (gobiernoCoberturaDeseadoCT, error) {
	if soporte == nil || !viasCoberturaCoherentesCT(vias) {
		return gobiernoCoberturaDeseadoCT{}, falloPostgreSQLCTDesarrollo(nil)
	}
	publicadaEn := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	vigencia := domain.VigenciaCatalogoCobertura{
		Desde: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Hasta: time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	definiciones := make([]domain.DefinicionViaCobertura, 0, len(vias))
	reglasVias := make([]domain.ReglaViaDecisionCobertura, 0, len(vias))
	for indice, via := range vias {
		definicion := domain.DefinicionViaCobertura{Clave: via.Clave, Orden: uint16(indice + 1)}
		regla := domain.ReglaViaDecisionCobertura{ViaClave: via.Clave, Prioridad: uint16(indice + 1)}
		for orden, clave := range via.Comprobaciones {
			definicion.Comprobaciones = append(definicion.Comprobaciones, domain.ComprobacionExigibleCobertura{
				Clave: clave, Orden: uint16(orden + 1), Obligatoria: true,
				Procedencia: domain.ProcedenciaComprobacionCobertura{
					Clave: via.Procedencia, DefinicionFuenteRef: backendFuenteCoberturaDesarrolloRef,
				},
			})
			regla.Comprobaciones = append(regla.Comprobaciones, domain.ReglaComprobacionDecisionCobertura{
				Clave:                  clave,
				ResultadosHabilitantes: []domain.ResultadoComprobacion{domain.ComprobacionAfirmativa},
				TratamientoAusencia:    domain.AusenciaCoberturaBloquea,
			})
		}
		definiciones = append(definiciones, definicion)
		reglasVias = append(reglasVias, regla)
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(domain.BorradorCatalogoViasCobertura{
		Referencia: "catalogo:ct:desarrollo:cobertura:" + sufijo, Version: version,
		PublicadoEn: publicadaEn, Vigencia: vigencia,
		ProcedenciaRef: "procedencia:ct:desarrollo:cobertura:" + sufijo,
		Vias:           definiciones,
	})
	if err != nil {
		return gobiernoCoberturaDeseadoCT{}, falloPostgreSQLCTDesarrollo(err)
	}
	politica, err := domain.PublicarPoliticaDecisionCobertura(domain.BorradorPoliticaDecisionCobertura{
		Referencia: "politica:ct:desarrollo:cobertura:" + sufijo, Version: version,
		Catalogo: catalogo.Identidad(), OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		FinalidadClave: "gestionar_cobertura_temporal", FinalidadRef: "finalidad:ct:desarrollo:cobertura",
		PublicadaEn: publicadaEn, Vigencia: vigencia,
		ProcedenciaRef: "procedencia:ct:desarrollo:politica-cobertura:" + sufijo,
		Vias:           reglasVias,
	}, catalogo)
	if err != nil {
		return gobiernoCoberturaDeseadoCT{}, falloPostgreSQLCTDesarrollo(err)
	}
	deseado := gobiernoCoberturaDeseadoCT{catalogo: catalogo.Publicacion(), politica: politica.Publicacion()}
	for _, configuracion := range []struct {
		accion domain.ClaveCatalogo
		nombre string
	}{
		{domain.AccionDecidirCoberturaGobernada, "decidir"},
		{domain.AccionRectificarCoberturaGobernada, "rectificar"},
	} {
		actuacion := cobertura.PublicacionPoliticaActuacionCobertura{
			Referencia: "actuacion:ct:desarrollo:cobertura:" + configuracion.nombre + ":" + sufijo,
			Version:    version, Canon: cobertura.CanonHuellaPoliticaActuacionCoberturaV1(),
			OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, Accion: configuracion.accion,
			Catalogo: catalogo.Identidad(), Politica: politica.Identidad(),
			FinalidadContratacionClave: "gestionar_cobertura_temporal", FinalidadContratacionRef: "finalidad:ct:desarrollo:cobertura",
			FinalidadAutorizacionVEC: domain.ClaveCatalogo(finalidadDecisionCoberturaDesarrollo), UnidadEjecutoraRef: unidadCoberturaContratacionTemporalDesarrollo,
			FaseDestino: "asignacion_unidad", EstadoDestino: domain.EstadoEnCurso,
			MotivoAutorizacionDecidir: soporte.motivoDecisionCobertura, MotivoAutorizacionRectificar: soporte.motivoRectificacionCobertura,
			PublicadaEn: publicadaEn, Vigencia: vigencia,
		}
		actuacion.HuellaSHA256, err = cobertura.CalcularHuellaSHA256PoliticaActuacionCobertura(actuacion)
		if err != nil || actuacion.Validar() != nil {
			return gobiernoCoberturaDeseadoCT{}, falloPostgreSQLCTDesarrollo(err)
		}
		deseado.actuaciones = append(deseado.actuaciones, actuacion)
	}
	return deseado, nil
}

// gobiernoCoberturaDeseadoParaCatalogoCT es la v2 de siempre si las vías
// coinciden con ella (sin catálogo, o con un catálogo que la repite) y, si no,
// una versión propia nombrada por la huella de su contenido.
func gobiernoCoberturaDeseadoParaCatalogoCT(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	vias []viaCoberturaCT,
) (gobiernoCoberturaDeseadoCT, error) {
	huella := huellaViasCoberturaCT(vias)
	if huella == huellaViasCoberturaCT(viasCoberturaPredeterminadasCT()) {
		return nuevoGobiernoCoberturaParaViasCT(soporte, vias, "v2", 2)
	}
	return nuevoGobiernoCoberturaParaViasCT(soporte, vias, "reglas-"+huella[:16], versionGobiernoCatalogoCT)
}

// eventoGobiernoCoberturaCatalogoCT nombra el evento por su secuencia y por la
// actuación que publica: repetir la misma publicación en la misma secuencia da
// el mismo evento, y el replay SQL la reconoce como «repetida».
func eventoGobiernoCoberturaCatalogoCT(secuencia uint64, huellaActuacion string) string {
	suma := sha256.Sum256([]byte(dominioEventoGobiernoCatalog + "|" +
		strconv.FormatUint(secuencia, 10) + "|" + huellaActuacion))
	return "evento_gobi_o404b_" + hex.EncodeToString(suma[:16])
}

// Resultados de un intento de publicación en una secuencia.
const (
	intentoGobiernoPublicado  = "publicada"
	intentoGobiernoOcupado    = "ocupada"
	intentoGobiernoReintentar = "reintentar"
)

// dependenciasSincronizacionGobiernoCT separa la lectura del puntero vigente
// (como ejecutor) de la publicación (como gobernador), para probar la
// asignación de secuencia sin PostgreSQL.
type dependenciasSincronizacionGobiernoCT struct {
	// vigente devuelve la huella de la actuación a la que apunta la acción,
	// o "" si no apunta a ninguna.
	vigente func(context.Context, domain.ClaveCatalogo) (string, error)
	// publicar intenta publicar en su secuencia y clasifica el resultado.
	publicar func(context.Context, publicacionGobiernoCoberturaDesarrollo) (string, error)
}

// sincronizarGobiernoCoberturaCT deja cada acción apuntando al contenido
// deseado. Si ya apunta a él no publica nada: dos arranques seguidos no añaden
// historia. Si no, publica en la primera secuencia libre tras las fijas; la
// secuencia es contigua y compartida con cualquier otra publicación o retirada,
// así que se busca pidiendo cada una: la ocupada se rechaza sin efectos.
func sincronizarGobiernoCoberturaCT(
	ctx context.Context,
	dependencias dependenciasSincronizacionGobiernoCT,
	deseado gobiernoCoberturaDeseadoCT,
	primeraSecuencia uint64,
) (int, error) {
	if dependencias.vigente == nil || dependencias.publicar == nil || primeraSecuencia == 0 {
		return 0, errGobiernoCoberturaCatalogoNoPublicado
	}
	publicadas := 0
	secuencia := primeraSecuencia
	for _, actuacion := range deseado.actuaciones {
		actual, err := dependencias.vigente(ctx, actuacion.Accion)
		if err != nil {
			return publicadas, errors.Join(errGobiernoCoberturaCatalogoNoPublicado, err)
		}
		if actual == actuacion.HuellaSHA256 {
			continue
		}
		publicada := false
		for sondeos, reintentos := 0, 0; !publicada; {
			if sondeos >= maximoSondeosSecuenciaCT || ctx.Err() != nil {
				return publicadas, errGobiernoCoberturaCatalogoNoPublicado
			}
			resultado, err := dependencias.publicar(ctx, publicacionGobiernoCoberturaDesarrollo{
				Esquema: esquemaGobiernoCoberturaDesarrollo, Secuencia: secuencia,
				EventoRef: eventoGobiernoCoberturaCatalogoCT(secuencia, actuacion.HuellaSHA256),
				Catalogo:  deseado.catalogo, Politica: deseado.politica, Actuacion: actuacion,
			})
			switch {
			case err != nil:
				return publicadas, errors.Join(errGobiernoCoberturaCatalogoNoPublicado, err)
			case resultado == intentoGobiernoPublicado:
				publicada = true
				publicadas++
			case resultado == intentoGobiernoOcupado:
				sondeos++
				reintentos = 0
			case resultado == intentoGobiernoReintentar && reintentos < maximoReintentosSerieCT:
				reintentos++
				continue
			default:
				return publicadas, errGobiernoCoberturaCatalogoNoPublicado
			}
			secuencia++
		}
	}
	return publicadas, nil
}

// sincronizarGobiernoCoberturaCatalogoPostgreSQLCT compone la sincronización
// real: lee el puntero con el ejecutor (función de resolución) y publica con
// el gobernador. Se llama al arrancar, después de las publicaciones fijas.
func sincronizarGobiernoCoberturaCatalogoPostgreSQLCT(
	ctx context.Context,
	gobierno *pgxpool.Pool,
	ejecucion *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) error {
	if ctx == nil || gobierno == nil || ejecucion == nil || soporte == nil {
		return errGobiernoCoberturaCatalogoNoPublicado
	}
	deseado, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, soporte.opcionesCatalogo.viasCoberturaVigentes())
	if err != nil {
		return errors.Join(errGobiernoCoberturaCatalogoNoPublicado, err)
	}
	resolutor, err := postgrescontratacion.NuevoResolutorGobiernoCoberturaO404BPostgreSQL(ejecucion)
	if err != nil {
		return errors.Join(errGobiernoCoberturaCatalogoNoPublicado, err)
	}
	fijas, err := nuevasPublicacionesGobiernoCoberturaDesarrollo(soporte)
	if err != nil {
		return err
	}
	_, err = sincronizarGobiernoCoberturaCT(ctx, dependenciasSincronizacionGobiernoCT{
		vigente: func(ctx context.Context, accion domain.ClaveCatalogo) (string, error) {
			return huellaActuacionVigenteCoberturaCT(ctx, resolutor, reloj, accion)
		},
		publicar: func(ctx context.Context, publicacion publicacionGobiernoCoberturaDesarrollo) (string, error) {
			return intentarPublicacionGobiernoCoberturaCT(ctx, gobierno, publicacion)
		},
	}, deseado, uint64(len(fijas))+1)
	return err
}

// huellaActuacionVigenteCoberturaCT resuelve la actuación vigente de la acción
// como lo hace una decisión real; sin puntero vigente devuelve "".
func huellaActuacionVigenteCoberturaCT(
	ctx context.Context,
	resolutor cobertura.ResolutorGobiernoOperacionCobertura,
	reloj relojContratacionTemporalDesarrollo,
	accion domain.ClaveCatalogo,
) (string, error) {
	construir := cobertura.NuevaSolicitudGobiernoDecisionCobertura
	if accion == domain.AccionRectificarCoberturaGobernada {
		construir = cobertura.NuevaSolicitudGobiernoRectificacionCobertura
	}
	solicitud, err := construir(organizacionAltaContratacionTemporalDesarrollo, expedienteSondeoGobiernoCT, 1)
	if err != nil {
		return "", err
	}
	gobierno, err := cobertura.ObtenerGobiernoOperacionCobertura(ctx, reloj, resolutor, solicitud)
	if errors.Is(err, cobertura.ErrGobiernoOperacionCoberturaNoDisponible) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	datos, err := gobierno.DesplegarPara(ctx, reloj, solicitud)
	if err != nil {
		return "", err
	}
	return datos.PoliticaActuacion.HuellaSHA256, nil
}

// intentarPublicacionGobiernoCoberturaCT publica en su propia transacción.
// «repetida» (la misma publicación ya estaba en esa secuencia) y el replay
// divergente (otra publicación la ocupa) significan secuencia ocupada; la
// transacción se revierte sin efectos.
func intentarPublicacionGobiernoCoberturaCT(
	ctx context.Context,
	pool *pgxpool.Pool,
	publicacion publicacionGobiernoCoberturaDesarrollo,
) (string, error) {
	resultado, err := publicarUnaGobiernoCoberturaDesarrollo(ctx, pool, publicacion)
	var errorPG *pgconn.PgError
	switch {
	case err == nil && resultado == "publicada":
		return intentoGobiernoPublicado, nil
	case err == nil && resultado == "repetida":
		return intentoGobiernoOcupado, nil
	case errors.As(err, &errorPG) && errorPG.Code == "23505" &&
		strings.Contains(errorPG.Message, "replay divergente"):
		return intentoGobiernoOcupado, nil
	case errors.As(err, &errorPG) && (errorPG.Code == "40001" || errorPG.Code == "40P01"):
		return intentoGobiernoReintentar, nil
	case err != nil:
		return "", err
	}
	return "", errGobiernoCoberturaCatalogoNoPublicado
}

// motivosAlternativaViasCoberturaCT ofrece el motivo de elección de RRHH en
// cada vía que no es la preferente: elegirla exige justificarlo.
func motivosAlternativaViasCoberturaCT(vias []viaCoberturaCT) []application.MotivoAlternativaCobertura {
	motivo := domain.ClaveCatalogo(motivoEleccionProcedimientoRRHHDesarrollo().EntradaClave)
	motivos := make([]application.MotivoAlternativaCobertura, 0, len(vias))
	for indice, via := range vias {
		if indice == 0 {
			continue
		}
		motivos = append(motivos, application.MotivoAlternativaCobertura{
			ViaClave: via.Clave, Clave: motivo,
			EtiquetaI18n: "contratacion_temporal.cobertura.motivo.eleccion_procedimiento_rrhh",
		})
	}
	return motivos
}

// catalogoAnalisisOpcional devuelve el primer catálogo no nulo, o nil.
func catalogoAnalisisOpcional(catalogos ...*catalogosAltaContratacionTemporalDesarrollo) *catalogosAltaContratacionTemporalDesarrollo {
	for _, catalogo := range catalogos {
		if catalogo != nil {
			return catalogo
		}
	}
	return nil
}
