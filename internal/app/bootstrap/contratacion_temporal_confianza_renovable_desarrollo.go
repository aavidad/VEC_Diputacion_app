package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/composicion/gobiernov3lector"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type relojConfianzaCTDesarrollo interface{ Ahora() time.Time }

// Una fuente por composición. Sólo conserva raíz y configuración públicas;
// los firmantes y emisores nominales siguen perteneciendo a sus proveedores.
// Cada servicio publicado es inmutable y una operación toma un único snapshot.
type fuenteConfianzaRenovableCTDesarrollo struct {
	mu       sync.Mutex
	reloj    relojConfianzaCTDesarrollo
	material materialAtestacionContratacionTemporalDesarrollo
	actual   *confianza.ServicioConfianzaAtestacionAutorizacionV3
	lector   *gobiernov3lector.Lector
	leer     func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error)
	renovar  func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error)
	// plazoIntento acota cada renovación programada; cero usa el valor
	// predeterminado. Sólo las pruebas lo reducen.
	plazoIntento time.Duration
}

func nuevaFuenteConfianzaRenovableCTDesarrollo(pool *pgxpool.Pool, m materialAtestacionContratacionTemporalDesarrollo, reloj relojConfianzaCTDesarrollo) (*fuenteConfianzaRenovableCTDesarrollo, error) {
	if pool == nil || reloj == nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	servicio, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(m.configuracion, reloj)
	if err != nil {
		return nil, err
	}
	// Copia explícita: ninguna referencia a secretos efímeros del arranque.
	publica := materialAtestacionContratacionTemporalDesarrollo{
		claveID: m.claveID, claveVersion: m.claveVersion, raiz: m.raiz,
		configuracion: m.configuracion, configuracionRef: m.configuracionRef,
		configuracionOrden: m.configuracionOrden, configuracionHuella: m.configuracionHuella,
		publicadaEn: m.publicadaEn, expiraEn: m.expiraEn, validaDesde: m.validaDesde,
		validaHasta: m.validaHasta, spki: append([]byte(nil), m.spki...), spkiHuella: m.spkiHuella,
	}
	f := &fuenteConfianzaRenovableCTDesarrollo{reloj: reloj, material: publica, actual: servicio,
		leer: func(ctx context.Context, anterior materialAtestacionContratacionTemporalDesarrollo, ahora time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
			var actual materialAtestacionContratacionTemporalDesarrollo
			err := ejecutarTransaccionGobiernoCTDesarrollo(ctx, pool, func(tx pgx.Tx) error {
				var e error
				actual, e = leerConfiguracionRenovableCTDesarrollo(ctx, tx, anterior, ahora)
				return e
			})
			return actual, err
		},
		renovar: func(ctx context.Context, anterior materialAtestacionContratacionTemporalDesarrollo, ahora time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
			return renovarConfiguracionConfianzaCTDesarrollo(ctx, pool, anterior, ahora)
		}}
	f.lector, err = f.nuevoLector()
	if err != nil {
		return nil, err
	}
	return f, nil
}

func publicacionConfianzaCT(m materialAtestacionContratacionTemporalDesarrollo) gobiernov3lector.Publicacion {
	return gobiernov3lector.Publicacion{Revision: m.configuracionRef, Secuencia: m.configuracionOrden,
		HuellaSHA256: m.configuracionHuella, PublicadaEn: m.publicadaEn, ExpiraEn: m.expiraEn}
}

func (f *fuenteConfianzaRenovableCTDesarrollo) nuevoLector() (*gobiernov3lector.Lector, error) {
	return gobiernov3lector.Nuevo(publicacionConfianzaCT(f.material), f.material.raiz, f.reloj,
		func(ctx context.Context, previa gobiernov3lector.Publicacion) (gobiernov3lector.Publicacion, error) {
			if f.leer == nil {
				return gobiernov3lector.Publicacion{}, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
			}
			anterior := f.material
			anterior.configuracionRef, anterior.configuracionOrden = previa.Revision, previa.Secuencia
			anterior.configuracionHuella, anterior.publicadaEn, anterior.expiraEn = previa.HuellaSHA256, previa.PublicadaEn, previa.ExpiraEn
			actual, err := f.leer(ctx, anterior, f.reloj.Ahora().UTC())
			if err != nil {
				return gobiernov3lector.Publicacion{}, err
			}
			return publicacionConfianzaCT(actual), nil
		})
}

func (f *fuenteConfianzaRenovableCTDesarrollo) instantanea(ctx context.Context) (*confianza.ServicioConfianzaAtestacionAutorizacionV3, error) {
	fallo := errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	if ctx == nil || f == nil {
		return nil, fallo
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reloj == nil || f.lector == nil || f.leer == nil || f.renovar == nil {
		return nil, fallo
	}
	ahora := f.reloj.Ahora().UTC()
	if ahora.Before(f.material.publicadaEn) || ahora.Before(f.material.validaDesde) || !ahora.Before(f.material.validaHasta) {
		return nil, fallo
	}
	if !ahora.Before(f.material.expiraEn) {
		// Único publicador: vec-server. La lectura posterior es separada y no
		// adopta una publicación que no pueda volver a leer del gobierno.
		if _, err := f.renovar(ctx, f.material, ahora); err != nil {
			return nil, err
		}
	}
	servicio, publicada, err := f.lector.Leer(ctx)
	if err != nil {
		return nil, fallo
	}
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(publicada.Revision, publicada.Secuencia, publicada.PublicadaEn, publicada.ExpiraEn, f.material.raiz)
	if err != nil {
		return nil, fallo
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	f.material.configuracionRef, f.material.configuracionOrden = publicada.Revision, publicada.Secuencia
	f.material.configuracionHuella, f.material.publicadaEn, f.material.expiraEn = publicada.HuellaSHA256, publicada.PublicadaEn, publicada.ExpiraEn
	f.material.configuracion = config
	f.actual = servicio
	return servicio, nil
}

// Renovación programada. La configuración vence a medianoche UTC y el
// protocolo existente sólo admite publicar la del día en curso: el puntero se
// establece en `publicadaEn` y la lectura exige `establecida_en <= ahora`, así
// que la del día siguiente no puede adelantarse sin otra vía de publicación.
// El temporizador despierta en el vencimiento, el primer instante en que la
// renovación es admisible, y reutiliza `instantanea`: misma transacción,
// mismo cerrojo consultivo y misma adopción idempotente que el disparo por
// uso, que se conserva. Así la continuidad no depende de tráfico CT (ni del
// lector de vec-interno, que no publica).
//
// Cada intento tiene plazo propio: `instantanea` retiene f.mu durante toda la
// transacción de gobierno y el contexto del temporizador sólo se cancela al
// cerrar, así que sin plazo un corte de red o un cerrojo consultivo ajeno
// bloquearía también cada petición CT. La espera se acota a unos minutos y se
// recalcula (los temporizadores de Go no avanzan con el equipo suspendido) y
// los reintentos crecen hasta un tope, registrando el primer fallo y después
// uno por reintento ya en el tope (cada 15 min), no una línea por minuto. El
// retroceso vuelve al inicial en cuanto el vencimiento avanza, también si la
// renovación la hizo el uso CT.
const (
	reintentoRenovacionProgramadaCTDesarrollo       = time.Minute
	reintentoMaximoRenovacionProgramadaCTDesarrollo = 15 * time.Minute
	maximoEsperaRenovacionProgramadaCT              = 10 * time.Minute
	plazoIntentoRenovacionProgramadaCTDesarrollo    = 30 * time.Second
)

type esperaRenovacionCTDesarrollo func(context.Context, time.Duration) error

func esperarTemporizadorCTDesarrollo(ctx context.Context, d time.Duration) error {
	temporizador := time.NewTimer(d)
	defer temporizador.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-temporizador.C:
		return nil
	}
}

func (f *fuenteConfianzaRenovableCTDesarrollo) vencimientoActual() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.material.expiraEn
}

// mantenerRenovacionProgramada bloquea hasta que ctx se cancela. Un fallo no
// adopta nada (la instantánea conserva la publicación confirmada) y se
// reintenta; nunca se interpreta como renovación.
func (f *fuenteConfianzaRenovableCTDesarrollo) mantenerRenovacionProgramada(ctx context.Context, esperar esperaRenovacionCTDesarrollo) {
	if f == nil || ctx == nil || esperar == nil || f.reloj == nil {
		return
	}
	reintento := reintentoRenovacionProgramadaCTDesarrollo
	var venceAnterior time.Time
	for ctx.Err() == nil {
		vence := f.vencimientoActual()
		if !vence.Equal(venceAnterior) {
			reintento = reintentoRenovacionProgramadaCTDesarrollo
			venceAnterior = vence
		}
		espera := min(max(vence.Sub(f.reloj.Ahora().UTC()), 0), maximoEsperaRenovacionProgramadaCT)
		if esperar(ctx, espera) != nil {
			return
		}
		ahora := f.reloj.Ahora().UTC()
		if ahora.Before(vence) {
			continue // Despertar anticipado: reloj de pared frente a monotónico.
		}
		plazo := f.plazoIntento
		if plazo <= 0 {
			plazo = plazoIntentoRenovacionProgramadaCTDesarrollo
		}
		intento, cancelarIntento := context.WithTimeout(ctx, plazo)
		_, err := f.instantanea(intento)
		cancelarIntento()
		if ctx.Err() != nil {
			return
		}
		if err == nil && f.vencimientoActual().After(ahora) {
			reintento = reintentoRenovacionProgramadaCTDesarrollo
			continue
		}
		if reintento == reintentoRenovacionProgramadaCTDesarrollo || reintento == reintentoMaximoRenovacionProgramadaCTDesarrollo {
			registrarFalloPostgreSQLContratacionTemporalDesarrollo(
				"renovacion_programada_confianza", "renovacion_no_confirmada",
			)
		}
		if esperar(ctx, reintento) != nil {
			return
		}
		reintento = min(2*reintento, reintentoMaximoRenovacionProgramadaCTDesarrollo)
	}
}

// iniciarRenovacionProgramadaCTDesarrollo arranca un único temporizador por
// fuente. La función devuelta lo cancela y espera a que termine; es
// idempotente.
func iniciarRenovacionProgramadaCTDesarrollo(f *fuenteConfianzaRenovableCTDesarrollo, esperar esperaRenovacionCTDesarrollo) func() {
	if f == nil || esperar == nil {
		return func() {}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	terminado := make(chan struct{})
	go func() {
		defer close(terminado)
		f.mantenerRenovacionProgramada(ctx, esperar)
	}()
	var una sync.Once
	return func() {
		una.Do(func() {
			cancelar()
			<-terminado
		})
	}
}

func renovarConfiguracionConfianzaCTDesarrollo(ctx context.Context, pool *pgxpool.Pool, anterior materialAtestacionContratacionTemporalDesarrollo, ahora time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
	siguiente := anterior
	err := ejecutarTransaccionGobiernoCTDesarrollo(ctx, pool, func(tx pgx.Tx) error {
		return renovarConfiguracionConfianzaCTEnTxDesarrollo(ctx, tx, anterior, ahora, &siguiente)
	})
	if err != nil {
		return materialAtestacionContratacionTemporalDesarrollo{}, err
	}
	return siguiente, nil
}

func renovarConfiguracionConfianzaCTEnTxDesarrollo(ctx context.Context, tx pgx.Tx, anterior materialAtestacionContratacionTemporalDesarrollo, ahora time.Time, siguiente *materialAtestacionContratacionTemporalDesarrollo) error {
	actual, err := leerConfiguracionRenovableCTDesarrollo(ctx, tx, anterior, ahora)
	if err != nil {
		return err
	}
	dia := time.Date(ahora.UTC().Year(), ahora.UTC().Month(), ahora.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if actual.publicadaEn.Equal(dia) && actual.expiraEn.Equal(dia.Add(24*time.Hour)) {
		*siguiente = actual
		return nil
	}
	if actual.expiraEn.After(dia) {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	siguiente.publicadaEn = dia
	siguiente.expiraEn = dia.Add(24 * time.Hour)
	siguiente.configuracionOrden = uint64(dia.Year()*10000 + int(dia.Month())*100 + dia.Day())
	if err := prepararConfiguracionGobiernoPostgreSQLContratacionTemporalDesarrollo(ctx, tx, siguiente); err != nil {
		return err
	}
	acto := "acto:ct:desarrollo:configuracion:r" + numeroDecimal64(siguiente.configuracionOrden)
	puntero := "acto:ct:desarrollo:puntero-configuracion:r" + numeroDecimal64(siguiente.configuracionOrden)
	if siguiente.configuracionRef == "confianza:atestacion:ct:desarrollo:"+dia.Format("2006-01-02") {
		acto = "acto:ct:desarrollo:configuracion:" + dia.Format("2006-01-02")
		puntero = "acto:ct:desarrollo:puntero-configuracion:" + dia.Format("2006-01-02")
	}
	// Sin ON CONFLICT: sólo la lectura exacta permite adoptar otra publicación.
	if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
 (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES ($1,$2,$3,$4,$5,$6)`, siguiente.configuracionRef, siguiente.configuracionOrden, siguiente.configuracionHuella, dia, siguiente.expiraEn, acto); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
 (configuracion_revision,raiz_clave_id,raiz_version) VALUES ($1,$2,$3)`, siguiente.configuracionRef, siguiente.claveID, siguiente.claveVersion); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err := tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
 (orden,configuracion_revision,establecida_en,acto_ref) VALUES ($1,$2,$3,$4)`, siguiente.configuracionOrden, siguiente.configuracionRef, dia, puntero); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	comprobada, err := leerConfiguracionRenovableCTDesarrollo(ctx, tx, anterior, ahora)
	if err != nil || comprobada.configuracionRef != siguiente.configuracionRef || comprobada.configuracionHuella != siguiente.configuracionHuella {
		return errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	return nil
}

// Las revocaciones, incluso programadas, impiden la renovación automática.
// No basta encontrar una raíz propia: el conjunto tiene que ser exactamente
// la raíz fijada, tanto en la revisión anterior como en la que se adopta.
func leerConfiguracionRenovableCTDesarrollo(ctx context.Context, tx pgx.Tx, anterior materialAtestacionContratacionTemporalDesarrollo, ahora time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
	vacia := materialAtestacionContratacionTemporalDesarrollo{}
	fallo := errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(ctx, tx)
	if err != nil || !propio {
		return vacia, fallo
	}
	var minima, raizMinima int64
	if err := tx.QueryRow(ctx, `SELECT configuracion_secuencia_minima,raiz_version_minima
 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id FOR UPDATE`).Scan(&minima, &raizMinima); err != nil {
		return vacia, fallo
	}
	actual := anterior
	var secuencia int64
	err = tx.QueryRow(ctx, `SELECT c.revision,c.secuencia,c.huella_configuracion_sha256,c.publicada_en,c.expira_en
 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
 JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision
 JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision
 JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON (r.clave_id,r.version)=(cr.raiz_clave_id,cr.raiz_version)
 WHERE p.orden=(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual)
 AND p.establecida_en<=pg_catalog.statement_timestamp() AND p.establecida_en=c.publicada_en
 AND p.orden=c.secuencia
 AND pg_catalog.starts_with(p.acto_ref,'acto:ct:desarrollo:puntero-configuracion:')
 AND pg_catalog.starts_with(c.acto_ref,'acto:ct:desarrollo:configuracion:')
 AND pg_catalog.starts_with(r.acto_ref,'acto:ct:desarrollo:raiz-atestacion:')
 AND r.clave_id=$1 AND r.version=$2 AND r.clave_publica_spki=$3 AND r.huella_spki_sha256=$4
 AND r.valida_desde=$5 AND r.valida_hasta=$6 AND r.suite=$7 AND r.audiencia_despliegue=$8
 AND r.valida_desde<=pg_catalog.statement_timestamp() AND r.valida_hasta>pg_catalog.statement_timestamp()
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz rr WHERE (rr.raiz_clave_id,rr.raiz_version)=(r.clave_id,r.version))
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion rc WHERE rc.configuracion_revision IN (c.revision,$9))
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_raiz x WHERE x.configuracion_revision=c.revision)=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_raiz x WHERE x.configuracion_revision=$9)=1
 AND EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version a
 JOIN vec_autorizacion_atestada_v3.configuracion_raiz ar ON ar.configuracion_revision=a.revision
 WHERE a.revision=$9 AND a.secuencia=$10 AND a.huella_configuracion_sha256=$11
 AND a.publicada_en=$12 AND a.expira_en=$13 AND ar.raiz_clave_id=$1 AND ar.raiz_version=$2)`,
		anterior.claveID, anterior.claveVersion, anterior.spki, anterior.spkiHuella, anterior.validaDesde, anterior.validaHasta,
		confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, audienciaAtestacionContratacionTemporalDesarrollo,
		anterior.configuracionRef, anterior.configuracionOrden, anterior.configuracionHuella, anterior.publicadaEn, anterior.expiraEn,
	).Scan(&actual.configuracionRef, &secuencia, &actual.configuracionHuella, &actual.publicadaEn, &actual.expiraEn)
	if err != nil || secuencia < 1 || secuencia > maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo || uint64(secuencia) < anterior.configuracionOrden || secuencia < minima || raizMinima < 0 || uint64(raizMinima) > anterior.claveVersion || ahora.Before(actual.publicadaEn) || ahora.Before(anterior.validaDesde) || !ahora.Before(anterior.validaHasta) {
		return vacia, fallo
	}
	actual.publicadaEn = actual.publicadaEn.UTC()
	actual.expiraEn = actual.expiraEn.UTC()
	dia := time.Date(actual.publicadaEn.Year(), actual.publicadaEn.Month(), actual.publicadaEn.Day(), 0, 0, 0, 0, time.UTC)
	if !actual.publicadaEn.Equal(dia) || !actual.expiraEn.Equal(dia.Add(24*time.Hour)) {
		return vacia, fallo
	}
	actual.configuracionOrden = uint64(secuencia)
	actual.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(actual.configuracionRef, actual.configuracionOrden, actual.publicadaEn, actual.expiraEn, actual.raiz)
	if err != nil {
		return vacia, fallo
	}
	huella, err := actual.configuracion.HuellaSHA256ParaGobierno()
	if err != nil || huella != actual.configuracionHuella {
		return vacia, fallo
	}
	return actual, nil
}

func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) instantaneaConfianza(ctx context.Context) (*confianza.ServicioConfianzaAtestacionAutorizacionV3, error) {
	if p == nil || p.confianza == nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if p.fuenteConfianza != nil {
		return p.fuenteConfianza.instantanea(ctx)
	}
	// Proveedores construidos aisladamente conservan su configuración fija.
	return p.confianza, nil
}
func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) Verificar(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, d core.DecisionAutorizacionLigadaV3, m core.ReferenciaEntradaCatalogo, c core.ResultadoContextoActorRegistradoV2, a vp.AtestacionAutorizacionV3) (confianza.PruebaConfianzaAtestacionAutorizacionV3, error) {
	snapshot, err := p.instantaneaConfianza(ctx)
	if err != nil {
		return confianza.PruebaConfianzaAtestacionAutorizacionV3{}, err
	}
	return snapshot.Verificar(ctx, s, d, m, c, a)
}

// El núcleo exige un verificador concreto: se compone su cadena por operación,
// con una única instantánea y los mismos PDP, firmante y emisor nominales.
type emisorMaterialRenovableCTDesarrollo struct {
	mu        sync.Mutex
	autoridad vp.AutorizadorSolicitudLigadaV3
	proveedor *proveedorMaterialAltaContratacionTemporalDesarrollo
}

func nuevoEmisorMaterialRenovableCTDesarrollo(a vp.AutorizadorSolicitudLigadaV3, p *proveedorMaterialAltaContratacionTemporalDesarrollo) (*emisorMaterialRenovableCTDesarrollo, error) {
	if p == nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(a, p.atestador, p.confianza, p.emisor); err != nil {
		return nil, err
	}
	return &emisorMaterialRenovableCTDesarrollo{autoridad: a, proveedor: p}, nil
}
func (e *emisorMaterialRenovableCTDesarrollo) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, c core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	fallo := errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	if e == nil || ctx == nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, fallo
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	snapshot, err := e.proveedor.instantaneaConfianza(ctx)
	if err != nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, err
	}
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(e.autoridad, e.proveedor.atestador, snapshot, e.proveedor.emisor)
	if err != nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, err
	}
	return emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
}
