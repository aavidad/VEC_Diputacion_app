package bootstrap

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const esquemaGobiernoCoberturaDesarrollo = "vec.contratacion-temporal.gobierno-cobertura.o4-04b.v1"

type publicacionGobiernoCoberturaDesarrollo struct {
	Esquema   string                                          `json:"esquema"`
	Secuencia uint64                                          `json:"secuencia"`
	EventoRef string                                          `json:"evento_ref"`
	Catalogo  domain.PublicacionCatalogoViasCobertura         `json:"catalogo"`
	Politica  domain.PublicacionPoliticaDecisionCobertura     `json:"politica"`
	Actuacion cobertura.PublicacionPoliticaActuacionCobertura `json:"actuacion"`
}

func nuevasPublicacionesGobiernoCoberturaDesarrollo(
	soporte *soporteAltaContratacionTemporalDesarrollo,
) ([]publicacionGobiernoCoberturaDesarrollo, error) {
	if soporte == nil {
		return nil, falloPostgreSQLCTDesarrollo(nil)
	}
	publicadaEn := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	vigencia := domain.VigenciaCatalogoCobertura{
		Desde: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Hasta: time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	comprobacion := domain.ComprobacionExigibleCobertura{
		Clave:       "existe_bolsa_vigente",
		Orden:       1,
		Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "bolsa",
			DefinicionFuenteRef: backendFuenteCoberturaDesarrolloRef,
		},
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:     "catalogo:ct:desarrollo:cobertura:v1",
			Version:        1,
			PublicadoEn:    publicadaEn,
			Vigencia:       vigencia,
			ProcedenciaRef: "procedencia:ct:desarrollo:cobertura:v1",
			Vias: []domain.DefinicionViaCobertura{{
				Clave:          "bolsa_vigente",
				Orden:          1,
				Comprobaciones: []domain.ComprobacionExigibleCobertura{comprobacion},
			}},
		},
	)
	if err != nil {
		return nil, falloPostgreSQLCTDesarrollo(err)
	}
	politica, err := domain.PublicarPoliticaDecisionCobertura(
		domain.BorradorPoliticaDecisionCobertura{
			Referencia:      "politica:ct:desarrollo:cobertura:v1",
			Version:         1,
			Catalogo:        catalogo.Identidad(),
			OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
			FinalidadClave:  "gestionar_cobertura_temporal",
			FinalidadRef:    "finalidad:ct:desarrollo:cobertura",
			PublicadaEn:     publicadaEn,
			Vigencia:        vigencia,
			ProcedenciaRef:  "procedencia:ct:desarrollo:politica-cobertura:v1",
			Vias: []domain.ReglaViaDecisionCobertura{{
				ViaClave:  "bolsa_vigente",
				Prioridad: 1,
				Comprobaciones: []domain.ReglaComprobacionDecisionCobertura{{
					Clave:                  comprobacion.Clave,
					ResultadosHabilitantes: []domain.ResultadoComprobacion{domain.ComprobacionAfirmativa},
					TratamientoAusencia:    domain.AusenciaCoberturaBloquea,
				}},
			}},
		},
		catalogo,
	)
	if err != nil {
		return nil, falloPostgreSQLCTDesarrollo(err)
	}
	acciones := []struct {
		accion     domain.ClaveCatalogo
		referencia string
		evento     string
	}{
		{
			accion:     domain.AccionDecidirCoberturaGobernada,
			referencia: "actuacion:ct:desarrollo:cobertura:decidir:v1",
			evento:     "evento_gobi_o404b_00000000000000000000000000000001",
		},
		{
			accion:     domain.AccionRectificarCoberturaGobernada,
			referencia: "actuacion:ct:desarrollo:cobertura:rectificar:v1",
			evento:     "evento_gobi_o404b_00000000000000000000000000000002",
		},
	}
	publicaciones := make([]publicacionGobiernoCoberturaDesarrollo, 0, len(acciones))
	for indice, configuracion := range acciones {
		actuacion := cobertura.PublicacionPoliticaActuacionCobertura{
			Referencia:                   configuracion.referencia,
			Version:                      1,
			Canon:                        cobertura.CanonHuellaPoliticaActuacionCoberturaV1(),
			OrganizacionRef:              organizacionAltaContratacionTemporalDesarrollo,
			Accion:                       configuracion.accion,
			Catalogo:                     catalogo.Identidad(),
			Politica:                     politica.Identidad(),
			FinalidadContratacionClave:   "gestionar_cobertura_temporal",
			FinalidadContratacionRef:     "finalidad:ct:desarrollo:cobertura",
			FinalidadAutorizacionVEC:     domain.ClaveCatalogo(finalidadDecisionCoberturaDesarrollo),
			UnidadEjecutoraRef:           unidadCoberturaContratacionTemporalDesarrollo,
			FaseDestino:                  "asignacion_unidad",
			EstadoDestino:                domain.EstadoEnCurso,
			MotivoAutorizacionDecidir:    soporte.motivoDecisionCobertura,
			MotivoAutorizacionRectificar: soporte.motivoRectificacionCobertura,
			PublicadaEn:                  publicadaEn,
			Vigencia:                     vigencia,
		}
		actuacion.HuellaSHA256, err =
			cobertura.CalcularHuellaSHA256PoliticaActuacionCobertura(actuacion)
		if err != nil || actuacion.Validar() != nil {
			return nil, falloPostgreSQLCTDesarrollo(err)
		}
		publicaciones = append(publicaciones, publicacionGobiernoCoberturaDesarrollo{
			Esquema:   esquemaGobiernoCoberturaDesarrollo,
			Secuencia: uint64(indice + 1),
			EventoRef: configuracion.evento,
			Catalogo:  catalogo.Publicacion(),
			Politica:  politica.Publicacion(),
			Actuacion: actuacion,
		})
	}
	// La v2 añade alternativas explícitas sin reescribir la publicación v1,
	// que permanece como antecedente verificable de las decisiones ya emitidas.
	// Son las vías de siempre; un catálogo con otras publica aparte su propia
	// versión (sincronizarGobiernoCoberturaCatalogoPostgreSQLCT).
	v2, err := nuevoGobiernoCoberturaParaViasCT(soporte, viasCoberturaPredeterminadasCT(), "v2", 2)
	if err != nil {
		return nil, err
	}
	for indice, evento := range []string{
		"evento_gobi_o404b_00000000000000000000000000000003",
		"evento_gobi_o404b_00000000000000000000000000000004",
	} {
		publicaciones = append(publicaciones, publicacionGobiernoCoberturaDesarrollo{
			Esquema: esquemaGobiernoCoberturaDesarrollo, Secuencia: uint64(indice + 3), EventoRef: evento,
			Catalogo: v2.catalogo, Politica: v2.politica, Actuacion: v2.actuaciones[indice],
		})
	}
	return publicaciones, nil
}

func publicarGobiernoCoberturaPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if ctx == nil || pool == nil || soporte == nil || ctx.Err() != nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	publicaciones, err := nuevasPublicacionesGobiernoCoberturaDesarrollo(soporte)
	if err != nil {
		return err
	}
	for _, publicacion := range publicaciones {
		resultado, err := publicarUnaGobiernoCoberturaDesarrollo(ctx, pool, publicacion)
		if err != nil || (resultado != "publicada" && resultado != "repetida") {
			return falloPostgreSQLCTDesarrollo(err)
		}
	}
	return nil
}

// publicarUnaGobiernoCoberturaDesarrollo publica una sola publicación en su
// propia transacción serializable y devuelve «publicada» o «repetida»; el error
// de PostgreSQL se devuelve tal cual para que quien llama lo clasifique.
func publicarUnaGobiernoCoberturaDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	publicacion publicacionGobiernoCoberturaDesarrollo,
) (string, error) {
	carga, err := json.Marshal(publicacion)
	if err != nil {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`); err != nil {
		return "", err
	}
	var resultado, evento, huella string
	if err := tx.QueryRow(ctx, `
		SELECT resultado, evento_ref, huella_evento_sha256
		  FROM vec_contratacion_temporal.gobi_o404b_publicar($1::jsonb)`,
		carga,
	).Scan(&resultado, &evento, &huella); err != nil {
		return "", err
	}
	if evento != publicacion.EventoRef || len(huella) != 64 ||
		(resultado != "publicada" && resultado != "repetida") {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return resultado, nil
}
