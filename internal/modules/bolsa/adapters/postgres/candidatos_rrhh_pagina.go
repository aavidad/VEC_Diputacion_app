package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	convoca "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type descifradorFilasRRHH interface {
	DescifrarFilasSeleccionadas(context.Context, []byte, string, string, []int) ([]convoca.FilaAceptada, error)
}

type LectorCandidatosRRHHPostgreSQL struct {
	pool        *pgxpool.Pool
	descifrador descifradorFilasRRHH
}

var _ ports.LectorCandidatosRRHHNominal = (*LectorCandidatosRRHHPostgreSQL)(nil)

func NuevoLectorCandidatosRRHHPostgreSQL(pool *pgxpool.Pool, descifrador descifradorFilasRRHH) (*LectorCandidatosRRHHPostgreSQL, error) {
	if pool == nil || descifrador == nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return &LectorCandidatosRRHHPostgreSQL{pool: pool, descifrador: descifrador}, nil
}

func (l *LectorCandidatosRRHHPostgreSQL) LeerCandidatosNominal(ctx context.Context, q ports.ConsultaCandidatosRRHHNominal, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.PaginaCandidatosRRHHNominal, error) {
	var vacio ports.PaginaCandidatosRRHHNominal
	if l == nil || l.pool == nil || l.descifrador == nil || ctx == nil || ctx.Err() != nil ||
		q.BolsaRef == "" || q.Limite < 1 || q.Limite > 100 || material.ValidarEstructura() != nil ||
		material.ResumenCapacidad().Operacion() != ports.AccionRRHHCandidatosConsultar ||
		material.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaRRHHCandidatosConsultar {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true)`); err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	var crudo []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.consultar_candidatos_rrhh_nominal_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::numeric,$16::numeric,$17,$18,$19,$20)`,
		q.BolsaRef, q.Estado, q.Texto, q.CursorToken, q.CursorRef, q.SnapshotSHA256, q.Limite, q.Modo, q.SeleccionRefs, q.CeseActivo,
		material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(),
		material.PersonaVersion(), material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI()).Scan(&crudo)
	if err != nil {
		return vacio, errorResumenNominal(err)
	}
	if len(crudo) == 0 || len(crudo) > 32<<20 {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	pagina, err := decodificarPaginaCandidatosRRHH(ctx, crudo, q, l.descifrador)
	if err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	if q.Modo == ports.ModoBarridoTextoCandidatosRRHH {
		seleccion := make([]string, 0, len(pagina.Candidatos))
		coincidentes := make([]ports.CandidatoRRHHNominal, 0, q.Limite)
		for _, candidato := range pagina.Candidatos {
			if !strings.Contains(strings.ToLower(candidato.NombreVisible+" "+candidato.DocumentoEnmascarado), q.Texto) {
				continue
			}
			seleccion = append(seleccion, candidato.ParticipacionRef)
			if len(coincidentes) < q.Limite {
				coincidentes = append(coincidentes, candidato)
			}
		}
		pagina.Candidatos, pagina.SeleccionRefs = coincidentes, seleccion
		pagina.TotalFiltrado = len(seleccion)
		pagina.HayMas = len(seleccion) > len(coincidentes)
		pagina.CursorRefSiguiente = ""
		if pagina.HayMas {
			pagina.CursorRefSiguiente = coincidentes[len(coincidentes)-1].ParticipacionRef
		}
		ids := make([]string, 0, len(coincidentes))
		for _, c := range coincidentes {
			ids = append(ids, c.ParticipacionRef)
		}
		var extras []byte
		err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.finalizar_barrido_rrhh_nominal_v1($1,$2,$3,$4)`,
			pagina.ConsumoHuellaSHA256, q.BolsaRef, pagina.SnapshotSHA256, ids).Scan(&extras)
		if err != nil {
			return vacio, errorResumenNominal(err)
		}
		if aplicarExtrasPaginaCandidatosRRHH(&pagina, extras) != nil {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
	} else {
		pagina.TotalFiltrado = pagina.TotalPrefiltrado
	}
	if ctx.Err() != nil || tx.Commit(ctx) != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	return pagina, nil
}

type candidatoPaginaSQL struct {
	ParticipacionRef string     `json:"participacion_ref"`
	FilaNumero       int        `json:"fila_numero"`
	Orden            *int       `json:"orden"`
	OrdenActa        int        `json:"orden_acta"`
	RazonOrden       string     `json:"razon_orden"`
	Estado           string     `json:"estado_clave"`
	EstadoDesde      time.Time  `json:"estado_desde"`
	DisponibleDesde  *time.Time `json:"disponible_desde"`
}

type paginaCandidatosSQL struct {
	GeneradoEn    time.Time  `json:"generado_en"`
	BolsaRef      string     `json:"bolsa_ref"`
	CategoriaRef  string     `json:"categoria_ref"`
	HuellaListado string     `json:"huella_listado_sha256"`
	ConfirmadaEn  time.Time  `json:"confirmada_en"`
	TipoLista     string     `json:"tipo_lista"`
	VigenteDesde  time.Time  `json:"vigente_desde"`
	VigenteHasta  *time.Time `json:"vigente_hasta"`
	Politica      struct {
		BolsaRef     string    `json:"bolsa_ref"`
		PoliticaRef  string    `json:"politica_ref"`
		Version      int64     `json:"version_politica"`
		Criterio     string    `json:"criterio"`
		TipoLista    string    `json:"tipo_lista"`
		Reposicion   string    `json:"reposicion"`
		Provisional  bool      `json:"provisional"`
		Rotulo       string    `json:"rotulo"`
		Actor        string    `json:"actor"`
		VigenteDesde time.Time `json:"vigente_desde"`
	} `json:"politica"`
	LlamamientosEnCurso int                  `json:"llamamientos_en_curso"`
	SnapshotSHA256      string               `json:"snapshot_sha256"`
	Total               int                  `json:"total"`
	TotalPrefiltrado    int                  `json:"total_prefiltrado"`
	PorEstado           map[string]int       `json:"por_estado"`
	Candidatos          []candidatoPaginaSQL `json:"candidatos"`
	Contactos           []struct {
		ContactoRef      string    `json:"contacto_ref"`
		ParticipacionRef string    `json:"participacion_ref"`
		LlamamientoRef   string    `json:"llamamiento_ref"`
		OfertaRef        string    `json:"oferta_ref"`
		EvidenciaRef     string    `json:"evidencia_ref"`
		EvidenciaHuella  string    `json:"evidencia_huella_sha256"`
		Canal            string    `json:"canal"`
		Instante         time.Time `json:"instante"`
		ActorRef         string    `json:"actor_ref"`
		Resultado        string    `json:"resultado"`
		Anotacion        string    `json:"anotacion"`
	} `json:"contactos"`
	Marcas         []marcaPaginaSQL `json:"marcas"`
	TurnoSiguiente *struct {
		ParticipacionRef string `json:"participacion_ref"`
		FilaNumero       int    `json:"fila_numero"`
		OrdenVigente     *int   `json:"orden_vigente"`
	} `json:"turno_siguiente"`
	TurnoUltimo *struct {
		ParticipacionRef string    `json:"participacion_ref"`
		FilaNumero       int       `json:"fila_numero"`
		OrdenVigente     *int      `json:"orden_vigente"`
		ContactoRef      string    `json:"contacto_ref"`
		LlamamientoRef   string    `json:"llamamiento_ref"`
		Instante         time.Time `json:"instante"`
		Canal            string    `json:"canal"`
		Resultado        string    `json:"resultado"`
	} `json:"turno_ultimo"`
	FilasProtegidas     json.RawMessage `json:"filas_protegidas"`
	HayMas              bool            `json:"hay_mas"`
	CursorRefSiguiente  string          `json:"cursor_ref_siguiente"`
	ConsumoHuellaSHA256 string          `json:"consumo_huella_sha256"`
}

type marcaPaginaSQL struct {
	ParticipacionRef           string `json:"participacion_ref"`
	PrestaServicios            string `json:"presta_servicios"`
	EnRevision                 string `json:"en_revision"`
	EncadenamientoDias         *int   `json:"encadenamiento_dias"`
	EncadenamientoUmbralMeses  *int   `json:"encadenamiento_umbral_meses"`
	EncadenamientoVentanaMeses *int   `json:"encadenamiento_ventana_meses"`
}

func decodificarPaginaCandidatosRRHH(ctx context.Context, crudo []byte, q ports.ConsultaCandidatosRRHHNominal, descifrador descifradorFilasRRHH) (ports.PaginaCandidatosRRHHNominal, error) {
	var s paginaCandidatosSQL
	var vacio ports.PaginaCandidatosRRHHNominal
	if json.Unmarshal(crudo, &s) != nil || s.BolsaRef != q.BolsaRef || s.CategoriaRef == "" ||
		s.GeneradoEn.IsZero() || s.ConfirmadaEn.IsZero() || s.VigenteDesde.IsZero() ||
		len(s.SnapshotSHA256) != 64 || len(s.HuellaListado) != 64 || len(s.ConsumoHuellaSHA256) != 64 || s.Total < 0 || s.TotalPrefiltrado < 0 ||
		len(s.Candidatos) > 5000 || len(s.Contactos) > 20000 || s.PorEstado == nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	p := bolsadominio.PoliticaOrdenBolsa{BolsaRef: s.Politica.BolsaRef, PoliticaRef: s.Politica.PoliticaRef,
		Version: uint64(s.Politica.Version), Criterio: s.Politica.Criterio, TipoLista: s.Politica.TipoLista,
		Reposicion: s.Politica.Reposicion, Provisional: s.Politica.Provisional, Rotulo: s.Politica.Rotulo,
		Actor: s.Politica.Actor, VigenteDesde: s.Politica.VigenteDesde}
	if s.Politica.Version <= 0 || p.BolsaRef != q.BolsaRef || p.Validar() != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	out := ports.PaginaCandidatosRRHHNominal{GeneradoEn: s.GeneradoEn, BolsaRef: s.BolsaRef,
		CategoriaRef: s.CategoriaRef, ConfirmadaEn: s.ConfirmadaEn, VigenteDesde: s.VigenteDesde,
		VigenteHasta: s.VigenteHasta, TipoLista: s.TipoLista, Politica: p,
		LlamamientosEnCurso: s.LlamamientosEnCurso, SnapshotSHA256: s.SnapshotSHA256,
		Total: s.Total, TotalPrefiltrado: s.TotalPrefiltrado, PorEstado: s.PorEstado,
		HayMas: s.HayMas, CursorRefSiguiente: s.CursorRefSiguiente, ConsumoHuellaSHA256: s.ConsumoHuellaSHA256}
	visibles := map[int]struct{ nombre, documento string }{}
	if len(s.FilasProtegidas) > 0 && string(s.FilasProtegidas) != "null" {
		var indice struct {
			Filas []struct {
				Numero int `json:"numero"`
			} `json:"filas"`
		}
		if json.Unmarshal(s.FilasProtegidas, &indice) != nil || len(indice.Filas) > 5000 {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		numeros := make([]int, 0, len(indice.Filas))
		for _, fila := range indice.Filas {
			numeros = append(numeros, fila.Numero)
		}
		filas, err := descifrador.DescifrarFilasSeleccionadas(ctx, s.FilasProtegidas, s.HuellaListado, s.CategoriaRef, numeros)
		if err != nil || len(filas) != len(numeros) {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		out.FilasDescifradas = len(filas)
		for _, fila := range filas {
			nombre := strings.Join(strings.Fields(strings.Join([]string{fila.Identidad.Nombre, fila.Identidad.PrimerApellido, fila.Identidad.SegundoApellido}, " ")), " ")
			visibles[fila.Numero] = struct{ nombre, documento string }{nombre, fila.Identidad.Documento}
		}
	}
	for _, candidata := range s.Candidatos {
		identidad, ok := visibles[candidata.FilaNumero]
		if !ok || candidata.ParticipacionRef == "" || candidata.OrdenActa < 1 || candidata.Estado == "" || candidata.EstadoDesde.IsZero() {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		out.Candidatos = append(out.Candidatos, ports.CandidatoRRHHNominal{ParticipacionRef: candidata.ParticipacionRef,
			Orden: candidata.Orden, OrdenActa: candidata.OrdenActa, RazonOrden: candidata.RazonOrden,
			NombreVisible: identidad.nombre, DocumentoEnmascarado: identidad.documento,
			Estado: candidata.Estado, EstadoDesde: candidata.EstadoDesde, DisponibleDesde: candidata.DisponibleDesde})
	}
	for _, c := range s.Contactos {
		contacto := bolsadominio.ContactoParticipacion{ContactoRef: c.ContactoRef, BolsaRef: s.BolsaRef,
			ParticipacionRef: c.ParticipacionRef, LlamamientoRef: c.LlamamientoRef,
			OfertaRef: c.OfertaRef, EvidenciaRef: c.EvidenciaRef, EvidenciaHuellaSHA256: c.EvidenciaHuella,
			Canal: c.Canal, Instante: c.Instante, Actor: c.ActorRef, Resultado: c.Resultado, Anotacion: c.Anotacion}
		if contacto.Validar() != nil {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		out.Contactos = append(out.Contactos, contacto)
	}
	if err := aplicarMarcasPaginaCandidatosRRHH(&out, s.Marcas); err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	if s.TurnoSiguiente != nil {
		identidad, ok := visibles[s.TurnoSiguiente.FilaNumero]
		if !ok || s.TurnoSiguiente.OrdenVigente == nil {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		out.TurnoSiguiente = &ports.CandidatoRRHHNominal{ParticipacionRef: s.TurnoSiguiente.ParticipacionRef,
			NombreVisible: identidad.nombre, Orden: s.TurnoSiguiente.OrdenVigente}
	}
	if s.TurnoUltimo != nil {
		identidad, ok := visibles[s.TurnoUltimo.FilaNumero]
		if !ok {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		contacto := bolsadominio.ContactoParticipacion{ContactoRef: s.TurnoUltimo.ContactoRef, BolsaRef: s.BolsaRef,
			ParticipacionRef: s.TurnoUltimo.ParticipacionRef, LlamamientoRef: s.TurnoUltimo.LlamamientoRef,
			Canal: s.TurnoUltimo.Canal, Instante: s.TurnoUltimo.Instante, Resultado: s.TurnoUltimo.Resultado}
		out.TurnoUltimo = &ports.TurnoCandidatoRRHHNominal{Candidato: ports.CandidatoRRHHNominal{ParticipacionRef: s.TurnoUltimo.ParticipacionRef,
			NombreVisible: identidad.nombre, Orden: s.TurnoUltimo.OrdenVigente}, Contacto: &contacto}
	}
	return out, nil
}

func aplicarExtrasPaginaCandidatosRRHH(p *ports.PaginaCandidatosRRHHNominal, crudo []byte) error {
	var s struct {
		Contactos []struct {
			ContactoRef      string    `json:"contacto_ref"`
			ParticipacionRef string    `json:"participacion_ref"`
			LlamamientoRef   string    `json:"llamamiento_ref"`
			OfertaRef        string    `json:"oferta_ref"`
			EvidenciaRef     string    `json:"evidencia_ref"`
			EvidenciaHuella  string    `json:"evidencia_huella_sha256"`
			Canal            string    `json:"canal"`
			Instante         time.Time `json:"instante"`
			ActorRef         string    `json:"actor_ref"`
			Resultado        string    `json:"resultado"`
			Anotacion        string    `json:"anotacion"`
		} `json:"contactos"`
		Marcas []marcaPaginaSQL `json:"marcas"`
	}
	if p == nil || len(crudo) == 0 || len(crudo) > 4<<20 || json.Unmarshal(crudo, &s) != nil || len(s.Contactos) > 20000 {
		return ports.ErrResumenBolsasNoDisponible
	}
	permitidas := map[string]bool{}
	for _, c := range p.Candidatos {
		permitidas[c.ParticipacionRef] = true
	}
	for _, c := range s.Contactos {
		contacto := bolsadominio.ContactoParticipacion{ContactoRef: c.ContactoRef, BolsaRef: p.BolsaRef,
			ParticipacionRef: c.ParticipacionRef, LlamamientoRef: c.LlamamientoRef,
			OfertaRef: c.OfertaRef, EvidenciaRef: c.EvidenciaRef, EvidenciaHuellaSHA256: c.EvidenciaHuella,
			Canal: c.Canal, Instante: c.Instante, Actor: c.ActorRef, Resultado: c.Resultado, Anotacion: c.Anotacion}
		if !permitidas[c.ParticipacionRef] || contacto.Validar() != nil {
			return ports.ErrResumenBolsasNoDisponible
		}
		p.Contactos = append(p.Contactos, contacto)
	}
	return aplicarMarcasPaginaCandidatosRRHH(p, s.Marcas)
}

func aplicarMarcasPaginaCandidatosRRHH(p *ports.PaginaCandidatosRRHHNominal, marcas []marcaPaginaSQL) error {
	if p == nil || len(marcas) > 100 {
		return ports.ErrResumenBolsasNoDisponible
	}
	if p.Marcas == nil {
		p.Marcas = make(map[string]bolsadominio.MarcasParticipacion, len(marcas))
	}
	permitidas := map[string]bool{}
	for _, c := range p.Candidatos {
		permitidas[c.ParticipacionRef] = true
	}
	for _, m := range marcas {
		if !permitidas[m.ParticipacionRef] || p.Marcas[m.ParticipacionRef].ParticipacionRef != "" {
			return ports.ErrResumenBolsasNoDisponible
		}
		marca := bolsadominio.MarcasParticipacion{ParticipacionRef: m.ParticipacionRef,
			PrestaServicios: m.PrestaServicios, EnRevision: m.EnRevision}
		if m.EncadenamientoDias != nil {
			marca.EncadenamientoDias = *m.EncadenamientoDias
		}
		if m.EncadenamientoUmbralMeses != nil {
			marca.EncadenamientoUmbralMeses = *m.EncadenamientoUmbralMeses
		}
		if m.EncadenamientoVentanaMeses != nil {
			marca.EncadenamientoVentanaMeses = *m.EncadenamientoVentanaMeses
		}
		if !marcaLeidaValida(marca, int64(marca.EncadenamientoDias)) {
			return ports.ErrResumenBolsasNoDisponible
		}
		p.Marcas[m.ParticipacionRef] = marca
	}
	return nil
}
